package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yupi/ai-passage-creator/internal/common"
	"github.com/yupi/ai-passage-creator/internal/config"
	"github.com/yupi/ai-passage-creator/internal/model"
	"github.com/yupi/ai-passage-creator/internal/store"
	"gorm.io/gorm"
)

// ArticleService 文章服务
type ArticleService struct {
	store        *store.ArticleStore
	agentSvc     *ArticleAgentService
	orchestrator AgentOrchestrator // 新增：多智能体编排器
	cfg          *config.Config    // 新增：配置对象
	quotaSvc     *QuotaService
	sseManager   *common.SSEManager
}

// AgentOrchestrator 智能体编排器接口（用于依赖注入）
type AgentOrchestrator interface {
	ExecutePhase1(ctx context.Context, state *model.ArticleState) error
	ExecutePhase2(ctx context.Context, state *model.ArticleState, streamHandler func(string)) error
	ExecutePhase3(ctx context.Context, state *model.ArticleState, streamHandler func(string)) error
}

// NewArticleService 创建文章服务
func NewArticleService(
	st *store.ArticleStore,
	agentSvc *ArticleAgentService,
	orchestrator AgentOrchestrator,
	cfg *config.Config,
	quotaSvc *QuotaService,
	sseManager *common.SSEManager,
) *ArticleService {
	return &ArticleService{
		store:        st,
		agentSvc:     agentSvc,
		orchestrator: orchestrator,
		cfg:          cfg,
		quotaSvc:     quotaSvc,
		sseManager:   sseManager,
	}
}

// Create 创建文章任务
func (s *ArticleService) Create(user *model.User, req *model.CreateArticleRequest) (string, error) {
	if req.Topic == "" {
		return "", common.ErrParams.WithMessage("选题不能为空")
	}

	// 处理配图方式（如果用户未选择，给普通用户设置默认的非 VIP 方式）
	finalImageMethods := s.processImageMethods(req.EnabledImageMethods, user)

	// 校验配图方式权限（普通用户不能使用 NANO_BANANA 和 SVG_DIAGRAM）
	if err := s.validateImageMethods(finalImageMethods, user); err != nil {
		return "", err
	}

	// 检查并消耗配额（原子操作）
	if err := s.quotaSvc.CheckAndConsumeQuota(user); err != nil {
		return "", err
	}

	// 生成任务 ID
	taskID := uuid.NewString()

	// 将 enabledImageMethods 转为 JSON（为空时设置为 nil）
	var methodsJSON *string
	if len(finalImageMethods) > 0 {
		methodsBytes, _ := json.Marshal(finalImageMethods)
		methodsStr := string(methodsBytes)
		methodsJSON = &methodsStr
	}

	// 创建文章记录
	article := &model.Article{
		TaskID:              taskID,
		UserID:              user.ID,
		Topic:               req.Topic,
		Style:               req.Style,
		EnabledImageMethods: methodsJSON,
		Status:              model.StatusPending,
		Phase:               model.PhasePending,
		CreateTime:          time.Now(),
	}

	if err := s.store.Create(article); err != nil {
		return "", common.ErrOperation
	}

	// 异步执行阶段1：生成标题方案
	go s.ExecutePhase1Async(taskID, req.Topic, req.Style)

	log.Printf("文章任务已创建, taskId=%s, userId=%d, style=%s", taskID, user.ID, req.Style)
	return taskID, nil
}

// GetByTaskID 根据任务ID获取文章
func (s *ArticleService) GetByTaskID(taskID string, userID int64, isAdmin bool) (*model.ArticleInfo, error) {
	article, err := s.store.GetByTaskID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrNotFound.WithMessage("文章不存在")
		}
		return nil, common.ErrSystem
	}

	// 权限校验：只能查看自己的文章（管理员除外）
	if !isAdmin && article.UserID != userID {
		return nil, common.ErrNoAuth
	}

	return article.ToArticleInfo(), nil
}

