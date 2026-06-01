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

const serverVersion = "v1.0.10"

func main() {
	// 0. 确保运行时目录存在
	ensureDirs()

	// 1. 初始化日志 (同时输出到控制台和文件)
	setupLogging()

	// 2. 加载配置 (自检)
	if err := loadConfig(); err != nil {
		log.Printf("❌ 初始加载配置文件失败: %v", err)
		// 这里可以选择退出，或者继续运行等待热更新
	}

	// 2.1 动态配置覆盖：从主订阅 YAML 文件读取端口和 Token
	confLock.RLock()
	readSub := conf.ReadMainSubConfig
	mainFile := conf.MainSubFile
	confLock.RUnlock()

	if readSub {
		port, token, err := ExtractConfigFromMainSub(mainFile)
		if err != nil {
			log.Printf("⚠️ 读取主订阅配置失败，使用 INI 默认值: %v", err)
		} else {
			confLock.Lock()
			if port != "" {
				conf.ListenPort = ":" + port
				log.Printf("🔧 端口已从订阅配置覆盖: %s", conf.ListenPort)
			}
			if token != "" {
				conf.ServerToken = token
				log.Printf("🔧 Token 已从订阅配置覆盖")
			}
			confLock.Unlock()
		}
	}

	// 打印配置摘要
	confLock.RLock()
	cfgListenPort := conf.ListenPort
	cfgCFTunnel := conf.CFTunnelContainer
	cfgSubUrls := len(conf.SubUrls)
	cfgSMTP := conf.SMTPHost != "" && conf.EmailTo != ""
	confLock.RUnlock()

	log.Println("========================================")
	log.Printf("  FlowCollect Server %s", serverVersion)
	log.Println("========================================")
	log.Printf("  监听端口: %s", cfgListenPort)
	if cfgCFTunnel != "" {
		log.Printf("  CF Tunnel 监控: 启用 (容器: %s, 检查间隔: 5分钟)", cfgCFTunnel)
	} else {
		log.Println("  CF Tunnel 监控: 未配置 (CFTunnelContainer 为空)")
	}
	log.Printf("  CSV 文件监听: 启用 (%s)", CSVFile)
	log.Printf("  INI 文件监听: 启用 (%s)", iniPath)
	log.Printf("  订阅源数量: %d", cfgSubUrls)
	if cfgSMTP {
		log.Println("  SMTP 邮件: 启用")
	} else {
		log.Println("  SMTP 邮件: 未配置")
	}
	log.Println("========================================")

	go watchConfig()
	go watchCSV()

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
	// 添加定时执行自动更新节点与规则的脚本任务（每天凌晨 4:00 执行一次）
	_, _ = c.AddFunc("0 4 * * *", func() {
		log.Println("⏰ 定时任务触发: 自动更新节点与规则...")
		// 利用 yaml_config.go 中暴露的内部触发逻辑
		triggerUpdateTask()
	})
	// CF Tunnel 健康监控（每 5 分钟检查一次）
	_, err := c.AddFunc("*/5 * * * *", func() {
		CFTunnelHealthCheck()
	})
	if err != nil {
		log.Printf("[CF Tunnel] cron 注册失败: %v", err)
	} else {
		log.Println("[CF Tunnel] cron 已注册: */5 * * * *")
	}
	c.Start()

	// 启动时立即执行一次 CF Tunnel 健康检查
	go CFTunnelHealthCheck()

	// 5. 启动时立即更新一次订阅数据
	go updateSubscriptionData()

	// 6. 启动 Web 服务
	r := gin.Default()

	// Cloudflare 隧道模式：信任 CF 边缘节点，从 CF-Connecting-IP 提取真实客户端 IP
	r.TrustedPlatform = gin.PlatformCloudflare

	// CORS 中间件（放行 WebSocket 升级头部 + Cloudflare 特有头部）
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers",
			"Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, "+
				"accept, origin, Cache-Control, X-Requested-With, "+
				"Upgrade, Sec-WebSocket-Key, Sec-WebSocket-Version, Sec-WebSocket-Protocol, Sec-WebSocket-Extensions, "+
				"CF-Connecting-IP, CF-IPCountry, CF-Ray")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// API 路由组
	api := r.Group("/api")
	{
		api.POST("/auth", handleAuth)

		// 增加 Token 鉴权中间件保护查询接口
		protected := api.Group("")
		protected.Use(TokenAuthMiddleware())
		{
			protected.GET("/stats", handleGetStats)
			protected.GET("/devices", handleGetDevices)
			protected.GET("/fake/stats", handleFakeGetStats)
			// 触发节点更新的接口，为了安全起见必须鉴权
			protected.POST("/trigger-update", HandleTriggerUpdate)
		}
	}

	// 动态订阅分发路由（自带 token 鉴权）
	r.GET("/sub", handleSub)

	// 模板文件原始分发（自带 token 鉴权，供 proxy-providers / rule-providers 拉取）
	r.GET("/templates/*filepath", handleTemplateFile)

	// 流量上报接口增加 Token 鉴权中间件
	r.POST("/report", TokenAuthMiddleware(), handleReport)

	// WebSocket 实时上报端点（自带鉴权）
	r.GET("/ws", handleWS)

	confLock.RLock()
	port := conf.ListenPort
	confLock.RUnlock()

	fmt.Printf("🚀 流量统计后端启动 | 监听端口 %s\n", port)
	r.Run(port)
}

// ensureDirs 启动时自动创建运行时必需目录，防止因目录缺失而 panic
func ensureDirs() {
	for _, dir := range []string{"./data", "./logs", "./configs", "./templates"} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("⚠️ 创建目录 %s 失败: %v\n", dir, err)
		}
	}
}

func setupLogging() {
	logFile, err := os.OpenFile("./logs/server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
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
