package rest

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// FuzzDecodeCursor feeds arbitrary cursor query parameters to decodeCursor:
// it must never panic, and any cursor it accepts must re-encode to one that
// decodes to the same position.
func FuzzDecodeCursor(f *testing.F) {
	f.Add(encodeCursor(time.Unix(1_700_000_000, 123_456_000), uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")))
	f.Add("")
	f.Add("not-base64!")
	f.Add("MTIzLm5vdC1hLXV1aWQ")          // "123.not-a-uuid"
	f.Add("LTkyMjMzNzIwMzY4NTQ3NzU4MDgu") // "-9223372036854775808."
	f.Fuzz(func(t *testing.T, s string) {
		c, err := decodeCursor(s)
		if err != nil || c == nil {
			return
		}
		again, err := decodeCursor(encodeCursor(c.CreatedAt, c.ID))
		if err != nil {
			t.Fatalf("re-encoded cursor rejected: %v", err)
		}
		if !again.CreatedAt.Equal(c.CreatedAt) || again.ID != c.ID {
			t.Fatalf("round trip changed cursor: %+v -> %+v", c, again)
		}
	})
}

// FuzzCursorRoundTrip checks encodeCursor/decodeCursor agree for every
// microsecond timestamp and id.
func FuzzCursorRoundTrip(f *testing.F) {
	f.Add(int64(0), []byte("0123456789abcdef"))
	f.Add(int64(-1), []byte("fedcba9876543210"))
	f.Add(int64(9_223_372_036_854_775_807), []byte("ffffffffffffffff"))
	f.Fuzz(func(t *testing.T, micros int64, idBytes []byte) {
		id, err := uuid.FromBytes(idBytes)
		if err != nil {
			return
		}
		at := time.UnixMicro(micros).UTC()
		c, err := decodeCursor(encodeCursor(at, id))
		if err != nil {
			t.Fatalf("decode(encode(%v, %s)): %v", at, id, err)
		}
		if c.CreatedAt.UnixMicro() != micros || c.ID != id {
			t.Fatalf("round trip: got (%d, %s), want (%d, %s)", c.CreatedAt.UnixMicro(), c.ID, micros, id)
		}
	})
}
