package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/shirou/gopsutil/v4/process"
)

// ResourceConfig holds target monitoring parameters.
type ResourceConfig struct {
	SparrowPID  int    `json:"sparrow_pid,omitempty"`
	PostgresPID int    `json:"postgres_pid,omitempty"`
	PostgresDSN string `json:"postgres_dsn,omitempty"`
}

// ResourceUsage holds summary metrics collected during the run.
type ResourceUsage struct {
	SparrowPID       int     `json:"sparrow_pid,omitempty"`
	SparrowCPUAvg    float64 `json:"sparrow_cpu_avg_pct"`
	SparrowCPUPeak   float64 `json:"sparrow_cpu_peak_pct"`
	SparrowPeakRSS   float64 `json:"sparrow_peak_rss_mb"`
	CPUMsPerDelivery float64 `json:"cpu_ms_per_delivery,omitempty"`

	PostgresPID     int     `json:"postgres_pid,omitempty"`
	PostgresCPUAvg  float64 `json:"postgres_cpu_avg_pct,omitempty"`
	PostgresCPUPeak float64 `json:"postgres_cpu_peak_pct,omitempty"`

	WALBytesWritten int64   `json:"wal_bytes_written,omitempty"`
	WALMBPerSec     float64 `json:"wal_mb_per_sec,omitempty"`
}

// ResourceSampler samples CPU, memory RSS, and WAL growth during a test run.
type ResourceSampler struct {
	cfg ResourceConfig

	sparrowProc  *process.Process
	postgresProc *process.Process
	db           *sql.DB

	sparrowCPUSamples  []float64
	sparrowRSSSamples  []uint64
	postgresCPUSamples []float64

	prevSparrowCPU  float64
	prevPostgresCPU float64
	prevSampleAt    time.Time

	startCPUTime float64
	endCPUTime   float64

	startWALBytes int64
	endWALBytes   int64

	startTime time.Time
	endTime   time.Time

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// NewResourceSampler initializes processes and DB connection for monitoring.
func NewResourceSampler(cfg ResourceConfig) (*ResourceSampler, error) {
	if cfg.SparrowPID == 0 && cfg.PostgresPID == 0 && cfg.PostgresDSN == "" {
		return nil, nil
	}

	s := &ResourceSampler{
		cfg:  cfg,
		done: make(chan struct{}),
	}

	if cfg.SparrowPID > 0 {
		proc, err := process.NewProcess(int32(cfg.SparrowPID))
		if err != nil {
			return nil, fmt.Errorf("attach sparrow pid %d: %w", cfg.SparrowPID, err)
		}
		s.sparrowProc = proc
	}

	if cfg.PostgresPID > 0 {
		proc, err := process.NewProcess(int32(cfg.PostgresPID))
		if err != nil {
			return nil, fmt.Errorf("attach postgres pid %d: %w", cfg.PostgresPID, err)
		}
		s.postgresProc = proc
	}

	if cfg.PostgresDSN != "" {
		db, err := sql.Open("pgx", cfg.PostgresDSN)
		if err != nil {
			return nil, fmt.Errorf("connect postgres dsn: %w", err)
		}
		s.db = db
	}

	return s, nil
}

func queryWALBytes(ctx context.Context, db *sql.DB) (int64, error) {
	if db == nil {
		return 0, nil
	}
	var bytes int64
	err := db.QueryRowContext(ctx, "SELECT pg_wal_lsn_diff(pg_current_wal_lsn(), '0/0')::bigint").Scan(&bytes)
	if err != nil {
		return 0, err
	}
	return bytes, nil
}

func getCPUTime(proc *process.Process) float64 {
	if proc == nil {
		return 0
	}
	times, err := proc.Times()
	if err != nil {
		return 0
	}
	return times.User + times.System
}

// cpuTimeWithChildren sums user+system CPU seconds of proc and its direct
// children. Postgres does its work in per-connection backend children, so
// sampling the postmaster alone reads ~0%.
// ponytail: direct children only; grandchildren (none in postgres) ignored.
func cpuTimeWithChildren(proc *process.Process) float64 {
	if proc == nil {
		return 0
	}
	total := getCPUTime(proc)
	children, err := proc.Children()
	if err != nil {
		return total
	}
	for _, c := range children {
		total += getCPUTime(c)
	}
	return total
}

// Start begins periodic background sampling.
func (s *ResourceSampler) Start(ctx context.Context) {
	if s == nil {
		return
	}
	s.startTime = time.Now()

	if s.sparrowProc != nil {
		s.startCPUTime = getCPUTime(s.sparrowProc)
	}
	s.prevSparrowCPU = s.startCPUTime
	s.prevPostgresCPU = cpuTimeWithChildren(s.postgresProc)
	s.prevSampleAt = s.startTime

	if s.db != nil {
		wal, err := queryWALBytes(ctx, s.db)
		if err != nil {
			log.Printf("resource sampler: start wal query failed: %v", err)
		} else {
			s.startWALBytes = wal
		}
	}

	sampleCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	go func() {
		defer close(s.done)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-sampleCtx.Done():
				return
			case <-ticker.C:
				s.sample()
			}
		}
	}()
}

