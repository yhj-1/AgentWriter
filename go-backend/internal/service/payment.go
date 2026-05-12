package service

import (
	"fmt"
	"log"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/refund"
	"github.com/stripe/stripe-go/v81/webhook"
	"github.com/yupi/ai-passage-creator/internal/common"
	"github.com/yupi/ai-passage-creator/internal/config"
	"github.com/yupi/ai-passage-creator/internal/model"
	"github.com/yupi/ai-passage-creator/internal/store"
	"gorm.io/gorm"
)

// PaymentService 支付服务
type PaymentService struct {
	config       *config.StripeConfig
	userStore    *store.UserStore
	paymentStore *store.PaymentStore
	db           *gorm.DB
}

// NewPaymentService 创建支付服务
func NewPaymentService(cfg *config.StripeConfig, userStore *store.UserStore, paymentStore *store.PaymentStore, db *gorm.DB) *PaymentService {
	// 初始化 Stripe API Key
	stripe.Key = cfg.APIKey

	return &PaymentService{
		config:       cfg,
		userStore:    userStore,
		paymentStore: paymentStore,
		db:           db,
	}
}

// CreateVipPaymentSession 创建 VIP 永久会员支付会话
func (s *PaymentService) CreateVipPaymentSession(userID int64) (string, error) {
	// 获取用户
	user, err := s.getUserOrError(userID)
	if err != nil {
		return "", err
	}

	// 验证用户不是 VIP
	if err := s.validateNotVip(user); err != nil {
		return "", err
	}

	// 创建 Stripe Session
	productType := common.ProductTypeVipPermanent
	sess, err := s.createStripeSession(userID, productType)
	if err != nil {
		log.Printf("创建 Stripe Session 失败: %v", err)
		return "", common.ErrSystem.WithMessage("创建支付会话失败")
	}

	// 保存支付记录
	if err := s.savePaymentRecord(userID, sess, productType); err != nil {
		log.Printf("保存支付记录失败: %v", err)
		return "", common.ErrSystem.WithMessage("保存支付记录失败")
	}

	log.Printf("创建支付会话成功, userId=%d, sessionId=%s", userID, sess.ID)
	return sess.URL, nil
}

// HandlePaymentSuccess 处理支付成功回调
func (s *PaymentService) HandlePaymentSuccess(sess *stripe.CheckoutSession) error {
	sessionID := sess.ID
	paymentIntentID := sess.PaymentIntent.ID

	// 查询支付记录
	record, err := s.paymentStore.GetBySessionID(sessionID)
	if err != nil {
		log.Printf("查询支付记录失败: %v", err)
		return err
	}
	if record == nil {
		log.Printf("支付记录不存在, sessionId=%s", sessionID)
		return fmt.Errorf("支付记录不存在")
	}

	// 幂等性检查
	if record.Status == string(common.PaymentStatusSucceeded) {
		log.Printf("支付记录已处理, sessionId=%s", sessionID)
		return nil
	}

	// 使用事务确保一致性
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 更新支付状态
		if err := s.paymentStore.UpdateStatus(record.ID, string(common.PaymentStatusSucceeded), paymentIntentID); err != nil {
			return err
		}

		// 升级用户为 VIP
		userID := record.UserID
		if err := s.userStore.UpgradeToVIP(userID); err != nil {
			return err
		}

		log.Printf("支付成功，用户已升级为 VIP, userId=%d, sessionId=%s", userID, sessionID)
		return nil
	})
}

