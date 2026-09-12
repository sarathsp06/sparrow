package main

import (
	"encoding/json"
	"errors"
	"flag"
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

// parseWithArg parses a subcommand's flags while accepting exactly one
// positional argument in either position: "push name -d x" or "push -d x name".
// Stdlib flag stops at the first positional, so a leading one is hoisted out
// before Parse.
func parseWithArg(fs *flag.FlagSet, args []string, usage string) (string, error) {
	arg, rest := "", args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		arg, rest = args[0], args[1:]
	}
	if err := fs.Parse(rest); err != nil {
		return "", err
	}
	switch {
	case arg == "" && fs.NArg() == 1:
		arg = fs.Arg(0)
	case arg != "" && fs.NArg() == 0:
	default:
		return "", errors.New("usage: " + usage)
	}
	return arg, nil
}

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
