package service

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
)

// agent2GenerateOutlineStream 智能体2：生成大纲（流式输出）
func (s *ArticleAgentService) agent2GenerateOutlineStream(ctx context.Context, state *model.ArticleState) error {
	// 创建日志记录
	startTime := time.Now()
	agentLog := &model.AgentLog{
		TaskID:    state.TaskID,
		AgentName: "agent2_generate_outline",
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

	// 构建 prompt，根据是否有用户补充描述插入对应部分
	descriptionSection := ""
	if state.UserDescription != "" {
		descriptionSection = strings.ReplaceAll(common.Agent2DescriptionSection, "{userDescription}", state.UserDescription)
	}

	prompt := strings.ReplaceAll(common.Agent2OutlinePrompt, "{mainTitle}", state.Title.MainTitle)
	prompt = strings.ReplaceAll(prompt, "{subTitle}", state.Title.SubTitle)
	prompt = strings.ReplaceAll(prompt, "{descriptionSection}", descriptionSection)
	prompt += s.getStylePrompt(state.Style)
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
			"type":    common.SSEMsgAgent2Streaming,
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

	content := contentBuilder.String()

	var outline model.OutlineResult
	if err := json.Unmarshal([]byte(content), &outline); err != nil {
		log.Printf("智能体2：大纲解析失败, content=%s", content)
		agentLog.Status = "FAILED"
		errMsg := "parse outline failed: " + err.Error()
		agentLog.ErrorMessage = &errMsg
		return fmt.Errorf("parse outline failed: %w", err)
	}

	state.Outline = &outline
	agentLog.Status = "SUCCESS"
	// 将输出数据转换为 JSON 格式
	outputDataJSON, _ := json.Marshal(map[string]interface{}{
		"sectionsCount": len(outline.Sections),
		"message":       fmt.Sprintf("生成 %d 个段落", len(outline.Sections)),
	})
	outputDataStr := string(outputDataJSON)
	agentLog.OutputData = &outputDataStr
	log.Printf("智能体2：大纲生成成功, sections=%d", len(outline.Sections))
	return nil
}

// mergeImagesIntoContent 图文合成：根据占位符将配图插入正文
func (s *ArticleAgentService) mergeImagesIntoContent(state *model.ArticleState) {
	// 创建日志记录
	startTime := time.Now()
	agentLog := &model.AgentLog{
		TaskID:    state.TaskID,
		AgentName: "agent6_merge_content",
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

	// 使用包含占位符的正文（Agent4 生成）
	content := state.ContentWithPlaceholders
	images := state.Images

	// 将输入数据转换为 JSON 格式
	inputDataJSON, _ := json.Marshal(map[string]interface{}{
		"placeholderCount": len(images),
		"imagesCount":      len(images),
	})
	inputDataStr := string(inputDataJSON)
	agentLog.InputData = &inputDataStr

	log.Printf("图文合成：ContentWithPlaceholders长度=%d, 图片数量=%d", len(content), len(images))

	if len(images) == 0 {
		state.FullContent = content
		agentLog.Status = "SUCCESS"
		// 将输出数据转换为 JSON 格式
		outputDataJSON, _ := json.Marshal(map[string]interface{}{
			"message": "无配图，直接使用原文",
		})
		outputDataStr := string(outputDataJSON)
		agentLog.OutputData = &outputDataStr
		return
	}

	fullContent := content

	// 遍历所有配图，根据占位符替换为实际图片
	replacedCount := 0
	for i, image := range images {
		placeholderID := image.PlaceholderID
		log.Printf("图文合成：处理图片[%d] placeholderID=%s, url=%s", i, placeholderID, image.URL)

		if placeholderID != "" {
			// 检查占位符是否存在
			if strings.Contains(fullContent, placeholderID) {
				imageMarkdown := fmt.Sprintf("![%s](%s)", image.Description, image.URL)
				fullContent = strings.ReplaceAll(fullContent, placeholderID, imageMarkdown)
				replacedCount++
				log.Printf("图文合成：成功替换占位符 %s", placeholderID)
			} else {
				log.Printf("图文合成：警告 - 占位符未找到: %s", placeholderID)
			}
		} else {
			log.Printf("图文合成：警告 - 图片[%d]的placeholderID为空", i)
		}
	}

	state.FullContent = fullContent
	agentLog.Status = "SUCCESS"
	// 将输出数据转换为 JSON 格式
	outputDataJSON, _ := json.Marshal(map[string]interface{}{
		"replacedCount": replacedCount,
		"message":       fmt.Sprintf("成功替换 %d 个占位符", replacedCount),
	})
	outputDataStr := string(outputDataJSON)
	agentLog.OutputData = &outputDataStr
	log.Printf("图文合成完成, fullContentLength=%d, 成功替换=%d个占位符", len(fullContent), replacedCount)
}
