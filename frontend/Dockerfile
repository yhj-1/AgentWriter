# ========================================
# AI 爆款文章创作器 - 前端 Dockerfile
# ========================================

# ============ 构建阶段 ============
# 如果拉取失败，请配置 Docker 镜像加速器，或使用以下国内镜像：
# FROM m.daocloud.io/docker.io/node:22-alpine AS build
FROM node:22-alpine AS build

WORKDIR /app

# 1. 先复制 package.json 并安装依赖（利用 Docker 缓存）
COPY package*.json ./
# 构建阶段需要所有依赖（包括 devDependencies）
RUN npm ci && \
    npm cache clean --force

# 2. 复制源代码
COPY . .

# 3. 生成 API 客户端（如果需要）
# RUN npm run openapi2ts

# 4. 构建生产版本
RUN npm run build

# ============ 运行阶段 ============
# 如果拉取失败，请配置 Docker 镜像加速器，或使用以下国内镜像：
# FROM m.daocloud.io/docker.io/nginx:alpine
FROM nginx:alpine

# 安装 curl（用于健康检查）
RUN apk add --no-cache curl

# 复制构建产物
COPY --from=build /app/dist /usr/share/nginx/html

# 复制 Nginx 配置
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 创建必要目录并设置权限（解决非 root 用户权限问题）
RUN mkdir -p /var/cache/nginx /var/log/nginx /tmp/nginx && \
    chown -R nginx:nginx /var/cache/nginx /var/log/nginx /tmp/nginx && \
    chown -R nginx:nginx /usr/share/nginx/html && \
    # 修改 nginx.conf 使用 /tmp/nginx/nginx.pid
    sed -i 's|/run/nginx.pid|/tmp/nginx/nginx.pid|g' /etc/nginx/nginx.conf && \
    # 确保 nginx 可以绑定 80 端口
    touch /tmp/nginx/nginx.pid && chown nginx:nginx /tmp/nginx/nginx.pid

# 使用非 root 用户运行（安全最佳实践）
USER nginx

# 暴露端口
EXPOSE 80

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -f http://localhost/ || exit 1

# 启动 Nginx
CMD ["nginx", "-g", "daemon off;"]
