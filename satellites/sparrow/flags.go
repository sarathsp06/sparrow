package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// kvFlag is a repeatable "key=value" flag collected into a map.
type kvFlag map[string]string

func (f kvFlag) String() string {
	pairs := make([]string, 0, len(f))
	for k, v := range f {
		pairs = append(pairs, k+"="+v)
	}
	return strings.Join(pairs, ",")
}

func (f kvFlag) Set(s string) error {
	k, v, ok := strings.Cut(s, "=")
	if !ok || k == "" {
		return fmt.Errorf("expected key=value, got %q", s)
	}
	f[k] = v
	return nil
}

func (f kvFlag) Type() string { return "key=value" }

// listFlag is a repeatable string flag.
type listFlag []string

func (f *listFlag) String() string { return strings.Join(*f, ",") }

func (f *listFlag) Set(s string) error {
	if s == "" {
		return fmt.Errorf("empty value")
	}
	*f = append(*f, s)
	return nil
}

func (f *listFlag) Type() string { return "string" }

// parseJSONArg parses inline JSON or, when prefixed with '@', the named file,
// into a JSON object.
func parseJSONArg(s string) (map[string]any, error) {
	raw := []byte(s)
	if strings.HasPrefix(s, "@") {
		data, err := os.ReadFile(s[1:])
		if err != nil {
			return nil, err
		}
		raw = data
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("payload must be a JSON object: %w", err)
	}
	return payload, nil
}
