//go:build client
// +build client

package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"gopkg.in/ini.v1"
)

// 定义配置结构体
type Config struct {
	MihomoAPIAddr string
	MihomoSecret  string
	RemoteServer  string
	RemoteToken   string
	DeviceID      string
	LocalLogFile  string
}

var (
	conf       Config
	confLock   sync.RWMutex
	lastStats  = make(map[string]Conn)
	iniPath    = "ClientSetting.ini"
	reportChan = make(chan ReportData, 100)
)

func init() {
	// 智能定位配置文件：如果当前目录找不到，尝试在可执行文件所在目录查找
	if _, err := os.Stat(iniPath); os.IsNotExist(err) {
		if exe, err := os.Executable(); err == nil {
			exeDir := filepath.Dir(exe)
			altPath := filepath.Join(exeDir, "ClientSetting.ini")
			if _, err := os.Stat(altPath); err == nil {
				iniPath = altPath
				fmt.Printf("当前目录未找到配置，已切换至可执行文件目录: %s\n", iniPath)
			}
		}
	}

	// 程序启动时首次加载
	if err := loadConfig(); err != nil {
		fmt.Printf("初始加载配置文件失败: %v，将使用代码内置默认值\n", err)
		applyDefaults()
	}
	// 启动后台监控协程
	go watchConfig()
}

// 加载配置的函数
func loadConfig() error {
	cfg, err := ini.Load(iniPath)
	if err != nil {
		return err
	}

	confLock.Lock()
	defer confLock.Unlock()

	section := cfg.Section("")
	conf = Config{
		MihomoAPIAddr: section.Key("MihomoAPIAddr").MustString("http://127.0.0.1:9090"),
		MihomoSecret:  section.Key("MihomoSecret").MustString(""),
		RemoteServer:  section.Key("RemoteServer").MustString(""),
		RemoteToken:   section.Key("RemoteToken").MustString("YourSecretToken"),
		DeviceID:      section.Key("DeviceID").MustString("PC-Windows"),
		LocalLogFile:  section.Key("LocalLogFile").MustString("node_traffic_stats.json"),
	}

	fmt.Printf("[%s] 配置文件加载/更新成功\n", time.Now().Format("15:04:05"))
	return nil
}

func applyDefaults() {
	confLock.Lock()
	defer confLock.Unlock()

	conf = Config{
		MihomoAPIAddr: "http://127.0.0.1:9090",
		MihomoSecret:  "abcd",
		RemoteServer:  "",
		RemoteToken:   "YourSecretToken",
		DeviceID:      "Device-Default",
		LocalLogFile:  "node_traffic_stats.json",
	}
}

// 监控文件变化的协程
func watchConfig() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("创建监听器失败:", err)
		return
	}
	defer watcher.Close()

	err = watcher.Add(iniPath)
	if err != nil {
		// 如果文件不存在，每隔5秒尝试重新添加，直到文件被创建
		go func() {
			for {
				time.Sleep(5 * time.Second)
				if err := watcher.Add(iniPath); err == nil {
					loadConfig()
					break
				}
			}
		}()
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// 当文件被修改时重新加载
			if event.Op&fsnotify.Write == fsnotify.Write {
				loadConfig()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Println("监听错误:", err)
		}
	}
}

// --- 业务模型保持不变 ---

type Conn struct {
	ID       string   `json:"id"`
	Upload   int64    `json:"upload"`
	Download int64    `json:"download"`
	Chains   []string `json:"chains"`
}

type ReportData struct {
	Timestamp   int64  `json:"timestamp"`
	DeviceID    string `json:"device_id"`
	NodeName    string `json:"node_name"`
	UpDelta     int64  `json:"up_delta"`
	DownDelta   int64  `json:"down_delta"`
	IsProxy     bool   `json:"is_proxy"`
	ActiveConns int    `json:"active_connections"` // 新增：活跃连接数
}

type NodeStats struct {
	Up   int64
	Down int64
}

func main() {
	confLock.RLock()
	fmt.Printf("精细化监控启动 [%s]...\n", conf.DeviceID)
	confLock.RUnlock()

	// 启动 WebSocket 管理器负责异步上报
	go websocketManager()

	fmt.Println("正在初始化连接快照 (静默模式)...")
	fetchAndProcess(true)

	ticker := time.NewTicker(10 * time.Second)
	fmt.Println("初始化完成，开始正式监控。")
	for range ticker.C {
		fetchAndProcess(false)
	}
}

