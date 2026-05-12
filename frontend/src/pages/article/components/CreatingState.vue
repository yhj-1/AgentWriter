<template>
  <div class="creating-state">
    <!-- 标题预览 -->
    <div v-if="article.mainTitle" class="preview-header">
      <h1 class="article-title">{{ article.mainTitle }}</h1>
      <p class="article-subtitle">{{ article.subTitle }}</p>
    </div>

    <!-- 大纲预览（流式解析展示） -->
    <div v-if="outlineRaw" class="outline-preview">
      <div class="section-label">
        <BulbOutlined />
        <span>文章大纲</span>
        <span v-if="isOutlineStreaming" class="typing-cursor">|</span>
      </div>
      <div class="outline-list">
        <div
          v-for="item in parsedOutline"
          :key="item.section"
          class="outline-item"
        >
          <div class="outline-title">{{ item.section }}. {{ item.title }}</div>
          <ul class="outline-points">
            <li v-for="(point, idx) in item.points" :key="idx">{{ point }}</li>
          </ul>
        </div>
      </div>
    </div>

    <!-- 正文预览（流式输出） -->
    <div v-if="article.content" class="content-preview">
      <div v-html="markdownToHtml(article.content)" class="markdown-body"></div>
      <span v-if="isStreaming" class="typing-cursor">|</span>
    </div>

    <!-- 配图进度 -->
    <div v-if="showImageProgress" class="image-progress-box">
      <div class="progress-header">
        <PictureOutlined />
        <span>正在生成配图</span>
      </div>
      <a-progress :percent="imageProgress" status="active" :stroke-color="{ from: '#22C55E', to: '#16A34A' }" />
      <p class="progress-hint">{{ imageCount }}/{{ totalImages }} 张图片已完成</p>
    </div>

    <!-- 加载占位 -->
    <div v-if="showLoadingPlaceholder" class="loading-placeholder">
      <a-spin size="large" />
      <p>AI 正在构思标题...</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { BulbOutlined, PictureOutlined } from '@ant-design/icons-vue'
import { markdownToHtml } from '@/utils/markdown'

// 大纲项类型
interface OutlineItem {
  title: string
  points: string[]
  section: number
}

const props = defineProps<{
  article: Partial<API.ArticleVO>
  outlineRaw: string
  isOutlineStreaming: boolean
  isStreaming: boolean
  currentStep: number
  imageCount: number
  totalImages: number
  imageProgress: number
}>()

// 显示配图进度
const showImageProgress = computed(() => props.currentStep === 4 && props.imageProgress > 0)

// 显示加载占位
const showLoadingPlaceholder = computed(() => props.currentStep === 0 && !props.article.mainTitle)

// 解析大纲 JSON（格式为 { "sections": [...] }）
const parsedOutline = computed<OutlineItem[]>(() => {
  if (!props.outlineRaw) return []

  const str = props.outlineRaw.trim()

  // 尝试解析完整的 JSON
  try {
    const parsed = JSON.parse(str)
    if (parsed && Array.isArray(parsed.sections)) {
      return parsed.sections
    }
    return []
  } catch {
    // JSON 不完整时，尝试解析已完成的部分
    try {
      // 找到最后一个完整的 section 对象 }
      // 格式: { "sections": [ {...}, {...} ] }
      const sectionsMatch = str.match(/"sections"\s*:\s*\[/)
      if (!sectionsMatch) return []

      const sectionsStart = str.indexOf('[', sectionsMatch.index)
      if (sectionsStart === -1) return []

      // 从 sections 数组开始，找到最后一个完整的 }
      const afterStart = str.substring(sectionsStart)
      const lastBrace = afterStart.lastIndexOf('}')

      if (lastBrace > 0) {
        const partialArray = afterStart.substring(0, lastBrace + 1) + ']'
        const parsed = JSON.parse(partialArray)
        if (Array.isArray(parsed)) {
          return parsed
        }
      }
      return []
    } catch {
      return []
    }
  }
})
</script>

<style scoped lang="scss">
.creating-state {
  max-width: 100%;
}

/* 标题区域 */
.preview-header {
  text-align: center;
  margin-bottom: 24px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--color-border-light);
}

.article-title {
  font-size: 28px;
  font-weight: 700;
  margin: 0 0 8px;
  color: var(--color-text);
  line-height: 1.4;
}

.article-subtitle {
  font-size: 16px;
  color: var(--color-text-secondary);
  margin: 0;
}

/* 大纲预览 */
.outline-preview {
  margin-bottom: 24px;
  padding: 20px 24px;
  background: var(--color-background-secondary);
  border-radius: var(--radius-lg);
}

.section-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-primary);
  margin-bottom: 16px;
}

.outline-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.outline-item {
  padding: 12px 16px;
  background: white;
  border-radius: var(--radius-md);
  border-left: 3px solid var(--color-primary);
}

.outline-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  margin-bottom: 8px;
}

.outline-points {
  margin: 0;
  padding-left: 18px;

  li {
    font-size: 13px;
    color: var(--color-text-secondary);
    line-height: 1.6;
    margin-bottom: 4px;

    &:last-child {
      margin-bottom: 0;
    }
  }
}

/* 正文预览 */
.content-preview {
  line-height: 1.8;
}

.markdown-body {
  line-height: 1.8;
  font-size: 15px;
  color: var(--color-text);

  :deep(h2) {
    font-size: 20px;
    font-weight: 600;
    margin: 24px 0 14px;
    padding-bottom: 10px;
    border-bottom: 1px solid var(--color-border);
    color: var(--color-text);
  }

  :deep(p) {
    margin-bottom: 14px;
    text-indent: 2em;
  }

  :deep(img) {
    display: block;
    max-width: 100%;
    margin: 20px auto;
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-md);
  }
}

.typing-cursor {
  display: inline-block;
  animation: blink 1s infinite;
  color: var(--color-primary);
  font-weight: bold;
  font-size: 18px;
}

.image-progress-box {
  background: var(--color-background-secondary);
  border-radius: var(--radius-lg);
  padding: 24px;
  margin-top: 24px;
  text-align: center;

  .progress-header {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    margin-bottom: 16px;
    font-size: 15px;
    font-weight: 600;
    color: var(--color-text);
  }

  .progress-hint {
    margin: 12px 0 0;
    font-size: 13px;
    color: var(--color-text-muted);
  }
}

.loading-placeholder {
  text-align: center;
  padding: 100px 0;

  p {
    margin: 16px 0 0;
    color: var(--color-text-secondary);
    font-size: 15px;
  }
}

@keyframes blink {
  0%, 50% { opacity: 1; }
  51%, 100% { opacity: 0; }
}
</style>
