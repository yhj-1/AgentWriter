package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/yupi/ai-passage-creator/internal/common"
	"github.com/yupi/ai-passage-creator/internal/config"
	"github.com/yupi/ai-passage-creator/internal/model"
)

// ArticleAgentService 文章智能体编排服务
type ArticleAgentService struct {
	llm             llms.Model
	imageStrategy   *ImageServiceStrategy
	agentLogService *AgentLogService
	sseManager      *common.SSEManager
}

// NewArticleAgentService 创建文章智能体服务
// 使用 LangChainGo OpenAI 客户端连接 DashScope（OpenAI 兼容）
func NewArticleAgentService(cfg *config.Config, imageStrategy *ImageServiceStrategy, agentLogService *AgentLogService, sseManager *common.SSEManager) (*ArticleAgentService, error) {
	baseURL := "https://dashscope.aliyuncs.com/compatible-mode/v1"

	// 添加调试日志
	log.Printf("初始化 DashScope 客户端: BaseURL=%s, Model=%s, APIKey=%s...",
		baseURL, cfg.AI.DashScope.Model, maskAPIKey(cfg.AI.DashScope.APIKey))

	llm, err := openai.New(
		openai.WithToken(cfg.AI.DashScope.APIKey),
		openai.WithModel(cfg.AI.DashScope.Model),
		openai.WithBaseURL(baseURL),
	)
	if err != nil {
		log.Printf("创建 DashScope 客户端失败: %v", err)
		return nil, fmt.Errorf("create dashscope client: %w", err)
	}

	log.Printf("DashScope 客户端初始化成功")

	return &ArticleAgentService{
		llm:             llm,
		imageStrategy:   imageStrategy,
		agentLogService: agentLogService,
		sseManager:      sseManager,
	}, nil
}

// maskAPIKey 遮蔽 API Key 用于日志
func maskAPIKey(key string) string {
	if len(key) <= 10 {
		return "***"
	}
	return key[:10] + "***"
}

// GetLLM 获取 LLM 实例（用于共享给 Orchestrator）
func (s *ArticleAgentService) GetLLM() llms.Model {
	return s.llm
}

// Execute 执行完整的文章生成流程（保留用于旧版本兼容）
func (s *ArticleAgentService) Execute(ctx context.Context, state *model.ArticleState) error {
	// 智能体1：生成标题
	log.Printf("智能体1：开始生成标题, taskId=%s", state.TaskID)
	if err := s.agent1GenerateTitle(ctx, state); err != nil {
		return fmt.Errorf("agent1 failed: %w", err)
	}
	s.sendMessage(state.TaskID, map[string]interface{}{
		"type":  common.SSEMsgAgent1Complete,
		"title": state.Title,
	})

	// 智能体2：生成大纲（流式）
	log.Printf("智能体2：开始生成大纲, taskId=%s", state.TaskID)
	if err := s.agent2GenerateOutlineStream(ctx, state); err != nil {
		return fmt.Errorf("agent2 failed: %w", err)
	}
	s.sendMessage(state.TaskID, map[string]interface{}{
		"type":    common.SSEMsgAgent2Complete,
		"outline": state.Outline.Sections,
	})

	// 智能体3：生成正文（流式）
	log.Printf("智能体3：开始生成正文, taskId=%s", state.TaskID)
	if err := s.agent3GenerateContent(ctx, state); err != nil {
		return fmt.Errorf("agent3 failed: %w", err)
	}
	s.sendMessage(state.TaskID, map[string]interface{}{
		"type": common.SSEMsgAgent3Complete,
	})

	// 智能体4：分析配图需求
	log.Printf("智能体4：开始分析配图需求, taskId=%s", state.TaskID)
	if err := s.agent4AnalyzeImageRequirements(ctx, state); err != nil {
		return fmt.Errorf("agent4 failed: %w", err)
	}
	s.sendMessage(state.TaskID, map[string]interface{}{
		"type":              common.SSEMsgAgent4Complete,
		"imageRequirements": state.ImageRequirements,
	})

	// 智能体5：生成配图
	log.Printf("智能体5：开始生成配图, taskId=%s", state.TaskID)
	if err := s.agent5GenerateImages(ctx, state); err != nil {
		return fmt.Errorf("agent5 failed: %w", err)
	}
	s.sendMessage(state.TaskID, map[string]interface{}{
		"type":   common.SSEMsgAgent5Complete,
		"images": state.Images,
	})

	// 图文合成：将配图插入正文
	log.Printf("开始图文合成, taskId=%s", state.TaskID)
	s.mergeImagesIntoContent(state)
	s.sendMessage(state.TaskID, map[string]interface{}{
		"type":        common.SSEMsgMergeComplete,
		"fullContent": state.FullContent,
	})

	log.Printf("文章生成完成, taskId=%s", state.TaskID)
	return nil
}

