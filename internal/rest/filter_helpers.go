package rest

import (
	"fmt"
	"strings"
	"time"
)

func parseLabelFilter(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	labels := make(map[string]string)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("labels must be comma-separated key=value pairs")
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			return nil, fmt.Errorf("labels must be comma-separated key=value pairs")
		}
		labels[key] = value
	}
	if len(labels) == 0 {
		return nil, nil
	}
	return labels, nil
}

// parseDateFilter parses a YYYY-MM-DD date (the start of that day, or its
// end when endOfDay) or an exact RFC3339 timestamp, such as an import's
// imported_at.
func parseDateFilter(raw string, endOfDay bool) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if ts, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return &ts, nil
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, fmt.Errorf("dates must use YYYY-MM-DD or RFC3339")
	}
	if endOfDay {
		t = t.Add(24*time.Hour - time.Nanosecond)
	}
	return &t, nil
}
