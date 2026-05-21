# API Key Authentication
Tags: auth

Health endpoints are always open. API endpoints work without key when not configured.

## Health Endpoint Is Always Open
* GET "/health" should return status "200"

## Ready Endpoint Is Always Open
* GET "/ready" should return status "200"

## API Endpoints Work Without Key When Not Configured
* POST "/webhook.EventService/ListEvents" with empty body should return status "200"
