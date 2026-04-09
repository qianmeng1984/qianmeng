package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"rag-knowledge-base/internal/model"
	"rag-knowledge-base/internal/service"
	"rag-knowledge-base/pkg/fileparser"
	"strings"
	"time"
)

// 定义请求/响应结构体 (从 router.go 移过来)
type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// 1. 修改请求和响应结构体
type ChatRequest struct {
	Question       string `json:"question"`
	ConversationID uint   `json:"conversation_id"` // 新增：前端传 0 代表新对话
}

type ChatResponse struct {
	Answer         string `json:"answer"`
	ConversationID uint   `json:"conversation_id"` // 新增：后端告诉前端是哪个会话
	Error          string `json:"error,omitempty"`
}

// Handler 结构体，持有 Service 依赖
type Handler struct {
	svc         *service.KnowledgeService
	authSvc     *service.AuthService
	feedbackSvc *service.FeedbackService // 👈 新增这一行
}

// NewHandler 构造函数 (注意入参变了)
func NewHandler(svc *service.KnowledgeService, authSvc *service.AuthService, fbSvc *service.FeedbackService) *Handler {
	return &Handler{
		svc:         svc,
		authSvc:     authSvc,
		feedbackSvc: fbSvc, // 👈 注入
	}
}

// ================= 用户相关 =================

// Register 注册接口实现
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求参数", http.StatusBadRequest)
		return
	}

	if err := h.authSvc.Register(req.Username, req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Write([]byte("注册成功"))
}

// Login 登录接口实现
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求参数", http.StatusBadRequest)
		return
	}

	token, _, roleName, err := h.authSvc.Login(req.Username, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
		"role":  roleName,
	})
}

// ================= 业务相关 =================

// Chat 流式对话接口实现
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 0. 验证是否支持流式输出
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	userID := r.Context().Value("user_id").(uint)

	// 1. 解析请求
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求参数", http.StatusBadRequest)
		return
	}

	// 2. 设置流式响应头 (Standard SSE headers)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// 禁用 buffer 缓冲
	w.Header().Set("X-Accel-Buffering", "no")

	// 3. 创建通道，用于 Service 层回传数据
	streamChan := make(chan string)

	// 4. 启动协程运行耗时的 RAG 业务
	go func() {
		// 调用新写的 AskWithRAGStream
		h.svc.AskWithRAGStream(req.Question, userID, req.ConversationID, streamChan)
	}()

	// 5. 主线程循环读取通道，并实时 Flush 给前端
	for msg := range streamChan {
		// 直接写入内容
		fmt.Fprint(w, msg)
		// ⚡️ 关键：立即推送到客户端
		flusher.Flush()
	}

	// 循环结束代表 streamChan 被 close，请求自然结束
}

// History 历史记录接口实现
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := r.Context().Value("user_id").(uint)
	list, _ := h.svc.GetConversationList(userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// 4. 新增 HistoryDetail 方法 (获取某个会话的详情)
func (h *Handler) HistoryDetail(w http.ResponseWriter, r *http.Request) {
	// GET /api/v1/history/messages?conversation_id=123
	userID := r.Context().Value("user_id").(uint)

	// 简单的参数解析
	idStr := r.URL.Query().Get("conversation_id")
	// 这里为了简单，假设 idStr 是合法的数字，实际可以用 strconv.Atoi
	var convID uint
	fmt.Sscanf(idStr, "%d", &convID)

	messages, _ := h.svc.GetMessagesByConversation(convID, userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// Upload 上传接口实现
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := r.Context().Value("user_id").(uint)
	role := r.Context().Value("role").(int)
	isPublic := r.FormValue("public") == "true"

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "文件读取失败", http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentBytes, _ := io.ReadAll(file)
	content, _ := fileparser.ParseContent(contentBytes, header.Filename)

	err = h.svc.AddDocument(content, header.Filename, userID, role, isPublic)

	if err != nil {
		if strings.Contains(err.Error(), "权限不足") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), 500)
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success"}`))
	}
}

// ... (之前的 Handler 结构体、NewHandler、Register、Login、Chat、History、Upload 保持不变) ...

// ================= 个人中心相关 (新增) =================

