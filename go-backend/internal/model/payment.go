package model

import "time"

// PaymentRecord 支付记录实体
type PaymentRecord struct {
	ID                    int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID                int64      `gorm:"column:userId;index:idx_userId" json:"userId"`
	StripeSessionID       string     `gorm:"column:stripeSessionId;size:128;index:idx_stripeSessionId" json:"stripeSessionId"`
	StripePaymentIntentID *string    `gorm:"column:stripePaymentIntentId;size:128" json:"stripePaymentIntentId"`
	Amount                float64    `gorm:"column:amount;type:decimal(10,2)" json:"amount"`
	Currency              string     `gorm:"column:currency;size:8;default:usd" json:"currency"`
	Status                string     `gorm:"column:status;size:32;index:idx_status" json:"status"`
	ProductType           string     `gorm:"column:productType;size:32" json:"productType"`
	Description           *string    `gorm:"column:description;size:256" json:"description"`
	RefundTime            *time.Time `gorm:"column:refundTime" json:"refundTime"`
	RefundReason          *string    `gorm:"column:refundReason;size:512" json:"refundReason"`
	CreateTime            time.Time  `gorm:"column:createTime;autoCreateTime;index:idx_createTime" json:"createTime"`
	UpdateTime            time.Time  `gorm:"column:updateTime;autoUpdateTime" json:"updateTime"`
}

// TableName 指定表名
func (PaymentRecord) TableName() string {
	return "payment_record"
}
