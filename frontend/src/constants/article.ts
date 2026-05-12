/**
 * 文章相关常量定义
 */

// 文章状态枚举
export enum ArticleStatus {
  PENDING = 'PENDING',
  PROCESSING = 'PROCESSING',
  COMPLETED = 'COMPLETED',
  FAILED = 'FAILED',
}

// 状态文本映射
export const STATUS_TEXT_MAP: Record<string, string> = {
  [ArticleStatus.PENDING]: '等待中',
  [ArticleStatus.PROCESSING]: '生成中',
  [ArticleStatus.COMPLETED]: '已完成',
  [ArticleStatus.FAILED]: '失败',
}

// 状态颜色映射（用于 Ant Design Tag）
export const STATUS_TAG_COLOR_MAP: Record<string, string> = {
  [ArticleStatus.PENDING]: 'default',
  [ArticleStatus.PROCESSING]: 'processing',
  [ArticleStatus.COMPLETED]: 'success',
  [ArticleStatus.FAILED]: 'error',
}

// 状态颜色映射（用于自定义样式）
export const STATUS_COLOR_MAP: Record<string, string> = {
  [ArticleStatus.PENDING]: '#6B7280',
  [ArticleStatus.PROCESSING]: '#3B82F6',
  [ArticleStatus.COMPLETED]: '#22C55E',
  [ArticleStatus.FAILED]: '#EF4444',
}

// 文章创作相关
export const MAX_TOPIC_LENGTH = 500
export const DEFAULT_TOTAL_IMAGES = 5

// 状态筛选选项
export const STATUS_OPTIONS = [
  { value: '', label: '全部状态' },
  { value: ArticleStatus.COMPLETED, label: '已完成' },
  { value: ArticleStatus.PROCESSING, label: '生成中' },
  { value: ArticleStatus.PENDING, label: '等待中' },
  { value: ArticleStatus.FAILED, label: '失败' },
]
