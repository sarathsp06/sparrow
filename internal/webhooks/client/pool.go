package client

import (
	"bytes"
	"sync"
)

// bufferPool is a sync.Pool for reusing byte buffers
var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// GetBuffer retrieves a buffer from the pool
func GetBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// PutBuffer returns a buffer to the pool
func PutBuffer(buf *bytes.Buffer) {
	if buf == nil {
		return
	}
	// Don't pool excessively large buffers (>64KB)
	if buf.Cap() > 64*1024 {
		return
	}
	buf.Reset()
	bufferPool.Put(buf)
}

// byteSlicePool is a sync.Pool for reusing byte slices
var byteSlicePool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 0, 4096) // 4KB default capacity
		return &b
	},
}

// GetByteSlice retrieves a byte slice from the pool
func GetByteSlice() *[]byte {
	return byteSlicePool.Get().(*[]byte)
}

// PutByteSlice returns a byte slice to the pool
func PutByteSlice(b *[]byte) {
	if b == nil {
		return
	}
	// Don't pool excessively large slices (>64KB)
	if cap(*b) > 64*1024 {
		return
	}
	*b = (*b)[:0]
	byteSlicePool.Put(b)
}

// headerMapPool is a sync.Pool for reusing header maps
var headerMapPool = sync.Pool{
	New: func() interface{} {
		return make(map[string]string, 8) // 8 headers typical capacity
	},
}

// GetHeaderMap retrieves a header map from the pool
func GetHeaderMap() map[string]string {
	return headerMapPool.Get().(map[string]string)
}

// PutHeaderMap returns a header map to the pool
func PutHeaderMap(m map[string]string) {
	if m == nil {
		return
	}
	// Clear the map
	for k := range m {
		delete(m, k)
	}
	headerMapPool.Put(m)
}
