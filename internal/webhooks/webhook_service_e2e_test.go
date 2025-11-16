package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/sarathsp06/sparrow/internal/logger"
	"github.com/sarathsp06/sparrow/internal/migration"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	storePostgres "github.com/sarathsp06/sparrow/pkg/storage/postgres"
	"github.com/sarathsp06/sparrow/pkg/types"
)

// TestWebhookRequest captures webhook request data
type TestWebhookRequest struct {
	Headers map[string][]string
	Body    []byte
	URL     string
	Method  string
}

// E2ETestSuite holds the test infrastructure
type E2ETestSuite struct {
	ctx          context.Context
	container    *postgres.PostgresContainer
	dsn          string
	dbPool       *pgxpool.Pool
	webhookRepo  store.RepositoryInterface
	queueManager *queue.Manager
	service      *WebhookService
}

// setupE2ETestSuite creates and initializes the test suite
func setupE2ETestSuite(t *testing.T) *E2ETestSuite {
	ctx := context.Background()
	dbName := "riverqueue"
	dbUser := "riveruser"
	dbPassword := "riverpass"

	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err, "failed to start postgres container")

	dsn, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	// Run migrations using the migration package
	log := logger.NewLogger("test-migration")
	err = migration.RunAllMigrations(ctx, dsn, "up", 0, 0, log)
	require.NoError(t, err, "failed to run migrations")

	// Create database pool
	dbPool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err, "failed to create database pool")

	err = dbPool.Ping(ctx)
	require.NoError(t, err, "failed to ping database")

	// Create repository
	sqlxDB, err := storePostgres.Open(dsn, 3)
	require.NoError(t, err, "failed to create sqlx connection")

	webhookRepo := store.NewRepository(sqlxDB)

	// Create queue manager
	queueManager, err := queue.NewManager(ctx, webhookRepo, dbPool)
	require.NoError(t, err, "failed to create queue manager")

	// Start queue manager in background
	go func() {
		_ = queueManager.Start(ctx)
	}()

	// Create webhook service
	service := NewWebhookService(queueManager.GetJobInserter(), webhookRepo)

	return &E2ETestSuite{
		ctx:          ctx,
		container:    postgresContainer,
		dsn:          dsn,
		dbPool:       dbPool,
		webhookRepo:  webhookRepo,
		queueManager: queueManager,
		service:      service,
	}
}

// teardown cleans up the test suite
func (suite *E2ETestSuite) teardown(t *testing.T) {
	if suite.queueManager != nil {
		_ = suite.queueManager.Stop(suite.ctx)
	}
	if suite.dbPool != nil {
		suite.dbPool.Close()
	}
	if suite.container != nil {
		err := testcontainers.TerminateContainer(suite.container)
		if err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}
}

// createTestWebhookServer creates a test HTTP server to capture webhook requests
func createTestWebhookServer(t *testing.T) (*httptest.Server, <-chan TestWebhookRequest) {
	requestChan := make(chan TestWebhookRequest, 10)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Capture the request
		request := TestWebhookRequest{
			Headers: make(map[string][]string),
			Body:    body,
			URL:     r.URL.String(),
			Method:  r.Method,
		}

		// Copy headers
		for name, values := range r.Header {
			request.Headers[name] = values
		}

		// Send to channel (non-blocking)
		select {
		case requestChan <- request:
		default:
			t.Logf("request channel is full, dropping request")
		}

		// Respond with success
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	return server, requestChan
}

