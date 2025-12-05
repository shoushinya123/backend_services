package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/logger"
	"go.uber.org/zap"
)

var VaultClient *Client

// Client Vault客户端
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
	enabled    bool
	logger     *zap.Logger
}

// NewClient 创建Vault客户端
func NewClient() (*Client, error) {
	cfg := config.AppConfig
	if cfg == nil {
		return nil, fmt.Errorf("config not loaded")
	}

	vaultCfg := cfg.Vault
	if !vaultCfg.Enabled {
		logger.Info("Vault is not enabled")
		return &Client{enabled: false}, nil
	}

	baseURL := vaultCfg.Address
	if baseURL == "" {
		baseURL = "http://localhost:8200"
	}

	// 确保URL以/v1开头
	if baseURL[len(baseURL)-3:] != "/v1" {
		if baseURL[len(baseURL)-1] != '/' {
			baseURL += "/v1"
		} else {
			baseURL += "v1"
		}
	}

	token := vaultCfg.Token
	if token == "" {
		token = "root" // 开发环境默认token
	}

	client := &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: baseURL,
		token:   token,
		enabled: true,
		logger:  logger.Logger,
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.HealthCheck(ctx); err != nil {
		logger.Warn("Failed to connect to Vault, will use fallback storage", zap.Error(err))
		return &Client{enabled: false}, nil
	}

	VaultClient = client
	logger.Info("✅ Vault connected successfully", zap.String("address", baseURL))
	return client, nil
}

// IsEnabled 检查Vault是否启用
func (c *Client) IsEnabled() bool {
	return c != nil && c.enabled
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(ctx context.Context) error {
	if !c.IsEnabled() {
		return fmt.Errorf("Vault is not enabled")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/sys/health", nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Vault-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusTooManyRequests {
		return fmt.Errorf("Vault health check failed: %s", resp.Status)
	}

	return nil
}

// WriteSecret 写入密钥
func (c *Client) WriteSecret(ctx context.Context, path string, data map[string]interface{}) error {
	if !c.IsEnabled() {
		return fmt.Errorf("Vault is not enabled")
	}

	url := c.baseURL + "/secret/data/" + path

	payload := map[string]interface{}{
		"data": data,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Vault-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to write secret: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to write secret: %s, body: %s", resp.Status, string(body))
	}

	return nil
}

// ReadSecret 读取密钥
func (c *Client) ReadSecret(ctx context.Context, path string) (map[string]interface{}, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("Vault is not enabled")
	}

	url := c.baseURL + "/secret/data/" + path

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Vault-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("secret not found: %s", path)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to read secret: %s, body: %s", resp.Status, string(body))
	}

	var result struct {
		Data struct {
			Data map[string]interface{} `json:"data"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data.Data, nil
}

// DeleteSecret 删除密钥
func (c *Client) DeleteSecret(ctx context.Context, path string) error {
	if !c.IsEnabled() {
		return fmt.Errorf("Vault is not enabled")
	}

	url := c.baseURL + "/secret/metadata/" + path

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Vault-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete secret: %s, body: %s", resp.Status, string(body))
	}

	return nil
}

// ListSecrets 列出密钥
func (c *Client) ListSecrets(ctx context.Context, path string) ([]string, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("Vault is not enabled")
	}

	url := c.baseURL + "/secret/metadata/" + path + "?list=true"

	req, err := http.NewRequestWithContext(ctx, "LIST", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Vault-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return []string{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list secrets: %s, body: %s", resp.Status, string(body))
	}

	var result struct {
		Data struct {
			Keys []string `json:"keys"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data.Keys, nil
}

// GetClient 获取全局Vault客户端
func GetClient() *Client {
	return VaultClient
}










