<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useLoginUserStore } from '@/stores/loginUser'
import { listArticle } from '@/api/articleController'
import dayjs from 'dayjs'
import {
  RocketOutlined,
  FileTextOutlined,
  OrderedListOutlined,
  EditOutlined,
  PictureOutlined,
  ThunderboltOutlined,
  ClockCircleOutlined,
  RightOutlined
} from '@ant-design/icons-vue'

const router = useRouter()
const loginUserStore = useLoginUserStore()

// 输入框
const topic = ref('')

// 最近文章
const recentArticles = ref<API.ArticleVO[]>([])
const loadingArticles = ref(false)

const goToCreate = () => {
  if (topic.value.trim()) {
    router.push({ path: '/create', query: { topic: topic.value } })
  } else {
    router.push('/create')
  }
}

const goToList = () => {
  router.push('/article/list')
}

const viewArticle = (article: API.ArticleVO) => {
  router.push(`/article/${article.taskId}`)
}

// 加载最近文章
const loadRecentArticles = async () => {
  if (!loginUserStore.loginUser.id) return

  loadingArticles.value = true
  try {
    const res = await listArticle({ pageNum: 1, pageSize: 6 })
    recentArticles.value = res.data.data?.records || []
  } catch (error) {
    console.error('加载文章失败:', error)
  } finally {
    loadingArticles.value = false
  }
}

// 格式化时间
const formatTime = (time: string | undefined) => {
  if (!time) return '--'
  return dayjs(time).format('MM-DD HH:mm')
}

// 功能卡片数据
const features = [
  {
    icon: FileTextOutlined,
    title: '智能生成标题',
    description: 'AI 自动分析选题，生成吸引眼球的爆款标题',
    color: '#22C55E'
  },
  {
    icon: OrderedListOutlined,
    title: '自动生成大纲',
    description: '智能规划文章结构，确保逻辑清晰完整',
    color: '#3B82F6'
  },
  {
    icon: EditOutlined,
    title: '流式生成正文',
    description: '实时展示创作过程，体验打字机般的流畅输出',
    color: '#8B5CF6'
  },
  {
    icon: PictureOutlined,
    title: '智能配图',
    description: '自动检索高质量无版权图片，完美匹配内容',
    color: '#F59E0B'
  },
  {
    icon: ThunderboltOutlined,
    title: '快速高效',
    description: '5-10分钟完成全文创作，效率提升10倍',
    color: '#EF4444'
  },
  {
    icon: ClockCircleOutlined,
    title: '历史管理',
    description: '随时查看和管理所有创作记录，支持导出',
    color: '#06B6D4'
  }
]

onMounted(() => {
  loadRecentArticles()
})
</script>

