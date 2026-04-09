package service

import (
	"encoding/json"
	"fmt"
	"github.com/pgvector/pgvector-go"
	"sort"
	"strings"

	"gorm.io/gorm"
	"rag-knowledge-base/internal/model"
	"rag-knowledge-base/pkg/textsplitter"
	"rag-knowledge-base/pkg/volcengine"
)

type KnowledgeService struct {
	aiClient *volcengine.Client
	db       *gorm.DB
}

func (s *KnowledgeService) GetDB() *gorm.DB {
	return s.db
}

func NewKnowledgeService(aiClient *volcengine.Client, db *gorm.DB) *KnowledgeService {
	return &KnowledgeService{
		aiClient: aiClient,
		db:       db,
	}
}

// =================================================================================
// 🧠 查询分析 (Query Analysis) —— 三合一超级大脑 (升级版)
// =================================================================================

// QueryAnalysisResult 定义 AI 返回的 JSON 结构 (新增 VectorQuery)
type QueryAnalysisResult struct {
	RewrittenQuery string   `json:"rewritten_query"` // 完整重写 (用于展示/上下文)
	VectorQuery    string   `json:"vector_query"`    // 向量专用 (去噪/去校名)
	Keywords       []string `json:"keywords"`        // 关键词 (用于数据库 LIKE)
	Intent         string   `json:"intent"`          // 🆕 新增：chat=闲聊/通用, query=知识库查询
}

