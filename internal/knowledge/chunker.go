package knowledge

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// DocumentType 文档类型枚举
type DocumentType int

const (
	DocTypeUnknown    DocumentType = iota
	DocTypeText                    // 纯文本文档
	DocTypeCode                    // 代码文件
	DocTypeMarkdown                // Markdown文档
	DocTypeHTML                    // HTML文档
	DocTypePDF                     // PDF文档
	DocTypeStructured              // 结构化文档（如JSON、XML）
	DocTypeLongForm                // 长文档（如小说、小说）
)

// ChunkStrategy 分块策略配置
type ChunkStrategy struct {
	ChunkSize         int  // 分块大小
	ChunkOverlap      int  // 分块重叠
	PreserveStructure bool // 是否保留结构（如代码块、标题等）
	SplitBySemantic   bool // 是否按语义边界分割
	MaxChunkSize      int  // 最大分块大小
	MinChunkSize      int  // 最小分块大小
}

// DocumentTypeDetector 文档类型检测器
type DocumentTypeDetector struct {
	patterns map[DocumentType][]*regexp.Regexp
}

// NewDocumentTypeDetector 创建文档类型检测器
func NewDocumentTypeDetector() *DocumentTypeDetector {
	detector := &DocumentTypeDetector{
		patterns: make(map[DocumentType][]*regexp.Regexp),
	}

	// 初始化检测模式
	detector.initPatterns()
	return detector
}

// initPatterns 初始化检测模式
func (d *DocumentTypeDetector) initPatterns() {
	// 代码文件模式
	d.patterns[DocTypeCode] = []*regexp.Regexp{
		regexp.MustCompile(`(?m)^(def |class |func |public |private |import |package |const |var |let |const )`),
		regexp.MustCompile(`(?m)^\s*[{}();]\s*$`),
		regexp.MustCompile(`(?m)<[^>]+>`), // HTML/XML标签
	}

	// Markdown模式
	d.patterns[DocTypeMarkdown] = []*regexp.Regexp{
		regexp.MustCompile("(?m)^#{1,6}\\s+"), // 标题
		regexp.MustCompile("(?m)^[-*+]\\s+"),  // 列表
		regexp.MustCompile("(?m)^>\\s+"),      // 引用
		regexp.MustCompile("(?m)```\\w*"),     // 代码块
	}

	// HTML模式
	d.patterns[DocTypeHTML] = []*regexp.Regexp{
		regexp.MustCompile("(?i)<html[^>]*>"),
		regexp.MustCompile("(?i)<head[^>]*>"),
		regexp.MustCompile("(?i)<body[^>]*>"),
		regexp.MustCompile("(?i)<div[^>]*>"),
	}

	// JSON/XML模式
	d.patterns[DocTypeStructured] = []*regexp.Regexp{
		regexp.MustCompile("^\\s*[\\{\\[\\[]"), // JSON开始
		regexp.MustCompile("(?i)^\\s*<\\?xml"), // XML声明
		regexp.MustCompile("(?i)^\\s*<[^>]+>"), // XML标签
	}

	// 长文档模式（小说等）
	d.patterns[DocTypeLongForm] = []*regexp.Regexp{
		regexp.MustCompile(`(?m)第[一二三四五六七八九十百千]+章`),
		regexp.MustCompile(`(?m)第[0-9]+章`),
		regexp.MustCompile(`(?m)[。！？][\n\r]+`), // 句子分段
	}
}

// DetectType 检测文档类型
func (d *DocumentTypeDetector) DetectType(filename, content string) DocumentType {
	// 首先根据文件扩展名判断
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".md", ".markdown":
		return DocTypeMarkdown
	case ".html", ".htm":
		return DocTypeHTML
	case ".json", ".xml", ".yaml", ".yml":
		return DocTypeStructured
	case ".pdf":
		return DocTypePDF
	case ".txt":
		// 对于txt文件，需要进一步分析内容
		if d.matchesType(content, DocTypeCode) {
			return DocTypeCode
		}
		if d.matchesType(content, DocTypeLongForm) {
			return DocTypeLongForm
		}
		return DocTypeText
	case ".go", ".py", ".js", ".ts", ".java", ".c", ".cpp", ".cs", ".php", ".rb", ".rs":
		return DocTypeCode
	}

	// 根据内容判断
	if d.matchesType(content, DocTypeCode) {
		return DocTypeCode
	}
	if d.matchesType(content, DocTypeMarkdown) {
		return DocTypeMarkdown
	}
	if d.matchesType(content, DocTypeHTML) {
		return DocTypeHTML
	}
	if d.matchesType(content, DocTypeStructured) {
		return DocTypeStructured
	}
	if d.matchesType(content, DocTypeLongForm) {
		return DocTypeLongForm
	}

	return DocTypeText
}