// GetUserInfo 获取当前用户信息
func (h *Handler) GetUserInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// 从 Context 获取 userID
	userID := r.Context().Value("user_id").(uint)

	user, err := h.authSvc.GetUserInfo(userID)
	if err != nil {
		http.Error(w, "获取用户信息失败", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// UpdateUserInfo 更新用户信息 (昵称、密码、头像)
func (h *Handler) UpdateUserInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := r.Context().Value("user_id").(uint)

	// 定义前端传来的格式
	var req struct {
		Nickname string `json:"nickname"`
		Password string `json:"password"` // 新密码
		Avatar   string `json:"avatar"`   // 头像路径
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求参数", http.StatusBadRequest)
		return
	}

	err := h.authSvc.UpdateUserInfo(userID, req.Nickname, req.Avatar, req.Password)
	if err != nil {
		http.Error(w, "更新失败: "+err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"success"}`))
}

// UploadAvatar 专门上传头像
func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. 限制大小 (例如 2MB)
	r.ParseMultipartForm(2 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "请选择文件", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 2. 准备保存目录 pkg/uploads/avatars/
	saveDir := "pkg/uploads/avatars"
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		http.Error(w, "无法创建目录", 500)
		return
	}

	// 3. 生成文件名 (防止重名，加个时间戳)
	ext := filepath.Ext(header.Filename)
	newFileName := fmt.Sprintf("avatar_%d%s", time.Now().UnixNano(), ext)
	dstPath := filepath.Join(saveDir, newFileName)

	// 4. 保存文件
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "文件保存失败", 500)
		return
	}
	defer dst.Close()
	io.Copy(dst, file)

	// 5. 返回给前端的访问路径 (注意：要是 URL 路径，不是系统路径)
	// 我们在 Router 里会把 /uploads/ 映射到 pkg/uploads/
	webPath := "/uploads/avatars/" + newFileName

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": webPath})
}

// GetKnowledgeList 获取知识库文件列表
func (h *Handler) GetKnowledgeList(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uint)

	files, err := h.svc.GetFileList(userID)
	if err != nil {
		http.Error(w, "获取列表失败", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

// DeleteKnowledge 删除知识库文件
func (h *Handler) DeleteKnowledge(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	userID := r.Context().Value("user_id").(uint)

	// 获取参数：文件名
	var req struct {
		FileName string `json:"file_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "参数错误", 400)
		return
	}

	if err := h.svc.DeleteFile(req.FileName, userID); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"success"}`))
}

// AdminGetUsers 获取用户列表
func (h *Handler) AdminGetUsers(w http.ResponseWriter, r *http.Request) {
	role := r.Context().Value("role").(int)
	if role != 1 { // 🔐 核心安全卡点
		http.Error(w, "无权访问", http.StatusForbidden)
		return
	}

	users, _ := h.authSvc.GetAllUsers()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// AdminDeleteUser 删除用户
func (h *Handler) AdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	role := r.Context().Value("role").(int)
	if role != 1 {
		http.Error(w, "无权访问", http.StatusForbidden)
		return
	}

	var req struct {
		ID uint `json:"id"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.ID == 1 { // 防止删除超级管理员自己
		http.Error(w, "不能删除根管理员", http.StatusBadRequest)
		return
	}

	h.authSvc.DeleteUser(req.ID)
	w.Write([]byte(`{"status":"success"}`))
}

// AdminUpdateUser 修改用户
func (h *Handler) AdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	role := r.Context().Value("role").(int)
	if role != 1 {
		http.Error(w, "无权访问", http.StatusForbidden)
		return
	}

	var req struct {
		ID       uint   `json:"id"`
		Nickname string `json:"nickname"`
		Password string `json:"password"` // 留空则不修改
	}
	json.NewDecoder(r.Body).Decode(&req)

	h.authSvc.AdminUpdateUser(req.ID, req.Nickname, req.Password)
	w.Write([]byte(`{"status":"success"}`))
}

// RenameConversation 重命名会话
func (h *Handler) RenameConversation(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uint)
	var req struct {
		ID    uint   `json:"id"`
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "参数错误", http.StatusBadRequest)
		return
	}

	if err := h.svc.RenameConversation(req.ID, userID, req.Title); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(`{"status":"success"}`))
}

