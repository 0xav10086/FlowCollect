// HTTP 路由处理函数（Report, Auth, Stats）

package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

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

	if validUser != "" && loginReq.Username == validUser && loginReq.Password == validPass {
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
	db.Model(&TrafficRecord{}).Select("node_name, SUM(up_delta + down_delta) as total, is_proxy").Where("timestamp >= ?", today).Group("node_name").Order("total DESC").Scan(&nodeStats)

	// 2. 统计今日代理 vs 本地总量
	var summary struct {
		ProxyUp   int64 `json:"proxy_up"`
		ProxyDown int64 `json:"proxy_down"`
		LocalUp   int64 `json:"local_up"`
		LocalDown int64 `json:"local_down"`
	}
	db.Model(&TrafficRecord{}).Where("timestamp >= ? AND is_proxy = ?", today, true).Select("COALESCE(SUM(up_delta), 0)").Scan(&summary.ProxyUp)
	db.Model(&TrafficRecord{}).Where("timestamp >= ? AND is_proxy = ?", today, true).Select("COALESCE(SUM(down_delta), 0)").Scan(&summary.ProxyDown)
	db.Model(&TrafficRecord{}).Where("timestamp >= ? AND is_proxy = ?", today, false).Select("COALESCE(SUM(up_delta), 0)").Scan(&summary.LocalUp)
	db.Model(&TrafficRecord{}).Where("timestamp >= ? AND is_proxy = ?", today, false).Select("COALESCE(SUM(down_delta), 0)").Scan(&summary.LocalDown)

	// 3. 获取最新的订阅快照
	var subStats []SubSnapshot
	confLock.RLock()
	subUrls := conf.SubUrls
	confLock.RUnlock()

	for _, url := range subUrls {
		var snap SubSnapshot
		if err := db.Where("sub_url = ?", url).Order("date desc, id desc").First(&snap).Error; err == nil {
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