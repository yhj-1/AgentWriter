package com.yupi.template.service.impl;

import com.mybatisflex.core.query.QueryWrapper;
import com.yupi.template.constant.UserConstant;
import com.yupi.template.mapper.ArticleMapper;
import com.yupi.template.mapper.UserMapper;
import com.yupi.template.model.entity.Article;
import com.yupi.template.model.entity.User;
import com.yupi.template.model.enums.ArticleStatusEnum;
import com.yupi.template.model.vo.StatisticsVO;
import com.yupi.template.service.AgentLogService;
import com.yupi.template.service.StatisticsService;
import jakarta.annotation.Resource;
import lombok.extern.slf4j.Slf4j;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Service;

import java.time.LocalDate;
import java.time.LocalDateTime;
import java.time.LocalTime;
import java.util.List;
import java.util.concurrent.TimeUnit;

/**
 * 统计服务实现
 *
 * @author <a href="https://codefather.cn">编程导航学习圈</a>
 */
@Service
@Slf4j
public class StatisticsServiceImpl implements StatisticsService {

    private static final String STATISTICS_CACHE_KEY = "statistics:overview";
    private static final long CACHE_EXPIRE_HOURS = 1L;

    @Resource
    private ArticleMapper articleMapper;

    @Resource
    private UserMapper userMapper;

    @Resource
    private AgentLogService agentLogService;

    @Resource
    private RedisTemplate<String, Object> redisTemplate;

    @Override
    public StatisticsVO getStatistics() {
        // 先从缓存获取
        StatisticsVO cachedStats = (StatisticsVO) redisTemplate.opsForValue().get(STATISTICS_CACHE_KEY);
        if (cachedStats != null) {
            log.info("从缓存获取统计数据");
            return cachedStats;
        }

        // 缓存不存在，重新计算
        // 今日创作数量
        Long todayCount = countArticlesByDateRange(getTodayStart(), LocalDateTime.now());

        // 本周创作数量
        Long weekCount = countArticlesByDateRange(getWeekStart(), LocalDateTime.now());

        // 本月创作数量
        Long monthCount = countArticlesByDateRange(getMonthStart(), LocalDateTime.now());

        // 总创作数量
        Long totalCount = countTotalArticles();

        // 成功率统计
        Double successRate = calculateSuccessRate();

        // 平均耗时统计
        Integer avgDurationMs = calculateAvgDuration();

        // 活跃用户统计（本周有创作的用户）
        Long activeUserCount = countActiveUsers(getWeekStart());

        // 总用户数
        Long totalUserCount = countTotalUsers();

        // VIP 用户数
        Long vipUserCount = countVipUsers();

        // 配额使用情况（总配额 - 剩余配额）
        Long quotaUsed = calculateQuotaUsed();

        StatisticsVO statistics = StatisticsVO.builder()
                .todayCount(todayCount)
                .weekCount(weekCount)
                .monthCount(monthCount)
                .totalCount(totalCount)
                .successRate(successRate)
                .avgDurationMs(avgDurationMs)
                .activeUserCount(activeUserCount)
                .totalUserCount(totalUserCount)
                .vipUserCount(vipUserCount)
                .quotaUsed(quotaUsed)
                .build();

        // 存入缓存，1 小时过期
        redisTemplate.opsForValue().set(STATISTICS_CACHE_KEY, statistics, CACHE_EXPIRE_HOURS, TimeUnit.HOURS);
        log.info("统计数据已缓存，过期时间: {} 小时", CACHE_EXPIRE_HOURS);

        return statistics;
    }

    /**
     * 统计指定时间范围内的文章数量
     */
    private Long countArticlesByDateRange(LocalDateTime start, LocalDateTime end) {
        QueryWrapper queryWrapper = QueryWrapper.create()
                .ge("createTime", start)
                .le("createTime", end);
        return articleMapper.selectCountByQuery(queryWrapper);
    }

    /**
     * 统计总文章数量
     */
    private Long countTotalArticles() {
        return articleMapper.selectCountByQuery(QueryWrapper.create());
    }

