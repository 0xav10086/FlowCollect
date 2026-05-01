// 配置管理。负责加载/监听 ServerSetting.ini

package main

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/ini.v1"
)

// 配置结构体
type ServerConfig struct {
	ListenPort  string
	ServerToken string
	DBPath      string
	SMTPHost    string
	SMTPPort    string
	EmailUser   string
	EmailPass   string
	EmailTo     string
	SubUrls     map[string]string // 变更为 map，键为 filename，值为 url
}

var (
	conf     ServerConfig
	confLock sync.RWMutex
	iniPath  = "ServerSetting.ini"
)

func loadConfig() error {
	cfg, err := ini.Load(iniPath)
	if err != nil {
		return err
	}

	confLock.Lock()
	defer confLock.Unlock()

	section := cfg.Section("server")
	smtpSec := cfg.Section("smtp")

	// 解析 SubUrls 字段
	// 期望格式：["bemly_node.yaml"]="https://...", ["cf_node.yaml"]="https://..."
	subUrlsMap := make(map[string]string)
	subUrlsStr := section.Key("SubUrls").String()

	// 按逗号分割，注意 URL 内部可能带有逗号，但通常来说 ini 的这种格式逗号在引号外或者不影响外层分割
	// 为稳妥起见，这里按 "]="" 分割或者简单的字符串处理
	parts := strings.Split(subUrlsStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// 寻找 [" 和 "]="
		startIdx := strings.Index(part, "[\"")
		if startIdx == -1 {
			startIdx = strings.Index(part, "[") // 容错：没有引号
		}

		equalIdx := strings.Index(part, "=")
		if startIdx != -1 && equalIdx != -1 && startIdx < equalIdx {
			// 提取文件名
			fileNamePart := part[startIdx+1 : equalIdx]
			fileNamePart = strings.TrimSpace(fileNamePart)
			fileNamePart = strings.TrimSuffix(fileNamePart, "]")
			fileNamePart = strings.Trim(fileNamePart, `"'`) // 去除内部引号

			// 提取 URL
			urlPart := part[equalIdx+1:]
			urlPart = strings.TrimSpace(urlPart)
			urlPart = strings.Trim(urlPart, `"'`) // 去除引号

			if fileNamePart != "" && urlPart != "" {
				subUrlsMap[fileNamePart] = urlPart
			}
		} else {
			log.Printf("警告: SubUrls 配置项 '%s' 格式不匹配 [\"filename\"]=\"url\"", part)
		}
	}

	conf = ServerConfig{
		ListenPort:  section.Key("ListenPort").MustString(":8686"),
		ServerToken: section.Key("ServerToken").MustString("YourSecretToken"),
		DBPath:      section.Key("DBPath").MustString("traffic.db"),
		SMTPHost:    smtpSec.Key("SMTPHost").MustString("smtp.qq.com"),
		SMTPPort:    smtpSec.Key("SMTPPort").MustString("587"),
		EmailUser:   smtpSec.Key("EmailUser").MustString(""),
		EmailPass:   smtpSec.Key("EmailPass").MustString(""),
		EmailTo:     smtpSec.Key("EmailTo").MustString(""),
		SubUrls:     subUrlsMap,
	}

	log.Printf("[%s] 服务端配置已更新，加载了 %d 个订阅链接", time.Now().Format("15:04:05"), len(subUrlsMap))
	return nil
}

func watchConfig() {
	watcher, _ := fsnotify.NewWatcher()
	defer watcher.Close()
	_ = watcher.Add(iniPath)
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				loadConfig()
			}
		}
	}
}
