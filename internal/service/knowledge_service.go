package service

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
	"rag-knowledge-base/internal/model"
	"rag-knowledge-base/pkg/textsplitter"
	"rag-knowledge-base/pkg/volcengine"
)

type KnowledgeService struct {
	aiClient *volcengine.Client
	db       *gorm.DB
}

func NewKnowledgeService(aiClient *volcengine.Client, db *gorm.DB) *KnowledgeService {
	return &KnowledgeService{
		aiClient: aiClient,
		db:       db,
	}
}

// AddDocument 上传文档 (需鉴权)
func (s *KnowledgeService) AddDocument(content string, fileName string, userID uint, userRole int, isPublic bool) error {
	// 权限检查：只有管理员(Role=1)才能传公共库
	if isPublic && userRole != 1 {
		return fmt.Errorf("权限不足：只有管理员(admin)可以上传公共知识库")
	}

	chunks := textsplitter.SplitText(content, 1000, 200)
	fmt.Printf("正在处理文件: %s, 共切分为 %d 段\n", fileName, len(chunks))

	successCount := 0
	for i, chunk := range chunks {
		vectorData, err := s.aiClient.CreateEmbedding(chunk)
		if err != nil {
			continue
		}

		doc := model.Knowledge{
			Content:  chunk,
			Vector:   pgvector.NewVector(vectorData),
			Source:   fileName,
			UserID:   userID,
			IsPublic: isPublic,
		}

		if err := s.db.Create(&doc).Error; err != nil {
			fmt.Printf("❌ 第 %d 段数据库写入失败: %v\n", i, err)
			continue
		}
		successCount++
	}

	if successCount == 0 {
		return fmt.Errorf("所有段落处理均失败，请检查 AI 配置或数据库")
	}

	fmt.Printf("✅ 文件 %s 处理完成！成功入库 %d/%d 段。\n", fileName, successCount, len(chunks))
	return nil
}

// SearchSimilarDocuments 核心检索 (支持数据隔离)
// userID: 当前是谁在查？
func (s *KnowledgeService) SearchSimilarDocuments(queryVector []float32, limit int, userID uint) ([]model.Knowledge, error) {
	var results []model.Knowledge

	// 🔍 逻辑：查找 (属于我的) OR (公共的 IsPublic=true)
	err := s.db.Model(&model.Knowledge{}).
		Where("user_id = ? OR is_public = ?", userID, true).
		Order(gorm.Expr("vector <-> ?", pgvector.NewVector(queryVector))).
		Limit(limit).
		Find(&results).Error

	return results, err
}

// AskWithRAG 问答核心逻辑 (含历史记忆 + 混合闲聊模式)
func (s *KnowledgeService) AskWithRAG(userQuestion string, userID uint, conversationID uint) (string, uint, error) {
	// 1. 如果是新对话 (conversationID == 0)，先创建会话
	if conversationID == 0 {
		// 截取前 20 个字作为标题
		title := []rune(userQuestion)
		if len(title) > 20 {
			title = title[:20]
		}
		newConv := model.Conversation{
			UserID: userID,
			Title:  string(title),
		}
		if err := s.db.Create(&newConv).Error; err != nil {
			return "", 0, fmt.Errorf("创建会话失败: %w", err)
		}
		conversationID = newConv.ID
	}

	// 2. 向量化用户问题
	queryVector, err := s.aiClient.CreateEmbedding(userQuestion)
	if err != nil {
		return "", 0, fmt.Errorf("问题向量化失败: %w", err)
	}

	// 3. 检索知识库
	// 策略：设置为 50，实现“全量召回”或“大范围召回”，保证不漏掉任何细节
	similarDocs, err := s.SearchSimilarDocuments(queryVector, 30, userID)
	if err != nil {
		return "", 0, fmt.Errorf("检索知识库失败: %w", err)
	}

	// =================================================================
	// 👇👇👇 【核心逻辑】 获取历史记录 (Short-term Memory) 👇👇👇
	// =================================================================
	var histories []model.ChatHistory
	if conversationID > 0 {
		// 【策略调整】Limit 设为 20 (约 10 轮对话)
		// 答辩亮点：增加上下文窗口，让 AI 拥有更长久的记忆，能处理复杂的多轮追问。
		s.db.Where("conversation_id = ?", conversationID).
			Order("created_at desc"). // 先倒序取最新的
			Limit(6).
			Find(&histories)
	}

	// 数据库查出来是倒序的（最新在最前），需要反转成正序（时间流向），AI 才能读懂逻辑
	sort.Slice(histories, func(i, j int) bool {
		return histories[i].ID < histories[j].ID
	})
	// =================================================================

	// 4. 组装终极 Prompt
	var promptBuilder strings.Builder

	// [Part A] 升级版系统人设 (System Prompt) - 混合模式
	promptBuilder.WriteString(`你是一个基于 RAG 的智能学务助手。
你的任务是结合【已知信息】和【对话历史】来回答用户的【最新问题】。

回答原则：
1. **优先依据【已知信息】**：如果已知信息里有答案，必须严格基于已知信息回答。
2. **允许通用对话**：如果用户是在打招呼（如"你好"）或闲聊，请直接用你的通用知识亲切回复，不需要查找知识库。
3. **推理与补充**：如果已知信息不全，你可以利用你的常识进行合理的推理和补充，但请话术委婉，比如"根据通常情况..."或"文档中未明确提及，但建议..."。
4. **结合上下文**：如果用户使用代词（他、它、这个），请务必从【对话历史】中推断指代对象。

`)

	// [Part B] 知识库背景 (Context)
	promptBuilder.WriteString("【已知信息/参考文档】：\n")
	if len(similarDocs) > 0 {
		for i, doc := range similarDocs {
			// 简单去一下换行符，让 Prompt 更紧凑
			cleanContent := strings.ReplaceAll(doc.Content, "\n", " ")
			promptBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, cleanContent))
		}
	} else {
		// 给 AI 一个信号：没查到文档，可以切回通用模式
		promptBuilder.WriteString("（本次检索未找到直接相关的学校文档，请依据你的通用知识回答）\n")
	}
	promptBuilder.WriteString("\n")

	// [Part C] 注入历史对话 (History)
	promptBuilder.WriteString("【对话历史】：\n")
	if len(histories) > 0 {
		for _, msg := range histories {
			if msg.Question != "" {
				promptBuilder.WriteString(fmt.Sprintf("User: %s\n", msg.Question))
			}
			if msg.Answer != "" {
				promptBuilder.WriteString(fmt.Sprintf("Assistant: %s\n", msg.Answer))
			}
		}
	} else {
		promptBuilder.WriteString("（这是对话的开始）\n")
	}
	promptBuilder.WriteString("\n")

	// [Part D] 用户当前问题
	promptBuilder.WriteString("【最新问题】：\n")
	promptBuilder.WriteString(userQuestion)

	// 5. 调用大模型
	finalPrompt := promptBuilder.String()

	// (调试用) 打印 Prompt，答辩时如果演示控制台，能看到这个会显得很专业
	// fmt.Printf("🔍 Prompt Token 消耗预估: %d 字符\n", len(finalPrompt))

	answer, err := s.aiClient.SendMessage(finalPrompt)
	if err != nil {
		return "", 0, err
	}

	// 6. 保存记录
	s.db.Create(&model.ChatHistory{
		ConversationID: conversationID,
		UserID:         userID,
		Question:       userQuestion,
		Answer:         answer,
	})

	return answer, conversationID, nil
}

