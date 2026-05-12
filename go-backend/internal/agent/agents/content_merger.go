package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/yupi/ai-passage-creator/internal/model"
	"github.com/yupi/ai-passage-creator/internal/service"
)

// ContentMergerAgent 图文合成 Agent
// 将配图插入到正文的相应位置（通过占位符替换）
type ContentMergerAgent struct {
	agentLogService *service.AgentLogService
}

// NewContentMergerAgent 创建图文合成 Agent
func NewContentMergerAgent(agentLogService *service.AgentLogService) *ContentMergerAgent {
	return &ContentMergerAgent{
		agentLogService: agentLogService,
	}
}

// Execute 执行图文合成任务
func (a *ContentMergerAgent) Execute(ctx context.Context, state *model.ArticleState) error {
	log.Printf("ContentMergerAgent 开始执行: 正文长度=%d, 图片数量=%d",
		len(state.ContentWithPlaceholders), len(state.Images))

	// 创建日志记录
	startTime := time.Now()
	agentLog := &model.AgentLog{
		TaskID:    state.TaskID,
		AgentName: "content_merger",
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

	// 将输入数据转换为 JSON 格式
	inputDataJSON, _ := json.Marshal(map[string]interface{}{
		"contentLength": len(state.ContentWithPlaceholders),
		"imagesCount":   len(state.Images),
	})
	inputDataStr := string(inputDataJSON)
	agentLog.InputData = &inputDataStr

	// 执行图文合成
	fullContent := a.mergeImagesIntoContent(state.ContentWithPlaceholders, state.Images)
	state.FullContent = fullContent

	agentLog.Status = "SUCCESS"
	// 将输出数据转换为 JSON 格式
	outputDataJSON, _ := json.Marshal(map[string]interface{}{
		"fullContentLength": len(fullContent),
		"message":           fmt.Sprintf("完整内容长度: %d 字符", len(fullContent)),
	})
	outputDataStr := string(outputDataJSON)
	agentLog.OutputData = &outputDataStr

	log.Printf("ContentMergerAgent：图文合成完成, fullContentLength=%d", len(fullContent))
	return nil
}

// mergeImagesIntoContent 将配图插入正文（使用占位符替换）
func (a *ContentMergerAgent) mergeImagesIntoContent(content string, images []model.ImageResult) string {
	if len(images) == 0 {
		return content
	}

	fullContent := content

	// 遍历所有配图，根据占位符替换为实际图片
	for _, image := range images {
		placeholder := image.PlaceholderID
		log.Printf("处理图片: position=%d, placeholderId=%s, url=%s",
			image.Position, placeholder, image.URL)

		if placeholder != "" {
			description := image.Description
			if description == "" {
				description = "配图"
			}
			imageMarkdown := fmt.Sprintf("![%s](%s)", description, image.URL)

			if strings.Contains(fullContent, placeholder) {
				fullContent = strings.Replace(fullContent, placeholder, imageMarkdown, 1)
				log.Printf("成功替换占位符: %s -> %s", placeholder, truncate(imageMarkdown, 50))
			} else {
				log.Printf("正文中未找到占位符: %s", placeholder)
			}
		} else {
			log.Printf("图片 position=%d 的 placeholderId 为空", image.Position)
		}
	}

	return fullContent
}

// truncate 截断字符串用于日志
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