// agent1GenerateTitle 智能体1：生成标题
func (s *ArticleAgentService) agent1GenerateTitle(ctx context.Context, state *model.ArticleState) error {
	prompt := strings.ReplaceAll(common.Agent1TitlePrompt, "{topic}", state.Topic)

	log.Printf("智能体1：发送请求到 LLM, promptLength=%d", len(prompt))

	content, err := llms.GenerateFromSinglePrompt(ctx, s.llm, prompt)
	if err != nil {
		log.Printf("智能体1：LLM 调用失败, error=%v", err)
		return fmt.Errorf("LLM call failed: %w", err)
	}

	log.Printf("智能体1：收到响应, contentLength=%d, content preview=%s...",
		len(content), truncateString(content, 100))

	var title model.TitleResult
	if err := json.Unmarshal([]byte(content), &title); err != nil {
		log.Printf("智能体1：标题解析失败, content=%s", content)
		return fmt.Errorf("parse title failed: %w", err)
	}

	state.Title = &title
	log.Printf("智能体1：标题生成成功, mainTitle=%s", title.MainTitle)
	return nil
}

// truncateString 截断字符串用于日志
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// agent2GenerateOutline 智能体2：生成大纲
func (s *ArticleAgentService) agent2GenerateOutline(ctx context.Context, state *model.ArticleState) error {
	prompt := strings.ReplaceAll(common.Agent2OutlinePrompt, "{mainTitle}", state.Title.MainTitle)
	prompt = strings.ReplaceAll(prompt, "{subTitle}", state.Title.SubTitle)

	content, err := llms.GenerateFromSinglePrompt(ctx, s.llm, prompt)
	if err != nil {
		return err
	}

	var outline model.OutlineResult
	if err := json.Unmarshal([]byte(content), &outline); err != nil {
		log.Printf("智能体2：大纲解析失败, content=%s", content)
		return fmt.Errorf("parse outline failed: %w", err)
	}

	state.Outline = &outline
	log.Printf("智能体2：大纲生成成功, sections=%d", len(outline.Sections))
	return nil
}

