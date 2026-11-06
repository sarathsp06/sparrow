package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"runtime"
	"runtime/debug"
	"sort"
	"sync/atomic"
	"time"
)

// ---------------------------------------------------------------------------
// Open-loop pacing
// ---------------------------------------------------------------------------

// Pacer hands out send times on a fixed schedule: slot k is due at
// start + k/rps, whether or not earlier requests have finished. Latency is
// then measured from the slot's due time, not from when a worker got round to
// sending it, so a stall shows up as latency instead of disappearing from the
// sample ("coordinated omission"). A worker that is behind takes the next slot
// at once, so the offered load stays at rps.
type Pacer struct {
	start    time.Time
	interval time.Duration
	next     atomic.Int64
}

// NewPacer starts a schedule of rps slots per second, the first due now.
func NewPacer(rps int) *Pacer {
	if rps < 1 {
		rps = 1
	}
	return &Pacer{start: time.Now(), interval: time.Second / time.Duration(rps)}
}

// Next blocks until the next slot is due and returns its due time; false when
// ctx ends first.
func (p *Pacer) Next(ctx context.Context) (time.Time, bool) {
	k := p.next.Add(1) - 1
	due := p.start.Add(time.Duration(k) * p.interval)
	if wait := time.Until(due); wait > 0 {
		t := time.NewTimer(wait)
		defer t.Stop()
		select {
		case <-t.C:
		case <-ctx.Done():
			return time.Time{}, false
		}
	}
	return due, ctx.Err() == nil
}

// ---------------------------------------------------------------------------
// Environment
// ---------------------------------------------------------------------------

// Environment records where and how a run happened, so a number can be traced
// back to the machine and build that produced it.
type Environment struct {
	StartedAt  time.Time `json:"started_at"`
	Hostname   string    `json:"hostname"`
	GOOS       string    `json:"goos"`
	GOARCH     string    `json:"goarch"`
	NumCPU     int       `json:"num_cpu"`
	GOMAXPROCS int       `json:"gomaxprocs"`
	GoVersion  string    `json:"go_version"`
	Revision   string    `json:"vcs_revision,omitempty"`
	Modified   bool      `json:"vcs_modified,omitempty"`
	Args       []string  `json:"args"`
}

func captureEnvironment() Environment {
	host, _ := os.Hostname()
	env := Environment{
		StartedAt:  time.Now().UTC(),
		Hostname:   host,
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		NumCPU:     runtime.NumCPU(),
		GOMAXPROCS: runtime.GOMAXPROCS(0),
		GoVersion:  runtime.Version(),
		Args:       os.Args[1:],
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				env.Revision = s.Value
			case "vcs.modified":
				env.Modified = s.Value == "true"
			}
		}
	}
	return env
}

func (e Environment) Print() {
	rev := e.Revision
	if rev == "" {
		rev = "unknown (build with `go build`, not `go run`, to record it)"
	} else if e.Modified {
		rev += " (uncommitted changes)"
	}
	fmt.Printf("Environment: %s, %s/%s, %d CPUs (GOMAXPROCS %d), %s, revision %s\n",
		e.Hostname, e.GOOS, e.GOARCH, e.NumCPU, e.GOMAXPROCS, e.GoVersion, rev)
}

// ---------------------------------------------------------------------------
// Repetitions
// ---------------------------------------------------------------------------

// Metric is one number taken from each run.
type Metric struct {
	Name   string
	Unit   string
	Values []float64
}

// MetricSummary describes a metric across runs: the mean with a 95%
// confidence interval (Student's t), the median and the spread.
type MetricSummary struct {
	Name   string  `json:"name"`
	Unit   string  `json:"unit"`
	Runs   int     `json:"runs"`
	Mean   float64 `json:"mean"`
	CI95   float64 `json:"ci95_half_width"` // 0 with fewer than two runs
	StdDev float64 `json:"stddev"`
	CV     float64 `json:"cv"` // stddev / mean
	Median float64 `json:"median"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
}

// Summarize computes the across-run statistics of m.
func Summarize(m Metric) MetricSummary {
	s := MetricSummary{Name: m.Name, Unit: m.Unit, Runs: len(m.Values)}
	n := len(m.Values)
	if n == 0 {
		return s
	}
	sorted := append([]float64(nil), m.Values...)
	sort.Float64s(sorted)
	s.Min, s.Max = sorted[0], sorted[n-1]
	if n%2 == 1 {
		s.Median = sorted[n/2]
	} else {
		s.Median = (sorted[n/2-1] + sorted[n/2]) / 2
	}
	var sum float64
	for _, v := range sorted {
		sum += v
	}
	s.Mean = sum / float64(n)
	if n < 2 {
		return s
	}
	var sq float64
	for _, v := range sorted {
		sq += (v - s.Mean) * (v - s.Mean)
	}
	s.StdDev = math.Sqrt(sq / float64(n-1))
	s.CI95 = tCritical95(n-1) * s.StdDev / math.Sqrt(float64(n))
	if s.Mean != 0 {
		s.CV = s.StdDev / s.Mean
	}
	return s
}

// tCritical95 is the two-sided 95% critical value of Student's t.
func tCritical95(df int) float64 {
	table := []float64{0, 12.706, 4.303, 3.182, 2.776, 2.571, 2.447, 2.365, 2.306, 2.262, 2.228,
		2.201, 2.179, 2.160, 2.145, 2.131, 2.120, 2.110, 2.101, 2.093, 2.086,
		2.080, 2.074, 2.069, 2.064, 2.060, 2.056, 2.052, 2.048, 2.045, 2.042}
	switch {
	case df < 1:
		return math.Inf(1)
	case df < len(table):
		return table[df]
	default:
		return 1.96
	}
}

// noisyCV is the coefficient of variation above which a comparison between
// two builds needs more runs (or a quieter machine) before it means anything.
const noisyCV = 0.10

// PrintSummaries writes the across-run table and says when a result is too
// noisy to compare.
func PrintSummaries(sums []MetricSummary) {
	if len(sums) == 0 {
		return
	}
	runs := sums[0].Runs
	fmt.Printf("\n=== Across %d runs ===\n\n", runs)
	if runs < 2 {
		fmt.Printf("One run gives no confidence interval. Use -runs 5 or more before comparing builds.\n\n")
	}
	fmt.Printf("  %-34s %14s %16s %12s %12s %12s %7s\n", "metric", "mean", "95% CI ±", "median", "min", "max", "CV")
	var noisy []string
	for _, s := range sums {
		fmt.Printf("  %-34s %14.2f %16.2f %12.2f %12.2f %12.2f %6.1f%%\n",
			s.Name+" ("+s.Unit+")", s.Mean, s.CI95, s.Median, s.Min, s.Max, 100*s.CV)
		if s.Runs > 1 && s.CV > noisyCV {
			noisy = append(noisy, s.Name)
		}
	}
	fmt.Println()
	if len(noisy) > 0 {
		fmt.Printf("Noisy (CV > %.0f%%): %v. Differences smaller than the confidence interval are not results.\n\n", 100*noisyCV, noisy)
	}
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
