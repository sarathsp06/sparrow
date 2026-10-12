// Package cron is the cron source: it pushes a fixed Sparrow event on each
// job's 5-field schedule.
package cron

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/sarathsp06/sparrow/satellites/sparrow-sources/sources"
)

// cronSpec is a parsed 5-field cron expression (min hour dom mon dow).
// Each field is a bitmask of allowed values.
type cronSpec struct {
	min, hour, dom, mon, dow uint64
	domStar, dowStar         bool
}

// parseCron parses a standard 5-field cron expression supporting *, numbers,
// ranges (a-b), steps (*/n, a-b/n) and comma lists.
func parseCron(expr string) (*cronSpec, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("expected 5 fields, got %d", len(fields))
	}
	bounds := [5]struct{ lo, hi int }{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 7}}
	var masks [5]uint64
	for i, f := range fields {
		m, err := parseCronField(f, bounds[i].lo, bounds[i].hi)
		if err != nil {
			return nil, fmt.Errorf("field %d %q: %w", i+1, f, err)
		}
		masks[i] = m
	}
	// dow: 7 means Sunday, fold into bit 0.
	if masks[4]&(1<<7) != 0 {
		masks[4] = (masks[4] &^ (1 << 7)) | 1
	}
	return &cronSpec{
		min: masks[0], hour: masks[1], dom: masks[2], mon: masks[3], dow: masks[4],
		domStar: fields[2] == "*",
		dowStar: fields[4] == "*",
	}, nil
}

// parseCronField parses one comma-separated field into a bitmask.
func parseCronField(field string, lo, hi int) (uint64, error) {
	var mask uint64
	for _, part := range strings.Split(field, ",") {
		rangePart, step := part, 1
		if idx := strings.IndexByte(part, '/'); idx >= 0 {
			rangePart = part[:idx]
			n, err := strconv.Atoi(part[idx+1:])
			if err != nil || n <= 0 {
				return 0, fmt.Errorf("invalid step %q", part[idx+1:])
			}
			step = n
		}
		start, end := lo, hi
		switch {
		case rangePart == "*":
			// full range
		case strings.Contains(rangePart, "-"):
			a, b, ok := strings.Cut(rangePart, "-")
			s, err1 := strconv.Atoi(a)
			e, err2 := strconv.Atoi(b)
			if !ok || err1 != nil || err2 != nil {
				return 0, fmt.Errorf("invalid range %q", rangePart)
			}
			start, end = s, e
		default:
			n, err := strconv.Atoi(rangePart)
			if err != nil {
				return 0, fmt.Errorf("invalid value %q", rangePart)
			}
			start, end = n, n
			if strings.IndexByte(part, '/') >= 0 {
				end = hi // "N/step" means N-hi/step, per vixie cron
			}
		}
		if start < lo || end > hi || start > end {
			return 0, fmt.Errorf("value out of range [%d,%d]: %q", lo, hi, part)
		}
		for v := start; v <= end; v += step {
			mask |= 1 << v
		}
	}
	if mask == 0 {
		return 0, fmt.Errorf("empty field")
	}
	return mask, nil
}

// Match reports whether t (minute resolution) satisfies the spec. Standard
// cron day semantics: when both dom and dow are restricted, either matches.
func (s *cronSpec) Match(t time.Time) bool {
	if s.min&(1<<t.Minute()) == 0 ||
		s.hour&(1<<t.Hour()) == 0 ||
		s.mon&(1<<int(t.Month())) == 0 {
		return false
	}
	domOK := s.dom&(1<<t.Day()) != 0
	dowOK := s.dow&(1<<int(t.Weekday())) != 0
	if !s.domStar && !s.dowStar {
		return domOK || dowOK
	}
	return domOK && dowOK
}

// Job emits a fixed event on a 5-field cron schedule (one cron: entry).
type Job struct {
	Schedule string            `yaml:"schedule"`
	Event    string            `yaml:"event"`
	Payload  map[string]any    `yaml:"payload"`
	Labels   map[string]string `yaml:"labels"`
}

// Emitter pushes fixed events on cron schedules. Build it with New, then
// start Run.
type Emitter struct {
	jobs []entry
}

type entry struct {
	Job
	spec    *cronSpec
	payload json.RawMessage
}

// New validates every job up front (event name, schedule, payload) so a
// bad cron: entry fails at startup, not at its first tick.
func New(jobs []Job) (*Emitter, error) {
	c := &Emitter{jobs: make([]entry, 0, len(jobs))}
	for i, job := range jobs {
		if job.Event == "" {
			return nil, fmt.Errorf("cron[%d]: event is required", i)
		}
		spec, err := parseCron(job.Schedule)
		if err != nil {
			return nil, fmt.Errorf("cron[%d] schedule %q: %w", i, job.Schedule, err)
		}
		payload := json.RawMessage(`{}`)
		if job.Payload != nil {
			if payload, err = json.Marshal(job.Payload); err != nil {
				return nil, fmt.Errorf("cron[%d] payload: %w", i, err)
			}
		}
		c.jobs = append(c.jobs, entry{Job: job, spec: spec, payload: payload})
	}
	return c, nil
}

// Run ticks once per minute until ctx is done and pushes each matching job's
// event. Jitterless: ticks are aligned to wall-clock minute boundaries. A
// failed push is logged and skipped; there is no catch-up. A nil log means
// slog.Default().
func (c *Emitter) Run(ctx context.Context, p sources.Pusher, log *slog.Logger) {
	log = cmp.Or(log, slog.Default())
	for {
		now := time.Now()
		next := now.Truncate(time.Minute).Add(time.Minute)
		select {
		case <-ctx.Done():
			return
		case <-time.After(next.Sub(now)):
		}
		c.tick(ctx, time.Now().Truncate(time.Minute), p, log)
	}
}

// tick pushes every job whose schedule matches the given minute.
func (c *Emitter) tick(ctx context.Context, tick time.Time, p sources.Pusher, log *slog.Logger) {
	for _, job := range c.jobs {
		if !job.spec.Match(tick) {
			continue
		}
		if err := p.PushEvent(ctx, job.Event, job.payload, job.Labels); err != nil {
			log.Error("cron push failed", "event", job.Event, "error", err)
			continue
		}
		log.Info("cron event pushed", "event", job.Event, "schedule", job.Schedule)
	}
}
