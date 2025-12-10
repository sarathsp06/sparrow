# Resource Benchmarking Guide

This guide explains how to benchmark and estimate resource requirements for the webhook delivery system.

## Quick Start

### Run Standard Benchmarks

```bash
# Run all benchmarks
go test -bench=. -benchmem ./internal/benchmarks/

# Run specific benchmark
go test -bench=BenchmarkWebhookDelivery_SmallPayload -benchmem ./internal/benchmarks/

# Run with CPU profiling
go test -bench=. -benchmem -cpuprofile=cpu.prof ./internal/benchmarks/

# Run with memory profiling
go test -bench=. -benchmem -memprofile=mem.prof ./internal/benchmarks/

# View detailed output (not short mode)
go test -bench=. -benchmem -v ./internal/benchmarks/
```

### Run Load Test Tool

```bash
# Build the benchmark tool
go build -o bin/benchmark ./cmd/benchmark/

# Run with default settings (1 minute, 100 RPS, 10KB payload)
./bin/benchmark

# Custom load test
./bin/benchmark \
  -duration=5m \
  -rps=1000 \
  -payload=50 \
  -concurrency=50 \
  -templates=true
```

## Benchmark Suite

### 1. Webhook Delivery Benchmarks

Tests webhook delivery performance with varying payload sizes:

- **BenchmarkWebhookDelivery_SmallPayload**: 1KB payloads
- **BenchmarkWebhookDelivery_MediumPayload**: 10KB payloads
- **BenchmarkWebhookDelivery_LargePayload**: 100KB payloads
- **BenchmarkWebhookDelivery_Concurrent**: 10 concurrent workers
- **BenchmarkWebhookDelivery_HighConcurrency**: 100 concurrent workers

**Metrics Reported:**
- Operations per second (ops/s)
- Memory per operation (KB/op)
- Data transferred per operation (KB-transferred/op)
- Active goroutines

**Example Output:**
```
BenchmarkWebhookDelivery_SmallPayload-8       24855    48587 ns/op    2.34 KB/op    1.12 KB-transferred/op    42 goroutines
```

### 2. Template Transformation Benchmarks

Tests template processing overhead:

- **BenchmarkTemplateTransformation_Small**: 1KB payloads with templates
- **BenchmarkTemplateTransformation_Large**: 100KB payloads with templates

**Use Cases:**
- Estimate overhead of payload transformations
- Compare cached vs uncached template performance
- Identify memory usage patterns

### 3. Cache Performance Benchmarks

Tests template cache efficiency:

- **BenchmarkCachePerformance_Hits**: Cache hit performance

**Metrics:**
- Cache hit ratio
- Memory overhead per cached template
- Lookup latency

### 4. Sustained Load Profile

Tests system behavior under prolonged load:

- **BenchmarkMemoryProfile_SustainedLoad**: 10-second sustained load test

**Measures:**
- Memory growth over time
- Goroutine leak detection
- GC impact
- Steady-state resource usage

## Load Testing Tool

The `cmd/benchmark` tool simulates realistic production workloads.

### Configuration Options

```bash
-duration duration
    Test duration (default 1m0s)
    
-rps int
    Target requests per second (default 100)
    
-payload int
    Payload size in KB (default 10)
    
-concurrency int
    Number of concurrent workers (default 10)
    
-templates
    Enable template transformations (default false)
```

### Example Scenarios

#### Baseline Performance Test
```bash
./bin/benchmark -duration=2m -rps=100 -payload=10 -concurrency=10
```

#### High Throughput Test
```bash
./bin/benchmark -duration=5m -rps=5000 -payload=10 -concurrency=100
```

#### Large Payload Test
```bash
./bin/benchmark -duration=2m -rps=500 -payload=100 -concurrency=50
```

#### Template Overhead Test
```bash
# Without templates
./bin/benchmark -duration=2m -rps=1000 -templates=false

# With templates
./bin/benchmark -duration=2m -rps=1000 -templates=true
```

## Resource Estimation

The load test tool automatically generates resource estimates for different load levels:

```
=== Resource Estimates for Production ===

Low Load (100 RPS):
  Estimated RAM: 150.00 MB (0.15 GB)
  Estimated Bandwidth: 2.50 MB/s (20.00 Mbps)
  Estimated Goroutines: 50
  Recommended CPU Cores: 1

Medium Load (1,000 RPS):
  Estimated RAM: 1500.00 MB (1.46 GB)
  Estimated Bandwidth: 25.00 MB/s (200.00 Mbps)
  Estimated Goroutines: 500
  Recommended CPU Cores: 1

High Load (10,000 RPS):
  Estimated RAM: 15000.00 MB (14.65 GB)
  Estimated Bandwidth: 250.00 MB/s (2000.00 Mbps)
  Estimated Goroutines: 5000
  Recommended CPU Cores: 6
```

