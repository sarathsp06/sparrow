package client

import (
	"sync"
	"testing"
	"time"
)

func TestNewFactory(t *testing.T) {
	config := &Config{
		Timeout:         10 * time.Second,
		MaxIdleConns:    50,
		MaxConnsPerHost: 5,
	}

	factory := NewFactory(config)
	if factory == nil {
		return
	}

	if factory.config.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", factory.config.Timeout)
	}

	if factory.clients == nil {
		t.Error("Expected clients map to be initialized")
	}
}

func TestNewFactoryWithNilConfig(t *testing.T) {
	factory := NewFactory(nil)

	if factory == nil {
		return
	}

	if factory.config == nil {
		t.Fatal("Expected default config to be set")
	}

	if factory.config.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", factory.config.Timeout)
	}
}

func TestGetClient(t *testing.T) {
	factory := NewFactory(nil)

	client1 := factory.GetClient("test1")
	if client1 == nil {
		t.Fatal("Expected non-nil client")
	}

	// Getting same client should return the same instance
	client2 := factory.GetClient("test1")
	if client1 != client2 {
		t.Error("Expected same client instance for same key")
	}

	// Different key should create new client
	client3 := factory.GetClient("test2")
	if client1 == client3 {
		t.Error("Expected different client instance for different key")
	}
}

func TestGetDefaultClient(t *testing.T) {
	factory := NewFactory(nil)

	client1 := factory.GetDefaultClient()
	if client1 == nil {
		t.Fatal("Expected non-nil default client")
	}

	// Should return same instance
	client2 := factory.GetDefaultClient()
	if client1 != client2 {
		t.Error("Expected same default client instance")
	}

	// Should be same as GetClient("default")
	client3 := factory.GetClient("default")
	if client1 != client3 {
		t.Error("Expected default client to match GetClient('default')")
	}
}

func TestFactoryConcurrency(t *testing.T) {
	factory := NewFactory(nil)
	var wg sync.WaitGroup

	// Simulate concurrent client requests
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := "client-1" // All goroutines request same client
			client := factory.GetClient(key)
			if client == nil {
				t.Error("Expected non-nil client")
			}
		}(i)
	}

	wg.Wait()

	// Should only have one client created
	factory.mu.RLock()
	clientCount := len(factory.clients)
	factory.mu.RUnlock()

	if clientCount != 1 {
		t.Errorf("Expected 1 client, got %d", clientCount)
	}
}

func TestFactoryMultipleClients(t *testing.T) {
	factory := NewFactory(nil)

	keys := []string{"client1", "client2", "client3", "client4", "client5"}
	clients := make(map[string]*WebhookClient)

	for _, key := range keys {
		clients[key] = factory.GetClient(key)
	}

	factory.mu.RLock()
	clientCount := len(factory.clients)
	factory.mu.RUnlock()

	if clientCount != len(keys) {
		t.Errorf("Expected %d clients, got %d", len(keys), clientCount)
	}

	// Verify all clients are different
	for i, key1 := range keys {
		for j, key2 := range keys {
			if i != j && clients[key1] == clients[key2] {
				t.Errorf("Expected different clients for %s and %s", key1, key2)
			}
		}
	}
}

func TestClose(t *testing.T) {
	factory := NewFactory(nil)

	// Create some clients
	factory.GetClient("test1")
	factory.GetClient("test2")
	factory.GetClient("test3")

	factory.mu.RLock()
	initialCount := len(factory.clients)
	factory.mu.RUnlock()

	if initialCount != 3 {
		t.Errorf("Expected 3 clients before close, got %d", initialCount)
	}

	err := factory.Close()
	if err != nil {
		t.Errorf("Unexpected error closing factory: %v", err)
	}

	factory.mu.RLock()
	finalCount := len(factory.clients)
	factory.mu.RUnlock()

	if finalCount != 0 {
		t.Errorf("Expected 0 clients after close, got %d", finalCount)
	}
}

func TestFactoryGetStats(t *testing.T) {
	factory := NewFactory(nil)

	// Create some clients
	factory.GetClient("test1")
	factory.GetClient("test2")

	stats := factory.GetStats()

	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	if len(stats) != 2 {
		t.Errorf("Expected stats for 2 clients, got %d", len(stats))
	}

	if _, exists := stats["test1"]; !exists {
		t.Error("Expected stats for test1 client")
	}

	if _, exists := stats["test2"]; !exists {
		t.Error("Expected stats for test2 client")
	}
}

func TestFactoryGetStatsEmpty(t *testing.T) {
	factory := NewFactory(nil)
	stats := factory.GetStats()

	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	if len(stats) != 0 {
		t.Errorf("Expected empty stats, got %d entries", len(stats))
	}
}

func BenchmarkGetClient(b *testing.B) {
	factory := NewFactory(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		factory.GetClient("benchmark-client")
	}
}

func BenchmarkGetClientConcurrent(b *testing.B) {
	factory := NewFactory(nil)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			factory.GetClient("benchmark-client")
		}
	})
}
