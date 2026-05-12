package context

import "context"

// streamHandlerKey 是用于在 context 中存储 StreamHandler 的 key 类型
type streamHandlerKey struct{}

// StreamHandler 流式输出处理器函数类型
// 接收字符串消息并进行处理（如通过 SSE 推送）
type StreamHandler func(message string)

// WithStreamHandler 将 StreamHandler 设置到 context 中
func WithStreamHandler(ctx context.Context, handler StreamHandler) context.Context {
	return context.WithValue(ctx, streamHandlerKey{}, handler)
}

// GetStreamHandler 从 context 中获取 StreamHandler
// 如果不存在则返回 nil
func GetStreamHandler(ctx context.Context) StreamHandler {
	if handler, ok := ctx.Value(streamHandlerKey{}).(StreamHandler); ok {
		return handler
	}
	return nil
}

// SendMessage 通过 context 中的 StreamHandler 发送消息
// 如果 handler 不存在则忽略
func SendMessage(ctx context.Context, message string) {
	if handler := GetStreamHandler(ctx); handler != nil {
		handler(message)
	}
}
