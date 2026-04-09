package model

import "gorm.io/gorm"

// SensitiveWord 敏感词库表
type SensitiveWord struct {
	gorm.Model
	Word  string `json:"word" gorm:"uniqueIndex;not null"`
	Level int    `json:"level" gorm:"default:2"` // 1=警告(替换为***), 2=直接阻断
}
