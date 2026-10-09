package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceSampler_Nil(t *testing.T) {
	var s *ResourceSampler
	s.Start(context.Background())
	usage := s.Stop(100)
	assert.Nil(t, usage)
}

func TestResourceSampler_Self(t *testing.T) {
	pid := os.Getpid()
	cfg := ResourceConfig{
		SparrowPID: pid,
	}

	sampler, err := NewResourceSampler(cfg)
	require.NoError(t, err)
	require.NotNil(t, sampler)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sampler.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	usage := sampler.Stop(10)
	require.NotNil(t, usage)
	assert.Equal(t, pid, usage.SparrowPID)
	assert.GreaterOrEqual(t, usage.SparrowPeakRSS, 0.0)
	assert.GreaterOrEqual(t, usage.SparrowCPUAvg, 0.0)
}

func TestResourceSampler_MemoryUsageTracking(t *testing.T) {
	pid := os.Getpid()
	cfg := ResourceConfig{
		SparrowPID: pid,
	}

	sampler, err := NewResourceSampler(cfg)
	require.NoError(t, err)
	require.NotNil(t, sampler)

	// Manually invoke sample before allocation
	sampler.sample()

	// Allocate a slice to change memory usage and ensure it is touched
	buf := make([]byte, 10*1024*1024)
	for i := 0; i < len(buf); i += 4096 {
		buf[i] = 1
	}

	// Sample again after allocation
	sampler.sample()

	usage := sampler.summarize(100)
	require.NotNil(t, usage)
	assert.Greater(t, usage.SparrowPeakRSS, 0.0, "peak RSS memory usage should be recorded and positive")

	// Ensure buf is retained until here
	assert.NotEmpty(t, buf)
}

func TestResourceSampler_Summarize(t *testing.T) {
	s := &ResourceSampler{
		cfg: ResourceConfig{
			SparrowPID:  1234,
			PostgresPID: 5678,
		},
		sparrowCPUSamples:  []float64{10.0, 20.0, 30.0},
		sparrowRSSSamples:  []uint64{100 * 1024 * 1024, 150 * 1024 * 1024},
		postgresCPUSamples: []float64{5.0, 15.0},
		startCPUTime:       10.0,
		endCPUTime:         15.0, // 5 seconds CPU time = 5000 ms
		startWALBytes:      1000000,
		endWALBytes:        11000000, // 10 MB WAL
		startTime:          time.Now().Add(-5 * time.Second),
		endTime:            time.Now(),
	}

	usage := s.summarize(1000) // 1000 deliveries
	require.NotNil(t, usage)

	assert.InDelta(t, 20.0, usage.SparrowCPUAvg, 0.01)
	assert.InDelta(t, 30.0, usage.SparrowCPUPeak, 0.01)
	assert.InDelta(t, 150.0, usage.SparrowPeakRSS, 0.01)

	assert.InDelta(t, 10.0, usage.PostgresCPUAvg, 0.01)
	assert.InDelta(t, 15.0, usage.PostgresCPUPeak, 0.01)

	// CPU ms per delivery: 5000 ms / 1000 = 5.0
	assert.InDelta(t, 5.0, usage.CPUMsPerDelivery, 0.01)

	// WAL: 10 MB over 5s = 2.0 MB/s
	assert.Equal(t, int64(10000000), usage.WALBytesWritten)
	assert.InDelta(t, 2.0, usage.WALMBPerSec, 0.1)
}
