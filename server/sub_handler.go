// 动态订阅分发 Handler
// 读取同级目录下的 *.yaml 节点模板、*.csv 规则清单和 RuleSet/ 规则集，
// 动态拼接成完整的 Clash 配置文件，以 text/yaml 格式返回。

package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// parseYAMLSection 从 YAML 内容中提取指定顶层 key 下的完整 block（包括 key 行及其下属缩进行）。
// 返回 key:content 形式的字符串；若 key 不存在或 block 为空则返回 ""。
func parseYAMLSection(content, key string) string {
	lines := strings.Split(content, "\n")
	startLine := -1

	for i, line := range lines {
		if strings.HasPrefix(line, key+":") {
			startLine = i
			break
		}
	}

	if startLine == -1 {
		return ""
	}

	// 收集从 key 行开始、后续所有缩进行（或空行），直到遇到下一个同级 key 或文件结束
	var blockLines []string
	blockLines = append(blockLines, lines[startLine])

	for i := startLine + 1; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			blockLines = append(blockLines, line)
			continue
		}
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' && strings.Contains(line, ":") {
			break
		}
		blockLines = append(blockLines, line)
	}

	return strings.Join(blockLines, "\n")
}

// loadTemplateSections 读取目录下所有 *.yaml 节点模板文件，
// 提取每个文件的 proxies 和 proxy-groups section，合并后返回。
func loadTemplateSections(dir string) (proxies string, proxyGroups string) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		log.Printf("[Sub] 搜索模板文件失败: %v", err)
		return
	}

	var allProxies []string
	var allGroups []string

	for _, path := range matches {
		baseName := filepath.Base(path)
		// 跳过 RuleSet 目录内的文件和非节点模板
		if strings.HasPrefix(baseName, "86") {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("[Sub] 读取模板 %s 失败: %v", baseName, err)
			continue
		}

		content := string(data)
		if strings.TrimSpace(content) == "" {
			continue
		}

		sec := parseYAMLSection(content, "proxies")
		if sec != "" {
			// 去掉第一行（"proxies:"），只保留列表项部分
			lines := strings.SplitN(sec, "\n", 2)
			if len(lines) > 1 {
				trimmed := strings.TrimSpace(lines[1])
				if trimmed != "" {
					allProxies = append(allProxies, lines[1])
				}
			}
		}

		sec = parseYAMLSection(content, "proxy-groups")
		if sec != "" {
			lines := strings.SplitN(sec, "\n", 2)
			if len(lines) > 1 {
				trimmed := strings.TrimSpace(lines[1])
				if trimmed != "" {
					allGroups = append(allGroups, lines[1])
				}
			}
		}
	}

	proxies = strings.Join(allProxies, "\n")
	proxyGroups = strings.Join(allGroups, "\n")
	return
}

// CSVRule 表示 CSV 中的一条规则映射记录
type CSVRule struct {
	Name     string
	Target   string
	Behavior string
	URL      string
}

// readCSVRules 解析规则集清单 CSV，返回有序的规则列表和去重后的 target 顺序。
func readCSVRules(csvPath string) (rules []CSVRule, orderedTargets []string) {
	data, err := os.ReadFile(csvPath)
	if err != nil {
		log.Printf("[Sub] 读取 CSV 文件失败 %s: %v", csvPath, err)
		return
	}

	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		log.Printf("[Sub] 解析 CSV 失败: %v", err)
		return
	}

	seen := make(map[string]bool)
	for _, record := range records {
		if len(record) < 5 {
			continue
		}
		name := strings.TrimSpace(record[0])
		target := strings.TrimSpace(record[1])
		behavior := strings.TrimSpace(record[2])
		url := strings.TrimSpace(record[4])

		if name == "" || strings.HasPrefix(name, "#") {
			continue
		}

		rules = append(rules, CSVRule{
			Name:     name,
			Target:   target,
			Behavior: behavior,
			URL:      url,
		})

		if !seen[target] {
			seen[target] = true
			orderedTargets = append(orderedTargets, target)
		}
	}
	return
}

// loadRuleSets 读取 RuleSet/ 目录下所有 86*.yaml 规则集文件，
// 返回 target → 文件完整内容 的映射。
func loadRuleSets(ruleDir string) map[string]string {
	result := make(map[string]string)
	matches, err := filepath.Glob(filepath.Join(ruleDir, "86*.yaml"))
	if err != nil {
		log.Printf("[Sub] 搜索规则集失败: %v", err)
		return result
	}

	for _, path := range matches {
		baseName := filepath.Base(path)
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("[Sub] 读取规则集 %s 失败: %v", baseName, err)
			continue
		}
		result[baseName] = string(data)
	}
	return result
}

// targetToFile 将 CSV 中的 target 名映射到对应的规则集文件名
func targetToFile(target string) string {
	switch target {
	case "Japan":
		return "86JPRules.yaml"
	case "HK":
		return "86HKRules.yaml"
	case "US":
		return "86USRules.yaml"
	case "bemly":
		return "86BemlyRules.yaml"
	case "Switch":
		return "86SwitchRules.yaml"
	case "DIRECT":
		return "86DirectRules.yaml"
	case "REJECT":
		return "86RejectRules.yaml"
	default:
		return ""
	}
}

