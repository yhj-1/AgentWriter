package common

// PaymentStatus 支付状态
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "PENDING"   // 待支付
	PaymentStatusSucceeded PaymentStatus = "SUCCEEDED" // 支付成功
	PaymentStatusFailed    PaymentStatus = "FAILED"    // 支付失败
	PaymentStatusRefunded  PaymentStatus = "REFUNDED"  // 已退款
)

// String 返回状态字符串
func (s PaymentStatus) String() string {
	return string(s)
}

// ProductType 产品类型
type ProductType string

const (
	ProductTypeVipPermanent ProductType = "VIP_PERMANENT" // 永久会员
)

// String 返回产品类型字符串
func (p ProductType) String() string {
	return string(p)
}

// GetProductPrice 获取产品价格（美元）
func GetProductPrice(productType ProductType) float64 {
	switch productType {
	case ProductTypeVipPermanent:
		return 199.00
	default:
		return 0
	}
}

// GetProductDescription 获取产品描述
func GetProductDescription(productType ProductType) string {
	switch productType {
	case ProductTypeVipPermanent:
		return "永久会员"
	default:
		return ""
	}
}

// Currency 相关常量
const (
	CurrencyUSD     = "usd"
	CentsMultiplier = 100 // 美元转美分的倍数
)
