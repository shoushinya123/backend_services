package knowledge

import (
	"context"
	"strings"
	"unicode"
)

// Chunk 表示分块后的文本结构
type Chunk struct {
	Index      int
	Text       string
	TokenCount int // Token数量
}

// Chunker 文本分块器
type Chunker struct {
	chunkSize    int
	chunkOverlap int
	tokenCounter TokenCounter // Token计数器（可选）
}

// TokenCounter Token计数接口
type TokenCounter interface {
	CountTokens(ctx context.Context, text string) (int, error)
}

// NewChunker 创建分块器
func NewChunker(chunkSize, overlap int) *Chunker {
	if chunkSize <= 0 {
		chunkSize = 800
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= chunkSize {
		overlap = chunkSize / 4
	}
	return &Chunker{
		chunkSize:    chunkSize,
		chunkOverlap: overlap,
	}
}

// SetTokenCounter 设置Token计数器
func (c *Chunker) SetTokenCounter(counter TokenCounter) {
	c.tokenCounter = counter
}

// Split 将文本切分为多个chunk
func (c *Chunker) Split(text string) []Chunk {
	return c.SplitWithContext(context.Background(), text)
}

// SplitWithContext 将文本切分为多个chunk（支持上下文和Token计数）
func (c *Chunker) SplitWithContext(ctx context.Context, text string) []Chunk {
	clean := normalizeWhitespace(text)
	if clean == "" {
		return nil
	}

	// 尝试在语义边界处分割（句子、段落）
	chunks := c.splitBySemanticBoundary(ctx, clean)
	if len(chunks) == 0 {
		// 降级到字符级分割
		chunks = c.splitByCharacter(ctx, clean)
	}

	return chunks
}

// splitBySemanticBoundary 按语义边界分割（句子、段落）
func (c *Chunker) splitBySemanticBoundary(ctx context.Context, text string) []Chunk {
	var chunks []Chunk
	paragraphs := strings.Split(text, "\n\n")
	
	currentChunk := strings.Builder{}
	currentSize := 0
	
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		
		// 估算段落大小（字符数）
		paraSize := len([]rune(para))
		
		// 如果当前块加上新段落会超过大小，先保存当前块
		if currentSize > 0 && currentSize+paraSize > c.chunkSize {
			chunkText := currentChunk.String()
			if chunkText != "" {
				tokenCount := c.estimateTokenCount(ctx, chunkText)
				chunks = append(chunks, Chunk{
					Index:      len(chunks),
					Text:       chunkText,
					TokenCount: tokenCount,
				})
			}
			currentChunk.Reset()
			currentSize = 0
		}
		
		// 如果单个段落就超过大小，按句子分割
		if paraSize > c.chunkSize {
			sentences := c.splitSentences(para)
			for _, sent := range sentences {
				sentSize := len([]rune(sent))
				if currentSize > 0 && currentSize+sentSize > c.chunkSize {
					chunkText := currentChunk.String()
					if chunkText != "" {
						tokenCount := c.estimateTokenCount(ctx, chunkText)
						chunks = append(chunks, Chunk{
							Index:      len(chunks),
							Text:       chunkText,
							TokenCount: tokenCount,
						})
					}
					currentChunk.Reset()
					currentSize = 0
				}
				if currentChunk.Len() > 0 {
					currentChunk.WriteString("\n")
				}
				currentChunk.WriteString(sent)
				currentSize += sentSize + 1
			}
		} else {
			if currentChunk.Len() > 0 {
				currentChunk.WriteString("\n\n")
			}
			currentChunk.WriteString(para)
			currentSize += paraSize + 2
		}
	}
	
	// 保存最后一个块
	if currentChunk.Len() > 0 {
		chunkText := currentChunk.String()
		tokenCount := c.estimateTokenCount(ctx, chunkText)
		chunks = append(chunks, Chunk{
			Index:      len(chunks),
			Text:       chunkText,
			TokenCount: tokenCount,
		})
	}
	
	return chunks
}

// splitByCharacter 按字符数分割（降级方案）
func (c *Chunker) splitByCharacter(ctx context.Context, text string) []Chunk {
	runes := []rune(text)
	var chunks []Chunk

	step := c.chunkSize - c.chunkOverlap
	if step <= 0 {
		step = c.chunkSize
	}

	for start := 0; start < len(runes); start += step {
		end := start + c.chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunkText := strings.TrimSpace(string(runes[start:end]))
		if chunkText == "" {
			continue
		}
		tokenCount := c.estimateTokenCount(ctx, chunkText)
		chunks = append(chunks, Chunk{
			Index:      len(chunks),
			Text:       chunkText,
			TokenCount: tokenCount,
		})

		if end == len(runes) {
			break
		}
	}

	return chunks
}

// splitSentences 分割句子
func (c *Chunker) splitSentences(text string) []string {
	// 简单的句子分割：按句号、问号、感叹号分割
	sentences := []string{}
	current := strings.Builder{}
	
	for _, r := range text {
		current.WriteRune(r)
		if r == '。' || r == '！' || r == '？' || r == '.' || r == '!' || r == '?' {
			sent := strings.TrimSpace(current.String())
			if sent != "" {
				sentences = append(sentences, sent)
			}
			current.Reset()
		}
	}
	
	if current.Len() > 0 {
		sent := strings.TrimSpace(current.String())
		if sent != "" {
			sentences = append(sentences, sent)
		}
	}
	
	return sentences
}

// estimateTokenCount 估算Token数量
func (c *Chunker) estimateTokenCount(ctx context.Context, text string) int {
	if c.tokenCounter != nil {
		count, err := c.tokenCounter.CountTokens(ctx, text)
		if err == nil {
			return count
		}
	}
	// 简单估算：中文字符*1.5 + 英文单词*1.3
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

func normalizeWhitespace(s string) string {
	var builder strings.Builder
	builder.Grow(len(s))

	var prevSpace bool
	for _, r := range s {
		if unicode.IsSpace(r) {
			if prevSpace {
				continue
			}
			builder.WriteRune(' ')
			prevSpace = true
			continue
		}
		builder.WriteRune(r)
		prevSpace = false
	}

	return strings.TrimSpace(builder.String())
}
