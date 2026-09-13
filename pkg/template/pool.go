package template

import (
	"bytes"
	"sync"
)

// bufferPool reuses byte buffers across template executions.
var bufferPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// getBuffer retrieves a reset buffer from the pool.
func getBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// putBuffer returns a buffer to the pool, dropping oversized ones (>64KB).
func putBuffer(buf *bytes.Buffer) {
	if buf == nil || buf.Cap() > 64*1024 {
		return
	}
	buf.Reset()
	bufferPool.Put(buf)
}
