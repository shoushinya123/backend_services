package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
)

func main() {
	var (
		input  = flag.String("input", "", "输入PDF文件路径（必需）")
		output = flag.String("output", "", "输出Markdown文件路径（可选，默认为输入文件名.md）")
	)
	flag.Parse()

	if *input == "" {
		fmt.Fprintf(os.Stderr, "错误: 必须指定输入PDF文件路径 (-input)\n")
		flag.Usage()
		os.Exit(1)
	}

	// 检查输入文件是否存在
	if _, err := os.Stat(*input); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "错误: 输入文件不存在: %s\n", *input)
		os.Exit(1)
	}

	// 确定输出文件路径
	outputPath := *output
	if outputPath == "" {
		ext := filepath.Ext(*input)
		outputPath = strings.TrimSuffix(*input, ext) + ".md"
	}

	// 转换PDF到Markdown
	if err := convertPDFToMarkdown(*input, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "错误: 转换失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ PDF转换成功: %s -> %s\n", *input, outputPath)
}

func convertPDFToMarkdown(inputPath, outputPath string) error {
	// 读取PDF文件
	pdfBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("读取PDF文件失败: %w", err)
	}

	// 创建PDF reader
	pdfReader, err := model.NewPdfReader(bytes.NewReader(pdfBytes))
	if err != nil {
		return fmt.Errorf("解析PDF失败: %w", err)
	}

	// 获取PDF信息
	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return fmt.Errorf("获取PDF页数失败: %w", err)
	}

	// 提取所有页面的文本
	var allText strings.Builder
	allText.WriteString(fmt.Sprintf("# %s\n\n", getTitleFromFilename(inputPath)))
	allText.WriteString(fmt.Sprintf("> 从PDF转换: %s\n", filepath.Base(inputPath)))
	allText.WriteString(fmt.Sprintf("> 总页数: %d\n\n", numPages))
	allText.WriteString("---\n\n")

	for i := 1; i <= numPages; i++ {
		page, err := pdfReader.GetPage(i)
		if err != nil {
			fmt.Fprintf(os.Stderr, "警告: 跳过第 %d 页: %v\n", i, err)
			continue
		}

		ex, err := extractor.New(page)
		if err != nil {
			fmt.Fprintf(os.Stderr, "警告: 提取第 %d 页失败: %v\n", i, err)
			continue
		}

		text, err := ex.ExtractText()
		if err != nil {
			fmt.Fprintf(os.Stderr, "警告: 提取第 %d 页文本失败: %v\n", i, err)
			continue
		}

		if strings.TrimSpace(text) != "" {
			// 添加页面分隔符
			allText.WriteString(fmt.Sprintf("## 第 %d 页\n\n", i))

			// 处理文本，尝试识别标题和段落
			formattedText := formatTextAsMarkdown(text)
			allText.WriteString(formattedText)
			allText.WriteString("\n\n---\n\n")
		}
	}

	// 写入Markdown文件
	if err := os.WriteFile(outputPath, []byte(allText.String()), 0644); err != nil {
		return fmt.Errorf("写入Markdown文件失败: %w", err)
	}

	return nil
}

// formatTextAsMarkdown 将提取的文本格式化为Markdown
func formatTextAsMarkdown(text string) string {
	lines := strings.Split(text, "\n")
	var result strings.Builder

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			result.WriteString("\n")
			continue
		}

		// 尝试识别标题（全大写、短行、包含数字编号等）
		if isLikelyHeading(line) {
			// 根据行长度决定标题级别
			if len([]rune(line)) < 30 {
				result.WriteString(fmt.Sprintf("### %s\n\n", line))
			} else {
				result.WriteString(fmt.Sprintf("#### %s\n\n", line))
			}
		} else {
			// 普通段落
			// 检查是否是列表项
			if isListItem(line) {
				result.WriteString(fmt.Sprintf("- %s\n", line))
			} else {
				result.WriteString(fmt.Sprintf("%s\n", line))
			}
		}

		// 在段落之间添加空行
		if i < len(lines)-1 && line != "" {
			nextLine := strings.TrimSpace(lines[i+1])
			if nextLine != "" && !isLikelyHeading(nextLine) {
				// 检查是否需要添加空行
				if !strings.HasSuffix(result.String(), "\n\n") {
					result.WriteString("\n")
				}
			}
		}
	}

	return result.String()
}

// isLikelyHeading 判断是否可能是标题
func isLikelyHeading(line string) bool {
	// 空行不是标题
	if line == "" {
		return false
	}

	runes := []rune(line)
	lineLen := len(runes)

	// 太长的行不太可能是标题
	if lineLen > 100 {
		return false
	}

	// 检查是否包含数字编号（如 "1. 标题"、"第一章"、"1.1" 等）
	headingPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^第[一二三四五六七八九十\d]+[章节部分]`),
		regexp.MustCompile(`^\d+[\.、]\s*`),
		regexp.MustCompile(`^[一二三四五六七八九十]+[、\.]\s*`),
		regexp.MustCompile(`^\d+\.\d+`), // 1.1, 2.3 等
	}

	for _, pattern := range headingPatterns {
		if pattern.MatchString(line) {
			return true
		}
	}

	// 检查是否全大写（可能是英文标题）
	upperCount := 0
	for _, r := range runes {
		if r >= 'A' && r <= 'Z' {
			upperCount++
		}
	}
	if len(runes) > 0 && float64(upperCount)/float64(len(runes)) > 0.7 {
		return true
	}

	// 短行且包含常见标题关键词
	if lineLen < 30 {
		headingKeywords := []string{"概述", "简介", "背景", "目标", "方案", "设计", "实现", "总结", "结论"}
		for _, keyword := range headingKeywords {
			if strings.Contains(line, keyword) {
				return true
			}
		}
	}

	return false
}

// isListItem 判断是否可能是列表项
func isListItem(line string) bool {
	// 检查是否以列表标记开头
	listPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^[•·▪▫◦]\s+`),
		regexp.MustCompile(`^[-*+]\s+`),
		regexp.MustCompile(`^\d+[\.、)]\s+`),
		regexp.MustCompile(`^[a-zA-Z][\.、)]\s+`),
	}

	for _, pattern := range listPatterns {
		if pattern.MatchString(line) {
			return true
		}
	}

	return false
}

// getTitleFromFilename 从文件名提取标题
func getTitleFromFilename(filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	title := strings.TrimSuffix(base, ext)

	// 移除常见的分隔符，替换为空格
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.ReplaceAll(title, "-", " ")

	// 清理多余空格
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")

	return strings.TrimSpace(title)
}