// ListByPage 分页查询文章列表
func (s *ArticleService) ListByPage(req *model.QueryArticleRequest, userID int64, isAdmin bool) (*model.PageResult, error) {
	// 设置默认值
	if req.PageNum <= 0 {
		req.PageNum = common.DefaultPageNum
	}
	if req.PageSize <= 0 {
		req.PageSize = common.DefaultPageSize
	}
	if req.PageSize > common.MaxPageSize {
		req.PageSize = common.MaxPageSize
	}

	// 非管理员查询时，强制使用当前用户ID
	queryUserID := &userID
	if isAdmin && req.UserID != nil {
		queryUserID = req.UserID
	}

	articles, total, err := s.store.List(queryUserID, req.Status, isAdmin, req.PageNum, req.PageSize)
	if err != nil {
		return nil, common.ErrSystem
	}

	// 转换为响应
	articleInfos := make([]model.ArticleInfo, 0, len(articles))
	for i := range articles {
		if info := articles[i].ToArticleInfo(); info != nil {
			articleInfos = append(articleInfos, *info)
		}
	}

	return &model.PageResult{
		Total:    total,
		Records:  articleInfos,
		PageNum:  req.PageNum,
		PageSize: req.PageSize,
	}, nil
}

// Delete 删除文章
func (s *ArticleService) Delete(id int64, userID int64, isAdmin bool) error {
	article, err := s.store.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.ErrNotFound
		}
		return common.ErrSystem
	}

	// 权限校验：只能删除自己的文章（管理员除外）
	if !isAdmin && article.UserID != userID {
		return common.ErrNoAuth
	}

	if err := s.store.Delete(id); err != nil {
		return common.ErrOperation
	}

	return nil
}

// ExecutePhase1Async 阶段1：异步生成标题方案
func (s *ArticleService) ExecutePhase1Async(taskID, topic, style string) {
	useOrchestrator := s.cfg.Agent.Orchestrator.Enabled
	log.Printf("阶段1异步任务开始, taskId=%s, topic=%s, style=%s, 使用多智能体编排=%v",
		taskID, topic, style, useOrchestrator)

	// 更新状态和阶段
	_ = s.store.UpdateStatus(taskID, model.StatusProcessing, nil)
	_ = s.UpdatePhase(taskID, model.PhaseTitleGenerating)

	// 创建状态对象
	state := &model.ArticleState{
		TaskID: taskID,
		Topic:  topic,
		Style:  style,
		Phase:  model.PhaseTitleGenerating,
	}

	// 执行智能体1：生成标题方案（根据配置选择执行方式）
	ctx := context.Background()
	var err error
	if useOrchestrator {
		err = s.orchestrator.ExecutePhase1(ctx, state)
	} else {
		err = s.agentSvc.ExecutePhase1(ctx, state)
	}

	if err != nil {
		log.Printf("阶段1异步任务失败, taskId=%s, error=%v", taskID, err)

		// 更新状态为失败
		errMsg := err.Error()
		_ = s.store.UpdateStatus(taskID, model.StatusFailed, &errMsg)

		// 推送错误消息
		s.sseManager.Send(taskID, map[string]interface{}{
			"type":    common.SSEMsgError,
			"message": errMsg,
		})
		s.sseManager.Complete(taskID)
		return
	}

	// 保存标题方案到数据库
	if err := s.SaveTitleOptions(taskID, state.TitleOptions); err != nil {
		log.Printf("保存标题方案失败, taskId=%s, error=%v", taskID, err)
		errMsg := "保存标题方案失败"
		_ = s.store.UpdateStatus(taskID, model.StatusFailed, &errMsg)
		return
	}

	// 更新阶段为等待选择标题
	_ = s.UpdatePhase(taskID, model.PhaseTitleSelecting)

	// 推送标题方案生成完成消息
	s.sseManager.Send(taskID, map[string]interface{}{
		"type":         common.SSEMsgTitlesGenerated,
		"titleOptions": state.TitleOptions,
	})

	log.Printf("阶段1异步任务完成, taskId=%s, optionsCount=%d", taskID, len(state.TitleOptions))
}

