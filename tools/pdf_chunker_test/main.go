package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aihub/backend-go/internal/knowledge"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <pdf_file> [chunkSize] [overlap]")
		fmt.Println("Example: go run main.go 31092-正文-1-122.pdf 800 120")
		os.Exit(1)
	}

	pdfPath := os.Args[1]
	chunkSize := 800
	overlap := 120

	if len(os.Args) > 2 {
		fmt.Sscanf(os.Args[2], "%d", &chunkSize)
	}
	if len(os.Args) > 3 {
		fmt.Sscanf(os.Args[3], "%d", &overlap)
	}

	fmt.Println("=" + strings.Repeat("=", 100))
	fmt.Println("PDF文件智能分块测试")
	fmt.Println("=" + strings.Repeat("=", 100))
	fmt.Printf("PDF文件: %s\n", pdfPath)
	fmt.Printf("分块配置: chunkSize=%d, overlap=%d\n", chunkSize, overlap)
	fmt.Printf("测试时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// 打开PDF文件
	file, err := os.Open(pdfPath)
	if err != nil {
		fmt.Printf("❌ 错误: 无法打开PDF文件: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// 使用PDFParser提取文本
	parser := &knowledge.PDFParser{}
	if !parser.Supports(pdfPath) {
		fmt.Printf("❌ 错误: 文件 %s 不是PDF格式\n", pdfPath)
		os.Exit(1)
	}

	fmt.Println("📄 正在提取PDF文本...")
	startParse := time.Now()
	text, err := parser.Parse(file, pdfPath)
	if err != nil {
		fmt.Printf("❌ 错误: PDF解析失败: %v\n", err)
		os.Exit(1)
	}
	parseDuration := time.Since(startParse)

	// 分析文本
	textRunes := []rune(text)
	textLen := len(textRunes)
	paragraphs := strings.Split(text, "\n\n")
	nonEmptyParagraphs := 0
	for _, p := range paragraphs {
		if strings.TrimSpace(p) != "" {
			nonEmptyParagraphs++
		}
	}

	fmt.Printf("✅ PDF解析完成 (耗时: %v)\n", parseDuration)
	fmt.Printf("   - 文本长度: %d 字符\n", textLen)
	fmt.Printf("   - 段落数量: %d (非空: %d)\n", len(paragraphs), nonEmptyParagraphs)
	fmt.Printf("   - 预估Token数: %d (估算)\n\n", estimateTokens(text))

	// 执行分块
	fmt.Println("🔪 正在执行智能分块...")
	startChunk := time.Now()
	chunker := knowledge.NewChunker(chunkSize, overlap)
	chunks := chunker.Split(text)
	chunkDuration := time.Since(startChunk)

	fmt.Printf("✅ 分块完成 (耗时: %v)\n", chunkDuration)
	fmt.Printf("   - 分块数量: %d\n", len(chunks))
	fmt.Printf("   - 平均每块: %.1f 字符\n", float64(textLen)/float64(len(chunks)))
	fmt.Println()

	// 显示分块结果
	fmt.Println("=" + strings.Repeat("=", 100))
	fmt.Println("分块结果详情")
	fmt.Println("=" + strings.Repeat("=", 100))

	totalChars := 0
	totalTokens := 0
	semanticBreaks := 0
	nonSemanticBreaks := 0

	for i, chunk := range chunks {
		chars := len([]rune(chunk.Text))
		totalChars += chars
		totalTokens += chunk.TokenCount

		fmt.Println(strings.Repeat("-", 100))
		fmt.Printf("块 #%d (索引: %d)\n", i+1, chunk.Index)
		fmt.Printf("  字符数: %d\n", chars)
		fmt.Printf("  Token数: %d\n", chunk.TokenCount)
		fmt.Printf("  大小占比: %.1f%% (相对于chunkSize=%d)\n", float64(chars)/float64(chunkSize)*100, chunkSize)

		// 检查语义边界
		if i < len(chunks)-1 {
			chunkRunes := []rune(chunk.Text)
			nextChunkRunes := []rune(chunks[i+1].Text)
			if len(chunkRunes) > 0 && len(nextChunkRunes) > 0 {
				lastRune := chunkRunes[len(chunkRunes)-1]
				nextFirstRune := nextChunkRunes[0]
				isSemanticBreak := isSentenceEnd(lastRune) || isParagraphBreak(chunk.Text, chunks[i+1].Text)
				if isSemanticBreak {
					semanticBreaks++
					fmt.Printf("  边界: ✅ 语义边界\n")
				} else {
					nonSemanticBreaks++
					fmt.Printf("  边界: ⚠️  非语义边界 (块#%d结尾: '%c', 块#%d开头: '%c')\n", 
						i+1, lastRune, i+2, nextFirstRune)
				}
			}
		}

		// 显示内容预览（前100字符）
		preview := chunk.Text
		if len([]rune(preview)) > 100 {
			preview = string([]rune(preview)[:100]) + "..."
		}
		fmt.Printf("  内容预览: %s\n", strings.ReplaceAll(preview, "\n", "\\n"))
	}

	// 统计信息
	fmt.Println("\n" + strings.Repeat("=", 100))
	fmt.Println("分块统计")
	fmt.Println(strings.Repeat("=", 100))
	fmt.Printf("总字符数: %d (原始: %d, 差异: %d)\n", totalChars, textLen, totalChars-textLen)
	fmt.Printf("总Token数: %d (估算)\n", totalTokens)
	fmt.Printf("平均块大小: %.1f 字符\n", float64(totalChars)/float64(len(chunks)))
	fmt.Printf("平均Token数: %.1f\n", float64(totalTokens)/float64(len(chunks)))
	
	if semanticBreaks+nonSemanticBreaks > 0 {
		semanticRate := float64(semanticBreaks) / float64(semanticBreaks+nonSemanticBreaks) * 100
		fmt.Printf("语义边界保持率: %.1f%% (%d/%d)\n", semanticRate, semanticBreaks, semanticBreaks+nonSemanticBreaks)
	}

	// 性能统计
	fmt.Println("\n" + strings.Repeat("=", 100))
	fmt.Println("性能统计")
	fmt.Println(strings.Repeat("=", 100))
	fmt.Printf("PDF解析时间: %v\n", parseDuration)
	fmt.Printf("分块处理时间: %v\n", chunkDuration)
	fmt.Printf("总处理时间: %v\n", parseDuration+chunkDuration)
	fmt.Printf("处理速度: %.0f 字符/秒\n", float64(textLen)/(parseDuration+chunkDuration).Seconds())

	fmt.Println("\n" + strings.Repeat("=", 100))
}

func isSentenceEnd(r rune) bool {
	return r == '。' || r == '！' || r == '？' || r == '.' || r == '!' || r == '?'
}

func isParagraphBreak(chunk1, chunk2 string) bool {
	chunk1Runes := []rune(chunk1)
	chunk2Runes := []rune(chunk2)
	if len(chunk1Runes) == 0 || len(chunk2Runes) == 0 {
		return false
	}
	return isSentenceEnd(chunk1Runes[len(chunk1Runes)-1]) && 
		   (chunk2Runes[0] >= 'A' && chunk2Runes[0] <= 'Z' || 
		    chunk2Runes[0] >= 'a' && chunk2Runes[0] <= 'z' ||
		    chunk2Runes[0] >= 0x4e00 && chunk2Runes[0] <= 0x9fff)
}

func estimateTokens(text string) int {
	chineseChars := 0
	englishWords := 0
	
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			chineseChars++
		}
	}
	
	words := strings.Fields(text)
	for _, word := range words {
		hasEnglish := false
		for _, r := range word {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				hasEnglish = true
				break
			}
		}
		if hasEnglish {
			englishWords++
		}
	}
	
	estimated := int(float64(chineseChars)*1.5 + float64(englishWords)*1.3)
	if estimated < len(text)/4 {
		estimated = len(text) / 4
	}
	return estimated
}

