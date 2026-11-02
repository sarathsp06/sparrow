package template

import (
	"encoding/json"
	"strings"
	"testing"
	"text/template"
	"time"
)

// render executes tmpl with the plain helper set (FuncMap, no budget
// wrappers) so each helper's own behaviour is what gets tested.
func render(t *testing.T, tmpl string, data any) (string, error) {
	t.Helper()
	tp, err := template.New("t").Funcs(FuncMap()).Parse(tmpl)
	if err != nil {
		t.Fatalf("parse %q: %v", tmpl, err)
	}
	var b strings.Builder
	err = tp.Execute(&b, data)
	return b.String(), err
}

func TestHelpersRender(t *testing.T) {
	data := map[string]any{
		"s":     "  Hello World  ",
		"name":  "ada lovelace",
		"n":     float64(7),
		"items": []any{"a", "b"},
		"user":  map[string]any{"address": map[string]any{"city": "Pune"}, "tags": "x"},
		"num":   json.Number("2.5"),
		"empty": "",
		"nil":   nil,
		"words": []string{"x", "y", "z"},
	}
	cases := []struct {
		tmpl, want string
	}{
		{`{{ json .items }}`, `["a","b"]`},
		{`{{ urlencode "a b@c" }}`, `a+b%40c`},
		{`{{ base64 "hi" }}`, `aGk=`},
		{`{{ base64decode "aGk=" }}`, `hi`},
		{`{{ formatTime "2006-01-02" (parseTime "2006-01-02T15:04:05Z07:00" "2024-03-05T10:00:00Z") }}`, `2024-03-05`},
		{`{{ upper "ab" }}{{ lower "CD" }}{{ title .name }}`, `ABcdAda Lovelace`},
		{`{{ trim "." "..x.." }}|{{ trimSpace .s }}`, `x|Hello World`},
		{`{{ join "-" (split "," "a,b,c") }}`, `a-b-c`},
		{`{{ join "+" .words }}`, `x+y+z`},
		{`{{ default "d" .empty }}{{ default "d" .nil }}{{ default "d" "v" }}`, `ddv`},
		{`{{ slice 1 3 "abcd" }}|{{ slice 2 99 "abcd" }}|{{ slice 9 10 "abcd" }}|{{ slice 3 1 "abcd" }}|{{ slice -1 2 "ab" }}`, `bc|cd|||`},
		{`{{ len "abc" }}{{ len .items }}{{ len .user }}{{ len .n }}`, `3220`},
		{`{{ truncate 2 "abcd" }}|{{ truncate 9 "ab" }}`, `ab|ab`},
		{`{{ ellipsis 5 "abcdefgh" }}|{{ ellipsis 2 "abcdefgh" }}|{{ ellipsis 9 "abc" }}`, `ab...|ab|abc`},
		{`{{ repeat 3 "ab" }}|{{ repeat -2 "ab" }}|`, `ababab||`},
		{`{{ contains "ell" "hello" }}{{ hasPrefix "he" "hello" }}{{ hasSuffix "lo" "hello" }}`, `truetruetrue`},
		{`{{ replace "o" "0" "foo" }}`, `f00`},
		{`{{ $d := dict "a" 1 "b" "two" }}{{ index $d "b" }}`, `two`},
		{`{{ $l := append (list 1 2) 3 }}{{ len $l }}`, `3`},
		{`{{ $m := merge (dict "a" 1) (dict "a" 2 "b" 3) }}{{ index $m "a" }}{{ index $m "b" }}`, `23`},
		{`{{ add .n 1 }} {{ sub .n "2" }} {{ mul .num 2 }} {{ div .n 2 }} {{ mod .n 4 }}`, `8 5 5 3.5 3`},
		{`{{ add 1.5 (toFloat "2") }} {{ toInt "9.7" }} {{ toString .n }}`, `3.5 9 7`},
		{`{{ dig "address" "city" "?" .user }}|{{ dig "address" "zip" "?" .user }}|{{ dig "tags" "inner" "?" .user }}|{{ dig "missing" "?" .user }}`, `Pune|?|?|?`},
	}
	for _, c := range cases {
		got, err := render(t, c.tmpl, data)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", c.tmpl, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %q, want %q", c.tmpl, got, c.want)
		}
	}
}