// ExecutePhase2Async 阶段2：异步生成大纲（用户确认标题后调用）
func (s *ArticleService) ExecutePhase2Async(taskID string) {
	useOrchestrator := s.cfg.Agent.Orchestrator.Enabled
	log.Printf("阶段2异步任务开始, taskId=%s, 使用多智能体编排=%v", taskID, useOrchestrator)

	// 获取文章信息
	article, err := s.store.GetByTaskID(taskID)
	if err != nil {
		log.Printf("阶段2获取文章失败, taskId=%s, error=%v", taskID, err)
		return
	}

	// 更新阶段
	_ = s.UpdatePhase(taskID, model.PhaseOutlineGenerating)

	// 创建状态对象
	state := &model.ArticleState{
		TaskID:          taskID,
		Style:           article.Style,
		UserDescription: "",
		Phase:           model.PhaseOutlineGenerating,
	}

	if article.UserDescription != nil {
		state.UserDescription = *article.UserDescription
	}

	// 设置标题
	state.Title = &model.TitleResult{
		MainTitle: *article.MainTitle,
		SubTitle:  *article.SubTitle,
	}

	// 执行智能体2：生成大纲（根据配置选择执行方式）
	ctx := context.Background()
	if useOrchestrator {
		err = s.orchestrator.ExecutePhase2(ctx, state, func(message string) {
			s.handleStreamMessage(taskID, message)
		})
	} else {
		err = s.agentSvc.ExecutePhase2(ctx, state)
	}

	if err != nil {
		log.Printf("阶段2异步任务失败, taskId=%s, error=%v", taskID, err)

		// 更新状态为失败
		errMsg := err.Error()
		_ = s.store.UpdateStatus(taskID, model.StatusFailed, &errMsg)

		// 推送错误消息
		s.sseManager.Send(taskID, map[string]interface{}{
			"type":    common.SSEMsgError,
			"message": errMsg,
		})
		s.sseManager.Complete(taskID)
		return
	}

	// 保存大纲到数据库
	outlineJSON, _ := json.Marshal(state.Outline.Sections)
	outlineStr := string(outlineJSON)
	article.Outline = &outlineStr
	_ = s.store.Update(article)

	// 更新阶段为等待编辑大纲
	_ = s.UpdatePhase(taskID, model.PhaseOutlineEditing)

	// 推送大纲生成完成消息
	s.sseManager.Send(taskID, map[string]interface{}{
		"type":    common.SSEMsgOutlineGenerated,
		"outline": state.Outline.Sections,
	})

	log.Printf("阶段2异步任务完成, taskId=%s", taskID)
}

// ExecutePhase3Async 阶段3：异步生成正文+配图（用户确认大纲后调用）
func (s *ArticleService) ExecutePhase3Async(taskID string) {
	useOrchestrator := s.cfg.Agent.Orchestrator.Enabled
	log.Printf("阶段3异步任务开始, taskId=%s, 使用多智能体编排=%v", taskID, useOrchestrator)

	// 获取文章信息
	article, err := s.store.GetByTaskID(taskID)
	if err != nil {
		log.Printf("阶段3获取文章失败, taskId=%s, error=%v", taskID, err)
		return
	}

	// 更新阶段
	_ = s.UpdatePhase(taskID, model.PhaseContentGenerating)

	// 创建状态对象
	state := &model.ArticleState{
		TaskID: taskID,
		Style:  article.Style,
		Phase:  model.PhaseContentGenerating,
	}

	// 从数据库获取允许的配图方式
	if article.EnabledImageMethods != nil && *article.EnabledImageMethods != "" {
		_ = json.Unmarshal([]byte(*article.EnabledImageMethods), &state.EnabledImageMethods)
	}

	// 设置标题
	state.Title = &model.TitleResult{
		MainTitle: *article.MainTitle,
		SubTitle:  *article.SubTitle,
	}

	// 设置大纲
	var outlineSections []model.OutlineSection
	if article.Outline != nil {
		_ = json.Unmarshal([]byte(*article.Outline), &outlineSections)
	}
	state.Outline = &model.OutlineResult{
		Sections: outlineSections,
	}

	// 执行智能体3-6：生成正文+配图（根据配置选择执行方式）
	// 多智能体编排模式支持配图并行生成
	ctx := context.Background()
	if useOrchestrator {
		err = s.orchestrator.ExecutePhase3(ctx, state, func(message string) {
			s.handleStreamMessage(taskID, message)
		})
	} else {
		err = s.agentSvc.ExecutePhase3(ctx, state)
	}

	if err != nil {
		log.Printf("阶段3异步任务失败, taskId=%s, error=%v", taskID, err)

		// 更新状态为失败
		errMsg := err.Error()
		_ = s.store.UpdateStatus(taskID, model.StatusFailed, &errMsg)

		// 推送错误消息
		s.sseManager.Send(taskID, map[string]interface{}{
			"type":    common.SSEMsgError,
			"message": errMsg,
		})
		s.sseManager.Complete(taskID)
		return
	}

	// 保存文章到数据库
	if err := s.saveArticle(taskID, state); err != nil {
		log.Printf("保存文章失败, taskId=%s, error=%v", taskID, err)
		errMsg := "保存文章失败"
		_ = s.store.UpdateStatus(taskID, model.StatusFailed, &errMsg)
		return
	}

	// 更新状态为已完成
	_ = s.store.UpdateStatus(taskID, model.StatusCompleted, nil)

	// 推送完成消息
	s.sseManager.Send(taskID, map[string]interface{}{
		"type":   common.SSEMsgAllComplete,
		"taskId": taskID,
	})
	s.sseManager.Complete(taskID)

	log.Printf("阶段3异步任务完成, taskId=%s", taskID)
}

