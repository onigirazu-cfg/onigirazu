package bufferpool

import (
	"bytes"
	"strings"
	"testing"
)

func TestGetBytesBuffer(t *testing.T) {
	buf := GetBytesBuffer()
	if buf == nil {
		t.Fatal("Expected buffer, got nil")
	}

	if buf.Len() != 0 {
		t.Errorf("Expected empty buffer, got length %d", buf.Len())
	}

	PutBytesBuffer(buf)
}

func TestPutBytesBuffer_Nil(t *testing.T) {
	// Should not panic
	PutBytesBuffer(nil)
}

func TestBytesBufferReuse(t *testing.T) {
	// Get buffer and write to it
	buf1 := GetBytesBuffer()
	buf1.WriteString("test data")

	// Return to pool
	PutBytesBuffer(buf1)

	// Get another buffer (should be the same one, but reset)
	buf2 := GetBytesBuffer()

	if buf2.Len() != 0 {
		t.Errorf("Expected empty buffer after reuse, got length %d", buf2.Len())
	}

	PutBytesBuffer(buf2)
}

func TestBytesBuffer_LargeBuffer(t *testing.T) {
	buf := GetBytesBuffer()

	// Write more than 64KB
	largeData := make([]byte, 65*1024)
	buf.Write(largeData)

	// Should not be returned to pool (too large)
	PutBytesBuffer(buf)

	// Get new buffer - should be a fresh one
	buf2 := GetBytesBuffer()
	if buf2.Cap() > 64*1024 {
		t.Error("Expected fresh buffer, got large buffer from pool")
	}

	PutBytesBuffer(buf2)
}

func TestGetStringsBuilder(t *testing.T) {
	sb := GetStringsBuilder()
	if sb == nil {
		t.Fatal("Expected builder, got nil")
	}

	if sb.Len() != 0 {
		t.Errorf("Expected empty builder, got length %d", sb.Len())
	}

	PutStringsBuilder(sb)
}

func TestPutStringsBuilder_Nil(t *testing.T) {
	// Should not panic
	PutStringsBuilder(nil)
}

func TestStringsBuilderReuse(t *testing.T) {
	// Get builder and write to it
	sb1 := GetStringsBuilder()
	sb1.WriteString("test data")

	// Return to pool
	PutStringsBuilder(sb1)

	// Get another builder (should be the same one, but reset)
	sb2 := GetStringsBuilder()

	if sb2.Len() != 0 {
		t.Errorf("Expected empty builder after reuse, got length %d", sb2.Len())
	}

	PutStringsBuilder(sb2)
}

func TestStringsBuilder_LargeBuilder(t *testing.T) {
	sb := GetStringsBuilder()

	// Write more than 64KB
	largeData := strings.Repeat("a", 65*1024)
	sb.WriteString(largeData)

	// Should not be returned to pool (too large)
	PutStringsBuilder(sb)

	// Get new builder - should be a fresh one
	sb2 := GetStringsBuilder()
	if sb2.Cap() > 64*1024 {
		t.Error("Expected fresh builder, got large builder from pool")
	}

	PutStringsBuilder(sb2)
}

func TestWithBytesBuffer(t *testing.T) {
	var result string

	err := WithBytesBuffer(func(buf *bytes.Buffer) error {
		buf.WriteString("hello world")
		result = buf.String()
		return nil
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", result)
	}
}

func TestWithStringsBuilder(t *testing.T) {
	var result string

	err := WithStringsBuilder(func(sb *strings.Builder) error {
		sb.WriteString("hello world")
		result = sb.String()
		return nil
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", result)
	}
}

func TestBytesBufferFunc(t *testing.T) {
	result, err := BytesBufferFunc(func(buf *bytes.Buffer) error {
		buf.WriteString("test")
		buf.WriteString(" ")
		buf.WriteString("data")
		return nil
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result != "test data" {
		t.Errorf("Expected 'test data', got '%s'", result)
	}
}

func TestStringsBuilderFunc(t *testing.T) {
	result, err := StringsBuilderFunc(func(sb *strings.Builder) error {
		sb.WriteString("test")
		sb.WriteString(" ")
		sb.WriteString("data")
		return nil
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result != "test data" {
		t.Errorf("Expected 'test data', got '%s'", result)
	}
}

// Benchmark tests
func BenchmarkBytesBuffer_WithPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := GetBytesBuffer()
		buf.WriteString("test data")
		_ = buf.String()
		PutBytesBuffer(buf)
	}
}

func BenchmarkBytesBuffer_WithoutPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := new(bytes.Buffer)
		buf.WriteString("test data")
		_ = buf.String()
	}
}

func BenchmarkStringsBuilder_WithPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sb := GetStringsBuilder()
		sb.WriteString("test data")
		_ = sb.String()
		PutStringsBuilder(sb)
	}
}

func BenchmarkStringsBuilder_WithoutPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sb := new(strings.Builder)
		sb.WriteString("test data")
		_ = sb.String()
	}
}
