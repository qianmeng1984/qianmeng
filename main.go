package main

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"net/http"
	"rag-knowledge-base/internal/model"
	"rag-knowledge-base/internal/router"
	"rag-knowledge-base/internal/service"
	"rag-knowledge-base/pkg/volcengine"
)

func main() {
	fmt.Println("正在初始化系统...")

	cfg := LoadConfig()

	// 1. 连接数据库
	fmt.Println("正在连接数据库...")
	dsn := "host=localhost user=rune password=123456 dbname=rag_db port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("❌ 连接数据库失败: " + err.Error())
	}

	// 2. 自动建表 (增加了 User 和 ChatHistory)
	err = db.AutoMigrate(&model.Knowledge{}, &model.User{}, &model.ChatHistory{}, &model.Conversation{}, &model.Feedback{}, &model.SensitiveWord{}, &model.Announcement{}, &model.PromptTemplate{}, &model.BlindSpot{})
	if err != nil {
		panic("❌ 自动建表失败: " + err.Error())
	}
	fmt.Println("✅ 数据库连接成功，表结构已同步！")

	// 3. 初始化 AI 客户端
	fmt.Println("正在连接火山引擎...")
	aiClient := volcengine.NewClient(
		cfg.VolcEngine.APIKey,
		cfg.VolcEngine.BaseURL,
		cfg.VolcEngine.ChatModelID,
		cfg.VolcEngine.EmbeddingModelID,
	)

	// 4. 初始化 Services
	svc := service.NewKnowledgeService(aiClient, db)
	authSvc := service.NewAuthService(db)         // 👈 新增这一行！
	feedbackSvc := service.NewFeedbackService(db) // 👈 新增这行

	// 5. 初始化 Router (把 authSvc 也传进去)
	r := router.NewRouter(svc, authSvc, feedbackSvc)

	// 6. 启动服务
	port := ":8080"
	fmt.Printf("🚀 服务启动成功！请访问 http://localhost%s\n", port)
	if err := http.ListenAndServe(port, r); err != nil {
		fmt.Printf("服务异常退出: %v\n", err)
	}
}
