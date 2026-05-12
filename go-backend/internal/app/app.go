package app

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/yupi/ai-passage-creator/internal/agent"
	"github.com/yupi/ai-passage-creator/internal/common"
	"github.com/yupi/ai-passage-creator/internal/config"
	"github.com/yupi/ai-passage-creator/internal/handler"
	"github.com/yupi/ai-passage-creator/internal/service"
	"github.com/yupi/ai-passage-creator/internal/store"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// App 应用程序
type App struct {
	Config      *config.Config
	DB          *gorm.DB
	RedisClient *redis.Client

	// Handlers
	UserHandler       *handler.UserHandler
	ArticleHandler    *handler.ArticleHandler
	HealthHandler     *handler.HealthHandler
	PaymentHandler    *handler.PaymentHandler
	WebhookHandler    *handler.WebhookHandler
	StatisticsHandler *handler.StatisticsHandler

	// Services (用于中间件)
	UserService *service.UserService
}

// New 创建应用实例
func New(cfg *config.Config) (*App, error) {
	// 初始化数据库
	db, err := initDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	// 初始化 Redis
	redisClient, err := initRedis(cfg)
	if err != nil {
		return nil, fmt.Errorf("init redis: %w", err)
	}

	// 初始化各层
	userStore := store.NewUserStore(db)
	articleStore := store.NewArticleStore(db)
	paymentStore := store.NewPaymentStore(db)
	agentLogStore := store.NewAgentLogStore(db)

	// SSE 管理器
	sseManager := common.NewSSEManager()

	// 服务层
	userService := service.NewUserService(userStore)
	quotaService := service.NewQuotaService(userStore)
	agentLogService := service.NewAgentLogService(agentLogStore)
	statisticsService := service.NewStatisticsService(db, userStore, articleStore, redisClient)

	// COS 服务（判断是否已配置）
	cosEnabled := cfg.COS.Bucket != "" && cfg.COS.SecretID != "" && cfg.COS.SecretKey != ""
	var cosService *service.CosService
	if cosEnabled {
		cosService = service.NewCosService(cfg.COS)
		log.Printf("COS 服务已启用, bucket=%s, region=%s", cfg.COS.Bucket, cfg.COS.Region)
	} else {
		log.Println("COS 服务未配置，图片将使用原始 URL")
	}

	// 初始化所有图片服务
	pexelsService := service.NewPexelsService(cfg)
	iconifyService := service.NewIconifyService(cfg.Iconify)
	mermaidService := service.NewMermaidService(cfg.Mermaid)
	nanoBananaService := service.NewNanoBananaService(cfg.NanoBanana)
	svgDiagramService := service.NewSVGDiagramService(cfg.SVGDiagram, cfg.AI)
	emojiPackService := service.NewEmojiPackService(cfg.EmojiPack)
	picsumService := service.NewPicsumService() // 降级服务

	// 初始化图片服务策略
	imageStrategy := service.NewImageServiceStrategy(cosService, cosEnabled)
	imageStrategy.RegisterService(pexelsService)
	imageStrategy.RegisterService(iconifyService)
	imageStrategy.RegisterService(mermaidService)
	imageStrategy.RegisterService(nanoBananaService)
	imageStrategy.RegisterService(svgDiagramService)
	imageStrategy.RegisterService(emojiPackService)
	imageStrategy.RegisterService(picsumService) // 注册降级服务

	log.Println("图片服务策略初始化完成，已注册 7 个图片服务（含降级服务）")

	// 智能体服务（注入 agentLogService）
	agentService, err := service.NewArticleAgentService(cfg, imageStrategy, agentLogService, sseManager)
	if err != nil {
		return nil, fmt.Errorf("init agent service: %w", err)
	}

	// 获取 LLM 实例（从 agentService 中获取，避免重复初始化）
	// 注意：这里我们需要从 agentService 暴露 llm，或者重新创建一个
	// 为了简化，我们创建多智能体编排器，它会在内部创建 LLM
	orchestrator := agent.NewArticleAgentOrchestrator(
		cfg,
		agentService.GetLLM(), // 假设我们添加一个 GetLLM 方法
		agentLogService,
		sseManager,
		imageStrategy,
	)

	log.Printf("智能体编排器初始化完成，启用状态: %v", cfg.Agent.Orchestrator.Enabled)

	articleService := service.NewArticleService(
		articleStore,
		agentService,
		orchestrator,
		cfg,
		quotaService,
		sseManager,
	)

	// 支付服务
	paymentService := service.NewPaymentService(&cfg.Stripe, userStore, paymentStore, db)

	// 处理器层
	userHandler := handler.NewUserHandler(userService)
	articleHandler := handler.NewArticleHandler(articleService, userService, agentLogService, sseManager)
	healthHandler := handler.NewHealthHandler()
	paymentHandler := handler.NewPaymentHandler(paymentService)
	webhookHandler := handler.NewWebhookHandler(paymentService)
	statisticsHandler := handler.NewStatisticsHandler(statisticsService)

	return &App{
		Config:            cfg,
		DB:                db,
		RedisClient:       redisClient,
		UserHandler:       userHandler,
		ArticleHandler:    articleHandler,
		HealthHandler:     healthHandler,
		PaymentHandler:    paymentHandler,
		WebhookHandler:    webhookHandler,
		StatisticsHandler: statisticsHandler,
		UserService:       userService,
	}, nil
}

// initDB 初始化数据库
func initDB(cfg *config.Config) (*gorm.DB, error) {
	dsn := cfg.Database.GetDSN()

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)

	log.Println("database connected")
	return db, nil
}

// initRedis 初始化 Redis
func initRedis(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.GetRedisAddr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// 测试连接
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	log.Println("redis connected")
	return client, nil
}

// Close 关闭资源
func (a *App) Close() error {
	// 关闭数据库
	sqlDB, err := a.DB.DB()
	if err != nil {
		return err
	}
	if err := sqlDB.Close(); err != nil {
		return err
	}

	// 关闭 Redis
	if a.RedisClient != nil {
		if err := a.RedisClient.Close(); err != nil {
			log.Printf("close redis: %v", err)
		}
	}

	return nil
}
