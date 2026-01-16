//go:build server
// +build server

// 业务逻辑。包含订阅抓取、日报处理、邮件发送等核心逻辑

package main

import (
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

func fetchSubInfo(url string) (used, total, expire int64, err error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, 0, 0, err
	}
	defer resp.Body.Close()

	info := resp.Header.Get("Subscription-Userinfo")
	if info == "" {
		return 0, 0, 0, fmt.Errorf("未找到流量头信息")
	}

	parts := strings.Split(info, ";")
	var up, down, exp int64
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "upload=") {
			up, _ = strconv.ParseInt(strings.Split(p, "=")[1], 10, 64)
		} else if strings.HasPrefix(p, "download=") {
			down, _ = strconv.ParseInt(strings.Split(p, "=")[1], 10, 64)
		} else if strings.HasPrefix(p, "total=") {
			total, _ = strconv.ParseInt(strings.Split(p, "=")[1], 10, 64)
		} else if strings.HasPrefix(p, "expire=") {
			exp, _ = strconv.ParseInt(strings.Split(p, "=")[1], 10, 64)
		}
	}
	return up + down, total, exp, nil
}

func processDailyReport() {
	today := time.Now().Truncate(24 * time.Hour)

	var proxyTotal, localTotal int64
	db.Model(&TrafficRecord{}).Where("timestamp >= ? AND is_proxy = ?", today, true).Select("SUM(up_delta + down_delta)").Scan(&proxyTotal)
	db.Model(&TrafficRecord{}).Where("timestamp >= ? AND is_proxy = ?", today, false).Select("SUM(up_delta + down_delta)").Scan(&localTotal)

	confLock.RLock()
	subUrls := conf.SubUrls
	confLock.RUnlock()

	subMsg := "【机场详情】\n"
	var totalAirportUsageToday int64
	for _, url := range subUrls {
		used, total, expire, err := fetchSubInfo(url)
		if err == nil {
			db.Create(&SubSnapshot{Date: today, SubUrl: url, Used: used, Total: total, Expire: expire})

			var lastSub SubSnapshot
			db.Where("sub_url = ? AND date < ?", url, today).Order("date desc").First(&lastSub)

			dailyUsed := used - lastSub.Used
			if dailyUsed < 0 {
				dailyUsed = 0
			}
			totalAirportUsageToday += dailyUsed

			daysLeft := int64(0)
			if dailyUsed > 0 {
				daysLeft = (total - used) / dailyUsed
			}

			subMsg += fmt.Sprintf("- 机场: %s...\n  今日消耗: %s | 剩余: %s | 预计还可用 %d 天\n",
				url[:15], formatBytes(dailyUsed), formatBytes(total-used), daysLeft)
		}
	}

	leakMsg := "【代理泄露检查】: 正常"
	diff := totalAirportUsageToday - proxyTotal
	if diff < 0 {
		diff = -diff
	}
	if diff > (100*1024*1024) && diff > (proxyTotal/5) {
		leakMsg = fmt.Sprintf("【⚠️ 代理泄露警告】\n机场扣除流量(%s) 与 本地统计代理流量(%s) 存在显著差异！差异值: %s",
			formatBytes(totalAirportUsageToday), formatBytes(proxyTotal), formatBytes(diff))
	}

	confLock.RLock()
	emailTo := conf.EmailTo
	confLock.RUnlock()

	subject := fmt.Sprintf("流量日报 - %s", today.Format("2006-01-02"))
	body := fmt.Sprintf("今日汇总:\n- 代理流量: %s\n- 本地流量: %s\n- 总计流量: %s\n\n%s\n\n%s",
		formatBytes(proxyTotal), formatBytes(localTotal), formatBytes(proxyTotal+localTotal), subMsg, leakMsg)

	sendEmail(subject, body)
	log.Printf("日报邮件已发送至: %s", emailTo)
}

func updateSubscriptionData() {
	log.Println("正在执行启动时订阅数据更新...")
	today := time.Now().Truncate(24 * time.Hour)

	confLock.RLock()
	subUrls := conf.SubUrls
	confLock.RUnlock()

	for _, url := range subUrls {
		used, total, expire, err := fetchSubInfo(url)
		if err == nil {
			db.Create(&SubSnapshot{Date: today, SubUrl: url, Used: used, Total: total, Expire: expire})
			log.Printf("订阅 [%s] 更新成功 | 已用: %s", url, formatBytes(used))
		} else {
			log.Printf("订阅 [%s] 更新失败: %v", url, err)
		}
	}
}

func sendEmail(subject, body string) {
	confLock.RLock()
	defer confLock.RUnlock()
	if conf.SMTPHost == "" || conf.EmailTo == "" {
		return
	}
	auth := smtp.PlainAuth("", conf.EmailUser, conf.EmailPass, conf.SMTPHost)
	msg := []byte("To: " + conf.EmailTo + "\r\nSubject: " + subject + "\r\n\r\n" + body)
	smtp.SendMail(conf.SMTPHost+":"+conf.SMTPPort, auth, conf.EmailUser, []string{conf.EmailTo}, msg)
}