// matchesType 检查内容是否匹配指定类型
func (d *DocumentTypeDetector) matchesType(content string, docType DocumentType) bool {
	patterns, exists := d.patterns[docType]
	if !exists {
		return false
	}

	// 计算匹配的模式数量
	matchCount := 0
	for _, pattern := range patterns {
		if pattern.MatchString(content) {
			matchCount++
		}
	}

	// 如果匹配多个模式，则认为是该类型
	return matchCount >= 2
}

// DynamicChunkStrategy 动态分块策略管理器
type DynamicChunkStrategy struct {
	detector   *DocumentTypeDetector
	strategies map[DocumentType]ChunkStrategy
}

// NewDynamicChunkStrategy 创建动态分块策略管理器
func NewDynamicChunkStrategy() *DynamicChunkStrategy {
	dcs := &DynamicChunkStrategy{
		detector:   NewDocumentTypeDetector(),
		strategies: make(map[DocumentType]ChunkStrategy),
	}
	dcs.initStrategies()
	return dcs
}

// initStrategies 初始化各类型文档的分块策略
func (dcs *DynamicChunkStrategy) initStrategies() {
	// 纯文本文档策略
	dcs.strategies[DocTypeText] = ChunkStrategy{
		ChunkSize:         800,
		ChunkOverlap:      200,
		PreserveStructure: false,
		SplitBySemantic:   true,
		MaxChunkSize:      1200,
		MinChunkSize:      400,
	}

	// 代码文件策略
	dcs.strategies[DocTypeCode] = ChunkStrategy{
		ChunkSize:         600,
		ChunkOverlap:      150,
		PreserveStructure: true,
		SplitBySemantic:   true,
		MaxChunkSize:      900,
		MinChunkSize:      300,
	}

	// Markdown文档策略
	dcs.strategies[DocTypeMarkdown] = ChunkStrategy{
		ChunkSize:         1000,
		ChunkOverlap:      250,
		PreserveStructure: true,
		SplitBySemantic:   true,
		MaxChunkSize:      1500,
		MinChunkSize:      500,
	}

	// HTML文档策略
	dcs.strategies[DocTypeHTML] = ChunkStrategy{
		ChunkSize:         700,
		ChunkOverlap:      175,
		PreserveStructure: true,
		SplitBySemantic:   false,
		MaxChunkSize:      1050,
		MinChunkSize:      350,
	}

	// PDF文档策略
	dcs.strategies[DocTypePDF] = ChunkStrategy{
		ChunkSize:         900,
		ChunkOverlap:      225,
		PreserveStructure: false,
		SplitBySemantic:   true,
		MaxChunkSize:      1350,
		MinChunkSize:      450,
	}

	// 结构化文档策略
	dcs.strategies[DocTypeStructured] = ChunkStrategy{
		ChunkSize:         500,
		ChunkOverlap:      125,
		PreserveStructure: true,
		SplitBySemantic:   false,
		MaxChunkSize:      750,
		MinChunkSize:      250,
	}

	// 长文档策略
	dcs.strategies[DocTypeLongForm] = ChunkStrategy{
		ChunkSize:         1200,
		ChunkOverlap:      300,
		PreserveStructure: false,
		SplitBySemantic:   true,
		MaxChunkSize:      1800,
		MinChunkSize:      600,
	}

	// 默认策略
	dcs.strategies[DocTypeUnknown] = ChunkStrategy{
		ChunkSize:         800,
		ChunkOverlap:      200,
		PreserveStructure: false,
		SplitBySemantic:   true,
		MaxChunkSize:      1200,
		MinChunkSize:      400,
	}
}

// GetStrategy 根据文档类型获取分块策略
func (dcs *DynamicChunkStrategy) GetStrategy(filename, content string) ChunkStrategy {
	docType := dcs.detector.DetectType(filename, content)
	strategy, exists := dcs.strategies[docType]
	if !exists {
		strategy = dcs.strategies[DocTypeUnknown]
	}
	return strategy
}

// Chunk 表示分块后的文本结构
type Chunk struct {
	Index      int
	Text       string
	TokenCount int // Token数量
}

// Chunker 文本分块器
type Chunker struct {
	chunkSize       int
	chunkOverlap    int
	tokenCounter    TokenCounter          // Token计数器（可选）
	strategy        ChunkStrategy         // 分块策略
	dynamicStrategy *DynamicChunkStrategy // 动态策略管理器
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
		chunkSize:       chunkSize,
		chunkOverlap:    overlap,
		dynamicStrategy: NewDynamicChunkStrategy(),
	}
}