// saveArticle 保存文章到数据库
func (s *ArticleService) saveArticle(taskID string, state *model.ArticleState) error {
	article, err := s.store.GetByTaskID(taskID)
	if err != nil {
		return err
	}

	outlineJSON, _ := json.Marshal(state.Outline.Sections)
	imagesJSON, _ := json.Marshal(state.Images)
	outlineStr := string(outlineJSON)
	imagesStr := string(imagesJSON)
	now := time.Now()

	article.MainTitle = &state.Title.MainTitle
	article.SubTitle = &state.Title.SubTitle
	article.Outline = &outlineStr
	article.Content = &state.Content
	article.FullContent = &state.FullContent
	article.Images = &imagesStr
	article.CompletedTime = &now

	return s.store.Update(article)
}

// ConfirmTitle 确认标题并输入补充描述
func (s *ArticleService) ConfirmTitle(taskID, mainTitle, subTitle string, userDescription *string, userID int64, isAdmin bool) error {
	// 获取文章
	article, err := s.store.GetByTaskID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.ErrNotFound.WithMessage("文章不存在")
		}
		return common.ErrSystem
	}

	// 权限校验
	if !isAdmin && article.UserID != userID {
		return common.ErrNoAuth
	}

	// 校验当前阶段
	if article.Phase != model.PhaseTitleSelecting {
		return common.ErrParams.WithMessage("当前阶段不允许确认标题")
	}

	// 更新标题和用户补充描述
	article.MainTitle = &mainTitle
	article.SubTitle = &subTitle
	article.UserDescription = userDescription

	if err := s.store.Update(article); err != nil {
		return common.ErrOperation
	}

	// 异步执行阶段2：生成大纲
	go s.ExecutePhase2Async(taskID)

	return nil
}

// ConfirmOutline 确认大纲
func (s *ArticleService) ConfirmOutline(taskID string, outline []model.OutlineSection, userID int64, isAdmin bool) error {
	// 获取文章
	article, err := s.store.GetByTaskID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.ErrNotFound.WithMessage("文章不存在")
		}
		return common.ErrSystem
	}

	// 权限校验
	if !isAdmin && article.UserID != userID {
		return common.ErrNoAuth
	}

	// 校验当前阶段
	if article.Phase != model.PhaseOutlineEditing {
		return common.ErrParams.WithMessage("当前阶段不允许确认大纲")
	}

	// 更新大纲
	outlineJSON, _ := json.Marshal(outline)
	outlineStr := string(outlineJSON)
	article.Outline = &outlineStr

	if err := s.store.Update(article); err != nil {
		return common.ErrOperation
	}

	// 异步执行阶段3：生成正文+配图
	go s.ExecutePhase3Async(taskID)

	return nil
}

// UpdatePhase 更新阶段
func (s *ArticleService) UpdatePhase(taskID, phase string) error {
	return s.store.UpdatePhase(taskID, phase)
}

