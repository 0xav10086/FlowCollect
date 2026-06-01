// CF Tunnel 健康监控
// 通过网络连通性检测 CF Tunnel 状态，掉线则通过 Docker API 自动重启容器

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

const dockerSocketPath = "/var/run/docker.sock"

var (
	cfTunnelFailCount int
	cfTunnelFailLock  sync.Mutex
)

// dockerClient 返回一个通过 Unix Socket 连接 Docker Engine 的 HTTP 客户端
func dockerClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.DialTimeout("unix", dockerSocketPath, 5*time.Second)
			},
		},
	}
}

// dockerRestartContainer 通过 Docker API 重启指定容器
func dockerRestartContainer(name string) error {
	client := dockerClient()
	url := fmt.Sprintf("http://localhost/containers/%s/restart?t=30", name)
	resp, err := client.Post(url, "", nil)
	if err != nil {
		return fmt.Errorf("Docker restart failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Docker restart returned status %d", resp.StatusCode)
	}
	return nil
}

// CheckAndRestartCFTunnel 检查 CF Tunnel 连通性，连续失败 3 次则重启容器
func CheckAndRestartCFTunnel(checkURL, containerName string) {
	if checkURL == "" || containerName == "" {
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(checkURL)

	cfTunnelFailLock.Lock()
	defer cfTunnelFailLock.Unlock()

	if err != nil {
		cfTunnelFailCount++
		log.Printf("[CF Tunnel] 连通性检查失败 (%d/3): %v", cfTunnelFailCount, err)
	} else {
		resp.Body.Close()
		if resp.StatusCode >= 500 {
			cfTunnelFailCount++
			log.Printf("[CF Tunnel] 服务端错误 (%d/3): HTTP %d", cfTunnelFailCount, resp.StatusCode)
		} else {
			if cfTunnelFailCount > 0 {
				log.Printf("[CF Tunnel] 连通恢复正常，重置失败计数")
			}
			cfTunnelFailCount = 0
			return
		}
	}

	if cfTunnelFailCount < 3 {
		return
	}

	log.Printf("[CF Tunnel] 连续失败 %d 次，正在重启容器 %s...", cfTunnelFailCount, containerName)
	cfTunnelFailCount = 0

	if err := dockerRestartContainer(containerName); err != nil {
		log.Printf("[CF Tunnel] 重启失败: %v", err)
		return
	}
	log.Printf("[CF Tunnel] 容器 %s 已重启", containerName)
}

// CFTunnelHealthCheck 执行 CF Tunnel 健康检查（供 cron 调用）
func CFTunnelHealthCheck() {
	confLock.RLock()
	container := conf.CFTunnelContainer
	token := conf.ServerToken
	listenPort := conf.ListenPort
	confLock.RUnlock()

	if container == "" {
		return
	}

	// 用本地端口做基本健康检查
	checkURL := fmt.Sprintf("http://127.0.0.1%s/sub?token=%s", listenPort, token)
	CheckAndRestartCFTunnel(checkURL, container)
}
