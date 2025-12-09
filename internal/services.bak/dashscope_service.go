package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DashScopeService DashScope API服务封装
type DashScopeService struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewDashScopeService 创建DashScope服务实例
func NewDashScopeService(apiKey string, baseURL string) *DashScopeService {
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}

	return &DashScopeService{
		apiKey:  apiKey,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ChatMessage 聊天消息
type DashScopeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest DashScope聊天请求
type DashScopeChatRequest struct {
	Model       string            `json:"model"`
	Messages    []DashScopeMessage `json:"messages"`
	Temperature float64           `json:"temperature,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	TopP        float64           `json:"top_p,omitempty"`
	TopK        int               `json:"top_k,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
	// 通义千问特有参数
	EnableThinking bool `json:"enable_thinking,omitempty"`
	ThinkingBudget int  `json:"thinking_budget,omitempty"`
	EnableSearch   bool `json:"enable_search,omitempty"`
}

// ChatResponse DashScope聊天响应
type DashScopeChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int    `json:"index"`
		Message      DashScopeMessage `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Chat 调用DashScope聊天API（非流式）
func (s *DashScopeService) Chat(req DashScopeChatRequest) (*DashScopeChatResponse, error) {
	url := fmt.Sprintf("%s/chat/completions", s.baseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头（DashScope API规范）
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API调用失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("DashScope API错误: %s (code: %s)", errorResp.Error.Message, errorResp.Error.Code)
		}
		return nil, fmt.Errorf("DashScope API错误: HTTP %d - %s", resp.StatusCode, string(body))
	}

	var chatResp DashScopeChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &chatResp, nil
}

// ChatStream 调用DashScope聊天API（流式）
func (s *DashScopeService) ChatStream(req DashScopeChatRequest, onChunk func([]byte) error) error {
	req.Stream = true
	url := fmt.Sprintf("%s/chat/completions", s.baseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头（DashScope API规范）
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("API调用失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return fmt.Errorf("DashScope API错误: %s (code: %s)", errorResp.Error.Message, errorResp.Error.Code)
		}
		return fmt.Errorf("DashScope API错误: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// 读取SSE流
	buf := make([]byte, 4096)
	var lineBuf []byte

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			data := buf[:n]
			lineBuf = append(lineBuf, data...)

			// 处理完整的行
			for {
				idx := bytes.IndexByte(lineBuf, '\n')
				if idx == -1 {
					break
				}

				line := lineBuf[:idx]
				lineBuf = lineBuf[idx+1:]

				// 处理SSE格式: data: {...}
				if len(line) > 6 && string(line[:6]) == "data: " {
					chunk := line[6:]
					if string(chunk) == "[DONE]" {
						return nil
					}
					if err := onChunk(chunk); err != nil {
						return err
					}
				}
			}
		}

		if err == io.EOF {
			// 处理最后一行
			if len(lineBuf) > 0 {
				if len(lineBuf) > 6 && string(lineBuf[:6]) == "data: " {
					chunk := lineBuf[6:]
					if string(chunk) != "[DONE]" {
						onChunk(chunk)
					}
				}
			}
			break
		}
		if err != nil {
			return fmt.Errorf("读取流失败: %w", err)
		}
	}

	return nil
}

// ValidateAPIKey 验证API Key是否有效
func (s *DashScopeService) ValidateAPIKey() error {
	// 发送一个简单的请求来验证API Key
	testReq := DashScopeChatRequest{
		Model: "qwen-turbo",
		Messages: []DashScopeMessage{
			{Role: "user", Content: "test"},
		},
		MaxTokens: 1,
	}

	_, err := s.Chat(testReq)
	if err != nil {
		// 如果是认证错误，返回更明确的错误信息
		if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
			return fmt.Errorf("API Key无效或已过期")
		}
		return fmt.Errorf("API Key验证失败: %w", err)
	}

	return nil
}

// GetModels 获取可用的模型列表（如果API支持）
func (s *DashScopeService) GetModels() ([]string, error) {
	// DashScope可能没有公开的模型列表API
	// 返回默认支持的模型列表
	return []string{
		"qwen-turbo",
		"qwen-plus",
		"qwen-max",
		"qwen-max-longcontext",
		"qwen-vl-plus",
		"qwen-7b-chat",
		"qwen-14b-chat",
	}, nil
}