// sample records one CPU/RSS observation. CPU % is computed from Times()
// deltas between ticks rather than gopsutil's CPUPercent, which can return
// NaN (poisoning averages and breaking JSON encoding) and, for Postgres,
// would only see the idle postmaster instead of its backend children.
func (s *ResourceSampler) sample() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	dt := now.Sub(s.prevSampleAt).Seconds()
	if dt <= 0 {
		return
	}

	if s.sparrowProc != nil {
		cur := getCPUTime(s.sparrowProc)
		if d := cur - s.prevSparrowCPU; d >= 0 {
			s.sparrowCPUSamples = append(s.sparrowCPUSamples, d/dt*100)
		}
		s.prevSparrowCPU = cur
		if mem, err := s.sparrowProc.MemoryInfo(); err == nil && mem != nil {
			s.sparrowRSSSamples = append(s.sparrowRSSSamples, mem.RSS)
		}
	}

	if s.postgresProc != nil {
		cur := cpuTimeWithChildren(s.postgresProc)
		// ponytail: backend churn between ticks skews one sample; skip negatives.
		if d := cur - s.prevPostgresCPU; d >= 0 {
			s.postgresCPUSamples = append(s.postgresCPUSamples, d/dt*100)
		}
		s.prevPostgresCPU = cur
	}

	s.prevSampleAt = now
}

// Stop finishes background sampling and calculates final metrics.
func (s *ResourceSampler) Stop(totalDelivered int64) *ResourceUsage {
	if s == nil {
		return nil
	}

	s.endTime = time.Now()
	if s.cancel != nil {
		s.cancel()
		<-s.done
	}

	// Final sample
	s.sample()

	if s.sparrowProc != nil {
		s.endCPUTime = getCPUTime(s.sparrowProc)
	}

	if s.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		wal, err := queryWALBytes(ctx, s.db)
		cancel()
		if err != nil {
			log.Printf("resource sampler: end wal query failed: %v", err)
		} else {
			s.endWALBytes = wal
		}
		_ = s.db.Close()
	}

	return s.summarize(totalDelivered)
}

func (s *ResourceSampler) summarize(totalDelivered int64) *ResourceUsage {
	s.mu.Lock()
	defer s.mu.Unlock()

	res := &ResourceUsage{
		SparrowPID:  s.cfg.SparrowPID,
		PostgresPID: s.cfg.PostgresPID,
	}

	if len(s.sparrowCPUSamples) > 0 {
		var sum, peak float64
		for _, cpu := range s.sparrowCPUSamples {
			sum += cpu
			if cpu > peak {
				peak = cpu
			}
		}
		res.SparrowCPUAvg = sum / float64(len(s.sparrowCPUSamples))
		res.SparrowCPUPeak = peak
	}

	if len(s.sparrowRSSSamples) > 0 {
		var peak uint64
		for _, rss := range s.sparrowRSSSamples {
			if rss > peak {
				peak = rss
			}
		}
		res.SparrowPeakRSS = float64(peak) / 1024 / 1024
	}

	if len(s.postgresCPUSamples) > 0 {
		var sum, peak float64
		for _, cpu := range s.postgresCPUSamples {
			sum += cpu
			if cpu > peak {
				peak = cpu
			}
		}
		res.PostgresCPUAvg = sum / float64(len(s.postgresCPUSamples))
		res.PostgresCPUPeak = peak
	}

	if s.endCPUTime > s.startCPUTime && totalDelivered > 0 {
		totalCPUMs := (s.endCPUTime - s.startCPUTime) * 1000.0
		res.CPUMsPerDelivery = totalCPUMs / float64(totalDelivered)
	}

	durationSec := s.endTime.Sub(s.startTime).Seconds()
	if s.endWALBytes > s.startWALBytes {
		res.WALBytesWritten = s.endWALBytes - s.startWALBytes
		if durationSec > 0 {
			res.WALMBPerSec = (float64(res.WALBytesWritten) / 1024 / 1024) / durationSec
		}
	}

	return res
}
