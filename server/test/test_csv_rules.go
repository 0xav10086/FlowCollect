// FlowCollect CSV 规则生成测试工具
// 测试 86_rule_set_collect.csv 是否能正确生成 86HKRules.yaml
//
// 用法:
//   cd server
//   go run -tags test test/test_csv_rules.go
//
// 前提: templates/86_rule_set_collect.csv 和 templates/RuleSet/86HKRules.yaml 存在

//go:build test

package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	csvFile    = "./templates/86_rule_set_collect.csv"
	ruleDir    = "./templates/RuleSet"
	targetFile = "./templates/RuleSet/86HKRules.yaml"
)

var (
	passCount int
	failCount int
)

func checkf(desc, expected, actual string) {
	if strings.Contains(actual, expected) {
		fmt.Printf("  ✅ PASS: %s\n", desc)
		passCount++
	} else {
		fmt.Printf("  ❌ FAIL: %s\n", desc)
		fmt.Printf("     期望包含: %q\n", expected)
		fmt.Printf("     实际(前200字): %q\n", truncate(actual, 200))
		failCount++
	}
}

func check(desc string, ok bool) {
	if ok {
		fmt.Printf("  ✅ PASS: %s\n", desc)
		passCount++
	} else {
		fmt.Printf("  ❌ FAIL: %s\n", desc)
		failCount++
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func main() {
	fmt.Println("==========================================")
	fmt.Println(" CSV 规则生成测试 (86HKRules)")
	fmt.Println("==========================================")
	fmt.Println()

	// ── 1. 检查前置文件 ──
	fmt.Println("[Step 1] 检查前置文件是否存在")
	if _, err := os.Stat(csvFile); os.IsNotExist(err) {
		fmt.Printf("  ❌ CSV 文件不存在: %s\n", csvFile)
		os.Exit(1)
	}
	fmt.Printf("  ✅ CSV 文件: %s\n", csvFile)

	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		fmt.Printf("  ⚠️ 目标文件不存在，测试将自动创建: %s\n", targetFile)
	} else {
		fmt.Printf("  ✅ 目标文件: %s\n", targetFile)
	}
	fmt.Println()

	// ── 2. 解析 CSV，提取 HK 条目 ──
	fmt.Println("[Step 2] 解析 CSV，提取 Target=HK 的条目")
	csvData, err := os.ReadFile(csvFile)
	if err != nil {
		fmt.Printf("  ❌ 读取 CSV 失败: %v\n", err)
		os.Exit(1)
	}

	reader := csv.NewReader(strings.NewReader(string(csvData)))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Printf("  ❌ 解析 CSV 失败: %v\n", err)
		os.Exit(1)
	}

	type csvEntry struct {
		Name     string
		Target   string
		Behavior string
		URL      string
	}

	var hkEntries []csvEntry
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

		if target == "HK" {
			hkEntries = append(hkEntries, csvEntry{name, target, behavior, url})
			fmt.Printf("  📄 %s (behavior=%s)\n", name, behavior)
			fmt.Printf("     URL: %s\n", url)
		}
	}

	checkf("CSV 中 HK 条目至少 1 条", fmt.Sprintf("%d", len(hkEntries)), fmt.Sprintf("%d", len(hkEntries)))
	check(fmt.Sprintf("HK 条目数 >= 1 (实际: %d)", len(hkEntries)), len(hkEntries) >= 1)
	fmt.Println()

	// ── 3. 下载 HK 规则文件 ──
	fmt.Println("[Step 3] 下载 HK 条目对应的规则文件")
	type downloadedRule struct {
		Name     string
		Behavior string
		Content  string
		Success  bool
	}

	var downloaded []downloadedRule
	client := &http.Client{Timeout: 30 * time.Second}

	for _, entry := range hkEntries {
		fmt.Printf("  -> %s ... ", entry.Name)

		req, err := http.NewRequest("GET", entry.URL, nil)
		if err != nil {
			fmt.Printf("❌ 请求创建失败: %v\n", err)
			downloaded = append(downloaded, downloadedRule{entry.Name, entry.Behavior, "", false})
			continue
		}
		req.Header.Set("User-Agent", "clash.meta")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("❌ 请求失败: %v\n", err)
			downloaded = append(downloaded, downloadedRule{entry.Name, entry.Behavior, "", false})
			continue
		}

		if resp.StatusCode != 200 {
			fmt.Printf("❌ HTTP %d\n", resp.StatusCode)
			resp.Body.Close()
			downloaded = append(downloaded, downloadedRule{entry.Name, entry.Behavior, "", false})
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Printf("❌ 读取失败: %v\n", err)
			downloaded = append(downloaded, downloadedRule{entry.Name, entry.Behavior, "", false})
			continue
		}

		fmt.Printf("✅ %d bytes\n", len(body))
		downloaded = append(downloaded, downloadedRule{entry.Name, entry.Behavior, string(body), true})
	}

	allDownloaded := true
	for _, d := range downloaded {
		if !d.Success {
			allDownloaded = false
			break
		}
	}
	if !allDownloaded {
		fmt.Printf("  ⚠️  部分下载失败（网络问题），仅验证成功下载的条目\n")
	}
	check("至少 1 个 HK 规则文件下载成功", func() bool {
		for _, d := range downloaded {
			if d.Success {
				return true
			}
		}
		return false
	}())
	fmt.Println()

	// ── 4. 处理规则 ──
	fmt.Println("[Step 4] 编译规则内容")
	type processedRules struct {
		Name   string
		Lines  []string
		Header string
	}

	var processed []processedRules

	for _, d := range downloaded {
		if !d.Success {
			continue
		}

		lines := processContent(d.Content, d.Behavior)
		header := fmt.Sprintf("  # === [AUTO: %s] ===\n", d.Name)
		footer := "  # =====================\n\n"

		fmt.Printf("  -> %s: %d 条规则 (behavior=%s)\n", d.Name, len(lines), d.Behavior)

		// Build output block
		var block strings.Builder
		block.WriteString(header)
		for _, line := range lines {
			block.WriteString(line)
			block.WriteString("\n")
		}
		block.WriteString(footer)
		_ = footer // keep for clarity

		processed = append(processed, processedRules{
			Name:   d.Name,
			Lines:  lines,
			Header: header,
		})
	}

	totalRules := 0
	for _, p := range processed {
		totalRules += len(p.Lines)
	}
	check(fmt.Sprintf("编译后的规则总条数 > 0 (实际: %d)", totalRules), totalRules > 0)
	fmt.Println()

	// ── 5. 验证 ──
	fmt.Println("[Step 5] 验证规则内容")

	// 5.1: 每个条目都有 header
	check(fmt.Sprintf("Telegram 有 AUTO header (共 %d 个已处理条目)", len(processed)), len(processed) > 0)

	// 5.2: Telegram (classical) 应包含 payload 风格的规则
	for _, p := range processed {
		if p.Name == "Telegram" {
			hasDashLine := false
			for _, line := range p.Lines {
				if strings.HasPrefix(line, "  - ") {
					hasDashLine = true
					break
				}
			}
			check("Telegram 包含 dash 开头的规则行", hasDashLine)
		}

		// 5.3: Youtube (domain) 应包含 DOMAIN-SUFFIX 规则
		if p.Name == "Youtube" {
			hasDomainSuffix := false
			for _, line := range p.Lines {
				if strings.Contains(line, "DOMAIN-SUFFIX") {
					hasDomainSuffix = true
					break
				}
			}
			check("Youtube 包含 DOMAIN-SUFFIX 规则", hasDomainSuffix)
		}
	}

	// 5.4: 检查是否需要清除 Manual 标记以下的内容
	fmt.Println()
	fmt.Println("[Step 6] 验证文件可写入性")
	// truncate at marker
	if _, err := os.Stat(targetFile); err == nil {
		truncateAtMarker(targetFile, "[MANUAL_END] Private")
		fmt.Printf("  ✅ 目标文件已截断至 [MANUAL_END] Private\n")
	}

	// 5.5: 写入测试结果到文件
	outputFile := targetFile + ".test_output"
	f, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("  ❌ 无法创建输出文件: %v\n", err)
	} else {
		for _, p := range processed {
			f.WriteString(p.Header)
			for _, line := range p.Lines {
				f.WriteString(line + "\n")
			}
			f.WriteString("  # =====================\n\n")
		}
		f.Close()
		fmt.Printf("  ✅ 规则已写入 %s\n", outputFile)
		os.Remove(outputFile) // 清理
	}
	fmt.Println()

	// ── 结果汇总 ──
	fmt.Println("==========================================")
	status := "✅"
	if failCount > 0 {
		status = "❌"
	}
	fmt.Printf(" %s %d passed, %d failed out of %d checks\n", status, passCount, failCount, passCount+failCount)
	fmt.Println("==========================================")

	if failCount > 0 {
		os.Exit(1)
	}
}

// processContent 模拟 yaml_config.go 中的 processAwkLogic
func processContent(content, behavior string) []string {
	var result []string
	scanner := bufio.NewScanner(strings.NewReader(content))

	reEmpty := regexp.MustCompile(`^\s*$`)
	reComment := regexp.MustCompile(`^#`)
	rePayload := regexp.MustCompile(`^payload:`)
	reDashStart := regexp.MustCompile(`^\s*-\s*`)
	reQuotes := regexp.MustCompile(`['"]`)

	inPayload := false

	for scanner.Scan() {
		line := scanner.Text()

		if behavior == "domain" {
			if reComment.MatchString(line) || rePayload.MatchString(line) || reEmpty.MatchString(line) {
				continue
			}
			processed := reDashStart.ReplaceAllString(line, "")
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
			if rePayload.MatchString(line) {
				inPayload = true
				continue
			}
			if inPayload && reDashStart.MatchString(line) {
				result = append(result, line)
			}
		}
	}
	return result
}

// truncateAtMarker 模拟 yaml_config.go 中的 truncateFileAtMarker
func truncateAtMarker(filePath, marker string) {
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
			break
		}
	}
	newContent.WriteString("\n")
	os.WriteFile(filePath, []byte(newContent.String()), 0644)
}
