package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	docs "github.com/yupi/ai-passage-creator/docs"
	"github.com/yupi/ai-passage-creator/internal/app"
	"github.com/yupi/ai-passage-creator/internal/common"
	"github.com/yupi/ai-passage-creator/internal/config"
	"github.com/yupi/ai-passage-creator/internal/middleware"
)

// @title AI Passage Creator API
// @version 1.0
// @description Go backend API 文档
// @BasePath /api
func main() {
	// 加载配置
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// 初始化应用
	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}
	defer application.Close()

	// 创建路由器
	r := gin.Default()
	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%d", cfg.Server.Port)
	docs.SwaggerInfo.BasePath = cfg.Server.ContextPath
	docs.SwaggerInfo.Schemes = []string{"http"}

	// 全局中间件
	r.Use(middleware.CORS())

	// 配置 Session
	if err := middleware.SetupSession(r, cfg); err != nil {
		log.Fatalf("setup session: %v", err)
	}

	// 注册路由
	api := r.Group(cfg.Server.ContextPath)
	{
		// OpenAPI 文档
		api.GET("/v3/api-docs", func(c *gin.Context) {
			c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(docs.SwaggerInfo.ReadDoc()))
		})
		api.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL(fmt.Sprintf("%s/v3/api-docs", cfg.Server.ContextPath))))

		// 健康检查
		api.GET("/health", application.HealthHandler.Check)

		// 用户路由
		user := api.Group("/user")
		{
			// 无需登录
			user.POST("/register", application.UserHandler.Register)
			user.POST("/login", application.UserHandler.Login)

			// 需要登录
			user.GET("/get/login", application.UserHandler.GetLoginUser)
			user.POST("/logout", application.UserHandler.Logout)
			user.GET("/get/vo", application.UserHandler.GetVO)

			// 需要管理员权限
			adminAuth := middleware.AuthCheck(application.UserService, common.AdminRole)
			user.POST("/add", adminAuth, application.UserHandler.Add)
			user.GET("/get", adminAuth, application.UserHandler.Get)
			user.POST("/delete", adminAuth, application.UserHandler.Delete)
			user.POST("/update", adminAuth, application.UserHandler.Update)
			user.POST("/list/page/vo", adminAuth, application.UserHandler.ListPageVO)
		}

		// 文章路由
		userAuth := middleware.AuthCheck(application.UserService, common.UserRole)
		article := api.Group("/article")
		{
			article.POST("/create", userAuth, application.ArticleHandler.Create)
			article.POST("/confirm-title", userAuth, application.ArticleHandler.ConfirmTitle)
			article.POST("/confirm-outline", userAuth, application.ArticleHandler.ConfirmOutline)
			article.POST("/ai-modify-outline", userAuth, application.ArticleHandler.AiModifyOutline)
			article.GET("/progress/:taskId", application.ArticleHandler.GetProgress)
			article.GET("/execution-logs/:taskId", application.ArticleHandler.GetExecutionLogs)
			article.GET("/:taskId", userAuth, application.ArticleHandler.Get)
			article.POST("/list", userAuth, application.ArticleHandler.List)
			article.POST("/delete", userAuth, application.ArticleHandler.Delete)
		}

		// 支付路由（需要登录）
		payment := api.Group("/payment")
		payment.Use(userAuth)
		{
			payment.POST("/create-vip-session", application.PaymentHandler.CreateVipSession)
			payment.POST("/refund", application.PaymentHandler.Refund)
			payment.GET("/records", application.PaymentHandler.GetRecords)
		}

		// Webhook 路由（不需要认证）
		webhook := api.Group("/webhook")
		{
			webhook.POST("/stripe", application.WebhookHandler.HandleStripeWebhook)
		}

		// 统计路由（仅管理员）
		adminAuth := middleware.AuthCheck(application.UserService, common.AdminRole)
		statistics := api.Group("/statistics")
		statistics.Use(adminAuth)
		{
			statistics.GET("/overview", application.StatisticsHandler.GetStatistics)
		}
	}

	// 启动服务器
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("server starting at http://localhost%s%s", addr, cfg.Server.ContextPath)
	if err := r.Run(addr); err != nil {
		log.Fatalf("start server: %v", err)
	}
}