// AnalyzeQuery 升级为：意图判断 + 重写 + 向量净化 + 关键词爆炸
// 返回值: (rewrittenQuery, vectorQuery, keywords, intent, cost)
func (s *KnowledgeService) AnalyzeQuery(userQuestion string, histories []model.ChatHistory) (string, string, []string, string, int) {
	// 1. 构建历史上下文
	var historyBuilder strings.Builder
	for _, h := range histories {
		historyBuilder.WriteString(fmt.Sprintf("User: %s\nAI: %s\n", h.Question, h.Answer))
	}

	// 2. 构造四合一 Prompt (🚀 升级版：保留原有去噪逻辑，增加意图识别)
	prompt := fmt.Sprintf(`你是一个河南科技学院（HIST）智能助手的后台**查询分析专家**。
请根据【对话历史】和【用户问题】，同时完成以下四个任务，并**严格以 JSON 格式输出**。

**任务一：意图分类 (Intent)**
判断用户意图，返回 "chat" 或 "query"。
- **chat (闲聊/通用)**：用户在进行问候、夸奖("你真棒")、辱骂、情感发泄、询问天气/常识、或者任何**与学校具体事务无关**的对话。
- **query (知识库查询)**：用户在询问关于河南科技学院的政策、历史、老师、课程、地点等具体信息。

**任务二：查询重写 (RewrittenQuery) [核心修正]**
1. 补全主语，生成一个通顺、完整的句子。
2. **必须保留用户的核心指令**：如果用户说“总结”、“概括”、“简述”、“列举”，**请务必保留这些动词**，不要把“总结新年贺词”改成“新年贺词是什么”。
   - 错误示范：用户="总结下校训" -> 重写="河南科技学院校训是什么" (❌ 意图丢失)
   - 正确示范：用户="总结下校训" -> 重写="请总结河南科技学院的校训内涵" (✅ 意图保留)

**任务三：向量检索优化 (VectorQuery) [核心去噪]**
1. **去噪原则**：剔除"河南科技学院"、"学校"等高频词。
2. **去功能词**：剔除"总结"、"查询"、"是什么"、"介绍一下"、"帮我找"等表示意图的动词/虚词。只保留核心实体名词！
   例："河南科技学院的新年贺词总结" -> "新年贺词" (去掉'总结'，防止向量被'总结'二字带偏)
   例："心理中心的电话是多少" -> "心理咨询中心电话" (去掉'是多少')
   例："介绍一下百农207" -> "百农207"
3. **特殊情况**：如果 任务一 判断为 "chat"，此字段请直接返回空字符串 ""。

**任务四：关键词爆炸 (Keywords)**
1. **拆解规则**：
   - **机构名泛化**："心理咨询中心" -> ["心理咨询中心", "心理中心", "咨询中心"]
   - **修饰语剥离**："新年贺词" -> ["新年贺词", "贺词"]
   - **特定型号拆解**："百农207小麦" -> ["百农207", "207"]
2. **去除通用词**：不要包含"河南科技学院"、"学校"、"查询"、"总结"等。
3. **特殊情况**：如果 任务一 判断为 "chat"，请直接返回 ["通用闲聊"]。

**输出格式要求：**
必须是合法的 JSON 格式，不要包含 Markdown 标记。
示例 1 (Query):
{
  "intent": "query",
  "rewritten_query": "河南科技学院心理咨询中心开放时间",
  "vector_query": "心理咨询中心开放时间",
  "keywords": ["心理咨询中心", "心理中心", "咨询中心"]
}
示例 2 (Chat):
{
  "intent": "chat",
  "rewritten_query": "你真棒",
  "vector_query": "",
  "keywords": ["通用闲聊"]
}

对话历史：
%s

当前问题：%s
JSON输出：`, historyBuilder.String(), userQuestion)

	// 3. 调用 AI
	resp, usage, err := s.aiClient.SendMessage(prompt)
	if err != nil {
		fmt.Printf("⚠️ 查询分析失败: %v，降级处理\n", err)
		// 降级时默认当做查询处理
		return userQuestion, userQuestion, []string{userQuestion}, "query", 0
	}

	// 4. 清洗与解析 JSON
	cleanResp := strings.TrimSpace(resp)
	cleanResp = strings.ReplaceAll(cleanResp, "```json", "")
	cleanResp = strings.ReplaceAll(cleanResp, "```", "")

	var result QueryAnalysisResult
	if err := json.Unmarshal([]byte(cleanResp), &result); err != nil {
		fmt.Printf("⚠️ JSON解析失败: %v | 原始内容: %s\n", err, cleanResp)
		// 兜底：简单分词，默认为查询
		return userQuestion, userQuestion, strings.Fields(userQuestion), "query", 0
	}

	// 兜底逻辑
	if result.Intent == "" {
		result.Intent = "query"
	}

	// 再次兜底清洗关键词 (仅在 Query 模式下执行，防止 AI 没遵守规则)
	var finalKeywords []string
	if result.Intent == "query" {
		for _, kw := range result.Keywords {
			if kw != "河南科技学院" && kw != "学校" && kw != "总结" && kw != "查询" {
				finalKeywords = append(finalKeywords, kw)
			}
		}
	} else {
		// 如果是 Chat 模式，直接信任 AI 返回的 ["通用闲聊"]，方便后续过滤
		finalKeywords = result.Keywords
	}

	// 👇 日志中增加意图显示
	fmt.Printf("🧠 [查询分析四合一] 消耗: %d Tokens | 意图: %s\n", usage.TotalTokens, result.Intent)
	fmt.Printf("   ├─ 📝 完整重写: %s\n", result.RewrittenQuery)
	fmt.Printf("   ├─ 🚀 向量专用: %s\n", result.VectorQuery)
	fmt.Printf("   └─ 💥 关键词爆炸: %v\n", finalKeywords)

	// 如果是 Query 模式且 AI 没生成 VectorQuery，兜底使用 RewrittenQuery
	if result.Intent == "query" && result.VectorQuery == "" {
		result.VectorQuery = result.RewrittenQuery
	}

	// 注意：返回值列表增加了 intent
	return result.RewrittenQuery, result.VectorQuery, finalKeywords, result.Intent, usage.TotalTokens
}

// =================================================================================
// 📂 文档管理 (AddDocument)
// =================================================================================

func (s *KnowledgeService) AddDocument(content string, fileName string, userID uint, userRole int, isPublic bool) error {
	if userRole != 1 {
		return fmt.Errorf("权限不足：仅管理员可维护知识库")
	}

	// 切片重叠量 150
	chunks := textsplitter.SplitText(content, 400, 150)
	fmt.Printf("📂 [上传] 文件: %s | 切片数: %d\n", fileName, len(chunks))

	totalTokens := 0
	successCount := 0

	for i, chunk := range chunks {
		// 注入文件名上下文
		enhancedChunk := fmt.Sprintf("【来源文档：%s】\n%s", fileName, chunk)
		vectorData, usage, err := s.aiClient.CreateEmbedding(enhancedChunk)
		if err != nil {
			continue
		}
		totalTokens += usage.TotalTokens

		doc := model.Knowledge{
			Content:  enhancedChunk,
			Vector:   pgvector.NewVector(vectorData),
			Source:   fileName,
			UserID:   userID,
			IsPublic: true,
		}
		if err := s.db.Create(&doc).Error; err == nil {
			successCount++
		}
		if (i+1)%5 == 0 {
			fmt.Printf("   ...已处理 %d/%d 段\n", i+1, len(chunks))
		}
	}

	if successCount == 0 {
		return fmt.Errorf("全部处理失败")
	}
	fmt.Printf("✅ 上传完成！入库: %d段 | 💰 总消耗: %d Tokens\n", successCount, totalTokens)
	return nil
}

