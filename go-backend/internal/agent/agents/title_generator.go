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

// TitleGeneratorAgent 标题生成 Agent
// 根据选题生成 3-5 个爆款标题方案
type TitleGeneratorAgent struct {
	llm             llms.Model
	agentLogService *service.AgentLogService
}

// NewTitleGeneratorAgent 创建标题生成 Agent
func NewTitleGeneratorAgent(llm llms.Model, agentLogService *service.AgentLogService) *TitleGeneratorAgent {
	return &TitleGeneratorAgent{
		llm:             llm,
		agentLogService: agentLogService,
	}
}

// Execute 执行标题生成任务
func (a *TitleGeneratorAgent) Execute(ctx context.Context, state *model.ArticleState) error {
	log.Printf("TitleGeneratorAgent 开始执行: topic=%s, style=%s", state.Topic, state.Style)

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
		a.agentLogService.SaveLogAsync(agentLog)
	}()

	// 构建 prompt
	prompt := strings.ReplaceAll(common.Agent1TitlePrompt, "{topic}", state.Topic)
	prompt += a.getStylePrompt(state.Style)
	agentLog.Prompt = &prompt

	log.Printf("TitleGeneratorAgent：发送请求到 LLM, promptLength=%d", len(prompt))

	// 调用 LLM
	content, err := llms.GenerateFromSinglePrompt(ctx, a.llm, prompt)
	if err != nil {
		log.Printf("TitleGeneratorAgent：LLM 调用失败, error=%v", err)
		agentLog.Status = "FAILED"
		errMsg := err.Error()
		agentLog.ErrorMessage = &errMsg
		return fmt.Errorf("LLM call failed: %w", err)
	}

	log.Printf("TitleGeneratorAgent：收到响应, contentLength=%d", len(content))

	// 解析标题方案列表
	var titleOptions []model.TitleOption
	if err := json.Unmarshal([]byte(content), &titleOptions); err != nil {
		log.Printf("TitleGeneratorAgent：标题方案解析失败, content=%s", content)
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
	log.Printf("TitleGeneratorAgent：标题方案生成成功, optionsCount=%d", len(titleOptions))
	return nil
}

// getStylePrompt 根据风格获取对应的 Prompt 附加内容
func (a *TitleGeneratorAgent) getStylePrompt(style string) string {
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
