package knowledge

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/unidoc/unioffice/document"
	"github.com/unidoc/unioffice/spreadsheet"
	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
)

// FileParser 文件解析器接口
type FileParser interface {
	Parse(reader io.Reader, filename string) (string, error)
	Supports(filename string) bool
}

// TextParser 文本文件解析器
type TextParser struct{}

func (p *TextParser) Supports(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".txt" || ext == ".md" || ext == ".markdown"
}

func (p *TextParser) Parse(reader io.Reader, filename string) (string, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}
	return string(content), nil
}

// PDFParser PDF文件解析器
type PDFParser struct{}

func (p *PDFParser) Supports(filename string) bool {
	return strings.ToLower(filepath.Ext(filename)) == ".pdf"
}

func (p *PDFParser) Parse(reader io.Reader, filename string) (string, error) {
	// 读取PDF内容
	pdfBytes, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("读取PDF文件失败: %w", err)
	}

	// 创建PDF reader
	pdfReader, err := model.NewPdfReader(bytes.NewReader(pdfBytes))
	if err != nil {
		return "", fmt.Errorf("解析PDF失败: %w", err)
	}

	// 提取文本
	var textBuilder strings.Builder
	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return "", fmt.Errorf("获取PDF页数失败: %w", err)
	}

	for i := 1; i <= numPages; i++ {
		page, err := pdfReader.GetPage(i)
		if err != nil {
			continue
		}

		ex, err := extractor.New(page)
		if err != nil {
			continue
		}

		text, err := ex.ExtractText()
		if err != nil {
			continue
		}

		textBuilder.WriteString(text)
		textBuilder.WriteString("\n")
	}

	return textBuilder.String(), nil
}

// WordParser Word文档解析器
type WordParser struct{}

func (p *WordParser) Supports(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".docx" || ext == ".doc"
}

func (p *WordParser) Parse(reader io.Reader, filename string) (string, error) {
	// 读取文档内容
	docBytes, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("读取Word文件失败: %w", err)
	}

	// 解析Word文档（仅支持.docx格式）
	if strings.ToLower(filepath.Ext(filename)) == ".doc" {
		return "", fmt.Errorf("暂不支持.doc格式，请使用.docx格式")
	}

	// 使用bytes.Reader实现ReaderAt接口
	readerAt := bytes.NewReader(docBytes)
	doc, err := document.Read(readerAt, int64(len(docBytes)))
	if err != nil {
		return "", fmt.Errorf("解析Word文档失败: %w", err)
	}
	defer doc.Close()

	// 提取文本
	var textBuilder strings.Builder
	for _, para := range doc.Paragraphs() {
		// 获取段落中的所有runs并提取文本
		for _, run := range para.Runs() {
			textBuilder.WriteString(run.Text())
		}
		textBuilder.WriteString("\n")
	}

	return textBuilder.String(), nil
}

// ExcelParser Excel文件解析器
type ExcelParser struct{}

func (p *ExcelParser) Supports(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".xlsx" || ext == ".xls"
}

func (p *ExcelParser) Parse(reader io.Reader, filename string) (string, error) {
	// 读取Excel内容
	excelBytes, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("读取Excel文件失败: %w", err)
	}

	// 解析Excel文档（仅支持.xlsx格式）
	if strings.ToLower(filepath.Ext(filename)) == ".xls" {
		return "", fmt.Errorf("暂不支持.xls格式，请使用.xlsx格式")
	}

	// 使用bytes.Reader实现ReaderAt接口
	readerAt := bytes.NewReader(excelBytes)
	ss, err := spreadsheet.Read(readerAt, int64(len(excelBytes)))
	if err != nil {
		return "", fmt.Errorf("解析Excel文档失败: %w", err)
	}
	defer ss.Close()

	// 提取文本
	var textBuilder strings.Builder
	for _, sheet := range ss.Sheets() {
		textBuilder.WriteString(fmt.Sprintf("工作表: %s\n", sheet.Name()))
		
		for _, row := range sheet.Rows() {
			var rowText []string
			for _, cell := range row.Cells() {
				cellText := cell.GetString()
				rowText = append(rowText, cellText)
			}
			if len(rowText) > 0 {
				textBuilder.WriteString(strings.Join(rowText, "\t"))
				textBuilder.WriteString("\n")
			}
		}
		textBuilder.WriteString("\n")
	}

	return textBuilder.String(), nil
}

// FileParserManager 文件解析器管理器
type FileParserManager struct {
	parsers []FileParser
}

// NewFileParserManager 创建文件解析器管理器
func NewFileParserManager() *FileParserManager {
	return &FileParserManager{
		parsers: []FileParser{
			&PDFParser{},
			&WordParser{},
			&ExcelParser{},
			&TextParser{},
		},
	}
}

// ParseFile 解析文件
func (m *FileParserManager) ParseFile(reader io.Reader, filename string) (string, error) {
	for _, parser := range m.parsers {
		if parser.Supports(filename) {
			return parser.Parse(reader, filename)
		}
	}
	return "", fmt.Errorf("不支持的文件格式: %s", filename)
}

// GetSupportedFormats 获取支持的文件格式
func (m *FileParserManager) GetSupportedFormats() []string {
	formats := make(map[string]bool)
	for _, parser := range m.parsers {
		switch parser.(type) {
		case *PDFParser:
			formats[".pdf"] = true
		case *WordParser:
			formats[".docx"] = true
			formats[".doc"] = true
		case *ExcelParser:
			formats[".xlsx"] = true
			formats[".xls"] = true
		case *TextParser:
			formats[".txt"] = true
			formats[".md"] = true
			formats[".markdown"] = true
		}
	}
	
	result := make([]string, 0, len(formats))
	for format := range formats {
		result = append(result, format)
	}
	return result
}

// ParseFileMetadata 解析文件元数据
func (m *FileParserManager) ParseFileMetadata(reader io.Reader, filename string) (map[string]interface{}, error) {
	metadata := map[string]interface{}{
		"filename": filename,
		"extension": filepath.Ext(filename),
		"supported": false,
	}

	// 检查是否支持
	for _, parser := range m.parsers {
		if parser.Supports(filename) {
			metadata["supported"] = true
			break
		}
	}

	// 尝试解析文件大小
	if seeker, ok := reader.(io.Seeker); ok {
		pos, _ := seeker.Seek(0, io.SeekCurrent)
		end, _ := seeker.Seek(0, io.SeekEnd)
		metadata["size"] = end - pos
		seeker.Seek(pos, io.SeekStart)
	}

	return metadata, nil
}

