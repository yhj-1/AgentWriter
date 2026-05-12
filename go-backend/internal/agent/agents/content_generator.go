package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	agentContext "github.com/yupi/ai-passage-creator/internal/agent/context"
	"github.com/yupi/ai-passage-creator/internal/common"
	"github.com/yupi/ai-passage-creator/internal/model"
	"github.com/yupi/ai-passage-creator/internal/service"
)

// ContentGeneratorAgent 正文生成 Agent
// 根据大纲生成文章正文内容（支持流式输出）
type ContentGeneratorAgent struct {
	llm             llms.Model
	agentLogService *service.AgentLogService
}

// NewContentGeneratorAgent 创建正文生成 Agent
func NewContentGeneratorAgent(llm llms.Model, agentLogService *service.AgentLogService) *ContentGeneratorAgent {
	return &ContentGeneratorAgent{
		llm:             llm,
		agentLogService: agentLogService,
	}
}

// Execute 执行正文生成任务（流式输出）
func (a *ContentGeneratorAgent) Execute(ctx context.Context, state *model.ArticleState) error {
	log.Printf("ContentGeneratorAgent 开始执行: mainTitle=%s", state.Title.MainTitle)

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
		a.agentLogService.SaveLogAsync(agentLog)
	}()

	// 构建 prompt
	outlineJSON, _ := json.Marshal(state.Outline.Sections)
	prompt := strings.ReplaceAll(common.Agent3ContentPrompt, "{mainTitle}", state.Title.MainTitle)
	prompt = strings.ReplaceAll(prompt, "{subTitle}", state.Title.SubTitle)
	prompt = strings.ReplaceAll(prompt, "{outline}", string(outlineJSON))
	prompt += a.getStylePrompt(state.Style)
	agentLog.Prompt = &prompt

	// 获取流式处理器
	streamHandler := agentContext.GetStreamHandler(ctx)

	// 调用 LLM（流式输出）
	content, err := a.callLLMWithStreaming(ctx, prompt, streamHandler)
	if err != nil {
		agentLog.Status = "FAILED"
		errMsg := err.Error()
		agentLog.ErrorMessage = &errMsg
		return err
	}

	state.Content = content
	agentLog.Status = "SUCCESS"
	// 将输出数据转换为 JSON 格式
	outputDataJSON, _ := json.Marshal(map[string]interface{}{
		"contentLength": len(content),
		"message":       fmt.Sprintf("正文长度: %d 字符", len(content)),
	})
	outputDataStr := string(outputDataJSON)
	agentLog.OutputData = &outputDataStr
	log.Printf("ContentGeneratorAgent：正文生成成功, length=%d", len(content))
	return nil
}

// callLLMWithStreaming 调用 LLM（流式输出）
func (a *ContentGeneratorAgent) callLLMWithStreaming(ctx context.Context, prompt string, streamHandler agentContext.StreamHandler) (string, error) {
	var contentBuilder strings.Builder

	// 流式生成
	_, err := a.llm.GenerateContent(ctx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		text := string(chunk)
		contentBuilder.WriteString(text)

		// 推送流式内容（带前缀）
		if streamHandler != nil {
			message := fmt.Sprintf("%s%s", common.SSEMsgAgent3Streaming+":", text)
			streamHandler(message)
		}
		return nil
	}))

	if err != nil {
		log.Printf("ContentGeneratorAgent：流式调用失败, error=%v", err)
		return "", fmt.Errorf("streaming LLM call failed: %w", err)
	}

	return contentBuilder.String(), nil
}

// getStylePrompt 根据风格获取对应的 Prompt 附加内容
func (a *ContentGeneratorAgent) getStylePrompt(style string) string {
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
