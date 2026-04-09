package model

import "gorm.io/gorm"

// Announcement 系统公告表
type Announcement struct {
	gorm.Model
	Title   string `json:"title" gorm:"not null"`
	Content string `json:"content" gorm:"type:text"`
	Status  int    `json:"status" gorm:"default:0"` // 0=草稿, 1=已发布
}
