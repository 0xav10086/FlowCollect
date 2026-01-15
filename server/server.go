//go:build server
// +build server

package main

import (
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/robfig/cron/v3"
	"gopkg.in/ini.v1"
	"gorm.io/gorm"
)

// 配置结构体
type ServerConfig struct {
	ListenPort  string
	ServerToken string
	DBPath      string
	SMTPHost    string
	SMTPPort    string
	EmailUser   string
	EmailPass   string
	EmailTo     string
	SubUrls     []string
}

var (
	conf     ServerConfig
	confLock sync.RWMutex
	db       *gorm.DB
	iniPath  = "ServerSetting.ini"
)

func init() {
	if err := loadConfig(); err != nil {
		log.Printf("初始加载配置文件失败: %v", err)
	}
	go watchConfig()
}

func loadConfig() error {
	cfg, err := ini.Load(iniPath)
	if err != nil {
		return err
	}

	confLock.Lock()
	defer confLock.Unlock()

	section := cfg.Section("server")
	smtpSec := cfg.Section("smtp")

	conf = ServerConfig{
		ListenPort:  section.Key("ListenPort").MustString(":8686"),
		ServerToken: section.Key("ServerToken").MustString("YourSecretToken"),
		DBPath:      section.Key("DBPath").MustString("traffic.db"),
		SMTPHost:    smtpSec.Key("SMTPHost").MustString("smtp.qq.com"),
		SMTPPort:    smtpSec.Key("SMTPPort").MustString("587"),
		EmailUser:   smtpSec.Key("EmailUser").MustString(""),
		EmailPass:   smtpSec.Key("EmailPass").MustString(""),
		EmailTo:     smtpSec.Key("EmailTo").MustString(""),
		SubUrls:     section.Key("SubUrls").Strings(","),
	}

	log.Printf("[%s] 服务端配置已更新", time.Now().Format("15:04:05"))
	return nil
}

func watchConfig() {
	watcher, _ := fsnotify.NewWatcher()
	defer watcher.Close()
	_ = watcher.Add(iniPath)
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				loadConfig()
			}
		}
	}
}

// 数据库模型保持不变
type TrafficRecord struct {
	ID        uint      `gorm:"primaryKey"`
	Timestamp time.Time `gorm:"index"`
	DeviceID  string
	NodeName  string
	UpDelta   int64
	DownDelta int64
	IsProxy   bool
}

type SubSnapshot struct {
	ID     uint      `gorm:"primaryKey"`
	Date   time.Time `gorm:"index"`
	SubUrl string
	Used   int64
	Total  int64
}

func initDB() {
	confLock.RLock()
	path := conf.DBPath
	confLock.RUnlock()

	var err error
	db, err = gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	db.AutoMigrate(&TrafficRecord{}, &SubSnapshot{})
}

func main() {
	initDB()

	// 定时任务
	c := cron.New()
	_, _ = c.AddFunc("55 23 * * *", func() {
		processDailyReport()
	})
	c.Start()

	r := gin.Default()

	// 1. 流量上报接口
	r.POST("/report", handleReport)

	// 2. 认证接口 (供 Vue3 使用)
	r.POST("/api/auth", handleAuth)

	// 3. API 数据接口 (供 Vue3 使用)
	r.GET("/api/stats", handleGetStats)

	confLock.RLock()
	port := conf.ListenPort
	confLock.RUnlock()

	fmt.Printf("流量统计后端启动 | 监听端口 %s\n", port)
	r.Run(port)
}

// 提取的上报处理逻辑
func handleReport(c *gin.Context) {
	confLock.RLock()
	token := conf.ServerToken
	confLock.RUnlock()

	if c.GetHeader("Authorization") != "Bearer "+token {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var data struct {
		Timestamp int64  `json:"timestamp"`
		DeviceID  string `json:"device_id"`
		NodeName  string `json:"node_name"`
		UpDelta   int64  `json:"up_delta"`
		DownDelta int64  `json:"down_delta"`
		IsProxy   bool   `json:"is_proxy"`
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db.Create(&TrafficRecord{
		Timestamp: time.Unix(data.Timestamp, 0),
		DeviceID:  data.DeviceID,
		NodeName:  data.NodeName,
		UpDelta:   data.UpDelta,
		DownDelta: data.DownDelta,
		IsProxy:   data.IsProxy,
	})
	c.Status(http.StatusOK)
}

// 处理登录认证
func handleAuth(c *gin.Context) {
	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	confLock.RLock()
	validUser := conf.EmailUser
	validPass := conf.ServerToken
	confLock.RUnlock()

	// 校验：EmailUser 必须已配置，且账号密码匹配
	if validUser != "" && loginReq.Username == validUser && loginReq.Password == validPass {
		// 登录成功，返回 ServerToken 作为前端的 Bearer Token
		c.JSON(http.StatusOK, gin.H{"token": validPass})
		return
	}

	c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误，请检查 ServerSetting.ini"})
}

// Vue3 后端接口逻辑
func handleGetStats(c *gin.Context) {
	today := time.Now().Truncate(24 * time.Hour)

	// 1. 统计今日各节点流量排行
	type NodeResult struct {
		NodeName string `json:"node_name"`
		Total    int64  `json:"total"`
		IsProxy  bool   `json:"is_proxy"`
	}
	var nodeStats []NodeResult
	db.Model(&TrafficRecord{}).
		Select("node_name, SUM(up_delta + down_delta) as total, is_proxy").
		Where("timestamp >= ?", today).
		Group("node_name").
		Order("total DESC").
		Scan(&nodeStats)

	// 2. 统计今日代理 vs 本地总量
	var summary struct {
		Proxy int64 `json:"proxy"`
		Local int64 `json:"local"`
	}
	db.Model(&TrafficRecord{}).Where("timestamp >= ? AND is_proxy = ?", today, true).Select("SUM(up_delta + down_delta)").Scan(&summary.Proxy)
	db.Model(&TrafficRecord{}).Where("timestamp >= ? AND is_proxy = ?", today, false).Select("SUM(up_delta + down_delta)").Scan(&summary.Local)

	// 3. 获取最新的订阅快照 (新增)
	var subStats []SubSnapshot
	confLock.RLock()
	subUrls := conf.SubUrls
	confLock.RUnlock()

	for _, url := range subUrls {
		var snap SubSnapshot
		// 查找该URL最新的记录
		if err := db.Where("sub_url = ?", url).Order("date desc").First(&snap).Error; err == nil {
			subStats = append(subStats, snap)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"date":       today.Format("2006-01-02"),
		"summary":    summary,
		"node_stats": nodeStats,
		"sub_stats":  subStats,
	})
}

// --- 业务逻辑 ---

func fetchSubInfo(url string) (used, total int64, err error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	// 解析机场特有的 Subscription-Userinfo 响应头
	info := resp.Header.Get("Subscription-Userinfo")
	if info == "" {
		return 0, 0, fmt.Errorf("未找到流量头信息")
	}

	parts := strings.Split(info, ";")
	var up, down int64
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "upload=") {
			up, _ = strconv.ParseInt(strings.Split(p, "=")[1], 10, 64)
		} else if strings.HasPrefix(p, "download=") {
			down, _ = strconv.ParseInt(strings.Split(p, "=")[1], 10, 64)
		} else if strings.HasPrefix(p, "total=") {
			total, _ = strconv.ParseInt(strings.Split(p, "=")[1], 10, 64)
		}
	}
	return up + down, total, nil
}

