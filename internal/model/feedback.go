package model

import "gorm.io/gorm"

// Feedback 用户反馈表
type Feedback struct {
	gorm.Model
	UserID uint `json:"user_id" gorm:"index"` // 谁反馈的
	User   User `json:"user" gorm:"foreignKey:UserID"`

	ChatHistoryID uint        `json:"chat_history_id" gorm:"index"` // 针对哪句话反馈的
	ChatHistory   ChatHistory `json:"chat_history" gorm:"foreignKey:ChatHistoryID"`

	Type       int    `json:"type"`        // 1=点赞(Like), 2=点踩(Dislike)
	Reason     string `json:"reason"`      // 差评原因 (点赞时通常为空)
	AdminReply string `json:"admin_reply"` // 管理员回复内容
	Status     int    `json:"status"`      // 0=待处理, 1=已处理
}