// TestWebhookDeliveryWithHMAC tests webhook delivery with HMAC signature generation
func TestWebhookDeliveryWithHMAC(t *testing.T) {
	suite := setupE2ETestSuite(t)
	defer suite.teardown(t)

	// Create test webhook server
	server, requestChan := createTestWebhookServer(t)
	defer server.Close()

	// Create webhook registration with secret
	webhook := &WebhookRegistration{
		ID:        "test-hmac-webhook",
		Namespace: "test",
		Events:    StringArray{"test.event"},
		URL:       server.URL,
		HTTPConfig: WebhookHTTPConfig{
			MaxRetries:            3,
			RetryBackoffSeconds:   60,
			CaptureResponseBody:   false,
			FollowRedirects:       true,
			VerifySSL:             true,
			RequestTimeoutSeconds: 30,
			ExpectedStatusCodes:   IntArray{200, 201, 202, 204},
			WebhookSecret:         "test-secret-key-123",
			UserAgent:             "TestAgent/1.0",
			ContentType:           "application/json",
		},
	}

	// Create delivery client and request
	deliveryClient := NewDeliveryClient(DefaultClientConfig())

	testPayload := map[string]interface{}{
		"event_type": "test.event",
		"timestamp":  time.Now().Unix(),
		"message":    "Hello, webhook with HMAC!",
		"data": map[string]interface{}{
			"id":   123,
			"name": "test item",
		},
	}

	deliveryRequest := &WebhookDeliveryRequest{
		Webhook:     webhook,
		EventID:     "evt-12345",
		DeliveryID:  "del-67890",
		Payload:     testPayload,
		AttemptNum:  1,
		MaxAttempts: 3,
	}

	t.Run("HMAC signature generation and delivery", func(t *testing.T) {
		// Deliver the webhook
		response := deliveryClient.DeliverWebhook(suite.ctx, deliveryRequest)

		// Verify delivery was successful
		assert.True(t, response.Success, "webhook delivery should be successful")
		assert.Equal(t, 200, response.StatusCode, "should receive 200 status code")
		assert.Empty(t, response.Error, "should not have delivery error")

		// Wait for webhook request
		var webhookRequest TestWebhookRequest
		select {
		case webhookRequest = <-requestChan:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for webhook request")
		}

		// Verify HMAC signature was sent
		signatureHeaders := webhookRequest.Headers["X-Webhook-Signature-256"]
		require.Len(t, signatureHeaders, 1, "should have exactly one signature header")
		signature := signatureHeaders[0]
		assert.NotEmpty(t, signature, "HMAC signature should be present")

		// Verify other webhook headers
		assert.Equal(t, []string{"evt-12345"}, webhookRequest.Headers["X-Webhook-Event-Id"])
		assert.Equal(t, []string{"del-67890"}, webhookRequest.Headers["X-Webhook-Delivery-Id"])
		assert.Equal(t, []string{"1"}, webhookRequest.Headers["X-Webhook-Attempt"])
		assert.Equal(t, []string{"TestAgent/1.0"}, webhookRequest.Headers["User-Agent"])
		assert.Equal(t, []string{"application/json"}, webhookRequest.Headers["Content-Type"])

		// Manually verify HMAC signature
		expectedPayload, err := json.Marshal(testPayload)
		require.NoError(t, err, "should marshal test payload")

		expectedSignature := generateHMAC(webhook.HTTPConfig.WebhookSecret, expectedPayload)
		assert.Equal(t, expectedSignature, signature, "HMAC signature should match expected value")

		// Verify request body - be flexible about JSON number types
		var receivedPayload map[string]interface{}
		err = json.Unmarshal(webhookRequest.Body, &receivedPayload)
		require.NoError(t, err, "should unmarshal webhook body")

		// Check specific fields instead of exact match due to JSON number type differences
		assert.Equal(t, "test.event", receivedPayload["event_type"])
		assert.Equal(t, "Hello, webhook with HMAC!", receivedPayload["message"])
		assert.NotNil(t, receivedPayload["timestamp"])

		if data, ok := receivedPayload["data"].(map[string]interface{}); ok {
			assert.Equal(t, float64(123), data["id"]) // JSON numbers are float64
			assert.Equal(t, "test item", data["name"])
		} else {
			t.Fatalf("data field should be a map")
		}
	})
}

