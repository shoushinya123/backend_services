package plugins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// PluginServiceClient 插件服务客户端（主系统调用插件服务）
type PluginServiceClient struct {
	baseURL    string
	httpClient *http.Client
	userID     uint
}

// NewPluginServiceClient 创建插件服务客户端
func NewPluginServiceClient(baseURL string, userID uint) *PluginServiceClient {
	if baseURL == "" {
		baseURL = "http://plugin-service:8002" // 默认内部服务地址
	}
	return &PluginServiceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userID: userID,
	}
}

// UploadPlugin 上传插件
func (c *PluginServiceClient) UploadPlugin(fileReader io.Reader, filename string) (map[string]interface{}, error) {
	// 读取文件内容到内存（因为需要多次使用）
	fileData, err := io.ReadAll(fileReader)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	// 创建multipart form
	body := &bytes.Buffer{}
	formWriter := multipart.NewWriter(body)
	
	// 添加文件字段
	part, err := formWriter.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("创建form文件字段失败: %w", err)
	}
	
	if _, err := part.Write(fileData); err != nil {
		return nil, fmt.Errorf("写入文件数据失败: %w", err)
	}
	
	if err := formWriter.Close(); err != nil {
		return nil, fmt.Errorf("关闭form writer失败: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/plugins/upload", c.baseURL), body)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", formWriter.FormDataContentType())
	req.Header.Set("X-User-Id", fmt.Sprintf("%d", c.userID))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	if resp.StatusCode != http.StatusOK {
		// 检查是否是Envoy返回的非JSON错误
		if strings.Contains(bodyStr, "no healthy upstream") || strings.Contains(bodyStr, "Service Unavailable") {
			return nil, fmt.Errorf("服务不可用: 插件服务未启动或健康检查失败 (HTTP %d)\n"+
				"请检查:\n"+
				"1. 插件服务是否正在运行\n"+
				"2. 插件服务的健康检查端点 /health 是否可访问\n"+
				"3. Envoy配置是否正确", resp.StatusCode)
		}
		// 尝试解析JSON错误
		var jsonErr struct {
			Error string `json:"error"`
			Data  struct {
				Error string `json:"error"`
			} `json:"data"`
		}
		if err := json.Unmarshal(bodyBytes, &jsonErr); err == nil {
			if jsonErr.Error != "" {
				return nil, fmt.Errorf("上传失败: %s", jsonErr.Error)
			}
			if jsonErr.Data.Error != "" {
				return nil, fmt.Errorf("上传失败: %s", jsonErr.Data.Error)
			}
		}
		return nil, fmt.Errorf("上传失败 (HTTP %d): %s", resp.StatusCode, bodyStr)
	}

	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w\n响应内容: %s", err, bodyStr)
	}

	return result.Data, nil
}

// ListPlugins 列出所有插件
func (c *PluginServiceClient) ListPlugins() ([]map[string]interface{}, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/plugins", c.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("X-User-Id", fmt.Sprintf("%d", c.userID))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	if resp.StatusCode != http.StatusOK {
		if strings.Contains(bodyStr, "no healthy upstream") || strings.Contains(bodyStr, "Service Unavailable") {
			return nil, fmt.Errorf("服务不可用: 插件服务未启动或健康检查失败 (HTTP %d)", resp.StatusCode)
		}
		return nil, fmt.Errorf("获取插件列表失败 (HTTP %d): %s", resp.StatusCode, bodyStr)
	}

	var result struct {
		Data struct {
			Plugins []map[string]interface{} `json:"plugins"`
		} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w\n响应内容: %s", err, bodyStr)
	}

	return result.Data.Plugins, nil
}

// GetModels 获取插件支持的模型
func (c *PluginServiceClient) GetModels(pluginID, apiKey string) (map[string][]string, error) {
	reqBody := map[string]string{
		"api_key": apiKey,
	}
	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/plugins/%s/models", c.baseURL, pluginID), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", fmt.Sprintf("%d", c.userID))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	if resp.StatusCode != http.StatusOK {
		if strings.Contains(bodyStr, "no healthy upstream") || strings.Contains(bodyStr, "Service Unavailable") {
			return nil, fmt.Errorf("服务不可用: 插件服务未启动或健康检查失败 (HTTP %d)", resp.StatusCode)
		}
		return nil, fmt.Errorf("获取模型列表失败 (HTTP %d): %s", resp.StatusCode, bodyStr)
	}

	var result struct {
		Data struct {
			Models map[string][]string `json:"models"`
		} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w\n响应内容: %s", err, bodyStr)
	}

	return result.Data.Models, nil
}

// EnablePlugin 启用插件
func (c *PluginServiceClient) EnablePlugin(pluginID string) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/plugins/%s/enable", c.baseURL, pluginID), nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("X-User-Id", fmt.Sprintf("%d", c.userID))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	if resp.StatusCode != http.StatusOK {
		if strings.Contains(bodyStr, "no healthy upstream") || strings.Contains(bodyStr, "Service Unavailable") {
			return fmt.Errorf("服务不可用: 插件服务未启动或健康检查失败 (HTTP %d)", resp.StatusCode)
		}
		return fmt.Errorf("启用插件失败 (HTTP %d): %s", resp.StatusCode, bodyStr)
	}

	return nil
}

