package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

func main() {
	// 1. 初始化日志 (同时输出到控制台和文件)
	setupLogging()

	// 2. 加载配置 (自检)
	if err := loadConfig(); err != nil {
		log.Printf("❌ 初始加载配置文件失败: %v", err)
		// 这里可以选择退出，或者继续运行等待热更新
	}
	go watchConfig()

	// 3. 初始化数据库
	initDB()

	// 4. 启动定时任务
	c := cron.New()
	_, _ = c.AddFunc("55 23 * * *", func() {
		processDailyReport()
	})
	// 添加定时清理过期数据的任务（例如每天凌晨 3:00 清理 30 天前的数据）
	_, _ = c.AddFunc("0 3 * * *", func() {
		cleanupOldData(30)
	})
	c.Start()

	// 5. 启动时立即更新一次订阅数据
	go updateSubscriptionData()

	// 6. 启动 Web 服务
	r := gin.Default()

	// API 路由组
	api := r.Group("/api")
	{
		api.POST("/auth", handleAuth)

		// 增加 Token 鉴权中间件保护查询接口
		protected := api.Group("")
		protected.Use(TokenAuthMiddleware())
		{
			protected.GET("/stats", handleGetStats)
			protected.GET("/fake/stats", handleFakeGetStats)
		}
	}
	// 流量上报接口增加 Token 鉴权中间件
	r.POST("/report", TokenAuthMiddleware(), handleReport)

	confLock.RLock()
	port := conf.ListenPort
	confLock.RUnlock()

	fmt.Printf("🚀 流量统计后端启动 | 监听端口 %s\n", port)
	r.Run(port)
}

func setupLogging() {
	logFile, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("无法创建日志文件:", err)
		return
	}
	// 同时写到文件和控制台
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	log.Println("✅ 日志系统初始化完成")
}

// TokenAuthMiddleware 验证 Authorization Header
func TokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		confLock.RLock()
		expectedToken := conf.ServerToken
		confLock.RUnlock()

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || authHeader != "Bearer "+expectedToken {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// cleanupOldData 清理指定天数之前的数据
func cleanupOldData(days int) {
	threshold := time.Now().AddDate(0, 0, -days)
	result := db.Where("timestamp < ?", threshold).Delete(&TrafficRecord{})
	if result.Error != nil {
		log.Printf("清理过期数据失败: %v", result.Error)
	} else {
		log.Printf("已清理 %d 天前的数据，共删除 %d 条记录", days, result.RowsAffected)
	}
}
