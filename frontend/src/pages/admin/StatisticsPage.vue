<template>
  <div class="statistics-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-container">
        <div class="header-content">
          <h1 class="page-title">数据分析</h1>
          <p class="page-subtitle">系统运营数据概览</p>
        </div>
        <a-button @click="loadData" :loading="loading" class="refresh-btn">
          <template #icon>
            <ReloadOutlined />
          </template>
          刷新数据
        </a-button>
      </div>
    </div>

    <div class="container">
      <a-spin :spinning="loading" tip="加载中...">
        <!-- 核心指标卡片 -->
        <div class="stats-grid">
          <div class="stat-card">
            <div class="stat-icon" style="background: rgba(34, 197, 94, 0.1)">
              <FileTextOutlined style="color: var(--color-primary)" />
            </div>
            <div class="stat-content">
              <div class="stat-label">今日创作</div>
              <div class="stat-value">{{ stats?.todayCount ?? 0 }}</div>
            </div>
          </div>

          <div class="stat-card">
            <div class="stat-icon" style="background: rgba(59, 130, 246, 0.1)">
              <BarChartOutlined style="color: #3B82F6" />
            </div>
            <div class="stat-content">
              <div class="stat-label">本周创作</div>
              <div class="stat-value">{{ stats?.weekCount ?? 0 }}</div>
            </div>
          </div>

          <div class="stat-card">
            <div class="stat-icon" style="background: rgba(168, 85, 247, 0.1)">
              <RiseOutlined style="color: #A855F7" />
            </div>
            <div class="stat-content">
              <div class="stat-label">本月创作</div>
              <div class="stat-value">{{ stats?.monthCount ?? 0 }}</div>
            </div>
          </div>

          <div class="stat-card">
            <div class="stat-icon" style="background: rgba(234, 179, 8, 0.1)">
              <CheckCircleOutlined style="color: #EAB308" />
            </div>
            <div class="stat-content">
              <div class="stat-label">成功率</div>
              <div class="stat-value">{{ (stats?.successRate ?? 0).toFixed(1) }}%</div>
            </div>
          </div>
        </div>

        <!-- 图表区域 -->
        <div class="charts-grid">
          <!-- 创作趋势图 -->
          <a-card :bordered="false" class="chart-card">
            <h3 class="chart-title">
              <LineChartOutlined />
              创作趋势
            </h3>
            <div ref="trendChartRef" class="chart-container"></div>
          </a-card>

          <!-- 性能统计图 -->
          <a-card :bordered="false" class="chart-card">
            <h3 class="chart-title">
              <ThunderboltOutlined />
              性能统计
            </h3>
            <div class="performance-stats">
              <div class="perf-item">
                <span class="perf-label">平均耗时</span>
                <span class="perf-value">{{ formatDuration(stats?.avgDurationMs ?? 0) }}</span>
              </div>
              <a-divider />
              <div class="perf-item">
                <span class="perf-label">总创作数</span>
                <span class="perf-value">{{ stats?.totalCount ?? 0 }}</span>
              </div>
            </div>
          </a-card>
        </div>

        <!-- 用户统计 -->
        <div class="charts-grid">
          <a-card :bordered="false" class="chart-card">
            <h3 class="chart-title">
              <TeamOutlined />
              用户分析
            </h3>
            <div ref="userChartRef" class="chart-container"></div>
          </a-card>

          <a-card :bordered="false" class="chart-card">
            <h3 class="chart-title">
              <CrownOutlined />
              配额使用情况
            </h3>
            <div ref="quotaChartRef" class="chart-container"></div>
          </a-card>
        </div>
      </a-spin>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { message } from 'ant-design-vue'
import {
  FileTextOutlined,
  BarChartOutlined,
  RiseOutlined,
  CheckCircleOutlined,
  LineChartOutlined,
  ThunderboltOutlined,
  TeamOutlined,
  CrownOutlined,
  ReloadOutlined
} from '@ant-design/icons-vue'
import { getStatistics } from '@/api/statisticsController'
import * as echarts from 'echarts'
import type { EChartsOption } from 'echarts'

