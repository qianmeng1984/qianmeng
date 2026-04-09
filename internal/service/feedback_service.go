package service

import (
	"gorm.io/gorm"
	"rag-knowledge-base/internal/model"
)

type FeedbackService struct {
	db *gorm.DB
}

func NewFeedbackService(db *gorm.DB) *FeedbackService {
	return &FeedbackService{db: db}
}

// CreateFeedback 用户提交或更新反馈
// 逻辑修改：如果已经存在反馈，则更新状态；如果不存在，则创建。
// fType: 1=赞, 2=踩, 0=取消反馈
func (s *FeedbackService) CreateFeedback(userID uint, chatID uint, fType int, reason string) error {
	var feedback model.Feedback

	// 1. 尝试查找该用户对这条消息是否已有反馈
	err := s.db.Where("user_id = ? AND chat_history_id = ?", userID, chatID).First(&feedback).Error

	if err == gorm.ErrRecordNotFound {
		// 2. 如果没有记录，且操作不是取消(0)，则创建新记录
		if fType == 0 {
			return nil // 本来就没有记录，又要取消，直接返回成功
		}
		newFeedback := model.Feedback{
			UserID:        userID,
			ChatHistoryID: chatID,
			Type:          fType,
			Reason:        reason,
			Status:        0,
		}
		return s.db.Create(&newFeedback).Error
	} else if err != nil {
		// 数据库查询出错
		return err
	}

	// 3. 如果已有记录，更新状态
	// 使用 map 进行更新，确保 fType=0 (零值) 也能被更新进去
	updates := map[string]interface{}{
		"type":   fType,
		"reason": reason,
		// "status": 0, // 可选：更新反馈后是否重置处理状态？视需求而定，这里暂时保留原状态或重置
	}

	return s.db.Model(&feedback).Updates(updates).Error
}

// GetMyFeedbacks 用户查看自己的反馈历史
func (s *FeedbackService) GetMyFeedbacks(userID uint) ([]model.Feedback, error) {
	var list []model.Feedback
	// Preload 加载关联的 ChatHistory，让用户知道是针对哪句话的反馈
	err := s.db.Where("user_id = ?", userID).
		Preload("ChatHistory").
		Order("created_at desc").
		Find(&list).Error
	return list, err
}

// GetAllFeedbacks 管理员查看所有反馈 (待处理的排前面)
func (s *FeedbackService) GetAllFeedbacks() ([]model.Feedback, error) {
	var list []model.Feedback
	// 加载 User(知道是谁) 和 ChatHistory(知道发生了什么)
	err := s.db.Preload("User").
		Preload("ChatHistory").
		Where("type = ?", 2).
		Order("status asc, created_at desc"). // status 0 (待处理) 排在前面
		Find(&list).Error
	return list, err
}

// ReplyFeedback 管理员回复并标记为已处理
func (s *FeedbackService) ReplyFeedback(id uint, reply string) error {
	return s.db.Model(&model.Feedback{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"admin_reply": reply,
			"status":      1, // 标记为已处理
		}).Error
}
