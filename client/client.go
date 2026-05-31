//go:build client
// +build client

package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
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
	"gopkg.in/yaml.v3"
)

// ── YAML 结构体：映射 Clash config.yaml ──

// FlowCollectExtension 自定义顶层扩展字段
type FlowCollectExtension struct {
	RemoteServer string `yaml:"remote-server"`
	RemoteToken  string `yaml:"remote-token"`
	DeviceID     string `yaml:"device-id"`
}

// ClashConfig 仅解析 FlowCollect 需要的字段，其余忽略
type ClashConfig struct {
	ExternalController string               `yaml:"external-controller"`
	Secret             string               `yaml:"secret"`
	FlowCollect        FlowCollectExtension `yaml:"x-flow-collect"`
}

// ── 运行时配置（已转换） ──

type Config struct {
	MihomoAPIAddr string
	MihomoSecret  string
	RemoteServer  string
	RemoteToken   string
	DeviceID      string
	LocalLogFile  string
}

// ── 全局变量 ──

var (
	conf           Config
	confLock       sync.RWMutex
	lastStats      = make(map[string]Conn)
	configPath     string
	reportChan     = make(chan ReportData, 100)
	mihomoClient   *http.Client // cached HTTP client for Mihomo API
	mihomoAPIAddr  string       // resolved Mihomo API base URL
)

// resolveMihomoAPI 将 Clash 的 external-controller 转换为可用的 HTTP URL
// Clash 格式: "0.0.0.0:9090" 或 "127.0.0.1:9090"
// 转换后: "http://127.0.0.1:9090"
func resolveMihomoAPI(controller string) string {
	if controller == "" {
		return "http://127.0.0.1:9090"
	}
	// 如果已经包含协议前缀，直接返回
	if strings.HasPrefix(controller, "http://") || strings.HasPrefix(controller, "https://") {
		return controller
	}
	// 将 0.0.0.0 替换为 127.0.0.1（本地连接）
	addr := strings.Replace(controller, "0.0.0.0", "127.0.0.1", 1)
	return "http://" + addr
}

// resolveMihomoClient 创建一个支持 HTTP + IPC 的 Mihomo API 客户端。
// 优先尝试 HTTP 连接，失败后回退到 IPC（Windows 命名管道 / Unix Socket）。
// 成功后缓存结果；都失败时不缓存，下次调用会重试。
func resolveMihomoClient(httpURL string) (*http.Client, string) {
	if mihomoClient != nil {
		return mihomoClient, mihomoAPIAddr
	}

	// 1. 尝试 HTTP
	if httpURL != "" {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get(httpURL + "/version")
		if err == nil {
			resp.Body.Close()
			fmt.Printf("[API] Mihomo HTTP API 可用: %s\n", httpURL)
			mihomoClient = client
			mihomoAPIAddr = httpURL
			return client, httpURL
		}
		fmt.Printf("[API] HTTP API 不可用: %v，尝试 IPC...\n", err)
	}

	// 2. 回退到 IPC
	ipcPath := knownIPCPath()
	if ipcPath != "" {
		transport := newIPCTransport(ipcPath)
		client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
		resp, err := client.Get("http://localhost/version")
		if err == nil {
			resp.Body.Close()
			fmt.Printf("[API] Mihomo IPC 可用: %s\n", ipcPath)
			mihomoClient = client
			mihomoAPIAddr = "http://localhost"
			return client, "http://localhost"
		}
		fmt.Printf("[API] IPC 不可用: %v\n", err)
	}

	// 3. 都可用，不缓存，下次调用重试
	fmt.Println("[API] 警告: Mihomo API 不可达，等待重试...")
	client := &http.Client{Timeout: 5 * time.Second}
	return client, httpURL
}

// loadConfig 从 Clash config.yaml 加载配置
func loadConfig() error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置失败: %w", err)
	}

	var cc ClashConfig
	if err := yaml.Unmarshal(data, &cc); err != nil {
		return fmt.Errorf("解析 YAML 失败: %w", err)
	}

	confLock.Lock()
	defer confLock.Unlock()

	// 配置变更时重置缓存的 Mihomo 客户端，下次请求时重新探测
	mihomoClient = nil
	mihomoAPIAddr = ""

	conf = Config{
		MihomoAPIAddr: resolveMihomoAPI(cc.ExternalController),
		MihomoSecret:  cc.Secret,
		RemoteServer:  cc.FlowCollect.RemoteServer,
		RemoteToken:   cc.FlowCollect.RemoteToken,
		DeviceID:      cc.FlowCollect.DeviceID,
		LocalLogFile:  "node_traffic_stats.json",
	}

	// 兜底：如果 x-flow-collect 未配置，使用默认值
	if conf.DeviceID == "" {
		hostname, _ := os.Hostname()
		conf.DeviceID = hostname
		if conf.DeviceID == "" {
			conf.DeviceID = "android-device"
		}
		// 自我修正：将检测到的主机名写回配置文件，避免下次仍为空
		go writeBackDeviceID(configPath, conf.DeviceID)
	}
	if conf.RemoteToken == "" {
		conf.RemoteToken = "YourSecretToken"
	}

	fmt.Printf("[%s] 配置加载成功 | MihomoAPI: %s | DeviceID: %s | Server: %s\n",
		time.Now().Format("15:04:05"), conf.MihomoAPIAddr, conf.DeviceID, conf.RemoteServer)
	return nil
}

