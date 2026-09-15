# API Key Enforcement
Tags: auth, enforcement

When SPARROW_API_KEY is configured, /v1 endpoints require the X-API-Key header.
Health endpoints stay open.

## Request Without Key Is Rejected
* Start authed sparrow server
* GET "/v1/event-types" on authed server without key should return status "401"

## Request With Wrong Key Is Rejected
* Start authed sparrow server
* GET "/v1/event-types" on authed server with key "wrong-key" should return status "401"

## Request With Correct Key Is Accepted
* Start authed sparrow server
* GET "/v1/event-types" on authed server with the correct key should return status "200"

## Health Endpoints Stay Open Without Key
* Start authed sparrow server
* GET "/health" on authed server without key should return status "200"
* GET "/ready" on authed server without key should return status "200"
* Stop authed sparrow server