// =================================================================================
// 🔍 混合检索 + 上下文扩展 (SearchSimilarDocuments & ExpandContext)
// =================================================================================

// ExpandContext 上下文扩展：核心算法
func (s *KnowledgeService) ExpandContext(hits []model.Knowledge) []model.Knowledge {
	if len(hits) == 0 {
		return nil
	}

	// 1. 收集所有需要查询的 ID (命中 ID + 前后邻居 ID)
	targetIDs := make(map[uint]bool)
	hitSourceMap := make(map[uint]string)

	for _, doc := range hits {
		hitSourceMap[doc.ID] = doc.Source
		targetIDs[doc.ID] = true
		if doc.ID > 1 {
			targetIDs[doc.ID-1] = true
		}
		targetIDs[doc.ID+1] = true
	}

	// 2. 将 map 转为 slice 进行数据库查询
	var ids []uint
	for id := range targetIDs {
		ids = append(ids, id)
	}

	var candidates []model.Knowledge
	s.db.Where("id IN ?", ids).Order("id asc").Find(&candidates)

	// 3. 智能过滤与去重
	var finalDocs []model.Knowledge
	processedIDs := make(map[uint]bool)

	for _, candidate := range candidates {
		keep := false
		// 校验是否同源
		for hitID, hitSource := range hitSourceMap {
			if candidate.Source == hitSource {
				if candidate.ID == hitID || candidate.ID == hitID-1 || candidate.ID == hitID+1 {
					keep = true
					break
				}
			}
		}
		if keep && !processedIDs[candidate.ID] {
			finalDocs = append(finalDocs, candidate)
			processedIDs[candidate.ID] = true
		}
	}

	return finalDocs
}