// agent3GenerateContent 智能体3：生成正文（流式）
func (s *ArticleAgentService) agent3GenerateContent(ctx context.Context, state *model.ArticleState) error {
	// 创建日志记录
	startTime := time.Now()
	agentLog := &model.AgentLog{
		TaskID:    state.TaskID,
		AgentName: "agent3_generate_content",
		StartTime: startTime,
		Status:    "RUNNING",
	}

	// 使用 defer 确保日志一定会被保存
	defer func() {
		endTime := time.Now()
		agentLog.EndTime = &endTime
		duration := int(time.Since(startTime).Milliseconds())
		agentLog.DurationMs = &duration
		s.agentLogService.SaveLogAsync(agentLog)
	}()

	outlineJSON, _ := json.Marshal(state.Outline.Sections)
	prompt := strings.ReplaceAll(common.Agent3ContentPrompt, "{mainTitle}", state.Title.MainTitle)
	prompt = strings.ReplaceAll(prompt, "{subTitle}", state.Title.SubTitle)
	prompt = strings.ReplaceAll(prompt, "{outline}", string(outlineJSON))
	agentLog.Prompt = &prompt

	var contentBuilder strings.Builder

	// 流式生成
	_, err := s.llm.GenerateContent(ctx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		text := string(chunk)
		contentBuilder.WriteString(text)

		// 推送流式内容
		s.sendMessage(state.TaskID, map[string]interface{}{
			"type":    common.SSEMsgAgent3Streaming,
			"content": text,
		})
		return nil
	}))

	if err != nil {
		agentLog.Status = "FAILED"
		errMsg := err.Error()
		agentLog.ErrorMessage = &errMsg
		return err
	}

	state.Content = contentBuilder.String()
	agentLog.Status = "SUCCESS"
	// 将输出数据转换为 JSON 格式
	outputDataJSON, _ := json.Marshal(map[string]interface{}{
		"contentLength": len(state.Content),
		"message":       fmt.Sprintf("正文长度: %d 字符", len(state.Content)),
	})
	outputDataStr := string(outputDataJSON)
	agentLog.OutputData = &outputDataStr
	log.Printf("智能体3：正文生成成功, length=%d", len(state.Content))
	return nil
}

// agent4AnalyzeImageRequirements 智能体4：分析配图需求（占位符方案）
func (s *ArticleAgentService) agent4AnalyzeImageRequirements(ctx context.Context, state *model.ArticleState) error {
	// 创建日志记录
	startTime := time.Now()
	agentLog := &model.AgentLog{
		TaskID:    state.TaskID,
		AgentName: "agent4_analyze_image_requirements",
		StartTime: startTime,
		Status:    "RUNNING",
	}

	// 使用 defer 确保日志一定会被保存
	defer func() {
		endTime := time.Now()
		agentLog.EndTime = &endTime
		duration := int(time.Since(startTime).Milliseconds())
		agentLog.DurationMs = &duration
		s.agentLogService.SaveLogAsync(agentLog)
	}()

	// 构建可用配图方式说明
	availableMethods := s.buildAvailableMethodsDescription(state.EnabledImageMethods)

	// 构建各配图方式的详细使用指南
	methodUsageGuide := s.buildMethodUsageGuide(state.EnabledImageMethods)

	prompt := strings.ReplaceAll(common.Agent4ImageRequirementsPrompt, "{mainTitle}", state.Title.MainTitle)
	prompt = strings.ReplaceAll(prompt, "{content}", state.Content)
	prompt = strings.ReplaceAll(prompt, "{availableMethods}", availableMethods)
	prompt = strings.ReplaceAll(prompt, "{methodUsageGuide}", methodUsageGuide)
	agentLog.Prompt = &prompt

	content, err := llms.GenerateFromSinglePrompt(ctx, s.llm, prompt)
	if err != nil {
		agentLog.Status = "FAILED"
		errMsg := err.Error()
		agentLog.ErrorMessage = &errMsg
		return err
	}

	// 解析包含占位符的结果
	var result struct {
		ContentWithPlaceholders string                   `json:"contentWithPlaceholders"`
		ImageRequirements       []model.ImageRequirement `json:"imageRequirements"`
	}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		log.Printf("智能体4：配图需求解析失败, content=%s", content)
		agentLog.Status = "FAILED"
		errMsg := "parse image requirements failed: " + err.Error()
		agentLog.ErrorMessage = &errMsg
		return fmt.Errorf("parse image requirements failed: %w", err)
	}

	// 更新正文为包含占位符的版本
	state.ContentWithPlaceholders = result.ContentWithPlaceholders

	// 验证并过滤配图需求
	validatedRequirements := s.validateAndFilterImageRequirements(result.ImageRequirements, state.EnabledImageMethods)
	state.ImageRequirements = validatedRequirements

	agentLog.Status = "SUCCESS"
	// 将输出数据转换为 JSON 格式
	outputDataJSON, _ := json.Marshal(map[string]interface{}{
		"requirementsCount": len(validatedRequirements),
		"message":           fmt.Sprintf("分析出 %d 个配图需求", len(validatedRequirements)),
	})
	outputDataStr := string(outputDataJSON)
	agentLog.OutputData = &outputDataStr

	log.Printf("智能体4：配图需求分析成功, count=%d, validated=%d, 已在正文中插入占位符",
		len(result.ImageRequirements), len(validatedRequirements))
	log.Printf("智能体4：ContentWithPlaceholders长度=%d, 前100字符=%s",
		len(state.ContentWithPlaceholders), truncateString(state.ContentWithPlaceholders, 100))

	// 打印配图需求的占位符信息
	for i, req := range validatedRequirements {
		log.Printf("智能体4：配图需求[%d] position=%d, placeholderId=%s", i, req.Position, req.PlaceholderID)
	}

	return nil
}