<template>
  <div id="homePage">
    <!-- Hero Section -->
    <div class="hero-section">
      <div class="hero-bg"></div>
      <div class="container">
        <div class="hero-badge">
          <ThunderboltOutlined />
          <span>AI 驱动的内容创作平台</span>
        </div>
        <h1 class="hero-title">AI 爆款文章创作器</h1>
        <p class="hero-subtitle">让每个人都能写出 10万+ 文章</p>

        <!-- 核心输入框 -->
        <div class="input-wrapper">
          <a-input
            v-model:value="topic"
            placeholder="输入您想创作的文章选题，例如：2026年AI如何改变职场"
            size="large"
            class="topic-input"
            @pressEnter="goToCreate"
          >
            <template #prefix>
              <EditOutlined class="input-icon" />
            </template>
          </a-input>
          <a-button type="primary" size="large" @click="goToCreate" class="cta-btn">
            <RocketOutlined />
            开始创作
          </a-button>
        </div>

        <p class="hero-tips">工作总结、心得体会、演讲稿、分析报告... 一键生成</p>
      </div>
    </div>

    <!-- Features Section -->
    <div class="features-section">
      <div class="container">
        <div class="section-header">
          <div class="section-badge">核心能力</div>
          <h2 class="section-title">专业人士的一站式AI写作工具</h2>
          <p class="section-subtitle">强大的 AI 能力，让创作变得简单高效</p>
        </div>
        <div class="features-grid">
          <div
            v-for="(feature, index) in features"
            :key="index"
            class="feature-card"
          >
            <div class="feature-icon-wrapper" :style="{ background: `${feature.color}15` }">
              <component :is="feature.icon" class="feature-icon" :style="{ color: feature.color }" />
            </div>
            <div class="feature-content">
              <h3 class="feature-title">{{ feature.title }}</h3>
              <p class="feature-description">{{ feature.description }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Recent Articles Section -->
    <div v-if="loginUserStore.loginUser.id && recentArticles.length > 0" class="articles-section">
      <div class="container">
        <div class="section-header-row">
          <div>
            <h2 class="section-title-sm">最近创作</h2>
            <p class="section-subtitle-sm">查看您最近创作的文章</p>
          </div>
          <a-button type="link" @click="goToList" class="view-all-btn">
            查看全部
            <RightOutlined />
          </a-button>
        </div>

        <a-spin :spinning="loadingArticles">
          <div class="articles-grid">
            <div
              v-for="article in recentArticles"
              :key="article.id"
              class="article-card"
              @click="viewArticle(article)"
            >
              <div class="article-cover">
                <img
                  v-if="article.coverImage"
                  :src="article.coverImage"
                  :alt="article.mainTitle"
                />
                <div v-else class="cover-placeholder">
                  <FileTextOutlined />
                </div>
              </div>
              <div class="article-info">
                <h4 class="article-title">{{ article.mainTitle || article.topic }}</h4>
                <div class="article-meta">
                  <span class="article-time">
                    <ClockCircleOutlined />
                    {{ formatTime(article.createTime) }}
                  </span>
                  <span :class="['article-status', `status-${article.status?.toLowerCase()}`]">
                    {{ article.status === 'COMPLETED' ? '已完成' : article.status === 'PROCESSING' ? '生成中' : '等待中' }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </a-spin>
      </div>
    </div>
  </div>
</template>

<style scoped>
#homePage {
  width: 100%;
  margin: 0;
  padding: 0;
  min-height: 100vh;
  background: var(--color-background);
}

/* Hero Section */
.hero-section {
  position: relative;
  padding: 80px 20px 100px;
  text-align: center;
  overflow: hidden;
}

.hero-bg {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--gradient-hero);
  z-index: 0;
}

.container {
  position: relative;
  z-index: 1;
  max-width: 900px;
  margin: 0 auto;
}

.hero-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: rgba(34, 197, 94, 0.1);
  border: 1px solid rgba(34, 197, 94, 0.2);
  border-radius: var(--radius-full);
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 24px;
  color: var(--color-primary-dark);
}