func processDailyReport() {
	today := time.Now().Truncate(24 * time.Hour)

	// 1. 本地统计
	var proxyTotal, localTotal int64
	db.Model(&TrafficRecord{}).Where("timestamp >= ? AND is_proxy = ?", today, true).Select("SUM(up_delta + down_delta)").Scan(&proxyTotal)
	db.Model(&TrafficRecord{}).Where("timestamp >= ? AND is_proxy = ?", today, false).Select("SUM(up_delta + down_delta)").Scan(&localTotal)

	// 2. 获取配置中的订阅URL
	confLock.RLock()
	subUrls := conf.SubUrls
	confLock.RUnlock()

	// 3. 机场统计与预测
	subMsg := "【机场详情】\n"
	var totalAirportUsageToday int64
	for _, url := range subUrls {
		used, total, err := fetchSubInfo(url)
		if err == nil {
			db.Create(&SubSnapshot{Date: today, SubUrl: url, Used: used, Total: total})

			var lastSub SubSnapshot
			db.Where("sub_url = ? AND date < ?", url, today).Order("date desc").First(&lastSub)

			dailyUsed := used - lastSub.Used
			if dailyUsed < 0 {
				dailyUsed = 0
			} // 防止机场重置流量导致负数
			totalAirportUsageToday += dailyUsed

			daysLeft := int64(0)
			if dailyUsed > 0 {
				daysLeft = (total - used) / dailyUsed
			}

			subMsg += fmt.Sprintf("- 机场: %s...\n  今日消耗: %s | 剩余: %s | 预计还可用 %d 天\n",
				url[:15], formatBytes(dailyUsed), formatBytes(total-used), daysLeft)
		}
	}

	// 4. 泄露检查
	leakMsg := "【代理泄露检查】: 正常"
	diff := totalAirportUsageToday - proxyTotal
	if diff < 0 {
		diff = -diff
	}
	// 差异超过 100MB 且比例超过 20% 时报警
	if diff > (100*1024*1024) && diff > (proxyTotal/5) {
		leakMsg = fmt.Sprintf("【⚠️ 代理泄露警告】\n机场扣除流量(%s) 与 本地统计代理流量(%s) 存在显著差异！差异值: %s",
			formatBytes(totalAirportUsageToday), formatBytes(proxyTotal), formatBytes(diff))
	}

	// 5. 获取邮件配置
	confLock.RLock()
	emailTo := conf.EmailTo
	confLock.RUnlock()

	// 6. 发送邮件
	subject := fmt.Sprintf("流量日报 - %s", today.Format("2006-01-02"))
	body := fmt.Sprintf("今日汇总:\n- 代理流量: %s\n- 本地流量: %s\n- 总计流量: %s\n\n%s\n\n%s",
		formatBytes(proxyTotal), formatBytes(localTotal), formatBytes(proxyTotal+localTotal), subMsg, leakMsg)

	sendEmail(subject, body)

	// 记录日志
	log.Printf("日报邮件已发送至: %s", emailTo)
}

func sendEmail(subject, body string) {
	// 获取邮件配置
	confLock.RLock()
	smtpHost := conf.SMTPHost
	smtpPort := conf.SMTPPort
	emailUser := conf.EmailUser
	emailPass := conf.EmailPass
	emailTo := conf.EmailTo
	confLock.RUnlock()

	// 检查配置是否完整
	if smtpHost == "" || smtpPort == "" || emailUser == "" || emailPass == "" || emailTo == "" {
		log.Println("邮件配置不完整，跳过发送")
		return
	}

	header := make(map[string]string)
	header["From"] = emailUser
	header["To"] = emailTo
	header["Subject"] = subject
	header["Content-Type"] = "text/plain; charset=UTF-8"

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	auth := smtp.PlainAuth("", emailUser, emailPass, smtpHost)

	// 使用 TLS 连接提高成功率
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, emailUser, []string{emailTo}, []byte(message))
	if err != nil {
		log.Println("邮件发送失败:", err)
	}
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
