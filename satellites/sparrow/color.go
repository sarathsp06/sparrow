package main

import (
	"io"
	"os"
)

// palette applies ANSI colors only when the writer is a real terminal and
// NO_COLOR is unset (https://no-color.org). Tests write to a buffer, so output
// stays plain and assertions keep matching.
type palette struct{ on bool }

func newPalette(w io.Writer) palette {
	if os.Getenv("NO_COLOR") != "" {
		return palette{}
	}
	f, ok := w.(*os.File)
	return palette{on: ok && isTerminal(f)}
}

func (p palette) paint(code, s string) string {
	if !p.on {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func (p palette) bold(s string) string   { return p.paint("1", s) }
func (p palette) dim(s string) string    { return p.paint("2", s) }
func (p palette) red(s string) string    { return p.paint("31", s) }
func (p palette) green(s string) string  { return p.paint("32", s) }
func (p palette) yellow(s string) string { return p.paint("33", s) }

// status colors a delivery status word (already padded to a column width).
func (p palette) status(status, cell string) string {
	switch status {
	case "success":
		return p.green(cell)
	case "failed":
		return p.red(cell)
	case "pending", "retrying":
		return p.yellow(cell)
	default:
		return cell
	}
}
