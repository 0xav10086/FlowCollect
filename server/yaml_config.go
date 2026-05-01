// 配置文件和规则集的更新与获取
package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 配置文件常量与 Nginx 映射的静态目录一致
const (
	WebRootDir = "/var/www/flow_collect"
	RuleDir    = WebRootDir + "/RuleSet"
	CSVFile    = WebRootDir + "/86_rule_set_collect.csv"
)

// HandleTriggerUpdate 触发 Go 版本的节点和规则更新，并发送邮件通知 (HTTP Handler)
func HandleTriggerUpdate(c *gin.Context) {
	go triggerUpdateTask()

	c.JSON(http.StatusOK, gin.H{
		"message": "更新任务已在后台触发，执行结果将通过邮件通知。",
	})
}

// triggerUpdateTask 是真正的执行函数，供 HTTP handler 和 cron 定时任务复用
func triggerUpdateTask() {
	var outputBuilder strings.Builder
	logWriter := io.MultiWriter(os.Stdout, &outputBuilder)
	logger := log.New(logWriter, "", log.LstdFlags)

	logger.Println("===================================================")
	logger.Printf("🚀 [%s] Auto-Update Task Started (Go Native)\n", time.Now().Format("2006-01-02 15:04:05"))
	logger.Println("===================================================")

	hasError := false

	// --- Phase 1: Update Subscriptions ---
	logger.Println("--> Phase 1: Updating subscriptions...")

	// 确保目录存在
	if err := os.MkdirAll(RuleDir, 0755); err != nil {
		logger.Printf("❌ 无法创建目录 %s: %v\n", RuleDir, err)
		hasError = true
	}

	confLock.RLock()
	subUrls := conf.SubUrls // 这是一个 map[string]string，键为文件名，值为 url
	confLock.RUnlock()

	for fileName, url := range subUrls {
		logger.Printf("⏳ Fetching nodes for [%s]...", fileName)
		targetFile := filepath.Join(WebRootDir, fileName)
		tempFile := targetFile + ".tmp"

		err := downloadFile(url, tempFile, logger)
		if err != nil {
			logger.Printf("❌ Failed to update [%s]: %v. Retaining old file.", fileName, err)
			hasError = true
			os.Remove(tempFile) // 清理临时文件
		} else {
			// 下载成功，覆盖原文件
			if err := os.Rename(tempFile, targetFile); err != nil {
				logger.Printf("❌ 覆盖文件失败 [%s]: %v", fileName, err)
				hasError = true
			} else {
				logger.Printf("✅ Successfully updated [%s].", fileName)
			}
		}
	}

	// --- Phase 2: Compile and Update Rule Sets ---
	logger.Println("\n--> Phase 2: Compiling cloud rule sets...")
	err := processRules(logger)
	if err != nil {
		logger.Printf("❌ Phase 2 Failed: %v\n", err)
		hasError = true
	} else {
		logger.Println("✅ [Success] All rule sets successfully compiled!")
	}

	logger.Println("===================================================")
	logger.Printf("🎉 [%s] Auto-Update Task Finished\n", time.Now().Format("2006-01-02 15:04:05"))
	logger.Println("===================================================")

	// 发送邮件通知
	subject := fmt.Sprintf("【FlowCollect】节点与规则更新通知 - %s", time.Now().Format("2006-01-02 15:04:05"))
	body := outputBuilder.String()
	if hasError {
		subject = "【⚠️警告】" + subject + " (包含错误)"
	}

	sendEmail(subject, body)
}

// downloadFile 辅助函数：使用原生 http.Client 下载文件并写入临时文件
func downloadFile(url, tempFilePath string, logger *log.Logger) error {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("User-Agent", "clash.meta")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP 状态码非200: %d", resp.StatusCode)
	}

	out, err := os.Create(tempFilePath)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	return nil
}

