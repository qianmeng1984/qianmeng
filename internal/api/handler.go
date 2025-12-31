package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	svc     *service.KnowledgeService
	authSvc *service.AuthService
}

// NewHandler 构造函数
func NewHandler(svc *service.KnowledgeService, authSvc *service.AuthService) *Handler {
	return &Handler{
		svc:     svc,
		authSvc: authSvc,
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

// Chat 对话接口实现
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// 从 Context 获取 userID (由中间件注入)
	userID := r.Context().Value("user_id").(uint)

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求参数", http.StatusBadRequest)
		return
	}

	answer, convID, err := h.svc.AskWithRAG(req.Question, userID, req.ConversationID)

	w.Header().Set("Content-Type", "application/json")
	response := ChatResponse{
		Answer:         answer,
		ConversationID: convID, // 返回会话ID
	}
	if err != nil {
		response.Error = err.Error()
	}
	json.NewEncoder(w).Encode(response)
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
