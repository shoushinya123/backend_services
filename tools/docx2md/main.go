package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/unidoc/unioffice/document"
)

func main() {
	var (
		input  = flag.String("input", "", "输入DOCX文件路径（必需）")
		output = flag.String("output", "", "输出Markdown文件路径（可选，默认为输入文件名.md）")
	)
	flag.Parse()

	if *input == "" {
		fmt.Fprintf(os.Stderr, "错误: 必须指定输入DOCX文件路径 (-input)\n")
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

	// 转换DOCX到Markdown
	if err := convertDocxToMarkdown(*input, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "错误: 转换失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ DOCX转换成功: %s -> %s\n", *input, outputPath)
}

func convertDocxToMarkdown(inputPath, outputPath string) error {
	// 读取DOCX文件
	docBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("读取DOCX文件失败: %w", err)
	}

	// 解析Word文档
	readerAt := bytes.NewReader(docBytes)
	doc, err := document.Read(readerAt, int64(len(docBytes)))
	if err != nil {
		return fmt.Errorf("解析Word文档失败: %w", err)
	}
	defer doc.Close()

	// 提取文本并格式化为Markdown
	var mdBuilder strings.Builder
	
	// 添加标题
	title := getTitleFromFilename(inputPath)
	mdBuilder.WriteString(fmt.Sprintf("# %s\n\n", title))
	mdBuilder.WriteString(fmt.Sprintf("> 从DOCX转换: %s\n\n", filepath.Base(inputPath)))
	mdBuilder.WriteString("---\n\n")

	// 遍历段落
	for _, para := range doc.Paragraphs() {
		paraText := extractParagraphText(para)
		
		if strings.TrimSpace(paraText) == "" {
			mdBuilder.WriteString("\n")
			continue
		}

		// 检查段落样式，判断是否为标题
		style := para.Style()
		if style != "" {
			// 根据样式判断标题级别
			if strings.Contains(strings.ToLower(style), "heading") || 
			   strings.Contains(strings.ToLower(style), "title") {
				// 尝试提取标题级别
				level := extractHeadingLevel(style)
				mdBuilder.WriteString(strings.Repeat("#", level) + " " + paraText + "\n\n")
				continue
			}
		}

		// 检查段落格式（加粗、字体大小等）来判断标题
		if isLikelyHeading(para) {
			mdBuilder.WriteString("## " + paraText + "\n\n")
			continue
		}

		// 普通段落（简化处理，不判断列表）
		mdBuilder.WriteString(paraText + "\n\n")
	}

	// 写入Markdown文件
	if err := os.WriteFile(outputPath, []byte(mdBuilder.String()), 0644); err != nil {
		return fmt.Errorf("写入Markdown文件失败: %w", err)
	}

	return nil
}

func extractParagraphText(para document.Paragraph) string {
	var textBuilder strings.Builder
	for _, run := range para.Runs() {
		text := run.Text()
		
		// 处理加粗文本
		if run.Properties().IsBold() {
			text = "**" + text + "**"
		}
		
		// 处理斜体文本
		if run.Properties().IsItalic() {
			text = "*" + text + "*"
		}
		
		textBuilder.WriteString(text)
	}
	return textBuilder.String()
}

func isLikelyHeading(para document.Paragraph) bool {
	// 检查是否有特殊的段落样式
	style := para.Style()
	if style != "" {
		styleLower := strings.ToLower(style)
		if strings.Contains(styleLower, "heading") ||
		   strings.Contains(styleLower, "title") {
			return true
		}
	}

	// 检查段落文本特征
	text := extractParagraphText(para)
	text = strings.TrimSpace(text)
	
	// 短行可能是标题
	if len([]rune(text)) < 50 && text != "" {
		// 检查是否包含标题关键词
		headingKeywords := []string{"概述", "简介", "背景", "目标", "方案", "设计", "实现", "总结", "结论", "第一章", "第二章", "第", "章"}
		for _, keyword := range headingKeywords {
			if strings.Contains(text, keyword) {
				return true
			}
		}
	}

	return false
}

func extractHeadingLevel(style string) int {
	styleLower := strings.ToLower(style)
	
	// 尝试从样式中提取级别
	if strings.Contains(styleLower, "heading 1") || strings.Contains(styleLower, "title") {
		return 1
	}
	if strings.Contains(styleLower, "heading 2") {
		return 2
	}
	if strings.Contains(styleLower, "heading 3") {
		return 3
	}
	if strings.Contains(styleLower, "heading 4") {
		return 4
	}
	
	// 默认返回2级标题
	return 2
}

func getTitleFromFilename(filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	title := strings.TrimSuffix(base, ext)
	
	// 移除常见的分隔符，替换为空格
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.ReplaceAll(title, "-", " ")
	
	// 清理多余空格
	title = strings.ReplaceAll(title, "  ", " ")
	
	return strings.TrimSpace(title)
}