// SearchSimilarDocuments 混合检索 (融合版：修复分数逻辑 + 保留完整日志)
func (s *KnowledgeService) SearchSimilarDocuments(queryVector []float32, limit int, userID uint, keywords []string) ([]model.Knowledge, error) {
	// 1. 向量检索 (使用修复后的排序逻辑)
	var vectorResults []model.Knowledge
	// 先把向量对象创建好，避免写两次
	vec := pgvector.NewVector(queryVector)

	s.db.Model(&model.Knowledge{}).
		// Select: 把距离算出来赋值给 score，供后续计算匹配度
		Select("*, vector <-> ? as score", vec).
		Where("user_id = ? OR is_public = ?", userID, true).
		// Order: 直接用表达式排序，防止 ambiguous 报错
		Order(gorm.Expr("vector <-> ?", vec)).
		Limit(limit).
		Find(&vectorResults)

	// 2. 关键词检索 (保持原有逻辑)
	var keywordResults []model.Knowledge
	if len(keywords) > 0 {
		db := s.db.Model(&model.Knowledge{}).Where("user_id = ? OR is_public = ?", userID, true)
		keywordQuery := s.db
		for i, kw := range keywords {
			if len([]rune(kw)) < 2 {
				continue
			}
			if i == 0 {
				keywordQuery = s.db.Where("content LIKE ?", "%"+kw+"%")
			} else {
				keywordQuery = keywordQuery.Or("content LIKE ?", "%"+kw+"%")
			}
		}
		db.Where(keywordQuery).Limit(5).Find(&keywordResults)
	}

	// ================= 📝 日志打印区域 (保留你原有的日志) =================

	// 辅助函数：提取 ID 和 文件名
	getDocInfo := func(docs []model.Knowledge) string {
		var infos []string
		for _, d := range docs {
			// 格式: [ID:文件名]
			infos = append(infos, fmt.Sprintf("[%d:%s]", d.ID, d.Source))
		}
		return strings.Join(infos, ", ")
	}

	fmt.Println("\n🔎 [检索透视镜]")
	fmt.Printf("   🔹 向量检索 Top%d: %s\n", limit, getDocInfo(vectorResults))
	fmt.Printf("   🔸 关键词检索:     %s\n", getDocInfo(keywordResults))

	// ================= 📌 核心修复：建立分数映射表 =================
	// 目的：为了在 Expansion 后还能认出谁是向量搜出来的，谁是关键词搜出来的

	scoreMap := make(map[uint]float32)
	sourceTypeMap := make(map[uint]int) // 1=Vector, 2=Keyword

	// 记录向量分 (Score < 2: 真实距离)
	for _, doc := range vectorResults {
		scoreMap[doc.ID] = doc.Score
		sourceTypeMap[doc.ID] = 1
	}
	// 记录关键词命中 (Score = 10)
	for _, doc := range keywordResults {
		if _, exists := sourceTypeMap[doc.ID]; !exists {
			sourceTypeMap[doc.ID] = 2
		}
	}

	// 3. 初步合并去重 (保留原有逻辑)
	finalMap := make(map[uint]model.Knowledge)
	for _, doc := range keywordResults {
		finalMap[doc.ID] = doc
	}
	for _, doc := range vectorResults {
		finalMap[doc.ID] = doc
	}

	var initialResults []model.Knowledge
	for _, doc := range finalMap {
		initialResults = append(initialResults, doc)
	}

	// 打印详细日志 (保留)
	fmt.Printf("📚 [检索初步] 向量: %d | 关键词: %d | 合并去重: %d 段\n",
		len(vectorResults), len(keywordResults), len(initialResults))

	// 4. 执行上下文扩展
	expandedResults := s.ExpandContext(initialResults)

	// ================= 📌 核心修复：回填分数与身份 =================
	// 扩展后的结果 Score 都是 0，我们需要根据 ID 把刚才存的分数填回去
	for i := range expandedResults {
		id := expandedResults[i].ID

		if val, ok := scoreMap[id]; ok {
			// Case A: 是向量检索命中的，恢复它的真实距离 (0.x)
			expandedResults[i].Score = val
		} else if sourceType, ok := sourceTypeMap[id]; ok && sourceType == 2 {
			// Case B: 是关键词命中的，给个标记分 (10.0)
			expandedResults[i].Score = 10.0
		} else {
			// Case C: 是扩展出来的邻居 (原文Score是0)，给个标记分 (20.0)
			expandedResults[i].Score = 20.0
		}
	}

	// 打印最终日志 (保留)
	fmt.Printf("📚 [检索最终] 扩展后: %d 段 (已包含前后邻居)\n", len(expandedResults))
	return expandedResults, nil
}

