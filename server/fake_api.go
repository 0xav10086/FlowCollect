// Test 模块。包含 Fake API 的处理逻辑和随机数据生成

package main

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	statsLock    sync.RWMutex
	currentStats = gin.H{
		"status": "active",
	}
)

// 仅供测试使用，这是一个随机流量生成器
func handleFakeGetStats(c *gin.Context) {
	statsLock.RLock()
	defer statsLock.RUnlock()

	// 1. 定义常量，方便维护
	const (
		MB         = 1024 * 1024
		MinTraffic = 1 * MB
		MaxTraffic = 2 * MB
	)

	nodeNames := []string{"Hong Kong 01", "Japan 02", "USA 03", "Singapore 04", "Taiwan 05", "Korea 06", "Direct"}

	var dist []gin.H
	var totalTraffic int64 = 0 // 用于存储累加后的总流量

	// 2. 循环生成每个节点的流量并计算总和
	for _, name := range nodeNames {
		// 生成 1MB 到 2MB 之间的随机流量
		// rand.Int63n(1MB) 会生成 0 到 1MB-1 的值，加上 1MB 后范围是 [1MB, 2MB)
		val := rand.Int64N((MaxTraffic - MinTraffic) + MinTraffic)

		totalTraffic += val

		dist = append(dist, gin.H{
			"name":  name,
			"value": val,
		})
	}

	// 3. 构建响应，确保 data 中的总流量等于 totalTraffic
	response := gin.H{
		"success": true,
		"message": "数据获取成功",
		// 关键点：这里不再单纯使用 currentStats，而是反映真实的累加值
		"data": gin.H{
			"total": totalTraffic, // 或者根据你 currentStats 的结构进行赋值
			// 如果 currentStats 里还有其他字段，可以这样：
			// "details": currentStats,
		},
		"system_info": gin.H{
			"uptime":       fmt.Sprintf("%dh", rand.IntN(100)+1),
			"load_average": []float64{rand.Float64() * 2, rand.Float64() * 2, rand.Float64() * 2},
			"last_updated": time.Now().Format("2006-01-02 15:04:05"),
		},
		"historical": []gin.H{
			{"time": time.Now().Add(-10 * time.Second).Format("15:04:05"), "value": formatNetworkBytes(float64(totalTraffic) * 0.8)}, // 模拟历史值
			{"time": time.Now().Add(-5 * time.Second).Format("15:04:05"), "value": formatNetworkBytes(float64(totalTraffic))},
		},
		"node_distribution": dist,
	}

	c.JSON(http.StatusOK, response)
}
