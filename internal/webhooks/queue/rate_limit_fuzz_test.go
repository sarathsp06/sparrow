package queue

import (
	"strconv"
	"testing"
	"time"
)

// FuzzParseRetryAfter checks that a receiver's Retry-After header always
// yields a delay in (0, maxRetryAfter], and that a larger number of seconds
// never yields a shorter delay than a smaller one.
func FuzzParseRetryAfter(f *testing.F) {
	for _, s := range []string{"", "0", "1", "120", "899", "900", "3600", "-5", "30.5",
		"9223372036", "9223372037", "18446744074", "99999999999999999999",
		"Thu, 01 Dec 2025 16:00:00 GMT", "not-a-date"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, header string) {
		d := parseRetryAfter(header)
		if d <= 0 || d > maxRetryAfter {
			t.Fatalf("parseRetryAfter(%q) = %v, want in (0, %v]", header, d, maxRetryAfter)
		}
		n, err := strconv.Atoi(header)
		if err != nil || n <= 0 {
			return
		}
		if n < int(maxRetryAfter/time.Second) && d != time.Duration(n)*time.Second {
			t.Fatalf("parseRetryAfter(%q) = %v, want %ds", header, d, n)
		}
		if n >= int(maxRetryAfter/time.Second) && d != maxRetryAfter {
			t.Fatalf("parseRetryAfter(%q) = %v, want max %v", header, d, maxRetryAfter)
		}
	})
}
