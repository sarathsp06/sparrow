package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderStructured(t *testing.T) {
	type row struct {
		Name   string `json:"event_name"`
		Active bool   `json:"active"`
	}
	v := row{Name: "user.signup", Active: true}

	// Empty format defers to the caller's human rendering.
	var buf bytes.Buffer
	done, err := renderStructured(&buf, "", v)
	if err != nil || done {
		t.Fatalf("empty format: done=%v err=%v, want done=false nil", done, err)
	}
	if buf.Len() != 0 {
		t.Fatalf("empty format wrote %q, want nothing", buf.String())
	}

	// JSON uses the json tag names.
	buf.Reset()
	done, err = renderStructured(&buf, "json", v)
	if !done || err != nil {
		t.Fatalf("json: done=%v err=%v", done, err)
	}
	if !strings.Contains(buf.String(), `"event_name": "user.signup"`) {
		t.Fatalf("json output missing tagged key: %q", buf.String())
	}

	// YAML routes through the json tags too (one source of truth).
	buf.Reset()
	done, err = renderStructured(&buf, "yaml", v)
	if !done || err != nil {
		t.Fatalf("yaml: done=%v err=%v", done, err)
	}
	if !strings.Contains(buf.String(), "event_name: user.signup") {
		t.Fatalf("yaml output not keyed by json tag: %q", buf.String())
	}

	// Unknown format is an error, and done=true so the caller does not also print.
	buf.Reset()
	done, err = renderStructured(&buf, "xml", v)
	if !done || err == nil {
		t.Fatalf("unknown format: done=%v err=%v, want done=true err!=nil", done, err)
	}
}

func TestFirstLineSkipsHeadings(t *testing.T) {
	desc := "# json\n\nConverts any value to a JSON string.\n\n## Usage\n"
	if got := firstLine(desc); got != "Converts any value to a JSON string." {
		t.Fatalf("firstLine = %q, want the prose line, not the heading", got)
	}
	if got := firstLine("   \n\n"); got != "" {
		t.Fatalf("firstLine of blank = %q, want empty", got)
	}
}