// writeBackDeviceID 将检测到的设备名写回配置文件的 x-flow-collect.device-id 字段
func writeBackDeviceID(path, deviceID string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	content := string(data)
	// 查找 "device-id:" 行并替换值
	lines := strings.Split(content, "\n")
	modified := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "device-id:") {
			// 提取缩进
			indent := line[:len(line)-len(trimmed)]
			lines[i] = indent + "device-id: \"" + deviceID + "\""
			modified = true
			break
		}
	}

	if !modified {
		return
	}

	output := strings.Join(lines, "\n")
	if err := os.WriteFile(path, []byte(output), 0644); err != nil {
		fmt.Printf("[Config] 写回 device-id 失败: %v\n", err)
		return
	}
	fmt.Printf("[Config] 已将 device-id 写回配置文件: %s\n", deviceID)
}

// watchConfig 监控配置文件变化并热重载
func watchConfig() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("创建文件监听器失败:", err)
		return
	}
	defer watcher.Close()

	if err := watcher.Add(configPath); err != nil {
		// 文件不存在时持续等待
		go func() {
			for {
				time.Sleep(5 * time.Second)
				if err := watcher.Add(configPath); err == nil {
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

// ── 业务模型 ──

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
	ActiveConns int    `json:"active_connections"`
}

type NodeStats struct {
	Up   int64
	Down int64
}

func main() {
	// ── 命令行参数 ──
	configFile := flag.String("c", "", "Clash config.yaml 路径 (必填或通过环境变量 FLOW_COLLECT_CONFIG)")
	flag.Parse()

	// 确定配置文件路径
	configPath = *configFile
	if configPath == "" {
		configPath = os.Getenv("FLOW_COLLECT_CONFIG")
	}
	if configPath == "" {
		// 尝试智能定位：当前目录 → 可执行文件目录 → Android 默认路径
		candidates := []string{
			"config.yaml",
			"../clash/config.yaml",
		}
		if exe, err := os.Executable(); err == nil {
			candidates = append(candidates, filepath.Join(filepath.Dir(exe), "config.yaml"))
		}
		// Android 默认路径
		candidates = append(candidates, "/data/adb/box/clash/config.yaml")

		for _, p := range candidates {
			if _, err := os.Stat(p); err == nil {
				configPath = p
				break
			}
		}
	}
	if configPath == "" {
		fmt.Fprintln(os.Stderr, "错误: 未找到配置文件，请使用 -c 参数指定 config.yaml 路径")
		fmt.Fprintln(os.Stderr, "用法: flow_collect_client -c /path/to/config.yaml")
		os.Exit(1)
	}

	// 验证文件存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "错误: 配置文件不存在: %s\n", configPath)
		os.Exit(1)
	}

	absPath, _ := filepath.Abs(configPath)
	fmt.Printf("配置文件: %s\n", absPath)

	// 加载配置
	if err := loadConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "初始加载配置失败: %v\n", err)
		os.Exit(1)
	}
	go watchConfig()

	// ── 启动监控 ──
	confLock.RLock()
	fmt.Printf("FlowCollect 审计客户端启动 [%s]...\n", conf.DeviceID)
	confLock.RUnlock()

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
	currConf := conf
	confLock.RUnlock()

	// 获取支持 HTTP + IPC 的客户端
	httpURL := resolveMihomoAPI(currConf.MihomoAPIAddr)
	client, apiAddr := resolveMihomoClient(httpURL)

	req, _ := http.NewRequest("GET", apiAddr+"/connections", nil)
	req.Header.Set("Authorization", "Bearer "+currConf.MihomoSecret)

	resp, err := client.Do(req)
	if err != nil {
		if !silent {
			fmt.Println("API 访问失败:", err)
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if !silent {
			fmt.Printf("API 鉴权失败! 状态码: %d\n", resp.StatusCode)
		}
		return
	}

	var data struct {
		Connections []Conn `json:"connections"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Println("解析 JSON 失败:", err)
		return
	}

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
	default:
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
	if strings.HasSuffix(u.Path, "/report") {
		u.Path = strings.TrimSuffix(u.Path, "/report") + "/ws"
	} else if u.Path == "" || u.Path == "/" {
		u.Path = "/ws"
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

		for data := range reportChan {
			err = wsConn.WriteJSON(data)
			if err != nil {
				fmt.Printf("[WebSocket] ❌ 发送错误: %v。断开并重新连接...\n", err)
				wsConn.Close()
				break
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
