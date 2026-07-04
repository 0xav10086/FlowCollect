// 通用 URL 可达性检查工具

package main

import (
	"log"
	"net/http"
	"time"
)

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

// HealthCheck 执行通用健康检查（供 cron 和启动时调用）
// 从配置读取 HealthCheckURL，仅报告状态，不自动重启容器
func HealthCheck() {
	confLock.RLock()
	checkURL := conf.HealthCheckURL
	confLock.RUnlock()

	if checkURL == "" {
		return
	}

	log.Printf("[HealthCheck] 检查 %s ...", checkURL)
	status, err := checkURLReachable(checkURL, 5*time.Second)
	if err != nil {
		log.Printf("[HealthCheck] 异常 (%s): %v", checkURL, err)
	} else if status >= 500 {
		log.Printf("[HealthCheck] 异常 (%s): HTTP %d", checkURL, status)
	} else {
		log.Printf("[HealthCheck] 正常: HTTP %d", status)
	}
}