// agent5GenerateImages 智能体5：生成配图（使用策略模式）
func (s *ArticleAgentService) agent5GenerateImages(ctx context.Context, state *model.ArticleState) error {
	// 创建日志记录
	startTime := time.Now()
	agentLog := &model.AgentLog{
		TaskID:    state.TaskID,
		AgentName: "agent5_generate_images",
		StartTime: startTime,
		Status:    "RUNNING",
	}

	// 使用 defer 确保日志一定会被保存
	defer func() {
		endTime := time.Now()
		agentLog.EndTime = &endTime
		duration := int(time.Since(startTime).Milliseconds())
		agentLog.DurationMs = &duration
		s.agentLogService.SaveLogAsync(agentLog)
	}()

	// 将输入数据转换为 JSON 格式
	inputDataJSON, _ := json.Marshal(map[string]interface{}{
		"requirementsCount": len(state.ImageRequirements),
		"message":           fmt.Sprintf("需要生成 %d 张配图", len(state.ImageRequirements)),
	})
	inputDataStr := string(inputDataJSON)
	agentLog.InputData = &inputDataStr

	var imageResults []model.ImageResult

	for _, req := range state.ImageRequirements {
		imageSource := req.ImageSource
		log.Printf("智能体5：开始获取配图, position=%d, imageSource=%s, keywords=%s, prompt=%s",
			req.Position, imageSource, req.Keywords, truncateString(req.Prompt, 50))

		// 构建图片请求对象
		imageRequest := &model.ImageRequest{
			Keywords: req.Keywords,
			Prompt:   req.Prompt,
			Position: req.Position,
			Type:     req.Type,
		}

		// 使用策略模式获取图片并统一上传到 COS
		result, err := s.imageStrategy.GetImageAndUpload(imageSource, imageRequest)
		if err != nil {
			log.Printf("智能体5：获取图片失败, position=%d, error=%v", req.Position, err)
			// 继续处理下一张图片，不中断流程
			continue
		}

		cosURL := result.URL
		method := result.Method

		// 创建配图结果（URL 已经是 COS 地址）
		imageResult := s.buildImageResult(&req, cosURL, method)
		imageResults = append(imageResults, imageResult)

		// 推送单张配图完成
		s.sendMessage(state.TaskID, map[string]interface{}{
			"type":  common.SSEMsgImageComplete,
			"image": imageResult,
		})

		log.Printf("智能体5：配图获取并上传成功, position=%d, method=%s, cosUrl=%s",
			req.Position, method, cosURL)
	}

	state.Images = imageResults
	agentLog.Status = "SUCCESS"
	// 将输出数据转换为 JSON 格式
	outputDataJSON, _ := json.Marshal(map[string]interface{}{
		"imagesCount": len(imageResults),
		"message":     fmt.Sprintf("成功生成 %d 张配图", len(imageResults)),
	})
	outputDataStr := string(outputDataJSON)
	agentLog.OutputData = &outputDataStr
	log.Printf("智能体5：所有配图生成并上传完成, count=%d", len(imageResults))
	return nil
}