func fetchAndProcess(silent bool) {
	confLock.RLock()
	currConf := conf // 获取当前配置快照
	confLock.RUnlock()

	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", currConf.MihomoAPIAddr+"/connections", nil)
	req.Header.Set("Authorization", "Bearer "+currConf.MihomoSecret)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("API 访问失败 (检查 Mihomo 配置):", err)
		return
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("API 鉴权失败! 状态码: %d，请检查 ClientSetting.ini 中的 MihomoSecret\n", resp.StatusCode)
		return
	}

	var data struct {
		Connections []Conn `json:"connections"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Println("解析 JSON 失败:", err)
		return
	}

	// 打印活跃连接数
	activeConns := len(data.Connections)
	if !silent {
		fmt.Printf("[%s] 活跃连接数: %d\n", time.Now().Format("15:04:05"), activeConns)
	}

	nodeStatsMap := make(map[string]*NodeStats)
	currentIDs := make(map[string]bool)

	for _, c := range data.Connections {
		currentIDs[c.ID] = true
		nodeName := "DIRECT"
		if len(c.Chains) > 0 {
			nodeName = c.Chains[len(c.Chains)-1]
		}

		last, exists := lastStats[c.ID]
		upDelta, downDelta := c.Upload, c.Download
		if exists {
			upDelta = c.Upload - last.Upload
			downDelta = c.Download - last.Download
		}

		if _, ok := nodeStatsMap[nodeName]; !ok {
			nodeStatsMap[nodeName] = &NodeStats{}
		}
		nodeStatsMap[nodeName].Up += upDelta
		nodeStatsMap[nodeName].Down += downDelta
		lastStats[c.ID] = c
	}

	// 清理已断开的连接
	for id := range lastStats {
		if !currentIDs[id] {
			delete(lastStats, id)
		}
	}

	if !silent {
		for name, stats := range nodeStatsMap {
			if stats.Up > 0 || stats.Down > 0 {
				dispatch(name, stats.Up, stats.Down, activeConns, currConf)
			}
		}
	}
}

func dispatch(nodeName string, up, down int64, activeConns int, currConf Config) {
	lowerName := strings.ToLower(nodeName)
	isProxy := (lowerName != "direct" && lowerName != "ua3f")

	payload := ReportData{
		Timestamp:   time.Now().Unix(),
		DeviceID:    currConf.DeviceID,
		NodeName:    nodeName,
		UpDelta:     up,
		DownDelta:   down,
		IsProxy:     isProxy,
		ActiveConns: activeConns,
	}

	saveLocal(payload, currConf.LocalLogFile)

	select {
	case reportChan <- payload:
		// 成功放入 channel
	default:
		// channel 满了，丢弃新数据，避免阻塞 fetchAndProcess
		fmt.Printf("[%s] ⚠️ 发送缓冲已满，丢弃数据 (Node: %s)\n", time.Now().Format("15:04:05"), nodeName)
	}
}

func saveLocal(data ReportData, filename string) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	b, _ := json.Marshal(data)
	f.WriteString(string(b) + "\n")
}

func getWSURL(serverURL string) string {
	u, err := url.Parse(serverURL)
	if err != nil {
		return strings.Replace(serverURL, "http", "ws", 1)
	}
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else if u.Scheme == "http" {
		u.Scheme = "ws"
	}
	// 将旧的 http://ip:port/report 转换为 ws://ip:port/ws
	if strings.HasSuffix(u.Path, "/report") {
		u.Path = strings.TrimSuffix(u.Path, "/report") + "/ws"
	}
	return u.String()
}

func websocketManager() {
	var wsConn *websocket.Conn
	var err error

	for {
		confLock.RLock()
		currConf := conf
		confLock.RUnlock()

		if currConf.RemoteServer == "" {
			time.Sleep(5 * time.Second)
			continue
		}

		wsURL := getWSURL(currConf.RemoteServer)

		headers := http.Header{}
		headers.Set("Authorization", "Bearer "+currConf.RemoteToken)
		dialer := websocket.DefaultDialer
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

		fmt.Printf("[WebSocket] 正在连接到 %s...\n", wsURL)
		wsConn, _, err = dialer.Dial(wsURL, headers)
		if err != nil {
			fmt.Printf("[WebSocket] 连接失败: %v。5秒后重试...\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		fmt.Println("[WebSocket] ✅ 连接成功，准备发送数据。")

		// 循环从 channel 读取并发送数据
		for data := range reportChan {
			err = wsConn.WriteJSON(data)
			if err != nil {
				fmt.Printf("[WebSocket] ❌ 发送错误: %v。断开并重新连接...\n", err)
				wsConn.Close()
				break // 跳出内层循环，重新连接
			} else {
				fmt.Printf("[已上报 WS] %s | 节点: %-15s ↑%-10s ↓%-10s\n",
					time.Now().Format("15:04:05"), data.NodeName, formatBytes(data.UpDelta), formatBytes(data.DownDelta))
			}
		}
	}
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
