package model

import "gorm.io/gorm"

// PromptTemplate 快捷指令库表
type PromptTemplate struct {
	gorm.Model
	Title      string `json:"title" gorm:"not null"`
	Content    string `json:"content" gorm:"type:text;not null"`
	IsActive   int    `json:"is_active" gorm:"default:1"`   // 1=启用 0=停用
	SortWeight int    `json:"sort_weight" gorm:"default:0"` // 排序权重，越大越靠前
}
