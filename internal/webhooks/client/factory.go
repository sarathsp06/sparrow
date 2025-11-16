package client

import (
	"context"
	"sync"
)

// Factory manages webhook client instances with different configurations
type Factory struct {
	clients map[string]*WebhookClient
	mu      sync.RWMutex
	config  *Config
}

// NewFactory creates a new webhook client factory
func NewFactory(config *Config) *Factory {
	if config == nil {
		config = DefaultConfig()
	}

	return &Factory{
		clients: make(map[string]*WebhookClient),
		config:  config,
	}
}

// GetClient returns a client for the given configuration key
// Creates a new client if one doesn't exist
func (f *Factory) GetClient(configKey string) *WebhookClient {
	f.mu.RLock()
	if client, exists := f.clients[configKey]; exists {
		f.mu.RUnlock()
		return client
	}
	f.mu.RUnlock()

	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := f.clients[configKey]; exists {
		return client
	}

	// Create new client
	client := NewWebhookClient(f.config)
	f.clients[configKey] = client

	return client
}

// GetDefaultClient returns the default webhook client
func (f *Factory) GetDefaultClient() *WebhookClient {
	return f.GetClient("default")
}

// PrewarmClients pre-establishes connections for all clients
func (f *Factory) PrewarmClients(ctx context.Context, hosts []string) error {
	f.mu.RLock()
	clients := make([]*WebhookClient, 0, len(f.clients))
	for _, client := range f.clients {
		clients = append(clients, client)
	}
	f.mu.RUnlock()

	for _, client := range clients {
		if err := client.PrewarmConnections(ctx, hosts); err != nil {
			return err
		}
	}

	return nil
}

// Close shuts down all clients
func (f *Factory) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, client := range f.clients {
		if err := client.Close(); err != nil {
			return err
		}
	}

	f.clients = make(map[string]*WebhookClient)
	return nil
}

// GetStats returns statistics for all clients
func (f *Factory) GetStats() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	stats := make(map[string]interface{})
	for key, client := range f.clients {
		stats[key] = client.GetStats()
	}

	return stats
}