// buildImageResult 构建配图结果对象
func (s *ArticleAgentService) buildImageResult(req *model.ImageRequirement, cosURL, method string) model.ImageResult {
	return model.ImageResult{
		Position:      req.Position,
		URL:           cosURL,
		Method:        method,
		Keywords:      req.Keywords,
		SectionTitle:  req.SectionTitle,
		Description:   req.Type,
		PlaceholderID: req.PlaceholderID,
	}
}

// sendMessage 发送 SSE 消息
func (s *ArticleAgentService) sendMessage(taskID string, data interface{}) {
	s.sseManager.Send(taskID, data)
}

// buildAvailableMethodsDescription 构建可用配图方式说明
func (s *ArticleAgentService) buildAvailableMethodsDescription(enabledMethods []string) string {
	// 如果为空，表示支持所有方式
	methods := enabledMethods
	if len(methods) == 0 {
		methods = common.GetAllMethods()
		// 移除 PICSUM（降级方案不展示）
		filteredMethods := []string{}
		for _, m := range methods {
			if !common.IsFallback(m) {
				filteredMethods = append(filteredMethods, m)
			}
		}
		methods = filteredMethods
	}

	var descriptions []string
	for _, method := range methods {
		desc := common.GetDescription(method)
		descriptions = append(descriptions, fmt.Sprintf("- %s: %s", method, desc))
	}

	return strings.Join(descriptions, "\n")
}

// buildMethodUsageGuide 构建各配图方式的详细使用指南
func (s *ArticleAgentService) buildMethodUsageGuide(enabledMethods []string) string {
	// 如果为空，表示支持所有方式
	methods := enabledMethods
	if len(methods) == 0 {
		methods = common.GetAllMethods()
	}

	guide := strings.Builder{}

	for _, method := range methods {
		if common.IsFallback(method) {
			continue // 降级方案不展示
		}

		switch method {
		case common.ImageMethodPexels:
			guide.WriteString("- PEXELS: 适合真实场景、产品照片、人物照片、自然风景等写实图片\n")
			guide.WriteString("  使用方式：提供英文搜索关键词(keywords)，要准确、具体\n\n")
		case common.ImageMethodNanoBanana:
			guide.WriteString("- NANO_BANANA: 适合创意插画、信息图表、需要文字渲染、抽象概念、艺术风格等 AI 生成图片\n")
			guide.WriteString("  使用方式：提供详细的英文生图提示词(prompt)，描述场景、风格、细节\n\n")
		case common.ImageMethodMermaid:
			guide.WriteString("- MERMAID: 适合流程图、架构图、时序图、关系图、甘特图等结构化图表\n")
			guide.WriteString("  使用方式：在 prompt 字段生成完整的 Mermaid 代码，keywords 留空\n\n")
		case common.ImageMethodIconify:
			guide.WriteString("- ICONIFY: 适合图标、符号、小型装饰性图标（如：箭头、勾选、星星、心形等）\n")
			guide.WriteString("  使用方式：提供英文图标关键词（keywords），如：check、arrow、star、heart，prompt 留空\n\n")
		case common.ImageMethodEmojiPack:
			guide.WriteString("- EMOJI_PACK: 适合表情包、搞笑图片、轻松幽默的配图\n")
			guide.WriteString("  使用方式：提供中文或英文关键词（keywords），描述表情内容，如：开心、哭笑、无语、疑问，prompt 留空\n")
			guide.WriteString("  注意：系统会自动在关键词后添加\"表情包\"进行搜索\n\n")
		case common.ImageMethodSVGDiagram:
			guide.WriteString("- SVG_DIAGRAM: 适合概念示意图、思维导图样式、逻辑关系展示（不涉及精确数据）\n")
			guide.WriteString("  使用方式：在 prompt 字段描述示意图需求（中文），说明要表达的概念和关系，keywords 留空\n")
			guide.WriteString("  示例：绘制思维导图样式的图，中心是\"自律\"，周围4个分支：习惯、环境、反馈、系统\n\n")
		}
	}

	return guide.String()
}

