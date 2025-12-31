package model

import (
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
	"time"
)

// User 用户表
type User struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"uniqueIndex;size:50"`
	Password string `gorm:"not null"`  // 明文密码
	Role     int    `gorm:"default:0"` // 0=普通用户, 1=管理员
	// === 新增字段 ===
	Nickname  string `gorm:"default:'新用户'"`                          // 显示名，可改
	Avatar    string `gorm:"default:'/uploads/avatars/default.png'"` // 头像路径，可改
	CreatedAt time.Time
}

// Knowledge 知识库表
type Knowledge struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Content string          `gorm:"type:text;not null"`
	Vector  pgvector.Vector `gorm:"type:vector(2048)"` // 确保维度对应你的模型
	Source  string          `gorm:"type:varchar(255)"`

	UserID   uint `gorm:"index"`         // 数据归属人
	IsPublic bool `gorm:"default:false"` // 是否公开
}

// Conversation 会话（左侧列表显示的项）
type Conversation struct {
	gorm.Model
	UserID uint   `json:"user_id"`
	Title  string `json:"title"` // 第一次对话的问题作为标题
}

// ChatHistory 具体的问答记录（右侧显示的每一条气泡）
type ChatHistory struct {
	gorm.Model
	ConversationID uint   `json:"conversation_id" gorm:"index"` // 关键：关联到哪个会话
	UserID         uint   `json:"user_id"`
	Question       string `json:"question"`
	Answer         string `json:"answer"`
}

// TableName 指定表名
func (Knowledge) TableName() string {
	return "knowledge_base"
}
