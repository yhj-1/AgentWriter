#!/bin/bash

# ========================================
# AI 爆款文章创作器 - 一键启动脚本
# ========================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印信息
info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# 检查 Docker 是否安装
check_docker() {
    if ! command -v docker &> /dev/null; then
        error "Docker 未安装，请先安装 Docker"
    fi
    
    # 检查 docker compose（新版）或 docker-compose（旧版）
    if docker compose version &> /dev/null; then
        DOCKER_COMPOSE="docker compose"
    elif command -v docker-compose &> /dev/null; then
        DOCKER_COMPOSE="docker-compose"
    else
        error "Docker Compose 未安装，请先安装 Docker Compose"
    fi
    
    info "Docker 和 Docker Compose 已安装 ✓"
}

# 检查 .env 文件
check_env() {
    if [ ! -f .env ]; then
        warn ".env 文件不存在，正在从 .env.example 创建..."
        cp .env.example .env
        warn "请编辑 .env 文件，填写必需的 API Key："
        warn "  - DASHSCOPE_API_KEY (通义千问)"
        warn "  - PEXELS_API_KEY (Pexels 图库)"
        echo ""
        read -p "是否现在编辑 .env 文件？(y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            ${EDITOR:-vi} .env
        else
            error "请先编辑 .env 文件，然后重新运行此脚本"
        fi
    fi
    
    # 检查必需的环境变量
    source .env
    
    if [ -z "$DASHSCOPE_API_KEY" ] || [ "$DASHSCOPE_API_KEY" = "your_dashscope_api_key" ]; then
        error "请在 .env 文件中设置 DASHSCOPE_API_KEY"
    fi
    
    if [ -z "$PEXELS_API_KEY" ] || [ "$PEXELS_API_KEY" = "your_pexels_api_key" ]; then
        error "请在 .env 文件中设置 PEXELS_API_KEY"
    fi
    
    info "环境变量配置检查通过 ✓"
}

# 停止并清理旧容器
cleanup() {
    info "停止并清理旧容器..."
    $DOCKER_COMPOSE down
}

# 构建并启动服务
start() {
    info "开始构建并启动服务..."
    info "这可能需要几分钟时间，请耐心等待..."
    echo ""
    
    $DOCKER_COMPOSE up -d --build
    
    echo ""
    info "服务启动成功！"
}

# 等待服务就绪
wait_for_services() {
    info "等待服务启动中..."
    echo ""
    
    # 等待 MySQL 就绪
    info "等待 MySQL 启动..."
    timeout=60
    counter=0
    until $DOCKER_COMPOSE exec -T mysql mysqladmin ping -h localhost -u root -p"${MYSQL_ROOT_PASSWORD:-123456}" --silent &> /dev/null; do
        sleep 2
        counter=$((counter + 2))
        if [ $counter -ge $timeout ]; then
            error "MySQL 启动超时"
        fi
        echo -n "."
    done
    echo ""
    info "MySQL 已就绪 ✓"
    
    # 等待 Redis 就绪
    info "等待 Redis 启动..."
    counter=0
    until $DOCKER_COMPOSE exec -T redis redis-cli ping &> /dev/null; do
        sleep 1
        counter=$((counter + 1))
        if [ $counter -ge 30 ]; then
            error "Redis 启动超时"
        fi
        echo -n "."
    done
    echo ""
    info "Redis 已就绪 ✓"
    
    # 等待后端就绪
    info "等待后端服务启动..."
    counter=0
    until curl -s http://localhost:${BACKEND_PORT:-8123}/api/health/ > /dev/null 2>&1; do
        sleep 3
        counter=$((counter + 3))
        if [ $counter -ge 180 ]; then
            error "后端服务启动超时"
        fi
        echo -n "."
    done
    echo ""
    info "后端服务已就绪 ✓"
    
    # 等待前端就绪
    info "等待前端服务启动..."
    counter=0
    until curl -s http://localhost:${FRONTEND_PORT:-80}/health > /dev/null 2>&1; do
        sleep 2
        counter=$((counter + 2))
        if [ $counter -ge 60 ]; then
            error "前端服务启动超时"
        fi
        echo -n "."
    done
    echo ""
    info "前端服务已就绪 ✓"
}

# 显示访问信息
show_info() {
    echo ""
    echo "========================================="
    echo "  🎉 AI 爆款文章创作器已成功启动！"
    echo "========================================="
    echo ""
    echo "📱 访问地址："
    echo "  前端页面: http://localhost:${FRONTEND_PORT:-80}"
    echo "  后端接口: http://localhost:${BACKEND_PORT:-8123}/api"
    echo "  接口文档: http://localhost:${BACKEND_PORT:-8123}/api/doc.html"
    echo ""
    echo "👤 测试账号："
    echo "  管理员: admin / 12345678"
    echo "  普通用户: user / 12345678"
    echo ""
    echo "📊 查看服务状态："
    echo "  docker compose ps"
    echo ""
    echo "📋 查看服务日志："
    echo "  docker compose logs -f [服务名]"
    echo "  服务名: backend, frontend, mysql, redis"
    echo ""
    echo "🛑 停止服务："
    echo "  docker compose down"
    echo ""
    echo "========================================="
}

# 主流程
main() {
    echo ""
    echo "========================================="
    echo "  AI 爆款文章创作器 - 一键启动"
    echo "========================================="
    echo ""
    
    check_docker
    check_env
    cleanup
    start
    wait_for_services
    show_info
}

# 执行主流程
main