// TestWebhookServiceRegistration tests webhook registration with HTTP configuration
func TestWebhookServiceRegistration(t *testing.T) {
	suite := setupE2ETestSuite(t)
	defer suite.teardown(t)

	// Create test webhook server
	server, _ := createTestWebhookServer(t)
	defer server.Close()

	t.Run("webhook registration with HTTP config", func(t *testing.T) {
		// Test webhook registration request with secret
		webhookRequest := WebhookRegistrationRequest{
			Namespace:   "test-service",
			Events:      []string{"user.created"},
			URL:         server.URL,
			Description: "Test service webhook",
			HTTPConfig: &WebhookHTTPConfig{
				WebhookSecret:         "service-test-secret-456",
				MaxRetries:            3,
				RetryBackoffSeconds:   120,
				CaptureResponseBody:   true,
				FollowRedirects:       false,
				VerifySSL:             true,
				RequestTimeoutSeconds: 45,
				ExpectedStatusCodes:   IntArray{200, 202},
				UserAgent:             "ServiceAgent/1.0",
				ContentType:           "application/json",
			},
		}

		// Convert to webhook registration
		webhook, err := webhookRequest.ToWebhookRegistration()
		require.NoError(t, err, "should convert request to registration")

		// Verify HTTP config was applied correctly
		assert.Equal(t, "service-test-secret-456", webhook.HTTPConfig.WebhookSecret)
		assert.Equal(t, 3, webhook.HTTPConfig.MaxRetries)
		assert.Equal(t, 120, webhook.HTTPConfig.RetryBackoffSeconds)
		assert.True(t, webhook.HTTPConfig.CaptureResponseBody)
		assert.False(t, webhook.HTTPConfig.FollowRedirects)
		assert.True(t, webhook.HTTPConfig.VerifySSL)
		assert.Equal(t, 45, webhook.HTTPConfig.RequestTimeoutSeconds)
		assert.Equal(t, IntArray{200, 202}, webhook.HTTPConfig.ExpectedStatusCodes)
		assert.Equal(t, "ServiceAgent/1.0", webhook.HTTPConfig.UserAgent)
		assert.Equal(t, "application/json", webhook.HTTPConfig.ContentType)
	})

	t.Run("webhook registration with defaults", func(t *testing.T) {
		// Test webhook registration with minimal config
		webhookRequest := WebhookRegistrationRequest{
			Namespace:   "test-minimal",
			Events:      []string{"order.created"},
			URL:         server.URL,
			Description: "Minimal webhook",
			HTTPConfig:  nil, // nil config should use defaults
		}

		// Convert to webhook registration
		webhook, err := webhookRequest.ToWebhookRegistration()
		require.NoError(t, err, "should convert request to registration with defaults")

		// Verify defaults were applied
		assert.Equal(t, 3, webhook.HTTPConfig.MaxRetries)
		assert.Equal(t, 60, webhook.HTTPConfig.RetryBackoffSeconds)
		assert.False(t, webhook.HTTPConfig.CaptureResponseBody)
		assert.True(t, webhook.HTTPConfig.FollowRedirects)
		assert.True(t, webhook.HTTPConfig.VerifySSL)
		assert.Equal(t, 30, webhook.HTTPConfig.RequestTimeoutSeconds)
		assert.Equal(t, IntArray{200, 201, 202, 204}, webhook.HTTPConfig.ExpectedStatusCodes)
		assert.Equal(t, "Sparrow-Webhook/1.0", webhook.HTTPConfig.UserAgent)
		assert.Equal(t, "application/json", webhook.HTTPConfig.ContentType)
	})
}

// TestWebhookSecretStorage tests that webhook secrets are properly stored in database
func TestWebhookSecretStorage(t *testing.T) {
	suite := setupE2ETestSuite(t)
	defer suite.teardown(t)

	// Create test webhook server
	server, _ := createTestWebhookServer(t)
	defer server.Close()

	t.Run("webhook secret database storage", func(t *testing.T) {
		namespace := "test-secret-storage"
		eventName := "test.event"
		webhookSecret := "super-secret-key-789"

		// Register an event first
		_, _, err := suite.service.RegisterEvent(suite.ctx, eventName, "Test event", nil, nil, true)
		require.NoError(t, err, "should register event")

		// Use the service method to register webhook with HTTP config
		webhookID, _, err := suite.service.RegisterWebhookWithHTTPConfig(
			suite.ctx,
			namespace,
			[]string{eventName},
			server.URL,
			nil,  // headers
			30,   // timeout
			true, // active
			"Test webhook with secret",
			&WebhookHTTPConfig{
				WebhookSecret: webhookSecret,
				MaxRetries:    3,
			},
		)
		require.NoError(t, err, "should register webhook")
		assert.NotEmpty(t, webhookID, "webhook ID should not be empty")

		// For this test, we'll just verify the secret was stored correctly in the database
		// by checking the webhook configuration
		stored, err := suite.webhookRepo.GetWebhookByID(suite.ctx, webhookID, namespace)
		require.NoError(t, err, "should retrieve stored webhook")
		assert.NotNil(t, stored, "webhook should exist")
		assert.Equal(t, webhookSecret, stored.WebhookSecret, "secret should match")

		// Verify that we can generate HMAC with the stored secret
		testPayload := []byte(`{"test": "data"}`)
		expectedHMAC := generateHMAC(webhookSecret, testPayload)
		actualHMAC := generateHMAC(stored.WebhookSecret, testPayload)
		assert.Equal(t, expectedHMAC, actualHMAC, "HMAC generation should work with stored secret")

		// If we got here, the secret storage and retrieval is working correctly
		t.Logf("Webhook secret storage test passed - secret stored and retrieved correctly")
	})
}

