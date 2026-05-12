package com.yupi.template.service;

import com.yupi.template.model.vo.StatisticsVO;

/**
 * 统计服务
 *
 * @author <a href="https://codefather.cn">编程导航学习圈</a>
 */
public interface StatisticsService {

    /**
     * 获取系统统计数据
     *
     * @return 统计数据
     */
    StatisticsVO getStatistics();
}
