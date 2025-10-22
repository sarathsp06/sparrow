#!/bin/bash

# sparrow Connect-RPC HTTP/JSON API Test Script
# This script demonstrates the Connect-RPC HTTP/JSON API
# Make sure the server is running: go run main.go

echo "🌐 sparrow Connect-RPC HTTP/JSON API Test"
echo "============================================="

BASE_URL="http://localhost:8080"

# Test 1: Health Check
echo -e "\n1. Testing health check..."
curl -s -w "\nStatus: %{http_code}\n" \
  "$BASE_URL/health" | jq 2>/dev/null || echo "Health check response received"

# Test 2: Register Webhook
echo -e "\n2. Registering webhook..."
WEBHOOK_RESPONSE=$(curl -s \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "test-app",
    "events": ["user.created", "user.updated"],
    "url": "https://webhook.site/test-endpoint",
    "headers": {
      "Authorization": "Bearer test-token",
      "Content-Type": "application/json"
    },
    "timeout": 30,
    "description": "Test webhook for user events"
  }' \
  "$BASE_URL/webhook.WebhookService/RegisterWebhook")

echo "Response:"
echo "$WEBHOOK_RESPONSE" | jq 2>/dev/null || echo "$WEBHOOK_RESPONSE"

# Extract webhook ID for next tests
WEBHOOK_ID=$(echo "$WEBHOOK_RESPONSE" | jq -r '.webhookId // .webhook_id // empty' 2>/dev/null)
if [ -z "$WEBHOOK_ID" ]; then
  echo "⚠️  Could not extract webhook ID, using placeholder"
  WEBHOOK_ID="placeholder-webhook-id"
fi

# Test 3: Push Event
echo -e "\n3. Pushing event..."
EVENT_RESPONSE=$(curl -s \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "test-app",
    "event": "user.created",
    "payload": "{\"user_id\":\"12345\",\"email\":\"user@example.com\",\"name\":\"John Doe\",\"action\":\"created\"}",
    "ttlSeconds": 3600,
    "metadata": {
      "source": "user-service",
      "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
    }
  }' \
  "$BASE_URL/webhook.WebhookService/PushEvent")

echo "Response:"
echo "$EVENT_RESPONSE" | jq 2>/dev/null || echo "$EVENT_RESPONSE"

# Extract event ID
EVENT_ID=$(echo "$EVENT_RESPONSE" | jq -r '.eventId // .event_id // empty' 2>/dev/null)

# Test 4: Get Webhook Status
echo -e "\n4. Getting webhook status..."
curl -s \
  -H "Content-Type: application/json" \
  -d '{
    "webhookId": "'$WEBHOOK_ID'"
  }' \
  "$BASE_URL/webhook.WebhookService/GetWebhookStatus" | \
  jq 2>/dev/null || echo "Webhook status response received"

# Test 5: List Webhooks
echo -e "\n5. Listing webhooks..."
curl -s \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "test-app",
    "activeOnly": true
  }' \
  "$BASE_URL/webhook.WebhookService/ListWebhooks" | \
  jq 2>/dev/null || echo "Webhook list response received"

echo -e "\n🎉 Connect-RPC HTTP/JSON API test completed!"
echo -e "\nTo test manually with curl:"
echo "curl -H 'Content-Type: application/json' \\"
echo "  -d '{\"namespace\":\"test\",\"events\":[\"test.event\"],\"url\":\"https://example.com/webhook\"}' \\"
echo "  http://localhost:8080/webhook.WebhookService/RegisterWebhook"

echo -e "\n📋 Key Features:"
echo "- HTTP/JSON API alongside gRPC"
echo "- Compatible with web clients" 
echo "- Standard HTTP status codes"
echo "- JSON request/response format"
echo "- Health check endpoint"