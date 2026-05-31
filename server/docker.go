// Docker Engine API 客户端（通过 Unix Socket 通信）
// 用于监控和管理同宿主机上的其他 Docker 容器

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

const dockerSocketPath = "/var/run/docker.sock"

// dockerHTTPClient 返回一个通过 Unix Socket 连接 Docker Engine 的 HTTP 客户端
func dockerHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.DialTimeout("unix", dockerSocketPath, 5*time.Second)
			},
		},
	}
}

// dockerContainerState 返回容器的运行状态
type dockerContainerState struct {
	Running  bool `json:"Running"`
	Paused   bool `json:"Paused"`
	Restarting bool `json:"Restarting"`
}

// dockerInspectResponse 精简的 Docker inspect 响应
type dockerInspectResponse struct {
	State dockerContainerState `json:"State"`
	Name  string               `json:"Name"`
}

// DockerInspect 检查指定容器是否正在运行
// 返回 (running, error)。容器不存在返回 false + error
func DockerInspect(containerName string) (bool, error) {
	socketExists, err := fileExists(dockerSocketPath)
	if err != nil || !socketExists {
		return false, fmt.Errorf("Docker socket not available at %s", dockerSocketPath)
	}

	client := dockerHTTPClient()
	url := fmt.Sprintf("http://localhost/containers/%s/json", containerName)

	resp, err := client.Get(url)
	if err != nil {
		return false, fmt.Errorf("Docker inspect failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, fmt.Errorf("container %s not found", containerName)
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("Docker inspect returned status %d", resp.StatusCode)
	}

	var result dockerInspectResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("Docker inspect decode failed: %w", err)
	}

	return result.State.Running, nil
}

// DockerRestart 重启指定容器
func DockerRestart(containerName string) error {
	socketExists, err := fileExists(dockerSocketPath)
	if err != nil || !socketExists {
		return fmt.Errorf("Docker socket not available at %s", dockerSocketPath)
	}

	client := dockerHTTPClient()
	// timeout=30 给容器 30 秒关闭时间
	url := fmt.Sprintf("http://localhost/containers/%s/restart?t=30", containerName)

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

// CheckAndRestartCFTunnel 检查 CF Tunnel 容器状态，掉线则自动重启
func CheckAndRestartCFTunnel(containerName string) {
	if containerName == "" {
		return
	}

	running, err := DockerInspect(containerName)
	if err != nil {
		log.Printf("[Docker] CF Tunnel 检查失败: %v", err)
		return
	}

	if running {
		return
	}

	log.Printf("[Docker] CF Tunnel 容器 %s 未运行，正在重启...", containerName)
	if err := DockerRestart(containerName); err != nil {
		log.Printf("[Docker] CF Tunnel 重启失败: %v", err)
		return
	}
	log.Printf("[Docker] CF Tunnel 容器 %s 已重启", containerName)
}

// fileExists 检查文件是否存在
func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