// validateAndFilterImageRequirements 验证并过滤配图需求
// 确保所有 imageSource 都在允许列表中
func (s *ArticleAgentService) validateAndFilterImageRequirements(requirements []model.ImageRequirement, enabledMethods []string) []model.ImageRequirement {
	// 如果 enabledMethods 为空，表示支持所有方式，不需要过滤
	if len(enabledMethods) == 0 {
		return requirements
	}

	// 创建允许的方式集合
	allowedSet := make(map[string]bool)
	for _, method := range enabledMethods {
		allowedSet[method] = true
	}

	// 验证并过滤配图需求
	var validated []model.ImageRequirement
	for _, req := range requirements {
		imageSource := req.ImageSource

		// 验证 imageSource 是否在允许列表中
		if allowedSet[imageSource] {
			validated = append(validated, req)
			log.Printf("配图需求验证通过, position=%d, imageSource=%s", req.Position, imageSource)
		} else {
			log.Printf("配图需求不符合限制被过滤, position=%d, imageSource=%s, enabledMethods=%v",
				req.Position, imageSource, enabledMethods)

			// 尝试替换为允许的方式（优先使用第一个允许的方式）
			if len(enabledMethods) > 0 {
				fallbackSource := enabledMethods[0]
				req.ImageSource = fallbackSource
				validated = append(validated, req)
				log.Printf("配图需求已替换为允许的方式, position=%d, fallback=%s",
					req.Position, fallbackSource)
			}
		}
	}

	return validated
}

// ExecutePhase1 阶段1：生成标题方案
func (s *ArticleAgentService) ExecutePhase1(ctx context.Context, state *model.ArticleState) error {
	log.Printf("阶段1：开始生成标题方案, taskId=%s", state.TaskID)

	// 智能体1：生成标题方案
	if err := s.agent1GenerateTitleOptions(ctx, state); err != nil {
		return fmt.Errorf("agent1 failed: %w", err)
	}
	s.sendMessage(state.TaskID, map[string]interface{}{
		"type":         common.SSEMsgAgent1Complete,
		"titleOptions": state.TitleOptions,
	})

	log.Printf("阶段1：标题方案生成完成, taskId=%s, optionsCount=%d", state.TaskID, len(state.TitleOptions))
	return nil
}

// ExecutePhase2 阶段2：生成大纲
func (s *ArticleAgentService) ExecutePhase2(ctx context.Context, state *model.ArticleState) error {
	log.Printf("阶段2：开始生成大纲, taskId=%s", state.TaskID)

	// 智能体2：生成大纲（流式）
	if err := s.agent2GenerateOutlineStream(ctx, state); err != nil {
		return fmt.Errorf("agent2 failed: %w", err)
	}
	s.sendMessage(state.TaskID, map[string]interface{}{
		"type":    common.SSEMsgAgent2Complete,
		"outline": state.Outline.Sections,
	})

	log.Printf("阶段2：大纲生成完成, taskId=%s", state.TaskID)
	return nil
}

