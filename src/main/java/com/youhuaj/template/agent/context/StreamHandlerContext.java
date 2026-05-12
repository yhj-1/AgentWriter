package com.yupi.template.agent.context;

import java.util.function.Consumer;

/**
 * 流式输出处理器上下文
 * 使用 ThreadLocal 保存 streamHandler，避免将其放入 StateGraph 状态中（无法序列化）
 *
 * @author AI Passage Creator
 */
public class StreamHandlerContext {

    private static final ThreadLocal<Consumer<String>> STREAM_HANDLER = new ThreadLocal<>();

    /**
     * 设置流式输出处理器
     *
     * @param handler 处理器
     */
    public static void set(Consumer<String> handler) {
        STREAM_HANDLER.set(handler);
    }

    /**
     * 获取流式输出处理器
     *
     * @return 处理器，可能为 null
     */
    public static Consumer<String> get() {
        return STREAM_HANDLER.get();
    }

    /**
     * 清理上下文
     * 务必在使用完毕后调用，避免内存泄漏
     */
    public static void clear() {
        STREAM_HANDLER.remove();
    }

    /**
     * 发送消息到流式输出
     * 如果 handler 不存在则忽略
     *
     * @param message 消息内容
     */
    public static void send(String message) {
        Consumer<String> handler = STREAM_HANDLER.get();
        if (handler != null && message != null) {
            handler.accept(message);
        }
    }
}
