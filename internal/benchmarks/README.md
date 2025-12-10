# Benchmarking Suite

This directory contains comprehensive benchmarking tools for estimating RAM, bandwidth, CPU, and other resource requirements for the webhook delivery system.

## Quick Start

### Run Benchmarks

```bash
# Run all benchmarks (in short mode to save time)
go test -bench=. -benchmem -short ./internal/benchmarks/

# Run specific benchmark
go test -bench=BenchmarkWebhookDelivery_MediumPayload -benchmem ./internal/benchmarks/

# Run with detailed output
go test -bench=. -benchmem -v ./internal/benchmarks/
```

### Run Load Test Tool

```bash
# Build
go build -o bin/benchmark ./cmd/benchmark/

# Run with defaults
./bin/benchmark

# Custom test
./bin/benchmark -duration=2m -rps=500 -payload=20 -concurrency=25
```

## Available Benchmarks

### Webhook Delivery
- `BenchmarkWebhookDelivery_SmallPayload` - 1KB payloads
- `BenchmarkWebhookDelivery_MediumPayload` - 10KB payloads
- `BenchmarkWebhookDelivery_LargePayload` - 100KB payloads
- `BenchmarkWebhookDelivery_Concurrent` - 10 concurrent workers
- `BenchmarkWebhookDelivery_HighConcurrency` - 100 concurrent workers

### Template Processing
- `BenchmarkTemplateTransformation_Small` - 1KB with templates
- `BenchmarkTemplateTransformation_Large` - 100KB with templates

### Cache Performance
- `BenchmarkCachePerformance_Hits` - Template cache efficiency

### Memory Profiling
- `BenchmarkMemoryProfile_SustainedLoad` - 10-second sustained load (skipped in short mode)

## Interpreting Results

### Example Output
```
BenchmarkWebhookDelivery_SmallPayload-12    100    45000 ns/op    4.08 KB-transferred/op    43.98 KB/op    7 goroutines
```

- `100` - number of iterations
- `45000 ns/op` - 45µs per operation
- `4.08 KB-transferred/op` - bandwidth per request
- `43.98 KB/op` - memory allocated per operation
- `7 goroutines` - active goroutines

## Load Test Tool

Simulates realistic production workloads and generates resource estimates.

### Flags
```bash
-duration    Test duration (default: 1m)
-rps         Target requests per second (default: 100)
-payload     Payload size in KB (default: 10)
-concurrency Worker count (default: 10)
-templates   Enable templates (default: false)
```

### Example Output
```
=== Resource Estimates for Production ===

Low Load (100 RPS):
  Estimated RAM: 150.00 MB (0.15 GB)
  Estimated Bandwidth: 2.50 MB/s (20.00 Mbps)
  Estimated Goroutines: 50
  Recommended CPU Cores: 1
```

## Documentation

See [docs/BENCHMARKING.md](../../docs/BENCHMARKING.md) for comprehensive documentation including:
- Profiling techniques
- Production deployment guidelines
- Troubleshooting tips
- Continuous benchmarking setup

## Notes

- Run benchmarks on dedicated hardware for consistent results
- Use `-short` flag for quick checks during development
- For production estimates, run longer tests (5-10 minutes)
- Results include 50% safety margin for production estimates
