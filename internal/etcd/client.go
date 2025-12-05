package etcd

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

// Client wraps the etcd client
type Client struct {
	client  *clientv3.Client
	enabled bool
	logger  *zap.Logger
}

// NewClient creates a new etcd client
func NewClient(endpoints []string, enabled bool, logger *zap.Logger) (*Client, error) {
	if !enabled {
		return &Client{enabled: false, logger: logger}, nil
	}

	if len(endpoints) == 0 {
		endpoints = []string{"http://localhost:2379"}
	}

	config := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	}

	client, err := clientv3.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err = client.Status(ctx, endpoints[0])
	if err != nil {
		logger.Warn("etcd connection test failed, will use fallback config", zap.Error(err))
		return &Client{enabled: false, logger: logger}, nil
	}

	logger.Info("etcd client initialized", zap.Strings("endpoints", endpoints))
	return &Client{
		client:  client,
		enabled: true,
		logger:  logger,
	}, nil
}

// IsEnabled returns whether etcd is enabled
func (c *Client) IsEnabled() bool {
	return c.enabled && c.client != nil
}

// GetClient returns the underlying etcd client
func (c *Client) GetClient() *clientv3.Client {
	return c.client
}

// Put stores a key-value pair in etcd
func (c *Client) Put(ctx context.Context, key, value string) error {
	if !c.IsEnabled() {
		return fmt.Errorf("etcd is not enabled")
	}

	_, err := c.client.Put(ctx, key, value)
	return err
}

// Get retrieves a value from etcd
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if !c.IsEnabled() {
		return "", fmt.Errorf("etcd is not enabled")
	}

	resp, err := c.client.Get(ctx, key)
	if err != nil {
		return "", err
	}

	if len(resp.Kvs) == 0 {
		return "", fmt.Errorf("key %s not found", key)
	}

	return string(resp.Kvs[0].Value), nil
}

// GetWithPrefix retrieves all keys with a prefix
func (c *Client) GetWithPrefix(ctx context.Context, prefix string) (map[string]string, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("etcd is not enabled")
	}

	resp, err := c.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, kv := range resp.Kvs {
		result[string(kv.Key)] = string(kv.Value)
	}

	return result, nil
}

// Delete deletes a key from etcd
func (c *Client) Delete(ctx context.Context, key string) error {
	if !c.IsEnabled() {
		return fmt.Errorf("etcd is not enabled")
	}

	_, err := c.client.Delete(ctx, key)
	return err
}

// DeleteWithPrefix deletes all keys with a prefix
func (c *Client) DeleteWithPrefix(ctx context.Context, prefix string) error {
	if !c.IsEnabled() {
		return fmt.Errorf("etcd is not enabled")
	}

	_, err := c.client.Delete(ctx, prefix, clientv3.WithPrefix())
	return err
}

// Watch watches a key for changes
func (c *Client) Watch(ctx context.Context, key string, callback func(string, string) error) error {
	if !c.IsEnabled() {
		return fmt.Errorf("etcd is not enabled")
	}

	watchChan := c.client.Watch(ctx, key)
	for watchResp := range watchChan {
		for _, event := range watchResp.Events {
			if event.Type == clientv3.EventTypePut {
				if err := callback(string(event.Kv.Key), string(event.Kv.Value)); err != nil {
					c.logger.Error("Error in etcd watch callback",
						zap.String("key", string(event.Kv.Key)),
						zap.Error(err),
					)
				}
			}
		}
	}

	return nil
}

// Close closes the etcd client
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}


