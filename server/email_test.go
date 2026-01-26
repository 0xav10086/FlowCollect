//go:build test
// +build test

package main

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestSendEmail(t *testing.T) {
	log.Println("=== 开始测试邮件发送功能 ===")

	// 1. 主动加载配置 (调用 config.go 中的 loadConfig)
	// 注意：确保 ServerSetting.ini 在当前运行目录下
	if err := loadConfig(); err != nil {
		t.Fatalf("❌ 配置文件加载失败: %v", err)
	}

	// 检查配置是否已加载 (sendEmail 依赖全局变量 conf)
	confLock.RLock()
	host := conf.SMTPHost
	to := conf.EmailTo
	confLock.RUnlock()

	if host == "" {
		t.Fatal("⚠️ 警告: conf.SMTPHost 为空，请检查 ServerSetting.ini")
	}
	log.Printf("配置已加载 | SMTP Host: %s | 发送给: %s", host, to)

	subject := fmt.Sprintf("FlowCollect Test - %s", time.Now().Format("2006-01-02 15:04:05"))
	body := "这是一封测试邮件，用于验证 sendEmail 函数。\n\n来自: server/test.go"

	log.Printf("准备发送邮件...\n主题: %s", subject)
	sendEmail(subject, body)
	log.Println("✅ sendEmail 函数调用结束，请检查收件箱。")
}