    /**
     * 计算成功率
     */
    private Double calculateSuccessRate() {
        Long totalCount = countTotalArticles();
        if (totalCount == 0) {
            return 0.0;
        }

        QueryWrapper successWrapper = QueryWrapper.create()
                .eq("status", ArticleStatusEnum.COMPLETED.getValue());
        Long successCount = articleMapper.selectCountByQuery(successWrapper);

        return (successCount.doubleValue() / totalCount.doubleValue()) * 100;
    }

    /**
     * 计算平均耗时（从创建到完成的平均时间）
     */
    private Integer calculateAvgDuration() {
        // 查询所有已完成的文章，计算 createTime 到 completedTime 的平均耗时
        QueryWrapper queryWrapper = QueryWrapper.create()
                .eq("status", ArticleStatusEnum.COMPLETED.getValue())
                .isNotNull("completedTime");
        
        try {
            List<Article> completedArticles = articleMapper.selectListByQuery(queryWrapper);
            if (completedArticles == null || completedArticles.isEmpty()) {
                return 0;
            }

            // 计算每篇文章的耗时（毫秒）
            double avgDuration = completedArticles.stream()
                    .filter(article -> article.getCreateTime() != null && article.getCompletedTime() != null)
                    .mapToLong(article -> {
                        long createMillis = java.sql.Timestamp.valueOf(article.getCreateTime()).getTime();
                        long completedMillis = java.sql.Timestamp.valueOf(article.getCompletedTime()).getTime();
                        return completedMillis - createMillis;
                    })
                    .average()
                    .orElse(0.0);

            return (int) avgDuration;
        } catch (Exception e) {
            log.warn("计算平均耗时失败", e);
        }
        
        return 0;
    }

    /**
     * 统计活跃用户数（本周有创作的用户）
     */
    private Long countActiveUsers(LocalDateTime weekStart) {
        QueryWrapper queryWrapper = QueryWrapper.create()
                .ge("createTime", weekStart);
        
        try {
            List<Article> articles = articleMapper.selectListByQuery(queryWrapper);
            // 统计去重后的用户数
            return articles.stream()
                    .map(Article::getUserId)
                    .distinct()
                    .count();
        } catch (Exception e) {
            log.warn("统计活跃用户失败", e);
        }
        
        return 0L;
    }

    /**
     * 统计总用户数
     */
    private Long countTotalUsers() {
        return userMapper.selectCountByQuery(QueryWrapper.create());
    }

    /**
     * 统计 VIP 用户数
     */
    private Long countVipUsers() {
        QueryWrapper queryWrapper = QueryWrapper.create()
                .eq("userRole", UserConstant.VIP_ROLE);
        return userMapper.selectCountByQuery(queryWrapper);
    }

    /**
     * 计算配额使用量
     */
    private Long calculateQuotaUsed() {
        // 配额使用量 = (普通用户数 * 初始配额) - 当前剩余配额总和
        QueryWrapper normalUserWrapper = QueryWrapper.create()
                .eq("userRole", UserConstant.DEFAULT_ROLE);
        
        try {
            List<User> normalUsers = userMapper.selectListByQuery(normalUserWrapper);
            Long normalUserCount = (long) normalUsers.size();
            
            // 统计剩余配额总和
            long remainingQuota = normalUsers.stream()
                    .mapToInt(user -> user.getQuota() != null ? user.getQuota() : 0)
                    .sum();
            
            return (normalUserCount * UserConstant.DEFAULT_QUOTA) - remainingQuota;
        } catch (Exception e) {
            log.warn("计算配额使用量失败", e);
        }
        
        return 0L;
    }

    /**
     * 获取今天开始时间
     */
    private LocalDateTime getTodayStart() {
        return LocalDateTime.of(LocalDate.now(), LocalTime.MIN);
    }

    /**
     * 获取本周开始时间（周一）
     */
    private LocalDateTime getWeekStart() {
        LocalDate today = LocalDate.now();
        LocalDate monday = today.minusDays(today.getDayOfWeek().getValue() - 1);
        return LocalDateTime.of(monday, LocalTime.MIN);
    }

    /**
     * 获取本月开始时间
     */
    private LocalDateTime getMonthStart() {
        LocalDate today = LocalDate.now();
        LocalDate firstDay = today.withDayOfMonth(1);
        return LocalDateTime.of(firstDay, LocalTime.MIN);
    }
}
