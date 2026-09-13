package main

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// renderStructured writes v as json or yaml when format is set. It returns
// done=true when it produced output (so the caller skips its human rendering);
// done=false means format was empty and the caller should render its table.
func renderStructured(out io.Writer, format string, v any) (done bool, err error) {
	switch format {
	case "":
		return false, nil
	case "json":
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return true, err
		}
		_, err = fmt.Fprintln(out, string(b))
		return true, err
	case "yaml", "yml":
		// Route through JSON so keys honor the json tags (one source of truth).
		j, err := json.Marshal(v)
		if err != nil {
			return true, err
		}
		var generic any
		if err := json.Unmarshal(j, &generic); err != nil {
			return true, err
		}
		b, err := yaml.Marshal(generic)
		if err != nil {
			return true, err
		}
		_, err = out.Write(b)
		return true, err
	default:
		return true, fmt.Errorf("unknown output format %q (want json or yaml)", format)
	}
}

// yesNo renders a bool as a short yes/no.
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
