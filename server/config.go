// 配置管理。负责加载/监听 ServerSetting.ini

package main

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
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

// parseINIValue 去除值两端的引号（单引号或双引号）
func parseINIValue(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			v = v[1 : len(v)-1]
		}
	}
	return v
}

// loadConfig 从 INI 文件加载配置
func loadConfig() error {
	f, err := os.Open(iniPath)
	if err != nil {
		return err
	}
	defer f.Close()

	subUrlsMap := make(map[string]string)

	// 默认值
	conf = ServerConfig{
		ListenPort:        ":8686",
		ServerToken:       "YourSecretToken",
		DBPath:            "./data/traffic.db",
		SMTPHost:          "smtp.qq.com",
		SMTPPort:          "587",
		SubUrls:           subUrlsMap,
		MainSubFile:       "main_sub.yaml",
		ReadMainSubConfig: false,
		CFTunnelContainer: "",
	}

	var currentSection string
	var lastKey string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// 跳过空行和注释
		if trimmed == "" || trimmed[0] == ';' || trimmed[0] == '#' {
			continue
		}

		// 检测 section
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			currentSection = strings.ToLower(trimmed[1 : len(trimmed)-1])
			continue
		}

		// 续行：以空格开头的行，追加到上一个 key 的值
		if line[0] == ' ' || line[0] == '\t' {
			if lastKey != "" {
				switch currentSection {
				case "server":
					switch lastKey {
					case "suburls":
						// SubUrls 续行：解析 ["name"]="url" 格式
						parseSubURL(trimmed, subUrlsMap)
					}
				}
			}
			continue
		}

		// 解析 key = value
		if idx := strings.Index(trimmed, "="); idx > 0 {
			key := strings.TrimSpace(trimmed[:idx])
			val := parseINIValue(trimmed[idx+1:])

			// 只有第一次遇到 SubUrls 时才解析首行
			if strings.EqualFold(key, "suburls") {
				lastKey = "suburls"
				parseSubURL(val, subUrlsMap)
				continue
			} else {
				lastKey = ""
			}

			// 根据当前 section 和 key 设置值
			lowerKey := strings.ToLower(key)
			switch currentSection {
			case "server":
				switch lowerKey {
				case "listenport":
					conf.ListenPort = val
				case "servertoken":
					conf.ServerToken = val
				case "dbpath":
					conf.DBPath = val
				case "mainsubfile":
					conf.MainSubFile = val
				case "readmainsubconfig":
					conf.ReadMainSubConfig = val == "true" || val == "1" || val == "yes"
				case "cftunnelcontainer":
					conf.CFTunnelContainer = val
				}
			case "smtp":
				switch lowerKey {
				case "smtphost":
					conf.SMTPHost = val
				case "smtpport":
					conf.SMTPPort = val
				case "emailuser":
					conf.EmailUser = val
				case "emailpass":
					conf.EmailPass = val
				case "emailto":
					conf.EmailTo = val
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	log.Printf("[%s] 服务端配置已更新，加载了 %d 个订阅链接", time.Now().Format("15:04:05"), len(subUrlsMap))
	return nil
}

// parseSubURL 解析单个 SubUrls 条目
// 输入格式: ["filename"]="url" 或 ["filename"]=url
func parseSubURL(val string, subUrlsMap map[string]string) {
	val = strings.TrimSpace(val)
	if val == "" {
		return
	}

	// 寻找 [" 和 "]="
	startIdx := strings.Index(val, "[\"")
	if startIdx == -1 {
		startIdx = strings.Index(val, "[")
		if startIdx == -1 {
			return
		}
	}

	equalIdx := strings.Index(val, "=")
	if equalIdx == -1 || equalIdx <= startIdx {
		return
	}

	// 提取文件名
	fileNamePart := val[startIdx+1 : equalIdx]
	fileNamePart = strings.TrimSpace(fileNamePart)
	fileNamePart = strings.TrimSuffix(fileNamePart, "]")
	fileNamePart = strings.Trim(fileNamePart, `"'`)

	// 提取 URL
	urlPart := val[equalIdx+1:]
	urlPart = strings.TrimSpace(urlPart)
	urlPart = strings.Trim(urlPart, `"'`)

	if fileNamePart != "" && urlPart != "" {
		subUrlsMap[fileNamePart] = urlPart
	}
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

	// 打印绝对路径
	if absPath, err := filepath.Abs(iniPath); err == nil {
		log.Printf("[Config] 绝对路径: %s", absPath)
	}

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
