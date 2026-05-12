package agent

import (
	"context"

	"github.com/yupi/ai-passage-creator/internal/model"
)

// Agent 智能体接口
// 所有智能体都实现此接口，负责文章生成流程中的某个具体任务
type Agent interface {
	// Execute 执行智能体任务
	// ctx: 上下文，可能包含 StreamHandler
	// state: 文章状态对象，智能体会修改此状态
	Execute(ctx context.Context, state *model.ArticleState) error
}