// ExecutePhase3 阶段3：生成正文+配图
func (s *ArticleAgentService) ExecutePhase3(ctx context.Context, state *model.ArticleState) error {
	log.Printf("阶段3：开始生成正文+配图, taskId=%s", state.TaskID)

	// 智能体3：生成正文（流式）
	log.Printf("智能体3：开始生成正文, taskId=%s", state.TaskID)
	if err := s.agent3GenerateContent(ctx, state); err != nil {
		return fmt.Errorf("agent3 failed: %w", err)
	}
	s.sendMessage(state.TaskID, map[string]interface{}{
		"type": common.SSEMsgAgent3Complete,
	})

	// 智能体4：分析配图需求
	log.Printf("智能体4：开始分析配图需求, taskId=%s", state.TaskID)
	if err := s.agent4AnalyzeImageRequirements(ctx, state); err != nil {
		return fmt.Errorf("agent4 failed: %w", err)
	}
	s.sendMessage(state.TaskID, map[string]interface{}{
		"type":              common.SSEMsgAgent4Complete,
		"imageRequirements": state.ImageRequirements,
	})

	// 智能体5：生成配图
	log.Printf("智能体5：开始生成配图, taskId=%s", state.TaskID)
	if err := s.agent5GenerateImages(ctx, state); err != nil {
		return fmt.Errorf("agent5 failed: %w", err)
	}
	s.sendMessage(state.TaskID, map[string]interface{}{
		"type":   common.SSEMsgAgent5Complete,
		"images": state.Images,
	})

	// 图文合成：将配图插入正文
	log.Printf("开始图文合成, taskId=%s", state.TaskID)
	s.mergeImagesIntoContent(state)
	s.sendMessage(state.TaskID, map[string]interface{}{
		"type":        common.SSEMsgMergeComplete,
		"fullContent": state.FullContent,
	})

	log.Printf("阶段3：正文生成完成, taskId=%s", state.TaskID)
	return nil
}

// agent1GenerateTitleOptions 智能体1：生成标题方案（3-5个）
func (s *ArticleAgentService) agent1GenerateTitleOptions(ctx context.Context, state *model.ArticleState) error {
	// 创建日志记录
	startTime := time.Now()
	agentLog := &model.AgentLog{
		TaskID:    state.TaskID,
		AgentName: "agent1_generate_titles",
		StartTime: startTime,
		Status:    "RUNNING",
	}

	// 使用 defer 确保日志一定会被保存
	defer func() {
		endTime := time.Now()
		agentLog.EndTime = &endTime
		duration := int(time.Since(startTime).Milliseconds())
		agentLog.DurationMs = &duration
		s.agentLogService.SaveLogAsync(agentLog)
	}()

	// 构建 prompt
	prompt := strings.ReplaceAll(common.Agent1TitlePrompt, "{topic}", state.Topic)
	prompt += s.getStylePrompt(state.Style)
	agentLog.Prompt = &prompt

	log.Printf("智能体1：发送请求到 LLM, promptLength=%d", len(prompt))

	// 调用 LLM
	content, err := llms.GenerateFromSinglePrompt(ctx, s.llm, prompt)
	if err != nil {
		log.Printf("智能体1：LLM 调用失败, error=%v", err)
		agentLog.Status = "FAILED"
		errMsg := err.Error()
		agentLog.ErrorMessage = &errMsg
		return fmt.Errorf("LLM call failed: %w", err)
	}

	log.Printf("智能体1：收到响应, contentLength=%d, content preview=%s...", len(content), truncateString(content, 100))

	// 解析标题方案列表
	var titleOptions []model.TitleOption
	if err := json.Unmarshal([]byte(content), &titleOptions); err != nil {
		log.Printf("智能体1：标题方案解析失败, content=%s", content)
		agentLog.Status = "FAILED"
		errMsg := "parse title options: " + err.Error()
		agentLog.ErrorMessage = &errMsg
		return fmt.Errorf("parse title options: %w", err)
	}

	state.TitleOptions = titleOptions
	agentLog.Status = "SUCCESS"
	// 将输出数据转换为 JSON 格式
	outputDataJSON, _ := json.Marshal(map[string]interface{}{
		"optionsCount": len(titleOptions),
		"message":      fmt.Sprintf("生成 %d 个标题方案", len(titleOptions)),
	})
	outputDataStr := string(outputDataJSON)
	agentLog.OutputData = &outputDataStr
	log.Printf("智能体1：标题方案生成成功, optionsCount=%d", len(titleOptions))
	return nil
}

