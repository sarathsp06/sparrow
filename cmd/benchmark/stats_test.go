package main

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestSummarize(t *testing.T) {
	s := Summarize(Metric{Name: "x", Values: []float64{10, 12, 14, 16, 18}})
	if s.Mean != 14 || s.Median != 14 || s.Min != 10 || s.Max != 18 {
		t.Fatalf("mean/median/min/max = %v/%v/%v/%v", s.Mean, s.Median, s.Min, s.Max)
	}
	// stddev = sqrt(40/4) = 3.1623; CI = 2.776 * 3.1623 / sqrt(5) = 3.926
	if math.Abs(s.StdDev-math.Sqrt(10)) > 1e-9 {
		t.Errorf("stddev = %v", s.StdDev)
	}
	if math.Abs(s.CI95-3.926) > 0.001 {
		t.Errorf("ci95 = %v, want 3.926", s.CI95)
	}

	one := Summarize(Metric{Values: []float64{7}})
	if one.CI95 != 0 || one.StdDev != 0 || one.Mean != 7 {
		t.Errorf("single run: %+v", one)
	}
	even := Summarize(Metric{Values: []float64{1, 2, 3, 10}})
	if even.Median != 2.5 {
		t.Errorf("even median = %v", even.Median)
	}
}

func TestTCritical95(t *testing.T) {
	if tCritical95(4) != 2.776 || tCritical95(1000) != 1.96 || !math.IsInf(tCritical95(0), 1) {
		t.Fatal("t table lookup")
	}
}

// A worker that falls behind must get slots that are already due, so the
// stall counts as latency instead of shrinking the offered load.
func TestPacerIsOpenLoop(t *testing.T) {
	p := NewPacer(100) // one slot every 10ms
	time.Sleep(55 * time.Millisecond)
	ctx := context.Background()
	first, _ := p.Next(ctx)
	sixth := first
	for range 5 {
		sixth, _ = p.Next(ctx)
	}
	if lag := time.Since(first); lag < 50*time.Millisecond {
		t.Fatalf("first slot should be ~55ms overdue, got %v", lag)
	}
	if got := sixth.Sub(first); got != 50*time.Millisecond {
		t.Fatalf("slots should be 10ms apart on the schedule, got %v between 1st and 6th", got)
	}
}

func TestPacerStopsOnCancel(t *testing.T) {
	p := NewPacer(1)
	ctx, cancel := context.WithCancel(context.Background())
	_, _ = p.Next(ctx) // slot 0 is due immediately
	cancel()
	if _, ok := p.Next(ctx); ok {
		t.Fatal("Next should report false after cancel")
	}
}