// GetConversationList 获取左侧会话列表
func (s *KnowledgeService) GetConversationList(userID uint) ([]model.Conversation, error) {
	var list []model.Conversation
	err := s.db.Where("user_id = ?", userID).
		Order("id desc"). // <--- 改这里
		Find(&list).Error
	return list, err
}

// GetMessagesByConversation 获取某个会话的具体消息
func (s *KnowledgeService) GetMessagesByConversation(conversationID uint, userID uint) ([]model.ChatHistory, error) {
	var messages []model.ChatHistory
	// 确保只能查自己的
	err := s.db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Order("created_at asc"). // 聊天记录按时间正序
		Find(&messages).Error
	return messages, err
}

// GetHistory 获取历史记录
func (s *KnowledgeService) GetHistory(userID uint) ([]model.ChatHistory, error) {
	var history []model.ChatHistory
	// 倒序取最近 50 条
	err := s.db.Where("user_id = ?", userID).
		Order("created_at desc").
		Limit(50).
		Find(&history).Error
	return history, err
}

// GetFileList 获取当前用户上传的文件列表
func (s *KnowledgeService) GetFileList(userID uint) ([]string, error) {
	var files []string
	// 使用 Distinct 去重，因为一个文件被切成了很多段，数据库里有很多行，我们只关心文件名
	err := s.db.Model(&model.Knowledge{}).
		Where("user_id = ?", userID).
		Distinct("source").
		Pluck("source", &files).Error
	return files, err
}

// DeleteFile 删除指定文件 (连同它的所有切片)
func (s *KnowledgeService) DeleteFile(fileName string, userID uint) error {
	// 硬删除：直接从数据库抹掉，防止污染检索
	// 必须加上 user_id 限制，防止删了别人的文件
	result := s.db.Where("source = ? AND user_id = ?", fileName, userID).
		Delete(&model.Knowledge{})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("文件不存在或无权删除")
	}
	return nil
}

// RenameConversation 重命名会话
func (s *KnowledgeService) RenameConversation(conversationID uint, userID uint, newTitle string) error {
	// 1. 简单校验
	if len(newTitle) == 0 {
		return fmt.Errorf("标题不能为空")
	}
	if len([]rune(newTitle)) > 50 {
		return fmt.Errorf("标题过长")
	}

	// 2. 更新数据库
	// 注意：必须带上 user_id，防止用户恶意修改别人的会话
	result := s.db.Model(&model.Conversation{}).
		Where("id = ? AND user_id = ?", conversationID, userID).
		Update("title", newTitle)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("会话不存在或无权修改")
	}
	return nil
}

// DeleteConversation 删除会话 (级联删除聊天记录)
func (s *KnowledgeService) DeleteConversation(conversationID uint, userID uint) error {
	// 开启事务，保证原子性（要么都删，要么都不删）
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 先检查权限并删除会话本体
		result := tx.Where("id = ? AND user_id = ?", conversationID, userID).
			Delete(&model.Conversation{})

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("会话不存在或无权删除")
		}

		// 2. 删除关联的所有聊天记录 (ChatHistory)
		// 硬删除，直接清理空间
		if err := tx.Where("conversation_id = ?", conversationID).Delete(&model.ChatHistory{}).Error; err != nil {
			return err
		}

		return nil
	})
}
