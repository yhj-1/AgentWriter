package com.yupi.template.service;

import com.yupi.template.model.entity.User;

/**
 * 配额服务接口
 *
 * @author <a href="https://codefather.cn">编程导航学习圈</a>
 */
public interface QuotaService {

    /**
     * 检查用户是否有足够的配额
     *
     * @param user 用户
     * @return 是否有配额
     */
    boolean hasQuota(User user);

    /**
     * 消耗配额（扣减1次）
     *
     * @param user 用户
     */
    void consumeQuota(User user);

    /**
     * 检查并消耗配额（原子操作）
     * 如果配额不足会抛出异常
     *
     * @param user 用户
     */
    void checkAndConsumeQuota(User user);
}