### Estimation Methodology

1. **RAM Estimation:**
   - Based on peak memory usage during test
   - Includes 50% safety margin
   - Accounts for connection pooling and buffering

2. **Bandwidth Estimation:**
   - Linear scaling from measured bandwidth
   - Includes both request and response data
   - Does not include retry traffic

3. **CPU Estimation:**
   - Rough estimate: 1 core per 1000 goroutines
   - Actual needs depend on workload characteristics

4. **Goroutine Estimation:**
   - Linear scaling from observed concurrency
   - Each webhook delivery creates temporary goroutines

## Production Deployment Guidelines

### Minimum Requirements (100 RPS)
- **RAM:** 512 MB
- **CPU:** 1 core
- **Bandwidth:** 10 Mbps
- **Database Connections:** 10

### Recommended Requirements (1,000 RPS)
- **RAM:** 2 GB
- **CPU:** 2 cores
- **Bandwidth:** 50 Mbps
- **Database Connections:** 25

### High-Scale Requirements (10,000 RPS)
- **RAM:** 16 GB
- **CPU:** 8 cores
- **Bandwidth:** 500 Mbps
- **Database Connections:** 50

### Additional Considerations

1. **Database Resources:**
   - Separate benchmarks needed for database performance
   - Connection pool sizing impacts memory
   - Query performance affects response times

2. **Network Latency:**
   - External webhook endpoints may have high latency
   - Consider timeout configurations
   - Plan for retry logic overhead

3. **Monitoring Overhead:**
   - OpenTelemetry tracing adds ~5-10% overhead
   - Metrics collection adds ~2-5% overhead
   - Logging can significantly impact performance

4. **Burst Capacity:**
   - Plan for 2-3x normal load during bursts
   - Queue-based architecture helps smooth spikes
   - Monitor queue depth and latency

## Profiling for Optimization

### CPU Profiling
```bash
go test -bench=BenchmarkWebhookDelivery_Concurrent \
  -cpuprofile=cpu.prof \
  ./internal/benchmarks/

go tool pprof -http=:8080 cpu.prof
```

### Memory Profiling
```bash
go test -bench=BenchmarkMemoryProfile_SustainedLoad \
  -memprofile=mem.prof \
  ./internal/benchmarks/

go tool pprof -http=:8080 mem.prof
```

### Trace Analysis
```bash
go test -bench=BenchmarkWebhookDelivery_HighConcurrency \
  -trace=trace.out \
  ./internal/benchmarks/

go tool trace trace.out
```

## Interpreting Results

### Memory Per Operation
- < 10 KB/op: Excellent (minimal allocation)
- 10-100 KB/op: Good (reasonable allocation)
- 100-1000 KB/op: Acceptable (moderate allocation)
- > 1 MB/op: Poor (high allocation, investigate)

### Requests Per Second
- Baseline: 1000+ RPS per core for simple operations
- With templates: 500+ RPS per core
- With database: 100-500 RPS per core (query dependent)

### Response Time
- P50 < 10ms: Excellent
- P50 < 50ms: Good
- P50 < 100ms: Acceptable
- P95 should be < 2x P50

## Troubleshooting

### High Memory Usage
- Check for goroutine leaks: `go tool pprof -http=:8080 mem.prof`
- Review connection pool settings
- Verify template cache size limits

### Poor Throughput
- Profile CPU usage to find bottlenecks
- Check database query performance
- Review network latency to webhook endpoints

### Inconsistent Results
- Run benchmarks multiple times
- Use `-benchtime=10s` for longer runs
- Ensure system is not under other load

## Best Practices

1. **Regular Benchmarking:**
   - Run benchmarks before major releases
   - Track performance trends over time
   - Set performance budgets and alerts

2. **Realistic Test Data:**
   - Use production-like payload sizes
   - Include edge cases (very large/small payloads)
   - Test with actual templates from production

3. **Environment Consistency:**
   - Run benchmarks on consistent hardware
   - Minimize background processes
   - Use dedicated benchmark environments

4. **Incremental Testing:**
   - Start with low load and increase gradually
   - Identify breaking points and bottlenecks
   - Test individual components in isolation

5. **Documentation:**
   - Record benchmark configurations
   - Document findings and optimizations
   - Share results with team

## Continuous Monitoring

Set up performance baselines and track them over time:

```bash
# Save baseline
go test -bench=. -benchmem ./internal/benchmarks/ > baseline.txt

# Compare with current
go test -bench=. -benchmem ./internal/benchmarks/ > current.txt
benchstat baseline.txt current.txt
```

## Additional Resources

- Go Performance Tuning: https://go.dev/doc/diagnostics
- pprof User Guide: https://github.com/google/pprof
- Benchmark Guidelines: https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go
