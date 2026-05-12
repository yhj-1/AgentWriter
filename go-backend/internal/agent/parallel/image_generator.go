package parallel

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	agentContext "github.com/yupi/ai-passage-creator/internal/agent/context"
	"github.com/yupi/ai-passage-creator/internal/common"
	"github.com/yupi/ai-passage-creator/internal/model"
	"github.com/yupi/ai-passage-creator/internal/service"
)

// ParallelImageGenerator 并行图片生成器
// 根据 imageSource 分组，并行执行不同类型的图片生成任务
type ParallelImageGenerator struct {
	imageStrategy *service.ImageServiceStrategy
}

// NewParallelImageGenerator 创建并行图片生成器
func NewParallelImageGenerator(imageStrategy *service.ImageServiceStrategy) *ParallelImageGenerator {
	return &ParallelImageGenerator{
		imageStrategy: imageStrategy,
	}
}

// Execute 执行并行图片生成任务
func (g *ParallelImageGenerator) Execute(ctx context.Context, state *model.ArticleState) error {
	log.Printf("ParallelImageGenerator 开始执行: 配图需求数量=%d", len(state.ImageRequirements))

	if len(state.ImageRequirements) == 0 {
		log.Printf("没有配图需求，跳过图片生成")
		state.Images = []model.ImageResult{}
		return nil
	}

	// 获取流式处理器
	streamHandler := agentContext.GetStreamHandler(ctx)

	// 按 imageSource 分组
	groupedBySource := g.groupByImageSource(state.ImageRequirements)

	log.Printf("配图需求按类型分组: %v", g.getGroupSummary(groupedBySource))

	// 并行执行不同类型的图片生成
	allImages := g.executeParallel(ctx, groupedBySource, streamHandler)

	// 按 position 排序
	g.sortByPosition(allImages)

	state.Images = allImages
	log.Printf("ParallelImageGenerator 执行完成: 成功生成 %d 张图片", len(allImages))
	return nil
}

// groupByImageSource 按 imageSource 分组
func (g *ParallelImageGenerator) groupByImageSource(requirements []model.ImageRequirement) map[string][]model.ImageRequirement {
	grouped := make(map[string][]model.ImageRequirement)
	for _, req := range requirements {
		source := req.ImageSource
		grouped[source] = append(grouped[source], req)
	}
	return grouped
}

// getGroupSummary 获取分组摘要（用于日志）
func (g *ParallelImageGenerator) getGroupSummary(grouped map[string][]model.ImageRequirement) map[string]int {
	summary := make(map[string]int)
	for source, reqs := range grouped {
		summary[source] = len(reqs)
	}
	return summary
}

// executeParallel 并行执行图片生成任务
// 不同 imageSource 类型并行执行，同一类型内部串行执行
func (g *ParallelImageGenerator) executeParallel(ctx context.Context, groupedBySource map[string][]model.ImageRequirement, streamHandler agentContext.StreamHandler) []model.ImageResult {
	// 使用 WaitGroup 等待所有 goroutine 完成
	var wg sync.WaitGroup
	// 使用 Mutex 保护共享的结果切片
	var mu sync.Mutex
	allImages := []model.ImageResult{}

	// 为每种 imageSource 创建异步任务
	for imageSource, requirements := range groupedBySource {
		wg.Add(1)
		// 闭包捕获变量
		go func(source string, reqs []model.ImageRequirement) {
			defer wg.Done()

			log.Printf("开始处理 %s 类型的图片，数量: %d", source, len(reqs))

			// 同一类型内部串行执行
			for _, req := range reqs {
				imageResult := g.generateSingleImage(ctx, req, streamHandler)
				if imageResult != nil {
					// 加锁保护共享切片
					mu.Lock()
					allImages = append(allImages, *imageResult)
					mu.Unlock()
				}
			}

			log.Printf("完成处理 %s 类型的图片", source)
		}(imageSource, requirements)
	}

	// 等待所有任务完成
	wg.Wait()

	return allImages
}

// generateSingleImage 生成单张图片
func (g *ParallelImageGenerator) generateSingleImage(ctx context.Context, req model.ImageRequirement, streamHandler agentContext.StreamHandler) *model.ImageResult {
	log.Printf("开始生成图片: position=%d, imageSource=%s, keywords=%s",
		req.Position, req.ImageSource, req.Keywords)

	// 构建图片请求对象
	imageRequest := &model.ImageRequest{
		Keywords: req.Keywords,
		Prompt:   req.Prompt,
		Position: req.Position,
		Type:     req.Type,
	}

	// 使用策略模式获取图片并统一上传到 COS
	result, err := g.imageStrategy.GetImageAndUpload(req.ImageSource, imageRequest)
	if err != nil {
		log.Printf("图片生成失败: imageSource=%s, position=%d, error=%v",
			req.ImageSource, req.Position, err)
		return nil
	}

	cosURL := result.URL
	method := result.Method

	// 创建配图结果（URL 已经是 COS 地址）
	imageResult := &model.ImageResult{
		Position:      req.Position,
		URL:           cosURL,
		Method:        method,
		Keywords:      req.Keywords,
		SectionTitle:  req.SectionTitle,
		Description:   req.Type,
		PlaceholderID: req.PlaceholderID,
	}

	// 推送单张配图完成（通过流式处理器）
	if streamHandler != nil {
		messageData := map[string]interface{}{
			"type":  common.SSEMsgImageComplete,
			"image": imageResult,
		}
		messageJSON, _ := json.Marshal(messageData)
		streamHandler(string(messageJSON))
	}

	log.Printf("图片生成成功: position=%d, method=%s, cosUrl=%s",
		req.Position, method, cosURL)

	return imageResult
}

// sortByPosition 按 position 排序
func (g *ParallelImageGenerator) sortByPosition(images []model.ImageResult) {
	// 简单冒泡排序（图片数量不多，性能足够）
	n := len(images)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if images[j].Position > images[j+1].Position {
				images[j], images[j+1] = images[j+1], images[j]
			}
		}
	}
}