func TestHelpersReportErrors(t *testing.T) {
	data := map[string]any{"m": map[string]any{"a": 1}, "obj": struct{}{}}
	cases := map[string]string{
		`{{ base64decode "%%%" }}`:          "illegal base64",
		`{{ repeat 1001 "x" }}`:             "exceeds maximum",
		`{{ dict "a" }}`:                    "even number",
		`{{ dict 1 2 }}`:                    "keys must be strings",
		`{{ add "x" 1 }}`:                   "cannot convert",
		`{{ add 1 "x" }}`:                   "cannot convert",
		`{{ sub "x" 1 }}`:                   "cannot convert",
		`{{ sub 1 "x" }}`:                   "cannot convert",
		`{{ mul "x" 1 }}`:                   "cannot convert",
		`{{ mul 1 "x" }}`:                   "cannot convert",
		`{{ div "x" 1 }}`:                   "cannot convert",
		`{{ div 1 "x" }}`:                   "cannot convert",
		`{{ div 1 0 }}`:                     "division by zero",
		`{{ mod "x" 1 }}`:                   "cannot convert",
		`{{ mod 1 "x" }}`:                   "cannot convert",
		`{{ mod 1 0 }}`:                     "modulo by zero",
		`{{ toInt .obj }}`:                  "cannot convert",
		`{{ dig "a" .m }}`:                  "at least one key",
		`{{ dig "a" "d" "notamap" }}`:       "must be a map",
		`{{ dig 1 "d" .m }}`:                "keys must be strings",
		`{{ parseTime "2006" "notayear" }}`: "cannot parse",
	}
	for tmpl, want := range cases {
		_, err := render(t, tmpl, data)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: error = %v, want it to mention %q", tmpl, err, want)
		}
	}
}

func TestToNumberAcceptsNumericTypes(t *testing.T) {
	for _, v := range []any{float64(2), float32(2), 2, int64(2), int32(2), json.Number("2"), " 2 "} {
		if f, err := toNumber(v); err != nil || f != 2 {
			t.Errorf("toNumber(%#v) = %v, %v; want 2", v, f, err)
		}
	}
	if _, err := toNumber(json.Number("x")); err == nil {
		t.Error("toNumber(json.Number(x)) succeeded, want an error")
	}
}

func TestNowIsCurrent(t *testing.T) {
	now := FuncMap()["now"].(func() time.Time)()
	if d := time.Since(now); d < 0 || d > time.Minute {
		t.Errorf("now() is %v away from the current time", d)
	}
}

func TestEveryHelperIsDocumentedOnce(t *testing.T) {
	seen := map[string]bool{}
	for _, f := range Functions() {
		if f.Name == "" || f.Func == nil {
			t.Errorf("helper %q has no name or function", f.Name)
		}
		if seen[f.Name] {
			t.Errorf("helper %q is listed twice", f.Name)
		}
		seen[f.Name] = true
	}
}

func TestTrimMatchesStringsTrim(t *testing.T) {
	for _, c := range []struct{ cutset, s string }{
		{" ", "  hello  "}, {".", "...a.b..."}, {"ü", "üüaüü"}, {"éü ", " éüaéb ü"},
		{"ü", "aüa"}, {"ü", ""}, {"", "ü"}, {"\xff", "\xffa\xff"},
	} {
		if got, want := trim(c.cutset, c.s), strings.Trim(c.s, c.cutset); got != want {
			t.Errorf("trim(%q, %q) = %q, want %q", c.cutset, c.s, got, want)
		}
	}
}

// strings.Trim with this cutset and string took about 20s of CPU.
func TestTrimLongUnicodeCutsetIsFast(t *testing.T) {
	cutset := strings.Repeat("ü", 1_000_000) + "é"
	s := strings.Repeat("é", 1_000_000)
	start := time.Now()
	if got := trim(cutset, s); got != "" {
		t.Fatalf("got %d bytes, want everything trimmed", len(got))
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("took %v", d)
	}
}