// DeleteConversation 删除会话
func (h *Handler) DeleteConversation(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uint)
	var req struct {
		ID uint `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "参数错误", http.StatusBadRequest)
		return
	}

	if err := h.svc.DeleteConversation(req.ID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(`{"status":"success"}`))
}

// AdminGetStats 获取仪表盘数据
func (h *Handler) AdminGetStats(w http.ResponseWriter, r *http.Request) {
	role := r.Context().Value("role").(int)
	if role != 1 {
		http.Error(w, "无权访问", http.StatusForbidden)
		return
	}

	stats, err := h.svc.GetDashboardStats()
	if err != nil {
		http.Error(w, "统计失败", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// ================= 反馈系统相关 (新增) =================

// SubmitFeedback 用户提交反馈
func (h *Handler) SubmitFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	userID := r.Context().Value("user_id").(uint)

	var req struct {
		ChatID int    `json:"chat_id"`
		Type   int    `json:"type"`   // 1=赞, 2=踩
		Reason string `json:"reason"` // 原因
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "参数错误", 400)
		return
	}

	if err := h.feedbackSvc.CreateFeedback(userID, uint(req.ChatID), req.Type, req.Reason); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte(`{"status":"success"}`))
}

// GetMyFeedbacks 用户获取自己的反馈列表
func (h *Handler) GetMyFeedbacks(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uint)
	list, err := h.feedbackSvc.GetMyFeedbacks(userID)
	if err != nil {
		http.Error(w, "获取失败", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// AdminGetFeedbacks 管理员获取所有反馈
func (h *Handler) AdminGetFeedbacks(w http.ResponseWriter, r *http.Request) {
	role := r.Context().Value("role").(int)
	if role != 1 {
		http.Error(w, "无权访问", 403)
		return
	}
	list, err := h.feedbackSvc.GetAllFeedbacks()
	if err != nil {
		http.Error(w, "获取失败", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// AdminReplyFeedback 管理员回复
func (h *Handler) AdminReplyFeedback(w http.ResponseWriter, r *http.Request) {
	role := r.Context().Value("role").(int)
	if role != 1 {
		http.Error(w, "无权访问", 403)
		return
	}
	var req struct {
		ID    int    `json:"id"`    // 反馈ID
		Reply string `json:"reply"` // 回复内容
	}
	json.NewDecoder(r.Body).Decode(&req)

	if err := h.feedbackSvc.ReplyFeedback(uint(req.ID), req.Reply); err != nil {
		http.Error(w, "回复失败", 500)
		return
	}
	w.Write([]byte(`{"status":"success"}`))
}

// internal/api/handler.go

// AdminGetContext 管理员获取会话上下文
func (h *Handler) AdminGetContext(w http.ResponseWriter, r *http.Request) {
	role := r.Context().Value("role").(int)
	if role != 1 {
		http.Error(w, "无权访问", 403)
		return
	}

	// 获取参数 conversation_id
	idStr := r.URL.Query().Get("conversation_id")
	var convID uint
	fmt.Sscanf(idStr, "%d", &convID)

	messages, err := h.svc.GetMessagesForAdmin(convID)
	if err != nil {
		http.Error(w, "获取失败", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// AdminGetSensitiveWords 获取敏感词列表
func (h *Handler) AdminGetSensitiveWords(w http.ResponseWriter, r *http.Request) {
	role := r.Context().Value("role").(int)
	if role != 1 {
		http.Error(w, "无权访问", 403)
		return
	}
	var list []model.SensitiveWord
	h.svc.GetDB().Order("id desc").Find(&list) // 需在 service 层暴露 db，或直接 h.authSvc.GetDB()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// AdminAddSensitiveWord 新增敏感词
func (h *Handler) AdminAddSensitiveWord(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value("role").(int) != 1 {
		http.Error(w, "无权", 403)
		return
	}
	var req model.SensitiveWord
	json.NewDecoder(r.Body).Decode(&req)

	// 这里你需要暴露出 GORM 实例，或者在 service 层写这几行。为简便直接操作 db
	// 如果由于私有变量无法访问 db，请参考下一步修改 service 暴露 DB
	if err := h.svc.GetDB().Create(&req).Error; err != nil {
		http.Error(w, "词汇已存在或添加失败", 500)
		return
	}
	w.Write([]byte(`{"status":"success"}`))
}

// AdminUpdateSensitiveWord 修改敏感词
func (h *Handler) AdminUpdateSensitiveWord(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value("role").(int) != 1 {
		http.Error(w, "无权", 403)
		return
	}
	var req model.SensitiveWord
	json.NewDecoder(r.Body).Decode(&req)
	h.svc.GetDB().Model(&model.SensitiveWord{}).Where("id = ?", req.ID).Updates(map[string]interface{}{
		"word":  req.Word,
		"level": req.Level,
	})
	w.Write([]byte(`{"status":"success"}`))
}

// AdminDeleteSensitiveWord 删除敏感词
func (h *Handler) AdminDeleteSensitiveWord(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value("role").(int) != 1 {
		http.Error(w, "无权", 403)
		return
	}
	var req struct {
		ID uint `json:"id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	h.svc.GetDB().Delete(&model.SensitiveWord{}, req.ID)
	w.Write([]byte(`{"status":"success"}`))
}

