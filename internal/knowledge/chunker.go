package knowledge

import (
	"strings"
	"unicode"
)

// Chunk 表示分块后的文本结构
type Chunk struct {
	Index int
	Text  string
}

// Chunker 文本分块器
type Chunker struct {
	chunkSize    int
	chunkOverlap int
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

// Split 将文本切分为多个chunk
func (c *Chunker) Split(text string) []Chunk {
	clean := normalizeWhitespace(text)
	if clean == "" {
		return nil
	}

	runes := []rune(clean)
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
		chunks = append(chunks, Chunk{
			Index: len(chunks),
			Text:  chunkText,
		})

		if end == len(runes) {
			break
		}
	}

	return chunks
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