const loading = ref(false)
const stats = ref<API.StatisticsVO | null>(null)

// ECharts 实例
const trendChartRef = ref<HTMLElement>()
const userChartRef = ref<HTMLElement>()
const quotaChartRef = ref<HTMLElement>()
let trendChart: echarts.ECharts | null = null
let userChart: echarts.ECharts | null = null
let quotaChart: echarts.ECharts | null = null

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await getStatistics()
    stats.value = res.data.data || null

    // 渲染图表
    setTimeout(() => {
      renderTrendChart()
      renderUserChart()
      renderQuotaChart()
    }, 100)
  } catch (error) {
    message.error((error as Error).message || '加载数据失败')
  } finally {
    loading.value = false
  }
}

// 渲染创作趋势图
const renderTrendChart = () => {
  if (!trendChartRef.value || !stats.value) return

  if (!trendChart) {
    trendChart = echarts.init(trendChartRef.value)
  }

  const option: EChartsOption = {
    tooltip: {
      trigger: 'axis'
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: ['今日', '本周', '本月', '总计'],
      axisLine: {
        lineStyle: {
          color: '#E2E8F0'
        }
      },
      axisLabel: {
        color: '#64748B'
      }
    },
    yAxis: {
      type: 'value',
      axisLine: {
        show: false
      },
      splitLine: {
        lineStyle: {
          color: '#F1F5F9'
        }
      },
      axisLabel: {
        color: '#64748B'
      }
    },
    series: [
      {
        name: '创作数量',
        type: 'bar',
        data: [
          stats.value.todayCount ?? 0,
          stats.value.weekCount ?? 0,
          stats.value.monthCount ?? 0,
          stats.value.totalCount ?? 0
        ],
        itemStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: '#4ADE80' },
            { offset: 1, color: '#22C55E' }
          ]),
          borderRadius: [4, 4, 0, 0]
        },
        barWidth: '40%'
      }
    ]
  }

  trendChart.setOption(option)
}

// 渲染用户分析图
const renderUserChart = () => {
  if (!userChartRef.value || !stats.value) return

  if (!userChart) {
    userChart = echarts.init(userChartRef.value)
  }

  const option: EChartsOption = {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} ({d}%)'
    },
    legend: {
      orient: 'vertical',
      right: '10%',
      top: 'center',
      textStyle: {
        color: '#64748B'
      }
    },
    series: [
      {
        name: '用户分布',
        type: 'pie',
        radius: ['40%', '70%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 8,
          borderColor: '#fff',
          borderWidth: 2
        },
        label: {
          show: false
        },
        emphasis: {
          label: {
            show: true,
            fontSize: 16,
            fontWeight: 'bold'
          }
        },
        data: [
          {
            value: stats.value.vipUserCount ?? 0,
            name: 'VIP 会员',
            itemStyle: { color: '#22C55E' }
          },
          {
            value: stats.value.activeUserCount ?? 0,
            name: '活跃用户',
            itemStyle: { color: '#3B82F6' }
          },
          {
            value: (stats.value.totalUserCount ?? 0) - (stats.value.activeUserCount ?? 0) - (stats.value.vipUserCount ?? 0),
            name: '其他用户',
            itemStyle: { color: '#94A3B8' }
          }
        ]
      }
    ]
  }

  userChart.setOption(option)
}

// 渲染配额使用图
const renderQuotaChart = () => {
  if (!quotaChartRef.value || !stats.value) return

  if (!quotaChart) {
    quotaChart = echarts.init(quotaChartRef.value)
  }

  const totalQuota = (stats.value.totalUserCount ?? 0) * 5
  const usedQuota = stats.value.quotaUsed ?? 0
  const remainingQuota = Math.max(0, totalQuota - usedQuota)

  const option: EChartsOption = {
    tooltip: {
      trigger: 'item'
    },
    series: [
      {
        name: '配额统计',
        type: 'pie',
        radius: '70%',
        center: ['50%', '50%'],
        data: [
          {
            value: usedQuota,
            name: '已使用',
            itemStyle: { color: '#EF4444' }
          },
          {
            value: remainingQuota,
            name: '剩余',
            itemStyle: { color: '#22C55E' }
          }
        ],
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)'
          }
        },
        label: {
          formatter: '{b}: {c}'
        }
      }
    ]
  }

  quotaChart.setOption(option)
}

