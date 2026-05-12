<template>
  <span :class="['status-badge', `status-${status?.toLowerCase()}`]">
    <span class="status-dot"></span>
    {{ statusText }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { getStatusText } from '@/utils/article'

const props = defineProps<{
  status: string
}>()

const statusText = computed(() => getStatusText(props.status))
</script>

<style scoped>
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 500;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.status-badge.status-completed {
  background: rgba(34, 197, 94, 0.1);
  color: var(--color-primary-dark);
}

.status-badge.status-completed .status-dot {
  background: var(--color-primary);
}

.status-badge.status-processing {
  background: rgba(59, 130, 246, 0.1);
  color: #2563EB;
}

.status-badge.status-processing .status-dot {
  background: #3B82F6;
  animation: pulse 1.5s infinite;
}

.status-badge.status-pending {
  background: var(--color-background-tertiary);
  color: var(--color-text-secondary);
}

.status-badge.status-pending .status-dot {
  background: var(--color-text-muted);
}

.status-badge.status-failed {
  background: rgba(239, 68, 68, 0.1);
  color: #DC2626;
}

.status-badge.status-failed .status-dot {
  background: #EF4444;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
</style>
