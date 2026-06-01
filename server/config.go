// 配置管理。负责加载/监听 ServerSetting.ini

package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/ini.v1"
)

// 配置结构体
type ServerConfig struct {
	ListenPort        string
	ServerToken       string
	DBPath            string
	SMTPHost          string
	SMTPPort          string
	EmailUser         string
	EmailPass         string
	EmailTo           string
	SubUrls           map[string]string // 变更为 map，键为 filename，值为 url
	MainSubFile       string            // 主订阅文件路径（相对于 templates/ 目录）
	ReadMainSubConfig bool              // 是否从主订阅文件读取端口和 Token 配置
	CFTunnelContainer string            // CF Tunnel 容器名（用于健康监控，留空则禁用）
}

var (
	conf     ServerConfig
	confLock sync.RWMutex
	iniPath  = "./configs/ServerSetting.ini"
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

	// ── 诊断：INI 文件解析详情 ──
	if absPath, err := filepath.Abs(iniPath); err == nil {
		log.Printf("[Config] 绝对路径: %s", absPath)
	}
	if raw, err := os.ReadFile(iniPath); err == nil {
		log.Printf("[Config] 文件原始内容:\n%s", string(raw))
	}
	serverKeys := section.KeyStrings()
	log.Printf("[Config] [server] section 包含 %d 个 key: %v", len(serverKeys), serverKeys)
	tunnelKey := section.Key("CFTunnelContainer")
	log.Printf("[Config] CFTunnelContainer 原始值: %q", tunnelKey.String())
	log.Printf("[Config] CFTunnelContainer MustString: %q", tunnelKey.MustString(""))
	log.Printf("[Config] CFTunnelContainer 是否存在: %v", tunnelKey.Value() != "")

	conf = ServerConfig{
		ListenPort:        section.Key("ListenPort").MustString(":8686"),
		ServerToken:       section.Key("ServerToken").MustString("YourSecretToken"),
		DBPath:            section.Key("DBPath").MustString("./data/traffic.db"),
		SMTPHost:          smtpSec.Key("SMTPHost").MustString("smtp.qq.com"),
		SMTPPort:          smtpSec.Key("SMTPPort").MustString("587"),
		EmailUser:         smtpSec.Key("EmailUser").MustString(""),
		EmailPass:         smtpSec.Key("EmailPass").MustString(""),
		EmailTo:           smtpSec.Key("EmailTo").MustString(""),
		SubUrls:           subUrlsMap,
		MainSubFile:       section.Key("MainSubFile").MustString("main_sub.yaml"),
		ReadMainSubConfig: section.Key("ReadMainSubConfig").MustBool(false),
		CFTunnelContainer: section.Key("CFTunnelContainer").MustString(""),
	}

	log.Printf("[%s] 服务端配置已更新，加载了 %d 个订阅链接", time.Now().Format("15:04:05"), len(subUrlsMap))
	return nil
}

func watchConfig() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("[Config] 创建文件监听器失败: %v", err)
		return
	}
	defer watcher.Close()

	if err := watcher.Add(iniPath); err != nil {
		log.Printf("[Config] 监听文件失败 %s: %v", iniPath, err)
		return
	}
	log.Printf("[Config] 正在监听 INI 文件: %s", iniPath)

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Printf("[Config] 检测到 INI 文件变化: %s", event.Name)
				loadConfig()
			}
		}
	}
}
