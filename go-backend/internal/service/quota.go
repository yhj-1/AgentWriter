package service

import (
	"log"

	"github.com/yupi/ai-passage-creator/internal/common"
	"github.com/yupi/ai-passage-creator/internal/model"
	"github.com/yupi/ai-passage-creator/internal/store"
)

// QuotaService 配额服务
type QuotaService struct {
	userStore *store.UserStore
}

// NewQuotaService 创建配额服务
func NewQuotaService(userStore *store.UserStore) *QuotaService {
	return &QuotaService{userStore: userStore}
}

// HasQuota 检查用户是否有足够的配额
func (s *QuotaService) HasQuota(user *model.User) bool {
	// 管理员和 VIP 用户无限配额
	if s.isAdmin(user) || s.isVIP(user) {
		return true
	}

	// 从数据库查询最新配额，避免使用缓存的旧数据
	freshUser, err := s.userStore.GetByID(user.ID)
	if err != nil || freshUser == nil {
		return false
	}

	return freshUser.Quota > 0
}

// ConsumeQuota 消耗配额（扣减1次）
func (s *QuotaService) ConsumeQuota(user *model.User) {
	// 管理员和 VIP 用户不消耗配额
	if s.isAdmin(user) || s.isVIP(user) {
		return
	}

	// 使用原子更新：UPDATE user SET quota = quota - 1 WHERE id = ? AND quota > 0
	affectedRows, err := s.userStore.DecrementQuota(user.ID)

	if err == nil && affectedRows > 0 {
		log.Printf("用户配额已消耗, userId=%d", user.ID)
	} else {
		log.Printf("用户配额扣减失败（可能配额不足或并发冲突）, userId=%d", user.ID)
	}
}

// CheckAndConsumeQuota 检查并消耗配额（原子操作）
// 如果配额不足会返回错误
func (s *QuotaService) CheckAndConsumeQuota(user *model.User) error {
	// 管理员和 VIP 用户跳过检查
	if s.isAdmin(user) || s.isVIP(user) {
		return nil
	}

	// 使用原子更新：检查与消费合并为一个原子操作
	affectedRows, err := s.userStore.DecrementQuota(user.ID)
	if err != nil {
		return common.ErrSystem
	}

	if affectedRows == 0 {
		// 影响行数为0，说明配额不足（已被其他请求消耗）
		return common.ErrOperation.WithMessage("配额不足，无法创建文章")
	}

	log.Printf("用户配额检查并消耗成功, userId=%d", user.ID)
	return nil
}

// isAdmin 判断是否为管理员
func (s *QuotaService) isAdmin(user *model.User) bool {
	return user.UserRole == common.AdminRole
}

// isVIP 判断是否为 VIP
func (s *QuotaService) isVIP(user *model.User) bool {
	return user.UserRole == common.VIPRole
}