// 格式化耗时
const formatDuration = (ms: number) => {
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

// 响应式处理
const handleResize = () => {
  trendChart?.resize()
  userChart?.resize()
  quotaChart?.resize()
}

onMounted(() => {
  loadData()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  trendChart?.dispose()
  userChart?.dispose()
  quotaChart?.dispose()
})
</script>

<style scoped lang="scss">
.statistics-page {
  background: var(--color-background-secondary);
  min-height: calc(100vh - 64px);
  padding-bottom: 60px;

  .page-header {
    background: var(--gradient-hero);
    padding: 32px 20px;
    margin-bottom: 24px;
  }

  .header-container {
    max-width: 1400px;
    margin: 0 auto;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .header-content {
    color: var(--color-text);
  }

  .page-title {
    font-size: 28px;
    font-weight: 700;
    margin: 0 0 6px;
    letter-spacing: -0.5px;
    color: var(--color-text);
  }

  .page-subtitle {
    font-size: 14px;
    color: var(--color-text-secondary);
    margin: 0;
  }

  .refresh-btn {
    height: 38px;
    border-radius: var(--radius-md);
    font-weight: 500;
  }

  .container {
    max-width: 1400px;
    margin: 0 auto;
    padding: 0 20px;
  }

  /* 统计卡片网格 */
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 20px;
    margin-bottom: 24px;
  }

  .stat-card {
    background: white;
    border-radius: var(--radius-lg);
    padding: 24px;
    border: 1px solid var(--color-border);
    display: flex;
    align-items: center;
    gap: 16px;
    transition: all var(--transition-normal);

    &:hover {
      box-shadow: var(--shadow-card-hover);
      transform: translateY(-2px);
    }

    .stat-icon {
      width: 56px;
      height: 56px;
      border-radius: var(--radius-lg);
      display: flex;
      align-items: center;
      justify-content: center;
      flex-shrink: 0;

      .anticon {
        font-size: 24px;
      }
    }

    .stat-content {
      flex: 1;
    }

    .stat-label {
      font-size: 13px;
      color: var(--color-text-secondary);
      margin-bottom: 6px;
      display: block;
    }

    .stat-value {
      font-size: 28px;
      font-weight: 700;
      color: var(--color-text);
      line-height: 1;
    }
  }

  /* 图表网格 */
  .charts-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 20px;
    margin-bottom: 24px;
  }

  .chart-card {
    border-radius: var(--radius-lg);
    border: 1px solid var(--color-border);
    overflow: hidden;

    :deep(.ant-card-body) {
      padding: 24px;
    }

    .chart-title {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 16px;
      font-weight: 600;
      margin: 0 0 20px;
      color: var(--color-text);

      .anticon {
        color: var(--color-primary);
        font-size: 18px;
      }
    }

    .chart-container {
      width: 100%;
      height: 300px;
    }

    .performance-stats {
      padding: 20px 0;

      .perf-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 16px 0;

        .perf-label {
          font-size: 14px;
          color: var(--color-text-secondary);
        }

        .perf-value {
          font-size: 24px;
          font-weight: 600;
          color: var(--color-primary);
        }
      }
    }
  }

  /* 响应式 */
  @media (max-width: 1200px) {
    .stats-grid {
      grid-template-columns: repeat(2, 1fr);
    }

    .charts-grid {
      grid-template-columns: 1fr;
    }
  }

  @media (max-width: 768px) {
    .page-header {
      padding: 24px 16px;
    }

    .header-container {
      flex-direction: column;
      gap: 16px;
      align-items: stretch;
    }

    .refresh-btn {
      width: 100%;
    }

    .stats-grid {
      grid-template-columns: 1fr;
      gap: 12px;
    }

    .stat-card {
      padding: 20px;
    }

    .chart-card {
      :deep(.ant-card-body) {
        padding: 20px;
      }

      .chart-container {
        height: 250px;
      }
    }
  }
}
</style>