// handleSub 处理 GET /sub 请求，动态生成并返回 Clash 订阅配置。
// 需要通过 ?token= 查询参数进行鉴权，token 必须匹配 ServerSetting.ini 中的 ServerToken。
func handleSub(c *gin.Context) {
	token := c.Query("token")

	confLock.RLock()
	expectedToken := conf.ServerToken
	confLock.RUnlock()

	if token == "" || token != expectedToken {
		log.Printf("[Sub] 鉴权失败: %s (token=%s)", c.ClientIP(), token)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: invalid or missing token"})
		return
	}

	log.Printf("[Sub] 收到订阅请求: %s (device=%s)", c.ClientIP(), c.Query("device"))

	// 1. 合并所有节点模板的 proxies 和 proxy-groups
	proxiesSection, proxyGroupsSection := loadTemplateSections(TemplatesDir)

	// 2. 读取 CSV 规则清单
	_, orderedTargets := readCSVRules(CSVFile)

	// 3. 加载所有规则集文件
	ruleSets := loadRuleSets(RuleDir)
	if len(ruleSets) == 0 {
		log.Println("[Sub] 警告: 未找到任何规则集文件，订阅将仅包含节点配置")
	}

	// 4. 构建完整的 Clash YAML 配置
	var sb strings.Builder

	// ── 基础配置 ──
	sb.WriteString("# FlowCollect Dynamic Subscription\n")
	sb.WriteString("# Auto-generated by server. Do not edit manually.\n\n")
	sb.WriteString("port: 7890\n")
	sb.WriteString("socks-port: 7891\n")
	sb.WriteString("allow-lan: true\n")
	sb.WriteString("mode: rule\n")
	sb.WriteString("log-level: info\n")
	sb.WriteString("external-controller: 127.0.0.1:9090\n\n")

	// ── DNS 配置 ──
	sb.WriteString("dns:\n")
	sb.WriteString("  enable: true\n")
	sb.WriteString("  listen: 0.0.0.0:53\n")
	sb.WriteString("  enhanced-mode: fake-ip\n")
	sb.WriteString("  fake-ip-range: 198.18.0.1/16\n")
	sb.WriteString("  nameserver:\n")
	sb.WriteString("    - 223.5.5.5\n")
	sb.WriteString("    - 119.29.29.29\n")
	sb.WriteString("  fallback:\n")
	sb.WriteString("    - tls://8.8.8.8:853\n")
	sb.WriteString("    - tls://1.1.1.1:853\n\n")

	// ── Proxies ──
	sb.WriteString("proxies:\n")
	if strings.TrimSpace(proxiesSection) != "" {
		sb.WriteString(proxiesSection)
		if !strings.HasSuffix(proxiesSection, "\n") {
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("  []\n")
	}
	sb.WriteString("\n")

	// ── Proxy Groups ──
	sb.WriteString("proxy-groups:\n")
	if strings.TrimSpace(proxyGroupsSection) != "" {
		sb.WriteString(proxyGroupsSection)
		if !strings.HasSuffix(proxyGroupsSection, "\n") {
			sb.WriteString("\n")
		}
	}
	sb.WriteString("  - name: DIRECT\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - DIRECT\n\n")

	// ── Rules ──
	sb.WriteString("rules:\n")

	ruleCount := 0
	for _, target := range orderedTargets {
		fileName := targetToFile(target)
		if fileName == "" {
			log.Printf("[Sub] 未知 target: %s，跳过", target)
			continue
		}

		content, ok := ruleSets[fileName]
		if !ok {
			log.Printf("[Sub] 规则集文件 %s 未找到，跳过 target=%s", fileName, target)
			continue
		}

		sb.WriteString(fmt.Sprintf("  # === Rules for %s ===\n", target))
		sb.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("\n")

		ruleCount++
	}

	if ruleCount == 0 {
		sb.WriteString("  - MATCH,DIRECT\n")
	}

	// ── 附加配置 ──
	sb.WriteString("mixed-port: 7890\n")
	sb.WriteString("sniffing:\n")
	sb.WriteString("  enable: true\n")
	sb.WriteString("  sniff:\n")
	sb.WriteString("    - TLS\n")
	sb.WriteString("    - HTTP\n")

	// 5. 返回 YAML 响应
	c.Data(http.StatusOK, "text/yaml; charset=utf-8", []byte(sb.String()))
	log.Printf("[Sub] 订阅已下发: %d 个模板, %d 条规则链", proxiesSection != "", ruleCount)
}

// handleTemplateFile 处理 GET /templates/*filepath 请求，返回 templates 目录下的原始文件。
// 支持 token 鉴权，用于 proxy-providers / rule-providers 拉取节点模板和规则集。
// 示例: /templates/shanhuyun_node.yaml?token=xxx, /templates/RuleSet/86JPRules.yaml?token=xxx
func handleTemplateFile(c *gin.Context) {
	token := c.Query("token")

	confLock.RLock()
	expectedToken := conf.ServerToken
	confLock.RUnlock()

	if token == "" || token != expectedToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: invalid or missing token"})
		return
	}

	filePath := c.Param("filepath")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing filepath"})
		return
	}

	// 安全检查：防止路径遍历
	cleanPath := filepath.Clean(filePath)
	if strings.Contains(cleanPath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filepath"})
		return
	}

	fullPath := filepath.Join(TemplatesDir, cleanPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	c.Data(http.StatusOK, "text/yaml; charset=utf-8", data)
	log.Printf("[Template] 文件下发: %s -> %s", c.ClientIP(), cleanPath)
}