// AskWithRAGStream 专为流式响应设计 (最终架构：AI意图分流 + 双模式响应)
func (s *KnowledgeService) AskWithRAGStream(userQuestion string, userID uint, conversationID uint, streamChan chan string) (uint, error) {
	// 结束后关闭通道，告诉 Handler 传输完毕
	defer close(streamChan)

	// =========================== 🚨 敏感词/安全合规拦截前置 ===========================
	var sensitiveWords []model.SensitiveWord
	s.db.Find(&sensitiveWords)

	for _, sw := range sensitiveWords {
		if strings.Contains(userQuestion, sw.Word) {
			if sw.Level == 2 {
				// Level 2: 危险词汇，直接阻断
				fmt.Printf("🛡️ [安全中心] 触发阻断级敏感词: %s\n", sw.Word)
				streamChan <- fmt.Sprintf("\n⛔ 【系统拦截】您的提问包含不当内容，请求已被安全策略阻断。请规范发言。")
				return conversationID, nil
			} else if sw.Level == 1 {
				// Level 1: 警告词汇，替换为 *** 后继续
				fmt.Printf("🛡️ [安全中心] 触发警告级敏感词，已自动打码: %s\n", sw.Word)
				userQuestion = strings.ReplaceAll(userQuestion, sw.Word, "***")
			}
		}
	}
	// =========================== 拦截结束 ===========================

	// 1. 准备历史记录
	var shortHistory []model.ChatHistory
	if conversationID > 0 {
		s.db.Where("conversation_id = ?", conversationID).Order("created_at desc").Limit(3).Find(&shortHistory)
		sort.Slice(shortHistory, func(i, j int) bool { return shortHistory[i].ID < shortHistory[j].ID })
	}

	// 2. 🧠 全局查询分析 (核心：由 AI 判断 Intent)
	// 不再使用本地规则拦截，完全依赖 AnalyzeQuery 的判断结果
	rewrittenQuery, vectorQuery, keywords, intent, analyzeCost := s.AnalyzeQuery(userQuestion, shortHistory)

	// 3. 创建会话 (如果需要)
	if conversationID == 0 {
		// 优先使用重写后的 Query 做标题，更准确
		title := rewrittenQuery
		if len([]rune(title)) > 15 {
			title = string([]rune(title)[:15])
		}
		newConv := model.Conversation{UserID: userID, Title: title}
		s.db.Create(&newConv)
		conversationID = newConv.ID
		streamChan <- fmt.Sprintf("CONF_ID:%d", conversationID)
	}

	// 定义最后要保存的变量
	var fullAnswer string
	var thinkingJsonStr string
	var totalTokens int

	// =========================== 🛣️ 分流逻辑 (基于 AI 意图) ===========================

	if intent == "chat" {
		// >>>>> 分支一：通用闲聊模式 (不走 RAG，省钱快跑) <<<<<
		fmt.Printf("🚀 [Router] AI 判定为闲聊 (Intent: chat): %s\n", userQuestion)

		// A. 发送“假”思维链 (为了前端展示不报错，且能看到是闲聊模式)
		thinkingData := map[string]interface{}{
			"rewritten_query": rewrittenQuery, // 原样显示或重写后的
			"vector_query":    "无 (通用对话模式)",
			"keywords":        []string{"通用闲聊"}, // 👈 配合黑名单，不统计热词
			"sources":         []string{},
			"analyze_cost":    analyzeCost,
		}
		thinkingBytes, _ := json.Marshal(thinkingData)
		thinkingJsonStr = string(thinkingBytes)
		streamChan <- "THINKING:" + thinkingJsonStr + "\n"

		// B. 构造闲聊 Prompt (保持你原有的自然风格配置)
		var chatPrompt strings.Builder
		chatPrompt.WriteString(`### Role
你现在的身份是河南科技学院的AI学务助手 "浅梦"。
设定：你是一位性格活泼、热情、像一位可爱的学姐。

### Rules
1. **自然对话**：这是闲聊模式，请不要机械地背诵人设。像朋友聊天一样自然。
2. **身份隐形**：除非用户明确问“你是谁”、“叫什么名字”，否则**不要**主动在每句话里强调自己是“浅梦”或“助手”。
   - 错误示范：你好，我是浅梦，河南科技学院的助手。请问有什么帮您？
   - 正确示范：嗨！同学你好呀，今天有什么新鲜事想聊聊吗？😊
3. **语气要求**：多用语气助词（比如“呀”、“呢”、“哦~”），禁止使用公文风或机器人语气。
4. **拒答原则**：如果问天气/实时新闻，请俏皮地回复你还在学校机房里，连不上外网，建议看手机。

### History
`)
		for _, msg := range shortHistory {
			chatPrompt.WriteString(fmt.Sprintf("User: %s\nAI: %s\n", msg.Question, msg.Answer))
		}
		chatPrompt.WriteString("\nUser: " + userQuestion + "\nAI:")

		// C. 调用 LLM
		var fullAnswerBuilder strings.Builder
		usage, err := s.aiClient.SendMessageStream(chatPrompt.String(), func(chunk string) {
			fullAnswerBuilder.WriteString(chunk)
			streamChan <- chunk
		})
		if err != nil {
			return 0, err
		}
		fullAnswer = fullAnswerBuilder.String()

		// 计算消耗 (分析消耗 + 闲聊消耗)
		totalTokens = analyzeCost + usage.TotalTokens

		// 打印账单
		fmt.Println("\n================= 🧾 本次请求真实账单 (闲聊模式) =================")
		fmt.Printf("1. 意图分析: %d Tokens\n", analyzeCost)
		fmt.Printf("2. 闲聊生成: %d Tokens\n", usage.TotalTokens)
		fmt.Printf("-------------------------------------------------------\n")
		fmt.Printf("💰 总消耗: %d Tokens (省去了 Embedding 和 检索)\n", totalTokens)
		fmt.Println("=======================================================\n")

	} else {
		// >>>>> 分支二：RAG 专家模式 (Intent: query) <<<<<
		fmt.Printf("🧠 [Router] AI 判定为查询 (Intent: query)，启动 RAG 引擎...\n")

		// 1. 向量化 (使用去噪后的 vectorQuery)
		var err error
		queryVector, embedUsage, err := s.aiClient.CreateEmbedding(vectorQuery)
		if err != nil {
			streamChan <- "ERR: 向量化服务异常"
			return 0, err
		}

		// 2. 混合检索 + 上下文扩展
		similarDocs, _ := s.SearchSimilarDocuments(queryVector, 3, userID, keywords)

		// 排序逻辑 (向量 > 关键词 > 扩展)
		sort.Slice(similarDocs, func(i, j int) bool {
			if similarDocs[i].Score == similarDocs[j].Score {
				return similarDocs[i].ID < similarDocs[j].ID
			}
			return similarDocs[i].Score < similarDocs[j].Score
		})

		// 👇👇👇 新增：知识盲区自动监控逻辑 (增强版) 👇👇👇
		isBlindSpot := true
		if len(similarDocs) > 0 {
			for _, doc := range similarDocs {
				if doc.Score >= 10.0 && doc.Score < 20.0 {
					// 情况 A: 命中了关键词检索 (Score=10)，说明文本有精确匹配，不是盲区
					isBlindSpot = false
					break
				} else if doc.Score < 10.0 {
					// 情况 B: 向量检索，计算它的实际余弦相似度
					distSq := doc.Score * doc.Score
					similarity := 1 - (distSq / 2)

					// ⚡ 核心阈值：如果检索到的切片中，有任何一个相似度大于 45%，说明懂点相关知识
					if similarity >= 0.45 {
						isBlindSpot = false
						break
					}
				}
			}
		}

		if isBlindSpot {
			blindSpot := model.BlindSpot{
				ConversationID: conversationID,
				Question:       userQuestion, // 保存学生最原始的提问
				Status:         0,            // 状态：待补充
			}
			s.db.Create(&blindSpot)
			fmt.Printf("🚨 [盲区监控] 向量最高匹配度不足或未命中，已自动捕获 Bad Case: %s\n", userQuestion)
		}
		// 👆👆👆 盲区逻辑结束 👆👆👆

		// 3. 准备思维链数据
		var sourceNames []string
		for _, doc := range similarDocs {
			cleanContent := strings.ReplaceAll(doc.Content, "\r", "")
			var sourceTag string

			if doc.Score >= 20.0 {
				sourceTag = "(上下文扩展)"
			} else if doc.Score >= 10.0 {
				sourceTag = "(关键词命中)"
			} else {
				// 欧氏距离 -> 余弦相似度转换
				distSq := doc.Score * doc.Score
				similarity := 1 - (distSq / 2)
				if similarity < 0 {
					similarity = 0
				}
				scorePercent := int(similarity * 100)
				sourceTag = fmt.Sprintf("(匹配度: %d%%)", scorePercent)
			}
			sourceNames = append(sourceNames, fmt.Sprintf("📄 %s %s | %s", doc.Source, sourceTag, cleanContent))
		}

		// 发送思维链
		totalTokens = analyzeCost + embedUsage.TotalTokens
		thinkingData := map[string]interface{}{
			"rewritten_query": rewrittenQuery,
			"vector_query":    vectorQuery,
			"keywords":        keywords,
			"sources":         sourceNames,
			"analyze_cost":    totalTokens,
		}
		thinkingBytes, _ := json.Marshal(thinkingData)
		thinkingJsonStr = string(thinkingBytes)
		streamChan <- "THINKING:" + thinkingJsonStr + "\n"

		// 4. 组装 RAG Prompt (严格严肃模式)
		var promptBuilder strings.Builder
		promptBuilder.WriteString(`### Role
你是由河南科技学院（HIST）开发的学生事务智能助手。你的名字叫 "浅梦"。
你性格热情、专业、像一位耐心的学长/学姐。

### Task
根据【已知信息】回答用户的【最新问题】,为用户提供详尽、准确的解答。

### Rules (至关重要)
1. **遵循指令意图**：
   - 如果用户要求**“总结”、“概括”**，请不要大段摘抄原文，需要适当的进行提炼和摘要。
   - 如果用户问**“有哪些”、“列举”**，请使用清晰的列表格式回答。
   - 如果用户问**“是什么”**，则可以详细引用原文。
2. **引用规范**：回答中尽量引用原文的关键句子，保持权威性。

3. **场景区分**：此时用户明确在询问学校事务，请认真回答。
   
4. **自我认知**：
   - 如果用户问"你是谁"，请回答："我是河南科技学院的AI学务助手，基于 RAG 技术构建。"

5. **诚实原则**：
如果【已知信息】完全没有提及答案，请直接说“抱歉，暂时没有查到相关信息”，不要瞎编。

6.**禁止套话**：直接开始回答，**严禁**使用“根据已知信息”、“文档显示”、“从文中可以看出”等机械表述。
   - ❌ 错误：根据文档，心理中心的电话是...
   - ✅ 正确：心理中心的电话是...

7.**细节/数据优先**：如果【已知信息】中包含具体的时间点、电话号码、人名名单、地址，**必须**完整列出，禁止只给概括性描述。
   - ❌ 错误：预约时间是周一到周日的上课时间。
   - ✅ 正确：预约时间是周一至周日。具体值班时段为：上午8:30-11:30，下午3:00-6:00（夏）/ 2:30-5:30（冬）。

### Context
【已知信息】：
`)
		// 动态拼接文档
		if len(similarDocs) > 0 {
			var lastDoc model.Knowledge
			index := 1
			for i, doc := range similarDocs {
				isContinuous := false
				if i > 0 && doc.ID == lastDoc.ID+1 && doc.Source == lastDoc.Source {
					isContinuous = true
				}
				cleanContent := strings.ReplaceAll(doc.Content, "\n", " ")
				if isContinuous {
					promptBuilder.WriteString(" " + cleanContent)
				} else {
					if i > 0 {
						promptBuilder.WriteString("\n")
					}
					promptBuilder.WriteString(fmt.Sprintf("%d. %s", index, cleanContent))
					index++
				}
				lastDoc = doc
			}
			promptBuilder.WriteString("\n")
		} else {
			promptBuilder.WriteString("（暂无相关文档）\n")
		}

		promptBuilder.WriteString("\n【对话历史】：\n")
		for _, msg := range shortHistory {
			promptBuilder.WriteString(fmt.Sprintf("User: %s\nAI: %s\n", msg.Question, msg.Answer))
		}
		promptBuilder.WriteString("\n\nUser: " + rewrittenQuery + "\nAssistant:")

		// 5. 调用 LLM
		var fullAnswerBuilder strings.Builder
		chatUsage, err := s.aiClient.SendMessageStream(promptBuilder.String(), func(chunk string) {
			fullAnswerBuilder.WriteString(chunk)
			streamChan <- chunk
		})
		if err != nil {
			streamChan <- "ERR: AI 服务异常"
			return 0, err
		}

		fullAnswer = fullAnswerBuilder.String()
		totalTokens += chatUsage.TotalTokens

		// 打印账单
		fmt.Println("\n================= 🧾 本次请求真实账单 (RAG模式) =================")
		fmt.Printf("1. 意图分析: %d Tokens\n", analyzeCost)
		fmt.Printf("2. 向量嵌入: %d Tokens\n", embedUsage.TotalTokens)
		fmt.Printf("3. 最终问答: %d Tokens\n", chatUsage.TotalTokens)
		fmt.Printf("-------------------------------------------------------\n")
		fmt.Printf("💰 总消耗: %d Tokens\n", totalTokens)
		fmt.Println("=======================================================\n")
	}

	// 🏁 最终保存 (两个分支汇聚于此)
	s.db.Create(&model.ChatHistory{
		ConversationID: conversationID,
		UserID:         userID,
		Question:       userQuestion,
		Answer:         fullAnswer,
		ThinkingLog:    thinkingJsonStr,
	})

	return conversationID, nil
}

