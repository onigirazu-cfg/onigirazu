package bufferpool

import (
	"bytes"
	"strings"
	"sync"
)

// BytesBufferPool is a pool of bytes.Buffer objects
var BytesBufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// StringsBuilderPool is a pool of strings.Builder objects
var StringsBuilderPool = sync.Pool{
	New: func() interface{} {
		return new(strings.Builder)
	},
}

// GetBytesBuffer gets a buffer from the pool
func GetBytesBuffer() *bytes.Buffer {
	buf := BytesBufferPool.Get().(*bytes.Buffer)
	buf.Reset() // Ensure buffer is clean
	return buf
}

// PutBytesBuffer returns a buffer to the pool
func PutBytesBuffer(buf *bytes.Buffer) {
	if buf == nil {
		return
	}
	// Don't pool buffers that are too large (> 64KB)
	// This prevents memory bloat from occasional large buffers
	if buf.Cap() > 64*1024 {
		return
	}
	buf.Reset()
	BytesBufferPool.Put(buf)
}

// GetStringsBuilder gets a strings.Builder from the pool
func GetStringsBuilder() *strings.Builder {
	sb := StringsBuilderPool.Get().(*strings.Builder)
	sb.Reset() // Ensure builder is clean
	return sb
}

// PutStringsBuilder returns a strings.Builder to the pool
func PutStringsBuilder(sb *strings.Builder) {
	if sb == nil {
		return
	}
	// Don't pool builders that are too large (> 64KB)
	if sb.Cap() > 64*1024 {
		return
	}
	sb.Reset()
	StringsBuilderPool.Put(sb)
}

// WithBytesBuffer executes a function with a pooled bytes.Buffer
// and automatically returns it to the pool
func WithBytesBuffer(fn func(*bytes.Buffer) error) error {
	buf := GetBytesBuffer()
	defer PutBytesBuffer(buf)
	return fn(buf)
}

// WithStringsBuilder executes a function with a pooled strings.Builder
// and automatically returns it to the pool
func WithStringsBuilder(fn func(*strings.Builder) error) error {
	sb := GetStringsBuilder()
	defer PutStringsBuilder(sb)
	return fn(sb)
}

// BytesBufferFunc is a helper that returns the buffer's string content
func BytesBufferFunc(fn func(*bytes.Buffer) error) (string, error) {
	buf := GetBytesBuffer()
	defer PutBytesBuffer(buf)

	if err := fn(buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// StringsBuilderFunc is a helper that returns the builder's string content
func StringsBuilderFunc(fn func(*strings.Builder) error) (string, error) {
	sb := GetStringsBuilder()
	defer PutStringsBuilder(sb)

	if err := fn(sb); err != nil {
		return "", err
	}

	return sb.String(), nil
}