// HandleRefund 处理退款
func (s *PaymentService) HandleRefund(userID int64, reason string) (bool, error) {
	// 获取用户
	user, err := s.getUserOrError(userID)
	if err != nil {
		return false, err
	}

	// 验证用户是 VIP
	if err := s.validateIsVip(user); err != nil {
		return false, err
	}

	// 查询最近的成功支付记录
	paymentRecord, err := s.paymentStore.GetLatestSuccessful(userID, string(common.ProductTypeVipPermanent))
	if err != nil {
		return false, common.ErrSystem
	}
	if paymentRecord == nil {
		return false, common.ErrNotFound.WithMessage("未找到支付记录")
	}
	if paymentRecord.StripePaymentIntentID == nil {
		return false, common.ErrOperation.WithMessage("支付记录无效")
	}

	// 调用 Stripe 退款
	refundObj, err := s.createStripeRefund(*paymentRecord.StripePaymentIntentID)
	if err != nil {
		log.Printf("创建退款失败: %v", err)
		return false, common.ErrSystem.WithMessage("创建退款失败")
	}

	if refundObj.Status != "succeeded" {
		return false, common.ErrOperation.WithMessage("退款失败")
	}

	// 使用事务确保一致性
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 更新退款记录
		if err := s.paymentStore.UpdateRefund(paymentRecord.ID, reason); err != nil {
			return err
		}

		// 撤销用户 VIP 身份
		if err := s.userStore.RevokeVIP(userID, common.DefaultQuota); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return false, common.ErrSystem
	}

	log.Printf("退款成功，已取消 VIP 身份, userId=%d, refundId=%s", userID, refundObj.ID)
	return true, nil
}

// ConstructEvent 验证 Webhook 签名并构造事件
func (s *PaymentService) ConstructEvent(payload []byte, signature string) (stripe.Event, error) {
	// 使用 ConstructEventWithOptions 忽略 API 版本不匹配警告
	// Stripe Webhook 的 API 版本可能与 SDK 版本不完全一致，这是正常现象
	return webhook.ConstructEventWithOptions(
		payload,
		signature,
		s.config.WebhookSecret,
		webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		},
	)
}

// GetPaymentRecords 获取用户的支付记录
func (s *PaymentService) GetPaymentRecords(userID int64) ([]*model.PaymentRecord, error) {
	return s.paymentStore.ListByUser(userID)
}

// ==================== 私有方法 ====================

// getUserOrError 获取用户或返回错误
func (s *PaymentService) getUserOrError(userID int64) (*model.User, error) {
	user, err := s.userStore.GetByID(userID)
	if err != nil {
		return nil, common.ErrNotFound.WithMessage("用户不存在")
	}
	return user, nil
}

// validateNotVip 验证用户不是 VIP
func (s *PaymentService) validateNotVip(user *model.User) error {
	if user.UserRole == common.VIPRole {
		return common.ErrOperation.WithMessage("您已经是永久会员")
	}
	return nil
}

// validateIsVip 验证用户是 VIP
func (s *PaymentService) validateIsVip(user *model.User) error {
	if user.UserRole != common.VIPRole {
		return common.ErrOperation.WithMessage("您不是会员，无法退款")
	}
	return nil
}

// createStripeSession 创建 Stripe 支付会话
func (s *PaymentService) createStripeSession(userID int64, productType common.ProductType) (*stripe.CheckoutSession, error) {
	price := common.GetProductPrice(productType)
	amountInCents := int64(price * common.CentsMultiplier)
	description := common.GetProductDescription(productType)

	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(s.config.SuccessURL),
		CancelURL:  stripe.String(s.config.CancelURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String(common.CurrencyUSD),
					UnitAmount: stripe.Int64(amountInCents),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(description),
						Description: stripe.String("解锁全部高级功能，无限创作配额，终身有效"),
					},
				},
				Quantity: stripe.Int64(1),
			},
		},
	}
	params.AddMetadata("userId", fmt.Sprintf("%d", userID))
	params.AddMetadata("productType", string(productType))

	return session.New(params)
}

// savePaymentRecord 保存支付记录
func (s *PaymentService) savePaymentRecord(userID int64, sess *stripe.CheckoutSession, productType common.ProductType) error {
	price := common.GetProductPrice(productType)
	desc := common.GetProductDescription(productType)

	record := &model.PaymentRecord{
		UserID:          userID,
		StripeSessionID: sess.ID,
		Amount:          price,
		Currency:        common.CurrencyUSD,
		Status:          string(common.PaymentStatusPending),
		ProductType:     string(productType),
		Description:     &desc,
	}

	return s.paymentStore.Create(record)
}

// createStripeRefund 创建 Stripe 退款
func (s *PaymentService) createStripeRefund(paymentIntentID string) (*stripe.Refund, error) {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(paymentIntentID),
		Reason:        stripe.String(string(stripe.RefundReasonRequestedByCustomer)),
	}
	return refund.New(params)
}