// NewDynamicChunker 创建动态分块器
func NewDynamicChunker() *Chunker {
	return &Chunker{
		chunkSize:       800, // 默认值，会被动态策略覆盖
		chunkOverlap:    200,
		dynamicStrategy: NewDynamicChunkStrategy(),
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

// SplitDocument 根据文档类型动态分块
func (c *Chunker) SplitDocument(ctx context.Context, filename, content string) []Chunk {
	// 获取动态分块策略
	strategy := c.dynamicStrategy.GetStrategy(filename, content)

	// 更新chunker的当前策略
	c.strategy = strategy

	clean := normalizeWhitespace(content)
	if clean == "" {
		return nil
	}

	var chunks []Chunk

	// 根据策略选择分割方法
	if strategy.SplitBySemantic {
		chunks = c.splitBySemanticBoundary(ctx, clean)
		if len(chunks) == 0 {
			chunks = c.splitByCharacter(ctx, clean)
		}
	} else {
		chunks = c.splitByCharacter(ctx, clean)
	}

	// 根据策略调整分块大小
	if strategy.PreserveStructure {
		chunks = c.adjustChunksForStructure(chunks, strategy)
	}

	return chunks
}

// splitBySemanticBoundary 按语义边界分割（句子、段落）
func (c *Chunker) splitBySemanticBoundary(ctx context.Context, text string) []Chunk {
	var chunks []Chunk

	// 使用动态策略的chunkSize，如果没有则使用默认值
	chunkSize := c.chunkSize
	if c.strategy.ChunkSize > 0 {
		chunkSize = c.strategy.ChunkSize
	}

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
		if currentSize > 0 && currentSize+paraSize > chunkSize {
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
		if paraSize > chunkSize {
			sentences := c.splitSentences(para)
			for _, sent := range sentences {
				sentSize := len([]rune(sent))
				if currentSize > 0 && currentSize+sentSize > chunkSize {
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

	// 使用动态策略的chunkSize和chunkOverlap
	chunkSize := c.chunkSize
	chunkOverlap := c.chunkOverlap
	if c.strategy.ChunkSize > 0 {
		chunkSize = c.strategy.ChunkSize
	}
	if c.strategy.ChunkOverlap > 0 {
		chunkOverlap = c.strategy.ChunkOverlap
	}

	step := chunkSize - chunkOverlap
	if step <= 0 {
		step = chunkSize
	}

	for start := 0; start < len(runes); start += step {
		end := start + chunkSize
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

// adjustChunksForStructure 根据结构调整分块
func (c *Chunker) adjustChunksForStructure(chunks []Chunk, strategy ChunkStrategy) []Chunk {
	var adjusted []Chunk

	for _, chunk := range chunks {
		// 检查分块大小是否在合理范围内
		if chunk.TokenCount > strategy.MaxChunkSize {
			// 如果太大，尝试进一步分割
			subChunks := c.splitLargeChunk(chunk, strategy)
			adjusted = append(adjusted, subChunks...)
		} else if chunk.TokenCount < strategy.MinChunkSize && len(adjusted) > 0 {
			// 如果太小，尝试与前一个分块合并
			lastIdx := len(adjusted) - 1
			if lastIdx >= 0 {
				combinedText := adjusted[lastIdx].Text + "\n" + chunk.Text
				combinedTokens := c.estimateTokenCount(context.Background(), combinedText)

				if combinedTokens <= strategy.MaxChunkSize {
					// 合并分块
					adjusted[lastIdx] = Chunk{
						Index:      adjusted[lastIdx].Index,
						Text:       combinedText,
						TokenCount: combinedTokens,
					}
					continue
				}
			}
			// 如果无法合并，保持原样
			adjusted = append(adjusted, chunk)
		} else {
			adjusted = append(adjusted, chunk)
		}
	}

	return adjusted
}

// splitLargeChunk 分割过大的分块
func (c *Chunker) splitLargeChunk(chunk Chunk, strategy ChunkStrategy) []Chunk {
	text := chunk.Text
	runes := []rune(text)

	// 计算分割点
	halfSize := len(runes) / 2

	// 寻找合适的分割点（句子边界或段落边界）
	splitIndex := halfSize
	for i := halfSize; i < len(runes) && i < halfSize+100; i++ {
		if runes[i] == '。' || runes[i] == '！' || runes[i] == '？' ||
			runes[i] == '.' || runes[i] == '!' || runes[i] == '?' ||
			runes[i] == '\n' {
			splitIndex = i + 1
			break
		}
	}

	if splitIndex >= len(runes) {
		splitIndex = halfSize
	}

	// 分割成两个子分块
	part1 := string(runes[:splitIndex])
	part2 := string(runes[splitIndex:])

	subChunks := []Chunk{
		{
			Index:      chunk.Index * 2,
			Text:       strings.TrimSpace(part1),
			TokenCount: c.estimateTokenCount(context.Background(), part1),
		},
		{
			Index:      chunk.Index*2 + 1,
			Text:       strings.TrimSpace(part2),
			TokenCount: c.estimateTokenCount(context.Background(), part2),
		},
	}

	return subChunks
}
