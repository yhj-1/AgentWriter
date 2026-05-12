package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/tmc/langchaingo/llms"
	agentContext "github.com/yupi/ai-passage-creator/internal/agent/context"
	"github.com/yupi/ai-passage-creator/internal/common"
	"github.com/yupi/ai-passage-creator/internal/model"
)

// OutlineGeneratorAgent 大纲生成 Agent
// 根据标题生成文章大纲（支持流式输出）
type OutlineGeneratorAgent struct {
	llm llms.Model
}

// NewOutlineGeneratorAgent 创建大纲生成 Agent
func NewOutlineGeneratorAgent(llm llms.Model) *OutlineGeneratorAgent {
	return &OutlineGeneratorAgent{
		llm: llm,
	}
}

// Execute 执行大纲生成任务（流式输出）
func (a *OutlineGeneratorAgent) Execute(ctx context.Context, state *model.ArticleState) error {
	log.Printf("OutlineGeneratorAgent 开始执行: mainTitle=%s, subTitle=%s",
		state.Title.MainTitle, state.Title.SubTitle)

	// 构建用户描述部分
	descriptionSection := ""
	if state.UserDescription != "" {
		descriptionSection = strings.ReplaceAll(
			common.Agent2DescriptionSection,
			"{userDescription}",
			state.UserDescription,
		)
	}

	// 构建 prompt
	prompt := strings.ReplaceAll(common.Agent2OutlinePrompt, "{mainTitle}", state.Title.MainTitle)
	prompt = strings.ReplaceAll(prompt, "{subTitle}", state.Title.SubTitle)
	prompt = strings.ReplaceAll(prompt, "{descriptionSection}", descriptionSection)
	prompt += a.getStylePrompt(state.Style)

	// 获取流式处理器
	streamHandler := agentContext.GetStreamHandler(ctx)

	// 调用 LLM（流式输出）
	content, err := a.callLLMWithStreaming(ctx, prompt, streamHandler)
	if err != nil {
		return err
	}

	// 解析结果
	var outlineResult model.OutlineResult
	if err := json.Unmarshal([]byte(content), &outlineResult); err != nil {
		log.Printf("OutlineGeneratorAgent：大纲解析失败, content=%s", content)
		return fmt.Errorf("parse outline failed: %w", err)
	}

	state.Outline = &outlineResult
	log.Printf("OutlineGeneratorAgent：大纲生成成功, sections=%d", len(outlineResult.Sections))
	return nil
}

// callLLMWithStreaming 调用 LLM（流式输出）
func (a *OutlineGeneratorAgent) callLLMWithStreaming(ctx context.Context, prompt string, streamHandler agentContext.StreamHandler) (string, error) {
	var contentBuilder strings.Builder

	// 流式生成
	_, err := a.llm.GenerateContent(ctx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		text := string(chunk)
		contentBuilder.WriteString(text)

		// 推送流式内容（带前缀）
		if streamHandler != nil {
			message := fmt.Sprintf("%s%s", common.SSEMsgAgent2Streaming+":", text)
			streamHandler(message)
		}
		return nil
	}))

	if err != nil {
		log.Printf("OutlineGeneratorAgent：流式调用失败, error=%v", err)
		return "", fmt.Errorf("streaming LLM call failed: %w", err)
	}

	return contentBuilder.String(), nil
}

// getStylePrompt 根据风格获取对应的 Prompt 附加内容
func (a *OutlineGeneratorAgent) getStylePrompt(style string) string {
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