// TestEndToEndWebhookDelivery is a simplified integration test
func TestEndToEndWebhookDelivery(t *testing.T) {
	suite := setupE2ETestSuite(t)
	defer suite.teardown(t)

	t.Run("webhook registration and service integration", func(t *testing.T) {
		namespace := "test-e2e"
		eventName := "user.registered"
		webhookSecret := "e2e-test-secret"

		// Register an event
		eventID, _, err := suite.service.RegisterEvent(suite.ctx, eventName, "User registration event", nil, nil, true)
		require.NoError(t, err, "should register event")
		assert.NotEmpty(t, eventID, "event ID should not be empty")

		// Test webhook registration with HTTP config (but don't wait for delivery)
		webhookID, _, err := suite.service.RegisterWebhookWithHTTPConfig(
			suite.ctx,
			namespace,
			[]string{eventName},
			"http://test-endpoint.example.com",
			nil,  // headers
			30,   // timeout
			true, // active
			"E2E test webhook",
			&WebhookHTTPConfig{
				WebhookSecret:       webhookSecret,
				MaxRetries:          2,
				RetryBackoffSeconds: 30,
				UserAgent:           "E2ETestAgent/1.0",
				ContentType:         "application/json",
			},
		)
		require.NoError(t, err, "should register webhook")
		assert.NotEmpty(t, webhookID, "webhook ID should not be empty")

		// Verify webhook was stored correctly by retrieving it
		webhooks, err := suite.service.GetRegisteredWebhooks(suite.ctx, namespace, webhookID, false)
		require.NoError(t, err, "should retrieve webhook")
		assert.Len(t, webhooks, 1, "should have exactly one webhook")
		assert.Equal(t, webhookID, webhooks[0].ID)
		assert.Equal(t, namespace, webhooks[0].Namespace)
		assert.Contains(t, webhooks[0].Events, eventName)
	})
}

// TestWebhookConfigurationValidation tests webhook HTTP configuration validation
func TestWebhookConfigurationValidation(t *testing.T) {
	suite := setupE2ETestSuite(t)
	defer suite.teardown(t)

	t.Run("webhook config validation", func(t *testing.T) {
		testCases := []struct {
			name          string
			config        WebhookHTTPConfig
			expectedError bool
		}{
			{
				name: "valid config",
				config: WebhookHTTPConfig{
					MaxRetries:            5,
					RetryBackoffSeconds:   120,
					RequestTimeoutSeconds: 60,
					ExpectedStatusCodes:   IntArray{200, 201, 202},
					UserAgent:             "TestAgent/1.0",
					ContentType:           "application/json",
				},
				expectedError: false,
			},
			{
				name: "invalid max retries",
				config: WebhookHTTPConfig{
					MaxRetries: -1, // Invalid
				},
				expectedError: true,
			},
			{
				name: "invalid timeout",
				config: WebhookHTTPConfig{
					RequestTimeoutSeconds: -5, // Invalid
				},
				expectedError: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.config.ValidateConfig()
				if tc.expectedError {
					assert.Error(t, err, "should have validation error")
				} else {
					assert.NoError(t, err, "should pass validation")
				}
			})
		}
	})
}

