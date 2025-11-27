package consul

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aihub/backend-go/internal/config"
	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

// ServiceRegistry handles service registration with Consul
type ServiceRegistry struct {
	client      *Client
	serviceID   string
	serviceName string
	logger      *zap.Logger
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(client *Client, serviceID, serviceName string, logger *zap.Logger) *ServiceRegistry {
	return &ServiceRegistry{
		client:      client,
		serviceID:   serviceID,
		serviceName: serviceName,
		logger:      logger,
	}
}

// Register registers the service with Consul
func (sr *ServiceRegistry) Register(cfg *config.Config) error {
	if !sr.client.IsEnabled() {
		sr.logger.Info("Consul is not enabled, skipping service registration")
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

	registration := &api.AgentServiceRegistration{
		ID:      sr.serviceID,
		Name:    sr.serviceName,
		Tags:    []string{"api", "go", "beego", cfg.Server.Env},
		Address: hostname,
		Port:    port,
		Check: &api.AgentServiceCheck{
			HTTP:                           healthCheckURL,
			Interval:                       "10s",
			Timeout:                        "3s",
			DeregisterCriticalServiceAfter: "30s",
		},
		Meta: map[string]string{
			"version": "1.0.0",
			"env":     cfg.Server.Env,
		},
	}

	if err := sr.client.RegisterService(registration); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	sr.logger.Info("Service registered with Consul",
		zap.String("service_id", sr.serviceID),
		zap.String("service_name", sr.serviceName),
		zap.String("address", hostname),
		zap.Int("port", port),
	)

	// Register signal handler for graceful shutdown
	sr.registerShutdownHandler()

	return nil
}

// Deregister deregisters the service from Consul
func (sr *ServiceRegistry) Deregister() error {
	if !sr.client.IsEnabled() {
		return nil
	}

	return sr.client.DeregisterService(sr.serviceID)
}

// registerShutdownHandler registers a signal handler to deregister on shutdown
func (sr *ServiceRegistry) registerShutdownHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		sr.logger.Info("Received shutdown signal, deregistering from Consul",
			zap.String("signal", sig.String()),
		)

		if err := sr.Deregister(); err != nil {
			sr.logger.Error("Failed to deregister from Consul", zap.Error(err))
		} else {
			sr.logger.Info("Successfully deregistered from Consul")
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

