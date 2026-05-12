package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/tmc/langchaingo/llms"
	"github.com/yupi/ai-passage-creator/internal/agent/agents"
	agentContext "github.com/yupi/ai-passage-creator/internal/agent/context"
	"github.com/yupi/ai-passage-creator/internal/agent/parallel"
	"github.com/yupi/ai-passage-creator/internal/common"
	"github.com/yupi/ai-passage-creator/internal/config"
	"github.com/yupi/ai-passage-creator/internal/model"
	"github.com/yupi/ai-passage-creator/internal/service"
)

// ArticleAgentOrchestrator 文章智能体编排器
// 使用多智能体架构编排文章生成流程，支持并行配图生成
type ArticleAgentOrchestrator struct {
	cfg                    *config.Config
	llm                    llms.Model
	agentLogService        *service.AgentLogService
	sseManager             *common.SSEManager
	imageStrategy          *service.ImageServiceStrategy
	titleGenerator         Agent
	outlineGenerator       Agent
	contentGenerator       Agent
	imageAnalyzer          Agent
	parallelImageGenerator Agent
	contentMerger          Agent
}

// NewArticleAgentOrchestrator 创建文章智能体编排器
func NewArticleAgentOrchestrator(
	cfg *config.Config,
	llm llms.Model,
	agentLogService *service.AgentLogService,
	sseManager *common.SSEManager,
	imageStrategy *service.ImageServiceStrategy,
) *ArticleAgentOrchestrator {
	return &ArticleAgentOrchestrator{
		cfg:                    cfg,
		llm:                    llm,
		agentLogService:        agentLogService,
		sseManager:             sseManager,
		imageStrategy:          imageStrategy,
		titleGenerator:         agents.NewTitleGeneratorAgent(llm, agentLogService),
		outlineGenerator:       agents.NewOutlineGeneratorAgent(llm),
		contentGenerator:       agents.NewContentGeneratorAgent(llm, agentLogService),
		imageAnalyzer:          agents.NewImageAnalyzerAgent(llm, agentLogService),
		parallelImageGenerator: parallel.NewParallelImageGenerator(imageStrategy),
		contentMerger:          agents.NewContentMergerAgent(agentLogService),
	}
}

// ExecutePhase1 阶段1：生成标题方案
func (o *ArticleAgentOrchestrator) ExecutePhase1(ctx context.Context, state *model.ArticleState) error {
	log.Printf("阶段1（多智能体编排）：开始生成标题方案, taskId=%s", state.TaskID)

	// 执行标题生成 Agent
	if err := o.titleGenerator.Execute(ctx, state); err != nil {
		return fmt.Errorf("title generator failed: %w", err)
	}

	// 推送完成消息
	o.sendMessage(state.TaskID, map[string]interface{}{
		"type":         common.SSEMsgAgent1Complete,
		"titleOptions": state.TitleOptions,
	})

	log.Printf("阶段1（多智能体编排）：标题方案生成完成, 数量=%d", len(state.TitleOptions))
	return nil
}

// ExecutePhase2 阶段2：生成大纲
func (o *ArticleAgentOrchestrator) ExecutePhase2(ctx context.Context, state *model.ArticleState, streamHandler func(string)) error {
	log.Printf("阶段2（多智能体编排）：开始生成大纲, taskId=%s", state.TaskID)

	// 将 streamHandler 设置到 context 中
	ctx = agentContext.WithStreamHandler(ctx, agentContext.StreamHandler(streamHandler))

	// 执行大纲生成 Agent
	if err := o.outlineGenerator.Execute(ctx, state); err != nil {
		return fmt.Errorf("outline generator failed: %w", err)
	}

	// 推送完成消息
	o.sendMessage(state.TaskID, map[string]interface{}{
		"type":    common.SSEMsgAgent2Complete,
		"outline": state.Outline.Sections,
	})

	log.Printf("阶段2（多智能体编排）：大纲生成完成, 章节数=%d", len(state.Outline.Sections))
	return nil
}

// ExecutePhase3 阶段3：生成正文+配图
// 流程：正文生成 -> 配图需求分析 -> 并行配图生成 -> 图文合成
func (o *ArticleAgentOrchestrator) ExecutePhase3(ctx context.Context, state *model.ArticleState, streamHandler func(string)) error {
	log.Printf("阶段3（多智能体编排）：开始生成正文+配图, taskId=%s", state.TaskID)

	// 将 streamHandler 设置到 context 中
	ctx = agentContext.WithStreamHandler(ctx, agentContext.StreamHandler(streamHandler))

	// 智能体3：生成正文（流式）
	log.Printf("智能体3（多智能体编排）：开始生成正文, taskId=%s", state.TaskID)
	if err := o.contentGenerator.Execute(ctx, state); err != nil {
		return fmt.Errorf("content generator failed: %w", err)
	}
	o.sendMessage(state.TaskID, map[string]interface{}{
		"type": common.SSEMsgAgent3Complete,
	})

	// 智能体4：分析配图需求
	log.Printf("智能体4（多智能体编排）：开始分析配图需求, taskId=%s", state.TaskID)
	if err := o.imageAnalyzer.Execute(ctx, state); err != nil {
		return fmt.Errorf("image analyzer failed: %w", err)
	}
	o.sendMessage(state.TaskID, map[string]interface{}{
		"type":              common.SSEMsgAgent4Complete,
		"imageRequirements": state.ImageRequirements,
	})

	// 智能体5：并行生成配图
	log.Printf("智能体5（多智能体编排）：开始并行生成配图, taskId=%s", state.TaskID)
	if err := o.parallelImageGenerator.Execute(ctx, state); err != nil {
		return fmt.Errorf("parallel image generator failed: %w", err)
	}
	o.sendMessage(state.TaskID, map[string]interface{}{
		"type":   common.SSEMsgAgent5Complete,
		"images": state.Images,
	})

	// 图文合成
	log.Printf("开始图文合成（多智能体编排）, taskId=%s", state.TaskID)
	if err := o.contentMerger.Execute(ctx, state); err != nil {
		return fmt.Errorf("content merger failed: %w", err)
	}
	o.sendMessage(state.TaskID, map[string]interface{}{
		"type":        common.SSEMsgMergeComplete,
		"fullContent": state.FullContent,
	})

	log.Printf("阶段3（多智能体编排）：正文+配图生成完成, 正文长度=%d, 图片数=%d",
		len(state.FullContent), len(state.Images))
	return nil
}

// sendMessage 发送 SSE 消息
func (o *ArticleAgentOrchestrator) sendMessage(taskID string, data interface{}) {
	// 将数据转换为 JSON 字符串（如果需要）
	switch v := data.(type) {
	case map[string]interface{}:
		// 已经是 map，直接发送
		o.sseManager.Send(taskID, v)
	case string:
		// 如果是字符串，尝试解析为 JSON
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(v), &jsonData); err == nil {
			o.sseManager.Send(taskID, jsonData)
		} else {
			// 如果解析失败，包装为简单消息
			o.sseManager.Send(taskID, map[string]interface{}{"message": v})
		}
	default:
		log.Printf("警告：不支持的消息类型: %T", v)
	}
}