// SaveTitleOptions 保存标题方案
func (s *ArticleService) SaveTitleOptions(taskID string, titleOptions []model.TitleOption) error {
	optionsJSON, _ := json.Marshal(titleOptions)
	optionsStr := string(optionsJSON)
	return s.store.UpdateTitleOptions(taskID, optionsStr)
}

// AiModifyOutline AI 修改大纲
func (s *ArticleService) AiModifyOutline(taskID, modifySuggestion string, user *model.User, isAdmin bool) ([]model.OutlineSection, error) {
	// 获取文章
	article, err := s.store.GetByTaskID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrNotFound.WithMessage("文章不存在")
		}
		return nil, common.ErrSystem
	}

	// 权限校验
	if !isAdmin && article.UserID != user.ID {
		return nil, common.ErrNoAuth
	}

	// 校验 VIP 权限
	if !s.isVipOrAdmin(user) {
		return nil, common.ErrNoAuth.WithMessage("AI 修改大纲功能仅限 VIP 会员使用")
	}

	// 校验当前阶段
	if article.Phase != model.PhaseOutlineEditing {
		return nil, common.ErrParams.WithMessage("当前阶段不允许修改大纲")
	}

	// 解析当前大纲
	var currentOutline []model.OutlineSection
	if article.Outline != nil {
		if err := json.Unmarshal([]byte(*article.Outline), &currentOutline); err != nil {
			return nil, common.ErrSystem.WithMessage("解析大纲失败")
		}
	}

	// 调用智能体修改大纲
	ctx := context.Background()
	modifiedOutline, err := s.agentSvc.AiModifyOutline(ctx, taskID, *article.MainTitle, *article.SubTitle, currentOutline, modifySuggestion)
	if err != nil {
		return nil, common.ErrOperation.WithMessage("AI修改大纲失败: " + err.Error())
	}

	return modifiedOutline, nil
}

// handleStreamMessage 处理流式消息
// 解析流式消息并通过 SSE 推送给客户端
func (s *ArticleService) handleStreamMessage(taskID, message string) {
	// 如果消息包含冒号分隔符，说明是流式消息（格式：type:content）
	if strings.Contains(message, ":") {
		parts := strings.SplitN(message, ":", 2)
		if len(parts) == 2 {
			msgType := parts[0]
			content := parts[1]
			s.sseManager.Send(taskID, map[string]interface{}{
				"type":    msgType,
				"content": content,
			})
			return
		}
	}

	// 尝试解析为 JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(message), &jsonData); err == nil {
		s.sseManager.Send(taskID, jsonData)
	} else {
		// 如果解析失败，作为普通消息发送
		s.sseManager.Send(taskID, map[string]interface{}{
			"message": message,
		})
	}
}

// processImageMethods 处理配图方式
// 如果用户未选择，给普通用户设置默认的非 VIP 方式，VIP 和管理员不限制
func (s *ArticleService) processImageMethods(methods []string, user *model.User) []string {
	// 如果用户已选择，直接返回
	if len(methods) > 0 {
		return methods
	}

	// VIP 和管理员：不限制，返回 nil 表示支持所有方式
	if s.isVipOrAdmin(user) {
		return nil
	}

	// 普通用户：返回默认的非 VIP 方式
	return []string{
		common.ImageMethodPexels,
		common.ImageMethodMermaid,
		common.ImageMethodIconify,
		common.ImageMethodEmojiPack,
	}
}

// validateImageMethods 校验配图方式权限
// 普通用户不能使用 NANO_BANANA 和 SVG_DIAGRAM
func (s *ArticleService) validateImageMethods(methods []string, user *model.User) error {
	if len(methods) == 0 {
		return nil
	}

	// VIP 和管理员无限制
	if s.isVipOrAdmin(user) {
		return nil
	}

	// 普通用户限制
	for _, method := range methods {
		if method == common.ImageMethodNanoBanana || method == common.ImageMethodSVGDiagram {
			return common.ErrNoAuth.WithMessage("高级配图功能（AI 生图、SVG 图表）仅限 VIP 会员使用")
		}
	}

	return nil
}

// isVipOrAdmin 判断是否为 VIP 或管理员
func (s *ArticleService) isVipOrAdmin(user *model.User) bool {
	return user.UserRole == common.AdminRole || user.UserRole == common.VIPRole
}
