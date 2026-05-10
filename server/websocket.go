// WebSocket 实时上报接收端点

package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for development; in production, restrict this
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// handleWS upgrades the HTTP connection to WebSocket and receives real-time traffic reports.
// The client connects to GET /ws with Authorization: Bearer <token> header.
func handleWS(c *gin.Context) {
	// Authenticate via header (sent during WebSocket handshake)
	confLock.RLock()
	token := conf.ServerToken
	confLock.RUnlock()

	authHeader := c.GetHeader("Authorization")
	if authHeader != "Bearer "+token {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WS] 升级失败: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("[WS] 客户端已连接: %s", c.ClientIP())

	for {
		var data struct {
			Timestamp   int64  `json:"timestamp"`
			DeviceID    string `json:"device_id"`
			NodeName    string `json:"node_name"`
			UpDelta     int64  `json:"up_delta"`
			DownDelta   int64  `json:"down_delta"`
			IsProxy     bool   `json:"is_proxy"`
			ActiveConns int    `json:"active_connections"`
		}

		err := conn.ReadJSON(&data)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[WS] 读取错误: %v", err)
			} else {
				log.Printf("[WS] 客户端断开: %s (%v)", c.ClientIP(), err)
			}
			break
		}

		// Write to database (same logic as handleReport)
		result := db.Create(&TrafficRecord{
			Timestamp:   time.Unix(data.Timestamp, 0),
			DeviceID:    data.DeviceID,
			NodeName:    data.NodeName,
			UpDelta:     data.UpDelta,
			DownDelta:   data.DownDelta,
			IsProxy:     data.IsProxy,
			ActiveConns: data.ActiveConns,
		})

		if result.Error != nil {
			log.Printf("[WS] 数据库写入失败: %v", result.Error)
		} else {
			log.Printf("[WS] 已接收 | 设备: %s | 节点: %s | ↑%d ↓%d | 连接数: %d",
				data.DeviceID, data.NodeName, data.UpDelta, data.DownDelta, data.ActiveConns)
		}
	}

	log.Printf("[WS] 连接关闭: %s", c.ClientIP())
}
