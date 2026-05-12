<template>
  <div class="brand-section">
    <div class="brand-bg"></div>
    <div class="brand-content">
      <div class="brand-logo">
        <img src="@/assets/logo.png" alt="Logo" class="logo-img" />
      </div>
      <h1 class="brand-title">AI 爆款文章创作器</h1>
      <p class="brand-subtitle">让每个人都能写出 10万+ 文章</p>
      <div class="brand-features">
        <div class="feature-item">
          <CheckCircleOutlined class="feature-check" />
          <span>智能生成标题与大纲</span>
        </div>
        <div class="feature-item">
          <CheckCircleOutlined class="feature-check" />
          <span>流式生成高质量正文</span>
        </div>
        <div class="feature-item">
          <CheckCircleOutlined class="feature-check" />
          <span>自动配图一键导出</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { CheckCircleOutlined } from '@ant-design/icons-vue'
</script>

<style scoped>
.brand-section {
  flex: 1;
  padding: 48px 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

.brand-bg {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, #22C55E 0%, #16A34A 50%, #15803D 100%);
}

.brand-bg::before {
  content: '';
  position: absolute;
  top: -50%;
  left: -50%;
  width: 200%;
  height: 200%;
  background: radial-gradient(circle, rgba(255,255,255,0.1) 0%, transparent 60%);
  animation: pulse-bg 8s ease-in-out infinite;
}

@keyframes pulse-bg {
  0%, 100% { transform: scale(1); opacity: 0.5; }
  50% { transform: scale(1.1); opacity: 0.3; }
}

.brand-content {
  position: relative;
  z-index: 1;
  text-align: center;
  color: white;
}

.brand-logo {
  margin-bottom: 24px;
}

.logo-img {
  width: 80px;
  height: 80px;
  object-fit: contain;
  background: rgba(255, 255, 255, 0.95);
  border-radius: var(--radius-xl);
  padding: 8px;
}

.brand-title {
  font-size: 26px;
  font-weight: 700;
  margin: 0 0 10px;
  letter-spacing: -0.5px;
}

.brand-subtitle {
  font-size: 15px;
  opacity: 0.9;
  margin: 0 0 36px;
}

.brand-features {
  text-align: left;
  background: rgba(255, 255, 255, 0.1);
  border-radius: var(--radius-lg);
  padding: 20px 24px;
  backdrop-filter: blur(8px);
}

.feature-item {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
  font-size: 14px;
}

.feature-item:last-child {
  margin-bottom: 0;
}

.feature-check {
  font-size: 18px;
  color: white;
}

/* 响应式 */
@media (max-width: 768px) {
  .brand-section {
    padding: 32px 24px;
  }

  .brand-title {
    font-size: 22px;
  }

  .brand-features {
    display: none;
  }
}
</style>