// ================= 系统公告管理 (新增) =================

// AdminGetAnnouncements 获取公告列表 (后台)
func (h *Handler) AdminGetAnnouncements(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value("role").(int) != 1 {
		http.Error(w, "无权访问", 403)
		return
	}
	var list []model.Announcement
	h.svc.GetDB().Order("id desc").Find(&list)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// AdminAddAnnouncement 新增公告
func (h *Handler) AdminAddAnnouncement(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value("role").(int) != 1 {
		http.Error(w, "无权", 403)
		return
	}
	var req model.Announcement
	json.NewDecoder(r.Body).Decode(&req)
	h.svc.GetDB().Create(&req)
	w.Write([]byte(`{"status":"success"}`))
}

// AdminUpdateAnnouncement 修改公告
func (h *Handler) AdminUpdateAnnouncement(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value("role").(int) != 1 {
		http.Error(w, "无权", 403)
		return
	}
	var req model.Announcement
	json.NewDecoder(r.Body).Decode(&req)
	h.svc.GetDB().Model(&model.Announcement{}).Where("id = ?", req.ID).Updates(map[string]interface{}{
		"title":   req.Title,
		"content": req.Content,
		"status":  req.Status,
	})
	w.Write([]byte(`{"status":"success"}`))
}

// AdminDeleteAnnouncement 删除公告
func (h *Handler) AdminDeleteAnnouncement(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value("role").(int) != 1 {
		http.Error(w, "无权", 403)
		return
	}
	var req struct {
		ID uint `json:"id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	h.svc.GetDB().Delete(&model.Announcement{}, req.ID)
	w.Write([]byte(`{"status":"success"}`))
}

// GetLatestAnnouncement 获取最新的一条已发布公告 (前台用，所有登录用户可看)
func (h *Handler) GetLatestAnnouncement(w http.ResponseWriter, r *http.Request) {
	var ann model.Announcement
	// 查找 status=1 的最新一条
	if err := h.svc.GetDB().Where("status = ?", 1).Order("updated_at desc").First(&ann).Error; err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`)) // 没有公告就返回空 JSON
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ann)
}

// ================= 快捷指令管理 (新增) =================
func (h *Handler) AdminGetPrompts(w http.ResponseWriter, r *http.Request) {
	var list []model.PromptTemplate
	h.svc.GetDB().Order("sort_weight desc, id desc").Find(&list)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (h *Handler) AdminAddPrompt(w http.ResponseWriter, r *http.Request) {
	var req model.PromptTemplate
	json.NewDecoder(r.Body).Decode(&req)
	h.svc.GetDB().Create(&req)
	w.Write([]byte(`{"status":"success"}`))
}

func (h *Handler) AdminUpdatePrompt(w http.ResponseWriter, r *http.Request) {
	var req model.PromptTemplate
	json.NewDecoder(r.Body).Decode(&req)
	h.svc.GetDB().Model(&model.PromptTemplate{}).Where("id = ?", req.ID).Updates(map[string]interface{}{
		"title":       req.Title,
		"content":     req.Content,
		"is_active":   req.IsActive,
		"sort_weight": req.SortWeight,
	})
	w.Write([]byte(`{"status":"success"}`))
}

func (h *Handler) AdminDeletePrompt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID uint `json:"id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	h.svc.GetDB().Delete(&model.PromptTemplate{}, req.ID)
	w.Write([]byte(`{"status":"success"}`))
}

// ================= 知识盲区监控管理 (新增) =================
func (h *Handler) AdminGetBlindSpots(w http.ResponseWriter, r *http.Request) {
	var list []model.BlindSpot
	// 优先显示未解决的 (status=0)，按时间倒序
	h.svc.GetDB().Order("status asc, id desc").Find(&list)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (h *Handler) AdminResolveBlindSpot(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID uint `json:"id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	h.svc.GetDB().Model(&model.BlindSpot{}).Where("id = ?", req.ID).Update("status", 1)
	w.Write([]byte(`{"status":"success"}`))
}

func (h *Handler) AdminDeleteBlindSpot(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID uint `json:"id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	h.svc.GetDB().Delete(&model.BlindSpot{}, req.ID)
	w.Write([]byte(`{"status":"success"}`))
}

// GetActivePrompts 获取已启用的快捷指令 (前台用)
func (h *Handler) GetActivePrompts(w http.ResponseWriter, r *http.Request) {
	var list []model.PromptTemplate
	// 只查已启用的，并按权重排序
	h.svc.GetDB().Where("is_active = ?", 1).Order("sort_weight desc, id desc").Find(&list)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}
