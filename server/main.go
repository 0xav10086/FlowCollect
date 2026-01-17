package main

import (
	"fmt"
	"io"
	"log"
	"os"

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
	c.Start()

	// 5. 启动时立即更新一次订阅数据
	go updateSubscriptionData()

	// 6. 启动 Web 服务
	r := gin.Default()
	r.POST("/report", handleReport)
	r.POST("/api/auth", handleAuth)
	r.GET("/api/stats", handleGetStats)
	r.GET("/api/fake/stats", handleFakeGetStats)

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