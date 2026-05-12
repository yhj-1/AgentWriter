package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yupi/ai-passage-creator/internal/common"
	"github.com/yupi/ai-passage-creator/internal/model"
	"github.com/yupi/ai-passage-creator/internal/service"
)

// PaymentHandler 支付控制器
type PaymentHandler struct {
	paymentSvc *service.PaymentService
}

// NewPaymentHandler 创建支付控制器
func NewPaymentHandler(paymentSvc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentSvc: paymentSvc}
}

// CreateVipSession 创建 VIP 支付会话
// POST /api/payment/create-vip-session
func (h *PaymentHandler) CreateVipSession(c *gin.Context) {
	// 获取登录用户
	user, exists := c.Get("loginUser")
	if !exists {
		c.JSON(http.StatusOK, common.Error(common.ErrNoAuth))
		return
	}
	loginUser := user.(*model.User)

	// 创建支付会话
	sessionURL, err := h.paymentSvc.CreateVipPaymentSession(loginUser.ID)
	if err != nil {
		appErr, ok := err.(*common.AppError)
		if ok {
			c.JSON(http.StatusOK, common.Error(appErr))
		} else {
			log.Printf("创建支付会话失败: %v", err)
			c.JSON(http.StatusOK, common.Error(common.ErrSystem))
		}
		return
	}

	c.JSON(http.StatusOK, common.Success(sessionURL))
}

// Refund 申请退款
// POST /api/payment/refund
func (h *PaymentHandler) Refund(c *gin.Context) {
	// 获取登录用户
	user, exists := c.Get("loginUser")
	if !exists {
		c.JSON(http.StatusOK, common.Error(common.ErrNoAuth))
		return
	}
	loginUser := user.(*model.User)

	// 解析请求参数
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// reason 可选，解析失败不影响
	}

	// 处理退款
	success, err := h.paymentSvc.HandleRefund(loginUser.ID, req.Reason)
	if err != nil {
		appErr, ok := err.(*common.AppError)
		if ok {
			c.JSON(http.StatusOK, common.Error(appErr))
		} else {
			log.Printf("退款失败: %v", err)
			c.JSON(http.StatusOK, common.Error(common.ErrSystem))
		}
		return
	}

	c.JSON(http.StatusOK, common.Success(success))
}

// GetRecords 获取当前用户的支付记录
// GET /api/payment/records
func (h *PaymentHandler) GetRecords(c *gin.Context) {
	// 获取登录用户
	user, exists := c.Get("loginUser")
	if !exists {
		c.JSON(http.StatusOK, common.Error(common.ErrNoAuth))
		return
	}
	loginUser := user.(*model.User)

	// 查询支付记录
	records, err := h.paymentSvc.GetPaymentRecords(loginUser.ID)
	if err != nil {
		log.Printf("查询支付记录失败: %v", err)
		c.JSON(http.StatusOK, common.Error(common.ErrSystem))
		return
	}

	c.JSON(http.StatusOK, common.Success(records))
}
