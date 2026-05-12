package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/yupi/ai-passage-creator/internal/common"
	"github.com/yupi/ai-passage-creator/internal/model"
	"github.com/yupi/ai-passage-creator/internal/service"
)

// ImageAnalyzerAgent 配图需求分析 Agent
// 分析文章内容，生成配图需求列表并在正文中插入占位符
type ImageAnalyzerAgent struct {
	llm             llms.Model
	agentLogService *service.AgentLogService
}

// NewImageAnalyzerAgent 创建配图需求分析 Agent
func NewImageAnalyzerAgent(llm llms.Model, agentLogService *service.AgentLogService) *ImageAnalyzerAgent {
	return &ImageAnalyzerAgent{
		llm:             llm,
		agentLogService: agentLogService,
	}
}

// Execute 执行配图需求分析任务
func (a *ImageAnalyzerAgent) Execute(ctx context.Context, state *model.ArticleState) error {
	log.Printf("ImageAnalyzerAgent 开始执行: mainTitle=%s, enabledMethods=%v",
		state.Title.MainTitle, state.EnabledImageMethods)

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
		a.agentLogService.SaveLogAsync(agentLog)
	}()

	// 构建可用配图方式说明
	availableMethods := a.buildAvailableMethodsDescription(state.EnabledImageMethods)

	// 构建各配图方式的详细使用指南
	methodUsageGuide := a.buildMethodUsageGuide(state.EnabledImageMethods)

	prompt := strings.ReplaceAll(common.Agent4ImageRequirementsPrompt, "{mainTitle}", state.Title.MainTitle)
	prompt = strings.ReplaceAll(prompt, "{content}", state.Content)
	prompt = strings.ReplaceAll(prompt, "{availableMethods}", availableMethods)
	prompt = strings.ReplaceAll(prompt, "{methodUsageGuide}", methodUsageGuide)
	agentLog.Prompt = &prompt

	content, err := llms.GenerateFromSinglePrompt(ctx, a.llm, prompt)
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
		log.Printf("ImageAnalyzerAgent：配图需求解析失败, content=%s", content)
		agentLog.Status = "FAILED"
		errMsg := "parse image requirements failed: " + err.Error()
		agentLog.ErrorMessage = &errMsg
		return fmt.Errorf("parse image requirements failed: %w", err)
	}

	// 更新正文为包含占位符的版本
	state.ContentWithPlaceholders = result.ContentWithPlaceholders

	// 验证并过滤配图需求
	validatedRequirements := a.validateAndFilterImageRequirements(result.ImageRequirements, state.EnabledImageMethods)
	state.ImageRequirements = validatedRequirements

	agentLog.Status = "SUCCESS"
	// 将输出数据转换为 JSON 格式
	outputDataJSON, _ := json.Marshal(map[string]interface{}{
		"requirementsCount": len(validatedRequirements),
		"message":           fmt.Sprintf("分析出 %d 个配图需求", len(validatedRequirements)),
	})
	outputDataStr := string(outputDataJSON)
	agentLog.OutputData = &outputDataStr

	log.Printf("ImageAnalyzerAgent：配图需求分析成功, count=%d, validated=%d, 已在正文中插入占位符",
		len(result.ImageRequirements), len(validatedRequirements))

	return nil
}

// buildAvailableMethodsDescription 构建可用配图方式说明
func (a *ImageAnalyzerAgent) buildAvailableMethodsDescription(enabledMethods []string) string {
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
func (a *ImageAnalyzerAgent) buildMethodUsageGuide(enabledMethods []string) string {
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
			guide.WriteString("- PEXELS: 提供英文搜索关键词(keywords)，要准确、具体。prompt 留空。\n")
		case common.ImageMethodNanoBanana:
			guide.WriteString("- NANO_BANANA: 提供详细的英文生图提示词(prompt)，描述场景、风格、细节。keywords 留空。\n")
		case common.ImageMethodMermaid:
			guide.WriteString("- MERMAID: 在 prompt 字段生成完整的 Mermaid 代码（如流程图、架构图）。keywords 留空。\n")
		case common.ImageMethodIconify:
			guide.WriteString("- ICONIFY: 提供英文图标关键词(keywords)，如：check、arrow、star、heart。prompt 留空。\n")
		case common.ImageMethodEmojiPack:
			guide.WriteString("- EMOJI_PACK: 提供中文或英文关键词(keywords)描述表情内容。prompt 留空。系统会自动添加\"表情包\"搜索。\n")
		case common.ImageMethodSVGDiagram:
			guide.WriteString("- SVG_DIAGRAM: 在 prompt 字段描述示意图需求（中文），说明要表达的概念和关系。keywords 留空。\n")
			guide.WriteString("  示例：绘制思维导图样式的图，中心是\"自律\"，周围4个分支：习惯、环境、反馈、系统\n")
		}
	}

	return guide.String()
}

// validateAndFilterImageRequirements 验证并过滤配图需求
// 确保所有 imageSource 都在允许列表中
func (a *ImageAnalyzerAgent) validateAndFilterImageRequirements(requirements []model.ImageRequirement, enabledMethods []string) []model.ImageRequirement {
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
