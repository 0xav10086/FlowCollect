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
	// 记录设备启动时间，保证 uptime 稳定增长
	deviceStartTimes = make(map[string]time.Time)
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
	// 临时存储节点流量值，用于后续分配给设备
	nodeValues := make(map[string]struct{ up, down int64 })

	var dist []gin.H
	var totalUp, totalDown int64

	// 2. 循环生成每个节点的流量并计算总和
	for _, name := range nodeNames {
		// 生成 up/down 流量, 上行通常小于下行
		upVal := rand.Int64N(MaxTraffic/4) + (MinTraffic / 10)
		downVal := rand.Int64N(MaxTraffic) + MinTraffic

		totalUp += upVal
		totalDown += downVal
		nodeValues[name] = struct{ up, down int64 }{up: upVal, down: downVal}

		dist = append(dist, gin.H{
			"name":  name,
			"up_value":   upVal,
			"down_value": downVal,
		})
	}

	// 3. 生成设备统计数据 (Device Stats)
	// 定义设备与节点的映射关系，确保数据逻辑自洽
	deviceMap := map[string][]string{
		"OpenWrt Gateway": {"Hong Kong 01", "Singapore 04", "Taiwan 05"},
		"Windows PC":      {"Japan 02", "USA 03", "Direct"},
		"Android Phone":   {"Korea 06"},
	}

	var deviceStats []gin.H
	for devName, nodes := range deviceMap {
		// 初始化设备启动时间 (如果不存在)
		if _, ok := deviceStartTimes[devName]; !ok {
			// 随机生成 1小时 到 7天前
			deviceStartTimes[devName] = time.Now().Add(-time.Duration(rand.Int64N(7*24*3600)+3600) * time.Second)
		}

		var devCurrentUp, devCurrentDown int64
		var devNodeDetails []gin.H

		// 计算该设备本次更新使用的流量（基于分配的节点）
		for _, nodeName := range nodes {
			if val, ok := nodeValues[nodeName]; ok {
				devCurrentUp += val.up
				devCurrentDown += val.down
				devNodeDetails = append(devNodeDetails, gin.H{
					"name":            nodeName,
					"up_value":        val.up,
					"down_value":      val.down,
					"formatted_value": formatNetworkBytes(float64(val.up + val.down)),
				})
			}
		}

		// 模拟总流量 (本次流量 + 随机历史基数)
		devTotalUp := devCurrentUp + rand.Int64N(200*MB)
		devTotalDown := devCurrentDown + rand.Int64N(800*MB)

		// 模拟连接数
		activeConns := rand.IntN(50) + 5
		closedConns := rand.IntN(200) + 20
		uptime := int64(time.Since(deviceStartTimes[devName]).Seconds())

		deviceStats = append(deviceStats, gin.H{
			"device_name":        devName,
			"current_up":         devCurrentUp,
			"current_down":       devCurrentDown,
			"formatted_current_up":   formatNetworkBytes(float64(devCurrentUp)),
			"formatted_current_down": formatNetworkBytes(float64(devCurrentDown)),
			"total_up":           devTotalUp,
			"total_down":         devTotalDown,
			"formatted_total_up":     formatNetworkBytes(float64(devTotalUp)),
			"formatted_total_down":   formatNetworkBytes(float64(devTotalDown)),
			"active_connections": activeConns,
			"closed_connections": closedConns,
			"total_connections":  activeConns + closedConns,
			"node_usage":         devNodeDetails,
			"uptime":             uptime,
		})
	}

	// 4. 构建响应
	response := gin.H{
		"success": true,
		"message": "数据获取成功",
		"system_info": gin.H{
			"uptime":       fmt.Sprintf("%dh", rand.IntN(100)+1),
			"load_average": []float64{rand.Float64() * 2, rand.Float64() * 2, rand.Float64() * 2},
			"last_updated": time.Now().Format("2006-01-02 15:04:05"),
		},
		"historical": []gin.H{
			// The first historical point is what the line chart fetches
			{"time": time.Now().Format("15:04:05"), "up_value": formatNetworkBytes(float64(totalUp)), "down_value": formatNetworkBytes(float64(totalDown))},
			{"time": time.Now().Add(-5 * time.Second).Format("15:04:05"), "up_value": formatNetworkBytes(float64(totalUp) * 0.8), "down_value": formatNetworkBytes(float64(totalDown) * 0.9)},
		},
		"node_distribution": dist,
		"device_stats":      deviceStats,
	}

	c.JSON(http.StatusOK, response)
}
