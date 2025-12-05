package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aihub/backend-go/internal/config"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

// ServiceInfo represents service registration information
type ServiceInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Address     string            `json:"address"`
	Port        int               `json:"port"`
	HealthCheck string            `json:"health_check"`
	Tags        []string          `json:"tags"`
	Meta        map[string]string `json:"meta"`
}

// ServiceRegistry handles service registration with etcd
type ServiceRegistry struct {
	client      *Client
	serviceID   string
	serviceName string
	serviceKey  string
	logger      *zap.Logger
	leaseID     clientv3.LeaseID
	keepAlive   <-chan *clientv3.LeaseKeepAliveResponse
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(client *Client, serviceID, serviceName string, logger *zap.Logger) *ServiceRegistry {
	// Service key format: /services/{serviceName}/instances/{serviceID}
	serviceKey := fmt.Sprintf("/services/%s/instances/%s", serviceName, serviceID)

	return &ServiceRegistry{
		client:      client,
		serviceID:    serviceID,
		serviceName:  serviceName,
		serviceKey:   serviceKey,
		logger:      logger,
	}
}

// Register registers the service with etcd
func (sr *ServiceRegistry) Register(cfg *config.Config) error {
	if !sr.client.IsEnabled() {
		sr.logger.Info("etcd is not enabled, skipping service registration")
		return nil
	}

	// Get hostname or use localhost
	hostname := os.Getenv("SERVICE_HOST")
	if hostname == "" {
		hostname = "localhost"
	}

	// Parse port
	port := 8000
	if cfg.Server.Port != "" {
		if p, err := parseInt(cfg.Server.Port); err == nil {
			port = p
		}
	}

	// Build health check URL
	healthCheckURL := fmt.Sprintf("http://%s:%d/health", hostname, port)

	// Create service info
	serviceInfo := ServiceInfo{
		ID:          sr.serviceID,
		Name:        sr.serviceName,
		Address:     hostname,
		Port:        port,
		HealthCheck: healthCheckURL,
		Tags:        []string{"api", "go", "beego", cfg.Server.Env},
		Meta: map[string]string{
			"version": "1.0.0",
			"env":     cfg.Server.Env,
		},
	}

	// Marshal service info to JSON
	serviceData, err := json.Marshal(serviceInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal service info: %w", err)
	}

	// Create lease for TTL (30 seconds)
	ctx := context.Background()
	lease, err := sr.client.GetClient().Grant(ctx, 30)
	if err != nil {
		return fmt.Errorf("failed to create lease: %w", err)
	}

	sr.leaseID = lease.ID

	// Put service info with lease
	_, err = sr.client.GetClient().Put(ctx, sr.serviceKey, string(serviceData), clientv3.WithLease(lease.ID))
	if err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	// Keep lease alive
	keepAlive, err := sr.client.GetClient().KeepAlive(ctx, lease.ID)
	if err != nil {
		return fmt.Errorf("failed to keep lease alive: %w", err)
	}

	sr.keepAlive = keepAlive

	// Start goroutine to handle keep-alive responses
	go func() {
		for ka := range keepAlive {
			if ka != nil {
				sr.logger.Debug("Service lease kept alive",
					zap.String("service_id", sr.serviceID),
					zap.Int64("lease_id", int64(ka.ID)),
				)
			}
		}
	}()

	sr.logger.Info("Service registered with etcd",
		zap.String("service_id", sr.serviceID),
		zap.String("service_name", sr.serviceName),
		zap.String("address", hostname),
		zap.Int("port", port),
		zap.String("key", sr.serviceKey),
	)

	// Register signal handler for graceful shutdown
	sr.registerShutdownHandler()

	return nil
}

// Deregister deregisters the service from etcd
func (sr *ServiceRegistry) Deregister() error {
	if !sr.client.IsEnabled() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Revoke lease (this will automatically delete the key)
	if sr.leaseID != 0 {
		_, err := sr.client.GetClient().Revoke(ctx, sr.leaseID)
		if err != nil {
			return fmt.Errorf("failed to revoke lease: %w", err)
		}
	} else {
		// Fallback: delete key directly
		_, err := sr.client.GetClient().Delete(ctx, sr.serviceKey)
		if err != nil {
			return fmt.Errorf("failed to delete service key: %w", err)
		}
	}

	sr.logger.Info("Service deregistered from etcd",
		zap.String("service_id", sr.serviceID),
		zap.String("key", sr.serviceKey),
	)

	return nil
}

// registerShutdownHandler registers a signal handler to deregister on shutdown
func (sr *ServiceRegistry) registerShutdownHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		sr.logger.Info("Received shutdown signal, deregistering from etcd",
			zap.String("signal", sig.String()),
		)

		if err := sr.Deregister(); err != nil {
			sr.logger.Error("Failed to deregister from etcd", zap.Error(err))
		} else {
			sr.logger.Info("Successfully deregistered from etcd")
		}

		os.Exit(0)
	}()
}

// parseInt safely parses an integer string
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

