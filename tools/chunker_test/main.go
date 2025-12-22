package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/aihub/backend-go/internal/knowledge"
)

func main() {
	// 测试文本
	text := `生理优势：这类犬体型中等、肌肉发达，耐力充沛且奔跑速度较快，能适应野外的山地、林地等复杂地形，足以跟上野兔的奔跑节奏。

狩猎本能：作为传统的工作猎犬，它们天生有较强的追踪和捕猎欲望，对小型跑动的动物（如兔子）会表现出明显的追逐倾向，过去常被东北农村的猎人用于协助捕猎小型兽类。`

	// 创建分块器
	// 使用较小的chunkSize来测试分块效果（实际使用中通常是800）
	chunkSize := 100 // 测试用：100字符
	overlap := 20    // 测试用：20字符重叠

	// 如果命令行参数提供了chunkSize，使用它
	if len(os.Args) > 1 {
		fmt.Sscanf(os.Args[1], "%d", &chunkSize)
	}
	if len(os.Args) > 2 {
		fmt.Sscanf(os.Args[2], "%d", &overlap)
	}

	chunker := knowledge.NewChunker(chunkSize, overlap)
	fmt.Printf("分块配置: chunkSize=%d, overlap=%d\n\n", chunkSize, overlap)

	// 执行分块
	chunks := chunker.Split(text)

	// 打印结果
	fmt.Println("=" + strings.Repeat("=", 80))
	fmt.Println("智能分块测试结果")
	fmt.Println("=" + strings.Repeat("=", 80))
	fmt.Printf("原始文本长度: %d 字符\n", len([]rune(text)))
	fmt.Printf("分块数量: %d\n\n", len(chunks))

	for i, chunk := range chunks {
		fmt.Println(strings.Repeat("-", 80))
		fmt.Printf("块 #%d (索引: %d)\n", i+1, chunk.Index)
		fmt.Printf("Token数: %d\n", chunk.TokenCount)
		fmt.Printf("字符数: %d\n", len([]rune(chunk.Text)))
		fmt.Printf("内容:\n%s\n", chunk.Text)
		fmt.Println()
	}

	fmt.Println("=" + strings.Repeat("=", 80))
	fmt.Println("分块分析:")
	fmt.Println("=" + strings.Repeat("=", 80))

	// 分析分块质量
	totalChars := 0
	for i, chunk := range chunks {
		chars := len([]rune(chunk.Text))
		totalChars += chars
		fmt.Printf("块 #%d: %d字符, %d tokens\n", i+1, chars, chunk.TokenCount)

		// 检查是否在句子中间断开
		if i < len(chunks)-1 {
			lastChar := chunk.Text[len(chunk.Text)-1]
			nextFirstChar := chunks[i+1].Text[0]
			if !isSentenceEnd(rune(lastChar)) && !isSentenceEnd(rune(nextFirstChar)) {
				fmt.Printf("  ⚠️  警告: 块 #%d 和块 #%d 之间可能在句子中间断开\n", i+1, i+2)
			} else {
				fmt.Printf("  ✅ 块 #%d 和块 #%d 在语义边界断开\n", i+1, i+2)
			}
		}
	}

	fmt.Printf("\n总字符数: %d (原始: %d, 差异: %d)\n",
		totalChars, len([]rune(text)), totalChars-len([]rune(text)))
}

func isSentenceEnd(r rune) bool {
	return r == '。' || r == '！' || r == '？' || r == '.' || r == '!' || r == '?' ||
		r == '\n' || r == '\r' || r == ' ' || r == '\t'
}