// AiModifyOutline AI 修改大纲
func (s *ArticleAgentService) AiModifyOutline(ctx context.Context, taskID, mainTitle, subTitle string, currentOutline []model.OutlineSection, modifySuggestion string) ([]model.OutlineSection, error) {
	// 创建日志记录
	startTime := time.Now()
	agentLog := &model.AgentLog{
		TaskID:    taskID,
		AgentName: "ai_modify_outline",
		StartTime: startTime,
		Status:    "RUNNING",
	}

	// 使用 defer 确保日志一定会被保存
	defer func() {
		endTime := time.Now()
		agentLog.EndTime = &endTime
		duration := int(time.Since(startTime).Milliseconds())
		agentLog.DurationMs = &duration
		s.agentLogService.SaveLogAsync(agentLog)
	}()

	// 构建当前大纲 JSON
	currentOutlineJSON, _ := json.Marshal(currentOutline)
	// 将输入数据转换为 JSON 格式
	inputDataJSON, _ := json.Marshal(map[string]interface{}{
		"outlineSectionsCount": len(currentOutline),
		"suggestionLength":     len(modifySuggestion),
	})
	inputDataStr := string(inputDataJSON)
	agentLog.InputData = &inputDataStr

	// 构建 prompt
	prompt := common.AiModifyOutlinePrompt
	prompt = strings.ReplaceAll(prompt, "{mainTitle}", mainTitle)
	prompt = strings.ReplaceAll(prompt, "{subTitle}", subTitle)
	prompt = strings.ReplaceAll(prompt, "{currentOutline}", string(currentOutlineJSON))
	prompt = strings.ReplaceAll(prompt, "{modifySuggestion}", modifySuggestion)
	agentLog.Prompt = &prompt

	log.Printf("AI修改大纲：发送请求到 LLM, promptLength=%d", len(prompt))

	// 调用 LLM
	content, err := llms.GenerateFromSinglePrompt(ctx, s.llm, prompt)
	if err != nil {
		log.Printf("AI修改大纲：LLM 调用失败, error=%v", err)
		agentLog.Status = "FAILED"
		errMsg := err.Error()
		agentLog.ErrorMessage = &errMsg
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	log.Printf("AI修改大纲：收到响应, contentLength=%d", len(content))

	// 解析修改后的大纲
	var outlineResult model.OutlineResult
	if err := json.Unmarshal([]byte(content), &outlineResult); err != nil {
		log.Printf("AI修改大纲：大纲解析失败, content=%s", content)
		agentLog.Status = "FAILED"
		errMsg := "parse outline: " + err.Error()
		agentLog.ErrorMessage = &errMsg
		return nil, fmt.Errorf("parse outline: %w", err)
	}

	agentLog.Status = "SUCCESS"
	// 将输出数据转换为 JSON 格式
	outputDataJSON, _ := json.Marshal(map[string]interface{}{
		"sectionsCount": len(outlineResult.Sections),
		"message":       fmt.Sprintf("修改后大纲段落数: %d", len(outlineResult.Sections)),
	})
	outputDataStr := string(outputDataJSON)
	agentLog.OutputData = &outputDataStr
	log.Printf("AI修改大纲成功, sectionsCount=%d", len(outlineResult.Sections))
	return outlineResult.Sections, nil
}

// getStylePrompt 根据风格获取对应的 Prompt 附加内容
func (s *ArticleAgentService) getStylePrompt(style string) string {
	if style == "" {
		return ""
	}

	switch style {
	case common.ArticleStyleTech:
		return common.StyleTechPrompt
	case common.ArticleStyleEmotional:
		return common.StyleEmotionalPrompt
	case common.ArticleStyleEducational:
		return common.StyleEducationalPrompt
	case common.ArticleStyleHumorous:
		return common.StyleHumorousPrompt
	default:
		return ""
	}
}
