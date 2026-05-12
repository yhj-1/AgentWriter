package handler

import (
	"encoding/json"
	"io"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v81"
	"github.com/yupi/ai-passage-creator/internal/service"
)

// WebhookHandler Webhook 控制器
type WebhookHandler struct {
	paymentSvc *service.PaymentService
}

// NewWebhookHandler 创建 Webhook 控制器
func NewWebhookHandler(paymentSvc *service.PaymentService) *WebhookHandler {
	return &WebhookHandler{paymentSvc: paymentSvc}
}

// HandleStripeWebhook 处理 Stripe Webhook 回调
// POST /api/webhook/stripe
func (h *WebhookHandler) HandleStripeWebhook(c *gin.Context) {
	// 读取请求体
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("读取 Webhook 请求体失败: %v", err)
		c.String(500, "error")
		return
	}

	// 获取签名头
	signature := c.GetHeader("Stripe-Signature")

	// 验证签名
	event, err := h.paymentSvc.ConstructEvent(payload, signature)
	if err != nil {
		log.Printf("验证 Webhook 签名失败: %v", err)
		c.String(400, "error")
		return
	}

	log.Printf("收到 Stripe Webhook 事件, type=%s", event.Type)

	// 处理事件
	switch event.Type {
	case "checkout.session.completed":
		// 支付成功
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			log.Printf("解析 Session 对象失败: %v", err)
			c.String(400, "error")
			return
		}
		if err := h.paymentSvc.HandlePaymentSuccess(&sess); err != nil {
			log.Printf("处理支付成功回调失败: %v", err)
			c.String(500, "error")
			return
		}

	case "checkout.session.async_payment_succeeded":
		// 异步支付成功
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			log.Printf("解析 Session 对象失败: %v", err)
			c.String(400, "error")
			return
		}
		if err := h.paymentSvc.HandlePaymentSuccess(&sess); err != nil {
			log.Printf("处理异步支付成功回调失败: %v", err)
			c.String(500, "error")
			return
		}

	default:
		log.Printf("未处理的事件类型: %s", event.Type)
	}

	c.String(200, "success")
}