// ... 辅助方法保持不变 ...

func (s *KnowledgeService) GetConversationList(userID uint) ([]model.Conversation, error) {
	var list []model.Conversation
	err := s.db.Where("user_id = ?", userID).Order("id desc").Find(&list).Error
	return list, err
}

func (s *KnowledgeService) GetMessagesByConversation(conversationID uint, userID uint) ([]model.ChatHistory, error) {
	var messages []model.ChatHistory
	err := s.db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).Order("created_at asc").Find(&messages).Error
	return messages, err
}

func (s *KnowledgeService) GetFileList(userID uint) ([]string, error) {
	var files []string
	err := s.db.Model(&model.Knowledge{}).Where("user_id = ?", userID).Distinct("source").Pluck("source", &files).Error
	return files, err
}

func (s *KnowledgeService) DeleteFile(fileName string, userID uint) error {
	result := s.db.Where("source = ? AND user_id = ?", fileName, userID).Delete(&model.Knowledge{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("文件不存在")
	}
	return nil
}

func (s *KnowledgeService) RenameConversation(conversationID uint, userID uint, newTitle string) error {
	result := s.db.Model(&model.Conversation{}).Where("id = ? AND user_id = ?", conversationID, userID).Update("title", newTitle)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("失败")
	}
	return nil
}