// TestWebhookRetryConfiguration tests webhook retry behavior
func TestWebhookRetryConfiguration(t *testing.T) {
	suite := setupE2ETestSuite(t)
	defer suite.teardown(t)

	t.Run("webhook retry configuration", func(t *testing.T) {
		// Create a server that fails initially
		attemptCount := 0
		var requestChan = make(chan TestWebhookRequest, 10)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++

			body, _ := io.ReadAll(r.Body)
			request := TestWebhookRequest{
				Headers: make(map[string][]string),
				Body:    body,
			}
			for name, values := range r.Header {
				request.Headers[name] = values
			}
			requestChan <- request

			if attemptCount < 3 {
				// Fail the first 2 attempts
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server Error"))
			} else {
				// Succeed on the 3rd attempt
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			}
		}))
		defer server.Close()

		// Create webhook with retry configuration
		webhook := &WebhookRegistration{
			ID:        "test-retry-webhook",
			Namespace: "test",
			Events:    StringArray{"retry.test"},
			URL:       server.URL,
			HTTPConfig: WebhookHTTPConfig{
				MaxRetries:            3,
				RetryBackoffSeconds:   1, // Short backoff for testing
				RequestTimeoutSeconds: 10,
				ExpectedStatusCodes:   IntArray{200, 201, 202},
				WebhookSecret:         "retry-test-secret",
				UserAgent:             "RetryTestAgent/1.0",
				ContentType:           "application/json",
			},
		}

		deliveryClient := NewDeliveryClient(DefaultClientConfig())
		testPayload := map[string]interface{}{
			"message": "retry test",
			"attempt": 1,
		}

		// Test first delivery (should fail)
		deliveryRequest := &WebhookDeliveryRequest{
			Webhook:     webhook,
			EventID:     "retry-evt-123",
			DeliveryID:  "retry-del-123",
			Payload:     testPayload,
			AttemptNum:  1,
			MaxAttempts: 3,
		}

		response := deliveryClient.DeliverWebhook(suite.ctx, deliveryRequest)
		assert.False(t, response.Success, "first attempt should fail")
		assert.Equal(t, 500, response.StatusCode, "should receive 500 status code")
		assert.True(t, response.ShouldRetry, "should indicate retry is needed")

		// Test second delivery (should also fail)
		deliveryRequest.AttemptNum = 2
		response = deliveryClient.DeliverWebhook(suite.ctx, deliveryRequest)
		assert.False(t, response.Success, "second attempt should fail")
		assert.True(t, response.ShouldRetry, "should indicate retry is needed")

		// Test third delivery (should succeed)
		deliveryRequest.AttemptNum = 3
		response = deliveryClient.DeliverWebhook(suite.ctx, deliveryRequest)
		assert.True(t, response.Success, "third attempt should succeed")
		assert.Equal(t, 200, response.StatusCode, "should receive 200 status code")
		assert.False(t, response.ShouldRetry, "should not need retry after success")

		// Verify all attempts were made with correct signatures
		assert.Equal(t, 3, attemptCount, "should have made exactly 3 attempts")

		// Check that we received all 3 webhook requests
		assert.Len(t, requestChan, 3, "should have captured all 3 requests")

		// Verify signatures on all attempts
		for i := 0; i < 3; i++ {
			select {
			case request := <-requestChan:
				signature := request.Headers["X-Webhook-Signature-256"][0]
				expectedSignature := generateHMAC(webhook.HTTPConfig.WebhookSecret, request.Body)
				assert.Equal(t, expectedSignature, signature, fmt.Sprintf("signature should be valid on attempt %d", i+1))
			default:
				t.Fatalf("expected request %d not received", i+1)
			}
		}
	})
}

// Helper function to generate HMAC signature for testing
func generateHMAC(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// Add RegisterWebhookWithHTTPConfig method to service for testing
// This is a helper method that should probably be added to the actual service
func (s *WebhookService) RegisterWebhookWithHTTPConfig(
	ctx context.Context,
	namespace string,
	events []string,
	url string,
	headers map[string]string,
	timeout int,
	active bool,
	description string,
	httpConfig *WebhookHTTPConfig,
) (string, int64, error) {
	// Create webhook registration request
	req := WebhookRegistrationRequest{
		Namespace:   namespace,
		Events:      events,
		URL:         url,
		Description: description,
		HTTPConfig:  httpConfig,
	}

	// Convert to webhook registration
	webhook, err := req.ToWebhookRegistration()
	if err != nil {
		return "", 0, fmt.Errorf("failed to create webhook registration: %w", err)
	}

	// Set additional fields
	if webhook.Headers == nil {
		webhook.Headers = make(map[string]interface{})
	}
	for k, v := range headers {
		webhook.Headers[k] = v
	}
	webhook.Timeout = timeout
	webhook.Active = active

	// Generate ID if not set
	if webhook.ID == "" {
		webhook.ID = "wh-" + generateID()
	}

	// Set created/updated timestamps
	now := time.Now()
	if webhook.CreatedAt.IsZero() {
		webhook.CreatedAt = now
	}
	webhook.UpdatedAt = now

	// Convert to store webhook registration for database storage
	storeWebhook := &store.WebhookRegistration{
		ID:            webhook.ID,
		Namespace:     webhook.Namespace,
		Events:        pq.StringArray(webhook.Events),
		URL:           webhook.URL,
		Timeout:       webhook.Timeout,
		WebhookSecret: webhook.HTTPConfig.WebhookSecret,
		Active:        webhook.Active,
		Description:   webhook.Description,
		Health:        store.WebhookHealth(webhook.Health),
		CreatedAt:     webhook.CreatedAt,
		UpdatedAt:     webhook.UpdatedAt,
	}

	// Convert headers
	if webhook.Headers != nil {
		headerMap := make(map[string]string)
		for k, v := range webhook.Headers {
			if str, ok := v.(string); ok {
				headerMap[k] = str
			}
		}
		storeWebhook.Headers = types.Map[string, string](headerMap)
	}

	// Register in database
	if err := s.webhookRepo.RegisterWebhook(ctx, storeWebhook); err != nil {
		return "", 0, fmt.Errorf("failed to register webhook: %w", err)
	}

	return webhook.ID, storeWebhook.CreatedAt.Unix(), nil
}

// Helper function to generate IDs
func generateID() string {
	// Simple ID generator for testing
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
