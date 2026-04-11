package queue

import (
	"testing"
	"time"
)

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected time.Duration
	}{
		{
			name:     "empty header returns default",
			header:   "",
			expected: defaultRetryAfter,
		},
		{
			name:     "integer seconds",
			header:   "120",
			expected: 120 * time.Second,
		},
		{
			name:     "one second",
			header:   "1",
			expected: 1 * time.Second,
		},
		{
			name:     "zero seconds returns default",
			header:   "0",
			expected: defaultRetryAfter,
		},
		{
			name:     "negative seconds returns default",
			header:   "-5",
			expected: defaultRetryAfter,
		},
		{
			name:     "exceeds max clamps to maxRetryAfter",
			header:   "3600",
			expected: maxRetryAfter,
		},
		{
			name:     "exactly at max",
			header:   "900",
			expected: maxRetryAfter, // 900s = 15min = maxRetryAfter
		},
		{
			name:     "just under max",
			header:   "899",
			expected: 899 * time.Second,
		},
		{
			name:     "unparseable string returns default",
			header:   "not-a-number-or-date",
			expected: defaultRetryAfter,
		},
		{
			name:     "float string returns default (Atoi rejects floats)",
			header:   "30.5",
			expected: defaultRetryAfter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRetryAfter(tt.header)
			if got != tt.expected {
				t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.header, got, tt.expected)
			}
		})
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	// Test with a future HTTP-date
	future := time.Now().Add(30 * time.Second).UTC().Format(time.RFC1123)
	got := parseRetryAfter(future)

	// The result should be approximately 30 seconds (within a 2-second tolerance)
	if got < 28*time.Second || got > 32*time.Second {
		t.Errorf("parseRetryAfter(%q) = %v, want ~30s", future, got)
	}
}

func TestParseRetryAfter_HTTPDate_Past(t *testing.T) {
	// A date in the past should return default
	past := time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC1123)
	got := parseRetryAfter(past)

	if got != defaultRetryAfter {
		t.Errorf("parseRetryAfter(past date) = %v, want %v (default)", got, defaultRetryAfter)
	}
}

func TestParseRetryAfter_HTTPDate_FarFuture(t *testing.T) {
	// A date far in the future should be capped to maxRetryAfter
	farFuture := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC1123)
	got := parseRetryAfter(farFuture)

	if got != maxRetryAfter {
		t.Errorf("parseRetryAfter(far future date) = %v, want %v (max)", got, maxRetryAfter)
	}
}

func TestDefaultAndMaxRetryAfterConstants(t *testing.T) {
	if defaultRetryAfter != 60*time.Second {
		t.Errorf("defaultRetryAfter = %v, want 60s", defaultRetryAfter)
	}
	if maxRetryAfter != 15*time.Minute {
		t.Errorf("maxRetryAfter = %v, want 15m", maxRetryAfter)
	}
}

func TestIsSuccessStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected []int64
		want     bool
	}{
		{"200 with empty expected (default 2xx)", 200, nil, true},
		{"201 with empty expected (default 2xx)", 201, nil, true},
		{"300 with empty expected (not 2xx)", 300, nil, false},
		{"429 with empty expected", 429, nil, false},
		{"500 with empty expected", 500, nil, false},
		{"200 in explicit list", 200, []int64{200, 201, 204}, true},
		{"201 in explicit list", 201, []int64{200, 201, 204}, true},
		{"202 not in explicit list", 202, []int64{200, 201, 204}, false},
		{"204 in explicit list", 204, []int64{200, 201, 204}, true},
		{"429 not in explicit list", 429, []int64{200, 201, 204}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSuccessStatusCode(tt.code, tt.expected)
			if got != tt.want {
				t.Errorf("isSuccessStatusCode(%d, %v) = %v, want %v", tt.code, tt.expected, got, tt.want)
			}
		})
	}
}
