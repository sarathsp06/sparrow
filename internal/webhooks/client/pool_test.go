package client

import (
	"bytes"
	"testing"
)

func TestBufferPool(t *testing.T) {
	t.Run("GetBuffer returns clean buffer", func(t *testing.T) {
		buf := GetBuffer()
		if buf == nil {
			t.Fatal("GetBuffer returned nil")
		}
		if buf.Len() != 0 {
			t.Errorf("Expected empty buffer, got length %d", buf.Len())
		}
	})

	t.Run("PutBuffer returns buffer to pool", func(t *testing.T) {
		buf1 := GetBuffer()
		buf1.WriteString("test data")
		PutBuffer(buf1)

		buf2 := GetBuffer()
		if buf2.Len() != 0 {
			t.Errorf("Expected buffer to be reset, got length %d", buf2.Len())
		}
		PutBuffer(buf2)
	})

	t.Run("Large buffers not returned to pool", func(t *testing.T) {
		buf := GetBuffer()
		// Write more than 64KB
		largeData := make([]byte, 65*1024)
		buf.Write(largeData)

		PutBuffer(buf) // Should not panic, just not pool it
	})

	t.Run("Nil buffer handled safely", func(t *testing.T) {
		PutBuffer(nil) // Should not panic
	})

	t.Run("Buffer reuse", func(t *testing.T) {
		buf1 := GetBuffer()
		ptr1 := buf1
		buf1.WriteString("first")
		PutBuffer(buf1)

		buf2 := GetBuffer()
		// Should get the same buffer back
		if buf2 != ptr1 {
			t.Log("Got different buffer (may happen with concurrent access)")
		}
		if buf2.String() != "" {
			t.Errorf("Expected empty buffer after reset, got: %s", buf2.String())
		}
		PutBuffer(buf2)
	})
}

func TestHeaderMapPool(t *testing.T) {
	t.Run("GetHeaderMap returns clean map", func(t *testing.T) {
		headers := GetHeaderMap()
		if headers == nil {
			t.Fatal("GetHeaderMap returned nil")
		}
		if len(headers) != 0 {
			t.Errorf("Expected empty map, got length %d", len(headers))
		}
	})

	t.Run("PutHeaderMap clears and returns map to pool", func(t *testing.T) {
		headers1 := GetHeaderMap()
		headers1["Content-Type"] = "application/json"
		headers1["Authorization"] = "Bearer token"
		PutHeaderMap(headers1)

		headers2 := GetHeaderMap()
		if len(headers2) != 0 {
			t.Errorf("Expected map to be cleared, got length %d", len(headers2))
		}
		PutHeaderMap(headers2)
	})

	t.Run("Nil map handled safely", func(t *testing.T) {
		PutHeaderMap(nil) // Should not panic
	})

	t.Run("Map reuse", func(t *testing.T) {
		headers1 := GetHeaderMap()
		headers1["X-Test"] = "value"
		PutHeaderMap(headers1)

		headers2 := GetHeaderMap()
		if _, exists := headers2["X-Test"]; exists {
			t.Error("Expected map to be cleared, but found old key")
		}
		PutHeaderMap(headers2)
	})
}

func BenchmarkBufferPool(b *testing.B) {
	b.Run("WithPool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := GetBuffer()
			buf.WriteString("test data for benchmark")
			_ = buf.Bytes()
			PutBuffer(buf)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := new(bytes.Buffer)
			buf.WriteString("test data for benchmark")
			_ = buf.Bytes()
		}
	})
}

func BenchmarkHeaderMapPool(b *testing.B) {
	b.Run("WithPool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			headers := GetHeaderMap()
			headers["Content-Type"] = "application/json"
			headers["Authorization"] = "Bearer token"
			headers["X-Request-ID"] = "123"
			_ = headers
			PutHeaderMap(headers)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			headers := make(map[string]string, 8)
			headers["Content-Type"] = "application/json"
			headers["Authorization"] = "Bearer token"
			headers["X-Request-ID"] = "123"
			_ = headers
		}
	})
}
