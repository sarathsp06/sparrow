# Benchmarking & Performance Guide

**Complete guide for benchmarking, load testing, and performance analysis of the Sparrow webhook delivery system.**

---

## Table of Contents

1. [Understanding What to Benchmark](#understanding-what-to-benchmark)
2. [Key Questions Before Benchmarking](#key-questions-before-benchmarking)
3. [Quick Start](#quick-start)
4. [Benchmark Suite](#benchmark-suite)
5. [Load Testing Tool](#load-testing-tool)
6. [Resource Estimation](#resource-estimation)
7. [Production Guidelines](#production-deployment-guidelines)
8. [Profiling & Analysis](#profiling-for-optimization)
9. [Best Practices](#best-practices)

---

## Understanding What to Benchmark

A webhook delivery system has multiple performance-critical components that should be measured independently and as a whole:

### Core Components

**1. Webhook Delivery Pipeline**
- End-to-end latency from event creation to delivery
- HTTP request preparation and execution
- Retry logic and backoff strategies
- Error handling overhead

**2. Template Transformation Engine**
- Template parsing and compilation time
- Payload transformation performance
- Template caching effectiveness
- Memory allocation patterns

**3. Queue Management**
- Job insertion and retrieval rates
- Queue depth and throughput
- Worker pool efficiency
- Concurrency handling

**4. Database Operations**
- Event registration and retrieval
- Delivery status updates
- Subscription management queries
- Connection pool utilization

**5. HTTP Client Layer**
- Connection pooling efficiency
- Request/response serialization
- Network I/O overhead
- Keep-alive effectiveness

### System-Wide Metrics

**Throughput Metrics:**
- Requests per second (RPS)
- Events processed per second
- Deliveries completed per second
- Bandwidth utilization (MB/s)

**Latency Metrics:**
- P50 (median) - typical user experience
- P90 - 90th percentile
- P95 - 95th percentile  
- P99 - worst-case for most users
- P999 - tail latency detection

**Resource Metrics:**
- Memory allocation and GC pressure
- CPU utilization per core
- Goroutine count and lifecycle
- Database connection pool usage
- File descriptor consumption

**Reliability Metrics:**
- Error rates by category
- Retry success rates
- Circuit breaker activation
- Timeout frequency

---

## Key Questions Before Benchmarking

### 1. What is Your Expected Load Profile?

**Traffic Patterns:**
- What is your baseline RPS? (e.g., 100 RPS, 1,000 RPS, 10,000 RPS)
- Do you have traffic spikes? (e.g., 10x spike during promotions)
- What is your peak load duration? (e.g., 5 minutes, 1 hour, sustained)
- Are events distributed evenly or bursty?

**Example Scenarios:**
```
E-commerce: 500 RPS baseline, 5,000 RPS during Black Friday
SaaS Platform: 2,000 RPS steady, 3,000 RPS during business hours
IoT System: 10,000 RPS sustained, 50,000 RPS during firmware updates
```

### 2. What is Your Payload Characteristics?

**Payload Size:**
- Typical payload size? (e.g., 1 KB, 10 KB, 100 KB)
- Maximum payload size? (e.g., 1 MB limit)
- Payload complexity? (deeply nested JSON, arrays, binary data)

**Template Usage:**
- How many unique templates? (10, 100, 1,000+)
- Template complexity? (simple field mapping vs complex transformations)
- Cache hit ratio expectations? (80%, 95%, 99%)

**Example Questions:**
```
- Are you sending minimal notification payloads (< 1 KB)?
- Are you forwarding complete order details (10-100 KB)?
- Do you include media URLs or base64 encoded data?
```

### 3. What is Your Concurrency Model?

**Concurrent Operations:**
- How many concurrent webhook deliveries?
- How many concurrent template transformations?
- How many database connections needed?
- How many goroutines are acceptable?

**Resource Constraints:**
- Available CPU cores? (2, 4, 8, 16)
- Available RAM? (512 MB, 2 GB, 8 GB, 16 GB)
- Network bandwidth? (100 Mbps, 1 Gbps, 10 Gbps)
- Database connection limits? (10, 50, 100)

### 4. What are Your Latency Requirements?

**SLA Targets:**
- What is acceptable P50 latency? (< 50ms, < 100ms, < 500ms)
- What is acceptable P95 latency? (< 200ms, < 1s, < 5s)
- What is acceptable P99 latency? (< 1s, < 5s, < 10s)
- What is your timeout threshold? (30s, 60s, 120s)

**Business Impact:**
```
Real-time notifications: P95 < 100ms
Payment confirmations: P99 < 500ms  
Batch processing: P50 < 5s acceptable
Analytics webhooks: P99 < 30s acceptable
```

### 5. What are Your Reliability Requirements?

**Error Tolerance:**
- Acceptable error rate? (0.1%, 1%, 5%)
- Retry strategy? (exponential backoff, fixed interval)
- Maximum retry attempts? (3, 5, 10)
- Dead letter queue policy?

**Failure Scenarios:**
- How to handle downstream endpoint failures?
- What happens during database outages?
- How to handle network partitions?
- Circuit breaker thresholds?

### 6. What is Your Deployment Environment?

**Infrastructure:**
- Container or VM deployment?
- Kubernetes or standalone?
- Shared or dedicated resources?
- Auto-scaling capabilities?

**External Dependencies:**
- Database type and configuration? (PostgreSQL, MySQL)
- Message queue if any? (Redis, RabbitMQ, Kafka)
- Observability stack? (Prometheus, Grafana, Jaeger)
- Load balancer configuration?

### 7. What Optimizations Have You Applied?

**Current State:**
- Connection pooling enabled?
- Template caching implemented?
- Memory pooling (sync.Pool) in use?
- Batch operations supported?

**Performance Features:**
```
✓ HTTP keep-alive connections
✓ LRU template cache (capacity?)
✓ sync.Pool for buffer reuse
✓ Database connection pooling
✓ Prepared statements
```

### 8. What Are You Optimizing For?

**Primary Goal:**
- Maximum throughput? (handle highest RPS)
- Lowest latency? (minimize P99)
- Minimal memory? (run in constrained environments)
- Cost efficiency? (maximize RPS per dollar)

**Trade-offs:**
```
High throughput may increase latency variance
Low memory may reduce cache effectiveness
Cost optimization may sacrifice peak performance
```

---

## Quick Start

### Running Standard Benchmarks


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

---

## Current Performance Characteristics

### Benchmark Tool Capabilities

The load testing tool (`cmd/benchmark/main_refactored.go`) is designed for production-grade performance testing with the following capabilities:

**Accurate Rate Limiting:**
- Token bucket algorithm for precise RPS control
- 99.5%+ accuracy at target RPS (e.g., 99.5/100 actual vs target)
- Smooth distribution across concurrent workers

**Scalability:**
- Tested up to 50,000+ RPS
- Configurable concurrency (10 to 1,000+ workers)
- Minimal resource overhead (160 KB fixed memory for latency tracking)

**Memory Efficiency:**
- Reservoir sampling for fixed memory usage
- Configurable sampling rate (0.01 to 1.0) for high RPS scenarios
- sync.Pool integration for buffer reuse

**Comprehensive Metrics:**
- Latency percentiles: P50, P90, P95, P99, P999
- Throughput: Requests/second, MB/second
- Resource usage: Memory, goroutines, CPU
- Error categorization and reporting

**Production Testing:**
- External URL support via `-url` flag
- Template transformation testing
- Graceful shutdown with signal handling
- Real-time resource monitoring

### Tool Configuration Flags

```bash
# Core parameters
-duration string      # Test duration (e.g., "30s", "5m", "1h")
-rps int             # Target requests per second
-concurrency int     # Number of concurrent workers

# Advanced options
-url string          # External server URL (optional, defaults to test server)
-payload string      # Custom JSON payload
-templates int       # Number of unique templates to test
-sampling float      # Sampling rate 0.01-1.0 (default 1.0)
-monitor duration    # Resource monitoring interval (e.g., "5s")
```

### Usage Examples

**Basic load test:**
```bash
go run cmd/benchmark/main_refactored.go \
  -duration 30s \
  -rps 1000 \
  -concurrency 50
```

**High throughput test:**
```bash
go run cmd/benchmark/main_refactored.go \
  -duration 60s \
  -rps 50000 \
  -concurrency 500 \
  -sampling 0.01
```

**Production endpoint test:**
```bash
go run cmd/benchmark/main_refactored.go \
  -url https://api.example.com/webhooks \
  -duration 120s \
  -rps 10000 \
  -payload '{"event":"order.created","data":{"id":"12345"}}'
```

**Template transformation test:**
```bash
go run cmd/benchmark/main_refactored.go \
  -duration 30s \
  -rps 5000 \
  -templates 100 \
  -concurrency 100
```

### Performance Optimizations in Use

The system incorporates several optimizations that affect benchmark results:

**1. sync.Pool for Memory Reuse**
- Buffer pool for HTTP request/response bodies
- Byte slice pool for temporary allocations
- Header map pool for HTTP headers
- Reduces allocations by 3.5-9.5x for repeated operations

**2. Template Caching**
- LRU cache for compiled templates
- Configurable capacity (default: 1000 templates)
- Cache hit ratio directly impacts transformation performance

**3. Connection Pooling**
- HTTP client with keep-alive connections
- Configurable idle connections and timeouts
- Reduces TCP handshake overhead

**4. Efficient Queue Management**
- Worker pool architecture for job processing
- Batch operations where applicable
- Context-based lifecycle management

### Interpreting Benchmark Results

**Latency Distribution:**
```
P50: 45ms     → Typical user experience
P90: 89ms     → 90% of requests complete within this time
P95: 125ms    → Service level target
P99: 280ms    → Tail latency detection
P999: 1.2s    → Outlier analysis
```

**Good Indicators:**
- P95 < 2x P50 (consistent performance)
- P99 < 5x P50 (limited tail latency)
- Low error rate (< 0.1%)
- Stable memory usage over time

**Warning Signs:**
- P99 > 10x P50 (high variance, potential issues)
- Increasing memory over time (potential leak)
- Rising goroutine count (goroutine leak)
- Error rate increasing with load (saturation)

### Benchmark Comparison Methodology

To compare performance before/after changes:

**1. Establish baseline:**
```bash
go test -bench=. -benchmem ./internal/benchmarks/ > baseline.txt
```

**2. Make changes and re-run:**
```bash
go test -bench=. -benchmem ./internal/benchmarks/ > current.txt
```

**3. Compare with benchstat:**
```bash
go install golang.org/x/perf/cmd/benchstat@latest
benchstat baseline.txt current.txt
```

**Example output:**
```
name                    old time/op    new time/op    delta
WebhookDelivery-8         1.50ms ± 2%    1.42ms ± 1%   -5.33%
TemplateTransform-8       125µs ± 1%      13µs ± 2%  -89.60%

name                    old alloc/op   new alloc/op   delta
WebhookDelivery-8         25.0kB ± 0%     2.5kB ± 0%  -90.00%
TemplateTransform-8       5.00kB ± 0%    0.50kB ± 0%  -90.00%

name                    old allocs/op  new allocs/op  delta
WebhookDelivery-8            45.0 ± 0%      15.0 ± 0%  -66.67%
TemplateTransform-8          12.0 ± 0%       1.0 ± 0%  -91.67%
```

---