.hero-title {
  font-size: 52px;
  font-weight: 700;
  margin: 0 0 16px;
  letter-spacing: -1.5px;
  line-height: 1.1;
  color: var(--color-text);
  background: linear-gradient(135deg, var(--color-primary-dark) 0%, var(--color-primary) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.hero-subtitle {
  font-size: 20px;
  margin: 0 0 40px;
  color: var(--color-text-secondary);
  font-weight: 400;
}

/* 核心输入框 */
.input-wrapper {
  display: flex;
  gap: 12px;
  max-width: 700px;
  margin: 0 auto 20px;
  padding: 8px;
  background: white;
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-lg);
  border: 1px solid var(--color-border);
}

.topic-input {
  flex: 1;
  border: none !important;
  box-shadow: none !important;
  font-size: 16px;
  padding: 8px 16px;
  background: transparent !important;
}

.topic-input:focus {
  box-shadow: none !important;
}

.input-icon {
  color: var(--color-text-muted);
  font-size: 18px;
}

.cta-btn {
  height: 52px !important;
  padding: 0 32px !important;
  font-size: 16px !important;
  font-weight: 600 !important;
  border-radius: var(--radius-lg) !important;
  background: var(--gradient-primary) !important;
  border: none !important;
  color: white !important;
  box-shadow: var(--shadow-green) !important;
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
  transition: opacity var(--transition-normal) !important;
}

.cta-btn:hover,
.cta-btn:focus,
.cta-btn:active {
  background: var(--gradient-primary) !important;
  border: none !important;
  color: white !important;
  box-shadow: var(--shadow-green) !important;
  opacity: 0.92;
}

.cta-btn :deep(.ant-wave) {
  display: none;
}

.hero-tips {
  font-size: 14px;
  color: var(--color-text-muted);
  margin: 0;
}

/* Features Section */
.features-section {
  padding: 80px 20px;
  background: var(--color-background-secondary);
}

.features-section .container {
  max-width: 1100px;
}

.section-header {
  text-align: center;
  margin-bottom: 48px;
}

.section-badge {
  display: inline-block;
  padding: 6px 14px;
  background: rgba(34, 197, 94, 0.1);
  border-radius: var(--radius-full);
  font-size: 13px;
  font-weight: 600;
  color: var(--color-primary-dark);
  margin-bottom: 16px;
}

.section-title {
  font-size: 32px;
  font-weight: 700;
  margin: 0 0 12px;
  color: var(--color-text);
  letter-spacing: -0.5px;
}

.section-subtitle {
  font-size: 16px;
  color: var(--color-text-secondary);
  margin: 0;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
}

.feature-card {
  background: white;
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  padding: 24px;
  display: flex;
  gap: 16px;
  align-items: flex-start;
  transition: all var(--transition-normal);
  cursor: pointer;
}

.feature-card:hover {
  border-color: var(--color-primary-light);
  box-shadow: var(--shadow-card-hover);
  transform: translateY(-2px);
}

.feature-icon-wrapper {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
  flex-shrink: 0;
}

.feature-icon {
  font-size: 22px;
}

.feature-content {
  flex: 1;
  min-width: 0;
}

.feature-title {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 6px;
  color: var(--color-text);
}

.feature-description {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin: 0;
  line-height: 1.5;
}

/* Articles Section */
.articles-section {
  padding: 60px 20px 80px;
  background: var(--color-background);
}

.articles-section .container {
  max-width: 1100px;
}

.section-header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 32px;
}

.section-title-sm {
  font-size: 24px;
  font-weight: 700;
  margin: 0 0 4px;
  color: var(--color-text);
}

.section-subtitle-sm {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin: 0;
}

.view-all-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--color-primary);
  font-weight: 500;
  padding: 0;
}

.view-all-btn:hover {
  color: var(--color-primary-dark);
}

.articles-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
}

.article-card {
  background: white;
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  overflow: hidden;
  transition: all var(--transition-normal);
  cursor: pointer;
}

.article-card:hover {
  border-color: var(--color-primary-light);
  box-shadow: var(--shadow-card-hover);
  transform: translateY(-2px);
}

.article-cover {
  height: 140px;
  background: var(--color-background-tertiary);
  overflow: hidden;
}

.article-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.cover-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  color: var(--color-text-muted);
}

.article-info {
  padding: 16px;
}

.article-title {
  font-size: 15px;
  font-weight: 600;
  margin: 0 0 12px;
  color: var(--color-text);
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.article-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.article-time {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--color-text-muted);
}

.article-status {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-weight: 500;
}

.article-status.status-completed {
  background: rgba(34, 197, 94, 0.1);
  color: var(--color-primary-dark);
}

.article-status.status-processing {
  background: rgba(59, 130, 246, 0.1);
  color: #2563EB;
}

.article-status.status-pending {
  background: var(--color-background-tertiary);
  color: var(--color-text-muted);
}

/* Responsive */
@media (max-width: 992px) {
  .features-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .articles-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .hero-section {
    padding: 60px 20px 80px;
  }

  .hero-title {
    font-size: 36px;
  }

  .hero-subtitle {
    font-size: 16px;
  }

  .input-wrapper {
    flex-direction: column;
    padding: 12px;
  }

  .cta-btn {
    width: 100%;
    justify-content: center;
  }

  .features-grid {
    grid-template-columns: 1fr;
  }

  .articles-grid {
    grid-template-columns: 1fr;
  }

  .section-title {
    font-size: 24px;
  }

  .section-header-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }
}
</style>