func (s *KnowledgeService) DeleteConversation(conversationID uint, userID uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND user_id = ?", conversationID, userID).Delete(&model.Conversation{}).Error; err != nil {
			return err
		}
		if err := tx.Where("conversation_id = ?", conversationID).Delete(&model.ChatHistory{}).Error; err != nil {
			return err
		}
		return nil
	})
}

type DashboardStats struct {
	TotalFiles  int64       `json:"total_files"`
	TotalChunks int64       `json:"total_chunks"`
	TypeDist    []NameValue `json:"type_dist"` // 饼图数据
	HotWords    []NameValue `json:"hot_words"` // 热词数据
}

type NameValue struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func (s *KnowledgeService) GetDashboardStats() (*DashboardStats, error) {
	stats := &DashboardStats{}

	// 1. 基础计数
	s.db.Model(&model.Knowledge{}).Count(&stats.TotalChunks)
	s.db.Model(&model.Knowledge{}).Distinct("source").Count(&stats.TotalFiles)

	// 2. 知识分布 (按源文件统计切片数)
	// 也就是看看哪个文件“知识含量”最高
	rows, err := s.db.Model(&model.Knowledge{}).Select("source, count(*) as total").Group("source").Rows()
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			var val int
			rows.Scan(&name, &val)
			// 简单的名称清洗，去掉后缀，让图表更好看
			cleanName := name
			if idx := strings.LastIndex(name, "."); idx != -1 {
				cleanName = name[:idx]
			}
			stats.TypeDist = append(stats.TypeDist, NameValue{Name: cleanName, Value: val})
		}
	}

	// 3. 热词挖掘 (从思维链日志中提取)
	var logs []string
	s.db.Model(&model.ChatHistory{}).
		Where("thinking_log IS NOT NULL").
		Where("thinking_log != ''").
		Pluck("thinking_log", &logs)

	wordFreq := make(map[string]int)

	// 🚫 定义黑名单：这些词不参与热词统计
	blacklist := map[string]bool{
		"通用闲聊":   true,
		"无需检索":   true,
		"通用对话模式": true,
		"无":      true,
	}

	for _, log := range logs {
		if log == "" {
			continue
		}

		var data struct {
			Keywords []string `json:"keywords"`
		}
		if err := json.Unmarshal([]byte(log), &data); err == nil {
			for _, kw := range data.Keywords {
				// ⚡️ 核心过滤逻辑
				// 1. 长度大于1
				// 2. 不在黑名单里
				if len([]rune(kw)) > 1 && !blacklist[kw] {
					wordFreq[kw]++
				}
			}
		}
	}

	// 转换为 Slice 并排序 (取 Top 20)
	for k, v := range wordFreq {
		stats.HotWords = append(stats.HotWords, NameValue{Name: k, Value: v})
	}
	// 简单的冒泡排序，把频率高的排前面
	sort.Slice(stats.HotWords, func(i, j int) bool {
		return stats.HotWords[i].Value > stats.HotWords[j].Value
	})
	if len(stats.HotWords) > 20 {
		stats.HotWords = stats.HotWords[:20]
	}

	return stats, nil
}

// GetMessagesForAdmin 管理员获取任意会话的上下文 (上帝视角)
func (s *KnowledgeService) GetMessagesForAdmin(conversationID uint) ([]model.ChatHistory, error) {
	var messages []model.ChatHistory
	// 去掉了 user_id = ? 的限制
	err := s.db.Where("conversation_id = ?", conversationID).
		Order("created_at asc").
		Find(&messages).Error
	return messages, err
}
