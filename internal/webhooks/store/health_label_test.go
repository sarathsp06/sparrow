package store

import "testing"

func TestHealthLabel(t *testing.T) {
	cases := []struct {
		name                string
		events              int
		rate                float64
		consecutiveFailures int
		want                string
	}{
		{"no data", 0, 0, 0, "unknown"},
		{"too few events", 2, 1.0, 0, "unknown"},
		{"healthy", 3, 1.0, 0, "healthy"},
		{"degraded", 5, 0.85, 0, "degraded"},
		{"unhealthy by rate", 10, 0.5, 0, "unhealthy"},
		{"unhealthy by run", 3, 1.0, 5, "unhealthy"},
		{"low rate but few events", 4, 0.5, 0, "unknown"},
	}
	for _, c := range cases {
		if got := healthLabel(c.events, c.rate, c.consecutiveFailures); got != c.want {
			t.Errorf("%s: healthLabel(%d, %.2f, %d) = %q, want %q", c.name, c.events, c.rate, c.consecutiveFailures, got, c.want)
		}
	}
}
