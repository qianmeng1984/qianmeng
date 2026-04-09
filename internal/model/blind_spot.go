package model

import "gorm.io/gorm"

// BlindSpot 知识库盲区表 (Bad Case 监控)
type BlindSpot struct {
	gorm.Model
	ConversationID uint   `json:"conversation_id"` // 用于关联上下文
	Question       string `json:"question" gorm:"type:text"`
	Status         int    `json:"status" gorm:"default:0"` // 0=待补充, 1=已解决
}
