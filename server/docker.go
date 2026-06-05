// CF Tunnel 健康监控
// 通过网络连通性检测 CF Tunnel 状态，掉线则通过 Docker API 自动重启容器

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

const dockerSocketPath = "/var/run/docker.sock"

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
	confLock.RUnlock()

	if container == "" {
		log.Println("[CF Tunnel] 容器名未配置 (CFTunnelContainer 为空)，跳过检查")
		return
	}

	// 检查外部 CF Tunnel URL（根路径，无需 token）
	externalURL := "https://nas.0xav10086.space/"
	status, err := checkURLReachable(externalURL, 5*time.Second)
	if err != nil || status >= 500 {
		if err != nil {
			log.Printf("[CF Tunnel] 异常 (%s): %v", externalURL, err)
		} else {
			log.Printf("[CF Tunnel] 异常 (%s): HTTP %d", externalURL, status)
		}
		log.Printf("[CF Tunnel] 正在重启容器 %s...", container)
		if err := dockerRestartContainer(container); err != nil {
			log.Printf("[CF Tunnel] 重启失败: %v", err)
		} else {
			log.Printf("[CF Tunnel] 容器 %s 已重启", container)
		}
	} else {
		log.Printf("[CF Tunnel] 正常: HTTP %d", status)
	}
}
