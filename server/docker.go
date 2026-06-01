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

// checkURLReachable 检查 URL 是否可达，返回 (statusCode, error)
func checkURLReachable(url string, timeout time.Duration) (int, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

// CFTunnelHealthCheck 执行 CF Tunnel 健康检查（供 cron 和启动时调用）
func CFTunnelHealthCheck() {
	log.Println("[CF Tunnel] 开始健康检查...")

	confLock.RLock()
	container := conf.CFTunnelContainer
	token := conf.ServerToken
	listenPort := conf.ListenPort
	confLock.RUnlock()

	if container == "" {
		log.Println("[CF Tunnel] 容器名未配置 (CFTunnelContainer 为空)，跳过检查")
		return
	}

	log.Printf("[CF Tunnel] 目标容器: %s", container)

	// 检查 1: 外部 CF Tunnel URL
	externalURL := fmt.Sprintf("https://nas.0xav10086.space/sub?token=%s", token)
	extStatus, extErr := checkURLReachable(externalURL, 10*time.Second)
	if extErr != nil {
		log.Printf("[CF Tunnel] 外部 URL 不可达 (%s): %v", externalURL, extErr)
	} else {
		log.Printf("[CF Tunnel] 外部 URL 正常: HTTP %d", extStatus)
	}

	// 检查 2: 本地端口
	localURL := fmt.Sprintf("http://127.0.0.1%s/sub?token=%s", listenPort, token)
	localStatus, localErr := checkURLReachable(localURL, 5*time.Second)
	if localErr != nil {
		log.Printf("[CF Tunnel] 本地端口不可达 (%s): %v", localURL, localErr)
	} else {
		log.Printf("[CF Tunnel] 本地端口正常: HTTP %d", localStatus)
	}

	// 综合判断
	cfTunnelFailLock.Lock()
	defer cfTunnelFailLock.Unlock()

	if extErr != nil || localErr != nil || extStatus >= 500 || localStatus >= 500 {
		cfTunnelFailCount++
		log.Printf("[CF Tunnel] 检查异常 (%d/3)", cfTunnelFailCount)
	} else {
		if cfTunnelFailCount > 0 {
			log.Printf("[CF Tunnel] 所有检查通过，重置失败计数")
		}
		cfTunnelFailCount = 0
		return
	}

	// 连续失败 3 次，执行重启
	if cfTunnelFailCount >= 3 {
		log.Printf("[CF Tunnel] 连续失败 %d 次，正在重启容器 %s...", cfTunnelFailCount, container)
		cfTunnelFailCount = 0
		if err := dockerRestartContainer(container); err != nil {
			log.Printf("[CF Tunnel] 重启失败: %v", err)
		} else {
			log.Printf("[CF Tunnel] 容器 %s 已重启", container)
		}
	}
}
