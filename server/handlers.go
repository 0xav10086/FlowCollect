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

	// 4. 统计设备信息 (新增，模仿 fake_api.go)
	var deviceStats []gin.H
	var deviceIDs []string
	db.Model(&TrafficRecord{}).Distinct("device_id").Pluck("device_id", &deviceIDs)

	// a. 预查询，避免N+1
	type DeviceTraffic struct {
		DeviceID string
		Up       int64
		Down     int64
	}
	var todayTraffics, totalTraffics []DeviceTraffic
	db.Model(&TrafficRecord{}).Select("device_id, SUM(up_delta) as up, SUM(down_delta) as down").Where("timestamp >= ?", today).Group("device_id").Scan(&todayTraffics)
	db.Model(&TrafficRecord{}).Select("device_id, SUM(up_delta) as up, SUM(down_delta) as down").Group("device_id").Scan(&totalTraffics)

	type DeviceNodeUsage struct {
		DeviceID, NodeName string
		Up, Down           int64
	}
	var nodeUsages []DeviceNodeUsage
	db.Model(&TrafficRecord{}).Select("device_id, node_name, SUM(up_delta) as up, SUM(down_delta) as down").Where("timestamp >= ?", today).Group("device_id, node_name").Scan(&nodeUsages)

	type DeviceFirstSeen struct {
		DeviceID  string
		FirstSeen time.Time
	}
	var firstSeens []DeviceFirstSeen
	db.Model(&TrafficRecord{}).Select("device_id, MIN(timestamp) as first_seen").Group("device_id").Scan(&firstSeens)

	// b. 数据整理到Map
	todayTrafficMap := make(map[string]DeviceTraffic)
	for _, t := range todayTraffics {
		todayTrafficMap[t.DeviceID] = t
	}
	totalTrafficMap := make(map[string]DeviceTraffic)
	for _, t := range totalTraffics {
		totalTrafficMap[t.DeviceID] = t
	}
	nodeUsageMap := make(map[string][]DeviceNodeUsage)
	for _, u := range nodeUsages {
		nodeUsageMap[u.DeviceID] = append(nodeUsageMap[u.DeviceID], u)
	}
	firstSeenMap := make(map[string]time.Time)
	for _, s := range firstSeens {
		firstSeenMap[s.DeviceID] = s.FirstSeen
	}

	// c. 组装最终数据
	for _, devID := range deviceIDs {
		devToday := todayTrafficMap[devID]
		devTotal := totalTrafficMap[devID]
		devNodes := nodeUsageMap[devID]
		var devNodeDetails []gin.H
		for _, nu := range devNodes {
			devNodeDetails = append(devNodeDetails, gin.H{"name": nu.NodeName, "up_value": nu.Up, "down_value": nu.Down, "formatted_value": formatNetworkBytes(float64(nu.Up + nu.Down))})
		}

		deviceStats = append(deviceStats, gin.H{
			"device_name": devID, "uptime": int64(time.Since(firstSeenMap[devID]).Seconds()),
			"current_up": devToday.Up, "current_down": devToday.Down, "formatted_current_up": formatNetworkBytes(float64(devToday.Up)), "formatted_current_down": formatNetworkBytes(float64(devToday.Down)),
			"total_up": devTotal.Up, "total_down": devTotal.Down, "formatted_total_up": formatNetworkBytes(float64(devTotal.Up)), "formatted_total_down": formatNetworkBytes(float64(devTotal.Down)),
			"active_connections": 0, "closed_connections": 0, "total_connections": 0, // 真实数据中无连接数统计
			"node_usage": devNodeDetails,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"date":       today.Format("2006-01-02"),
		"summary":    summary,
		"node_stats": nodeStats,
		"sub_stats":  subStats,
		"device_stats": deviceStats,
	})
}