// DisablePlugin 禁用插件
func (c *PluginServiceClient) DisablePlugin(pluginID string) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/plugins/%s/disable", c.baseURL, pluginID), nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("X-User-Id", fmt.Sprintf("%d", c.userID))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	if resp.StatusCode != http.StatusOK {
		if strings.Contains(bodyStr, "no healthy upstream") || strings.Contains(bodyStr, "Service Unavailable") {
			return fmt.Errorf("服务不可用: 插件服务未启动或健康检查失败 (HTTP %d)", resp.StatusCode)
		}
		return fmt.Errorf("禁用插件失败 (HTTP %d): %s", resp.StatusCode, bodyStr)
	}

	return nil
}

// DeletePlugin 删除插件
func (c *PluginServiceClient) DeletePlugin(pluginID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/api/plugins/%s", c.baseURL, pluginID), nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("X-User-Id", fmt.Sprintf("%d", c.userID))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	if resp.StatusCode != http.StatusOK {
		if strings.Contains(bodyStr, "no healthy upstream") || strings.Contains(bodyStr, "Service Unavailable") {
			return fmt.Errorf("服务不可用: 插件服务未启动或健康检查失败 (HTTP %d)", resp.StatusCode)
		}
		return fmt.Errorf("删除插件失败 (HTTP %d): %s", resp.StatusCode, bodyStr)
	}

	return nil
}

// Embed 向量化文本
func (c *PluginServiceClient) Embed(pluginID, text string) ([]float32, int, error) {
	reqBody := map[string]string{
		"text": text,
	}
	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/plugins/%s/embed", c.baseURL, pluginID), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, 0, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	if resp.StatusCode != http.StatusOK {
		if strings.Contains(bodyStr, "no healthy upstream") || strings.Contains(bodyStr, "Service Unavailable") {
			return nil, 0, fmt.Errorf("服务不可用: 插件服务未启动或健康检查失败 (HTTP %d)", resp.StatusCode)
		}
		return nil, 0, fmt.Errorf("向量化失败 (HTTP %d): %s", resp.StatusCode, bodyStr)
	}

	var result struct {
		Data struct {
			Embedding  []float32 `json:"embedding"`
			Dimensions int       `json:"dimensions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, 0, fmt.Errorf("解析响应失败: %w\n响应内容: %s", err, bodyStr)
	}

	return result.Data.Embedding, result.Data.Dimensions, nil
}

// Rerank 重排序文档
func (c *PluginServiceClient) Rerank(pluginID, query string, documents []RerankDocument) ([]RerankResult, error) {
	reqBody := map[string]interface{}{
		"query":      query,
		"documents":  documents,
	}
	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/plugins/%s/rerank", c.baseURL, pluginID), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	if resp.StatusCode != http.StatusOK {
		if strings.Contains(bodyStr, "no healthy upstream") || strings.Contains(bodyStr, "Service Unavailable") {
			return nil, fmt.Errorf("服务不可用: 插件服务未启动或健康检查失败 (HTTP %d)", resp.StatusCode)
		}
		return nil, fmt.Errorf("重排序失败 (HTTP %d): %s", resp.StatusCode, bodyStr)
	}

	var result struct {
		Data struct {
			Results []RerankResult `json:"results"`
		} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w\n响应内容: %s", err, bodyStr)
	}

	return result.Data.Results, nil
}

// UpdatePluginConfig 更新插件配置
func (c *PluginServiceClient) UpdatePluginConfig(pluginID string, config map[string]interface{}) error {
	jsonData, _ := json.Marshal(config)

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/api/plugins/%s/config", c.baseURL, pluginID), bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", fmt.Sprintf("%d", c.userID))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	if resp.StatusCode != http.StatusOK {
		if strings.Contains(bodyStr, "no healthy upstream") || strings.Contains(bodyStr, "Service Unavailable") {
			return fmt.Errorf("服务不可用: 插件服务未启动或健康检查失败 (HTTP %d)", resp.StatusCode)
		}
		return fmt.Errorf("更新配置失败 (HTTP %d): %s", resp.StatusCode, bodyStr)
	}

	return nil
}

// GetPluginConfig 获取插件配置
func (c *PluginServiceClient) GetPluginConfig(pluginID string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/plugins/%s/config", c.baseURL, pluginID), nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("X-User-Id", fmt.Sprintf("%d", c.userID))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	if resp.StatusCode != http.StatusOK {
		if strings.Contains(bodyStr, "no healthy upstream") || strings.Contains(bodyStr, "Service Unavailable") {
			return nil, fmt.Errorf("服务不可用: 插件服务未启动或健康检查失败 (HTTP %d)", resp.StatusCode)
		}
		return nil, fmt.Errorf("获取配置失败 (HTTP %d): %s", resp.StatusCode, bodyStr)
	}

	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w\n响应内容: %s", err, bodyStr)
	}

	return result.Data, nil
}

