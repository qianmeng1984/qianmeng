package router

import (
	"context"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"rag-knowledge-base/internal/api" // 引入刚才创建的 handler
	"rag-knowledge-base/internal/service"
	"strings"
)

func NewRouter(svc *service.KnowledgeService, authSvc *service.AuthService) http.Handler {
	// 1. 初始化 Handler
	handler := api.NewHandler(svc, authSvc)

	mux := http.NewServeMux()

	// 2. 静态文件服务 (前端页面)
	fileServer := http.FileServer(http.Dir("web"))
	mux.Handle("/", fileServer)

	// 3. 【核心修复】图片资源服务
	// 这一段之前漏掉了！加上它，后端才知道去 pkg/uploads 找图片
	imgServer := http.StripPrefix("/uploads/", http.FileServer(http.Dir("pkg/uploads")))
	mux.Handle("/uploads/", imgServer)

	// 4. 注册 API 路由
	prefix := "/api/v1"

	// --- 公开接口 ---
	mux.HandleFunc(prefix+"/register", handler.Register)
	mux.HandleFunc(prefix+"/login", handler.Login)

	// --- 需要鉴权的接口 (包裹 AuthMiddleware) ---
	mux.HandleFunc(prefix+"/chat", AuthMiddleware(handler.Chat))
	mux.HandleFunc(prefix+"/history", AuthMiddleware(handler.History))
	mux.HandleFunc(prefix+"/upload", AuthMiddleware(handler.Upload))

	// 个人中心相关
	mux.HandleFunc(prefix+"/user/info", AuthMiddleware(handler.GetUserInfo))      // 获取信息
	mux.HandleFunc(prefix+"/user/update", AuthMiddleware(handler.UpdateUserInfo)) // 修改信息
	mux.HandleFunc(prefix+"/upload/avatar", AuthMiddleware(handler.UploadAvatar)) // 上传头像

	// 【新增】获取会话详情
	mux.HandleFunc(prefix+"/history/messages", AuthMiddleware(handler.HistoryDetail))

	// 【新增】列表和删除
	mux.HandleFunc(prefix+"/knowledge/list", AuthMiddleware(handler.GetKnowledgeList))
	mux.HandleFunc(prefix+"/knowledge/delete", AuthMiddleware(handler.DeleteKnowledge))

	// 管理员接口
	mux.HandleFunc(prefix+"/admin/users", AuthMiddleware(handler.AdminGetUsers))
	mux.HandleFunc(prefix+"/admin/user/delete", AuthMiddleware(handler.AdminDeleteUser))
	mux.HandleFunc(prefix+"/admin/user/update", AuthMiddleware(handler.AdminUpdateUser))

	// 👇👇👇 【新增】会话管理接口 👇👇👇
	mux.HandleFunc(prefix+"/history/rename", AuthMiddleware(handler.RenameConversation))
	mux.HandleFunc(prefix+"/history/delete", AuthMiddleware(handler.DeleteConversation))
	// 5. 挂载 CORS 中间件
	return CORSMiddleware(mux)
}

// ================= 中间件 (保持不变) =================

// CORSMiddleware 处理跨域请求
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			return
		}

		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware 鉴权中间件
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "未登录", http.StatusUnauthorized)
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Token格式错误", http.StatusUnauthorized)
			return
		}

		tokenStr := parts[1]
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return service.SecretKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "无效的Token", http.StatusUnauthorized)
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			userID := uint(claims["user_id"].(float64))
			role := int(claims["role"].(float64))

			ctx := context.WithValue(r.Context(), "user_id", userID)
			ctx = context.WithValue(ctx, "role", role)

			next(w, r.WithContext(ctx))
		} else {
			http.Error(w, "Token解析失败", http.StatusUnauthorized)
		}
	}
}
