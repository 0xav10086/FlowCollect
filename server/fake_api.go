//go:build server
// +build server

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

	response := gin.H{
		"success": true,
		"message": "数据获取成功",
		"data":    currentStats,
		"system_info": gin.H{
			"uptime":       fmt.Sprintf("%dh", rand.IntN(100)+1), // 修复: Intn -> IntN
			"load_average": []float64{rand.Float64() * 2, rand.Float64() * 2, rand.Float64() * 2},
			"last_updated": time.Now().Format("2006-01-02 15:04:05"),
		},
		"historical": []gin.H{
			{"time": time.Now().Add(-10 * time.Second).Format("15:04:05"), "value": formatNetworkBytes(rand.Float64() * 1e11)},
			{"time": time.Now().Add(-5 * time.Second).Format("15:04:05"), "value": formatNetworkBytes(rand.Float64() * 1e11)},
		},
	}
	c.JSON(http.StatusOK, response)
}