// getTargetFile 根据 target 映射对应的文件名
func getTargetFile(target string) string {
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

// processRules 处理规则生成逻辑
func processRules(logger *log.Logger) error {
	if _, err := os.Stat(CSVFile); os.IsNotExist(err) {
		return fmt.Errorf("Rule collection CSV not found: %s", CSVFile)
	}

	// 1. Initialize: Clear content after [MANUAL_END] Private
	files, err := filepath.Glob(filepath.Join(RuleDir, "86*.yaml"))
	if err == nil {
		for _, file := range files {
			truncateFileAtMarker(file, "[MANUAL_END] Private")
		}
	}

	// 2. Parse CSV and process rules
	csvData, err := os.ReadFile(CSVFile)
	if err != nil {
		return fmt.Errorf("读取 CSV 失败: %v", err)
	}

	reader := csv.NewReader(strings.NewReader(string(csvData)))
	reader.FieldsPerRecord = -1 // 允许不一致的列数
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("解析 CSV 失败: %v", err)
	}

	tempRawFile := filepath.Join(WebRootDir, "raw.tmp")
	defer os.Remove(tempRawFile) // 确保退出时删除临时文件

	for _, record := range records {
		if len(record) < 5 {
			continue
		}
		name := strings.TrimSpace(record[0])
		target := strings.TrimSpace(record[1])
		behavior := strings.TrimSpace(record[2])
		// localPath := strings.TrimSpace(record[3]) // Not used in bash
		url := strings.TrimSpace(record[4])

		if name == "" || strings.HasPrefix(name, "#") {
			continue
		}

		targetFile := filepath.Join(RuleDir, getTargetFile(target))
		if getTargetFile(target) == "" {
			logger.Printf("⚠️  [Warn] Invalid target '%s' for %s", target, name)
			continue
		}
		if _, err := os.Stat(targetFile); os.IsNotExist(err) {
			logger.Printf("⚠️  [Warn] Target file for policy group %s not found (%s), skipping %s", target, targetFile, name)
			continue
		}

		logger.Printf("  -> Fetching and compiling: %s (-> %s)", name, target)

		// Append header
		appendToFile(targetFile, fmt.Sprintf("  # === [AUTO: %s] ===\n", name))

		// Download to temp file
		err := downloadFile(url, tempRawFile, logger)
		if err != nil {
			logger.Printf("    ❌ 下载规则失败: %v", err)
			continue
		}

		// Read and process downloaded rules
		rawContent, err := os.ReadFile(tempRawFile)
		if err != nil {
			logger.Printf("    ❌ 读取规则失败: %v", err)
			continue
		}

		// Process lines based on behavior
		processedLines := processAwkLogic(string(rawContent), behavior)
		for _, line := range processedLines {
			appendToFile(targetFile, line+"\n")
		}

		// Append footer
		appendToFile(targetFile, "  # =====================\n\n")
	}

	return nil
}

// truncateFileAtMarker 模拟 bash 中的 sed -i '/\[MANUAL_END\] Private/q'
func truncateFileAtMarker(filePath, marker string) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	var newContent strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		newContent.WriteString(line + "\n")
		if strings.Contains(line, marker) {
			break // 到达标记，停止读取后续内容
		}
	}
	newContent.WriteString("\n") // 补充一个空行

	// 重新写入
	os.WriteFile(filePath, []byte(newContent.String()), 0644)
}

// appendToFile 追加内容到文件
func appendToFile(filePath, content string) {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(content)
}

// processAwkLogic 模拟 bash 脚本中复杂的 awk 处理逻辑
func processAwkLogic(content, behavior string) []string {
	var result []string
	scanner := bufio.NewScanner(strings.NewReader(content))

	// 正则表达式预编译
	reEmpty := regexp.MustCompile(`^\s*$`)
	reComment := regexp.MustCompile(`^#`)
	rePayload := regexp.MustCompile(`^payload:`)
	reDashStart := regexp.MustCompile(`^\s*-\s*`)
	reQuotes := regexp.MustCompile(`['"]`)

	inPayload := false

	for scanner.Scan() {
		line := scanner.Text()

		if behavior == "domain" {
			// /^#/ || /^payload:/ || /^[[:space:]]*$/ {next}
			if reComment.MatchString(line) || rePayload.MatchString(line) || reEmpty.MatchString(line) {
				continue
			}

			// sub(/^[[:space:]]*-[[:space:]]*/, "");
			processed := reDashStart.ReplaceAllString(line, "")
			// gsub(/[\047\042]/, "");
			processed = reQuotes.ReplaceAllString(processed, "")

			if strings.HasPrefix(processed, "+.") {
				result = append(result, "  - DOMAIN-SUFFIX,"+processed[2:])
			} else if strings.HasPrefix(processed, "+") {
				result = append(result, "  - DOMAIN-SUFFIX,"+processed[1:])
			} else if strings.HasPrefix(processed, "full:") {
				result = append(result, "  - DOMAIN,"+processed[5:])
			} else if strings.HasPrefix(processed, "domain:") {
				result = append(result, "  - DOMAIN-SUFFIX,"+processed[7:])
			} else {
				result = append(result, "  - DOMAIN-SUFFIX,"+processed)
			}

		} else {
			// BEGIN { in_payload=0 }
			// /^payload:/ { in_payload=1; next }
			if rePayload.MatchString(line) {
				inPayload = true
				continue
			}
			// in_payload && /^[[:space:]]*-[[:space:]]*/ { print $0 }
			if inPayload && reDashStart.MatchString(line) {
				result = append(result, line)
			}
		}
	}
	return result
}
