package store

import (
	"time"

	"github.com/yupi/ai-passage-creator/internal/model"
	"gorm.io/gorm"
)

// PaymentStore 支付记录数据访问层
type PaymentStore struct {
	db *gorm.DB
}

// NewPaymentStore 创建支付记录存储
func NewPaymentStore(db *gorm.DB) *PaymentStore {
	return &PaymentStore{db: db}
}

// Create 创建支付记录
func (s *PaymentStore) Create(record *model.PaymentRecord) error {
	return s.db.Create(record).Error
}

// GetBySessionID 根据 Stripe Session ID 查询支付记录
func (s *PaymentStore) GetBySessionID(sessionID string) (*model.PaymentRecord, error) {
	var record model.PaymentRecord
	err := s.db.Where("stripeSessionId = ?", sessionID).First(&record).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetLatestSuccessful 查询用户最近的成功支付记录
func (s *PaymentStore) GetLatestSuccessful(userID int64, productType string) (*model.PaymentRecord, error) {
	var record model.PaymentRecord
	err := s.db.Where("userId = ? AND status = ? AND productType = ?",
		userID, "SUCCEEDED", productType).
		Order("createTime DESC").
		First(&record).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// UpdateStatus 更新支付状态
func (s *PaymentStore) UpdateStatus(id int64, status string, paymentIntentID string) error {
	updates := map[string]interface{}{
		"status":                status,
		"stripePaymentIntentId": paymentIntentID,
	}
	return s.db.Model(&model.PaymentRecord{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateRefund 更新退款信息
func (s *PaymentStore) UpdateRefund(id int64, reason string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":       "REFUNDED",
		"refundTime":   now,
		"refundReason": reason,
	}
	return s.db.Model(&model.PaymentRecord{}).Where("id = ?", id).Updates(updates).Error
}

// ListByUser 查询用户的所有支付记录
func (s *PaymentStore) ListByUser(userID int64) ([]*model.PaymentRecord, error) {
	var records []*model.PaymentRecord
	err := s.db.Where("userId = ?", userID).
		Order("createTime DESC").
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}
