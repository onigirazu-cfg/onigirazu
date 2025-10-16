package state

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// CompressionConfig holds compression configuration
type CompressionConfig struct {
	Enabled           bool   // Enable compression
	Algorithm         string // "gzip" (only option for now)
	CompressionLevel  int    // 0-9, where 0 is no compression and 9 is max
	FileSizeThreshold int64  // Only compress if file size exceeds this threshold (in bytes)
}

// DefaultCompressionConfig returns sensible defaults
func DefaultCompressionConfig() *CompressionConfig {
	return &CompressionConfig{
		Enabled:           true,
		Algorithm:         "gzip",
		CompressionLevel:  6,          // Balance between speed and compression
		FileSizeThreshold: 100 * 1024, // 100KB
	}
}

// CompressionManager handles state compression and decompression
type CompressionManager struct {
	config *CompressionConfig
}

// NewCompressionManager creates a new compression manager
func NewCompressionManager(config *CompressionConfig) *CompressionManager {
	if config == nil {
		config = DefaultCompressionConfig()
	}
	return &CompressionManager{
		config: config,
	}
}

// CompressState compresses a state to bytes (JSON + gzip if enabled)
func (cm *CompressionManager) CompressState(state *types.State) ([]byte, error) {
	// Marshal state to JSON
	jsonData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state to JSON: %w", err)
	}

	// Check if compression should be applied
	if !cm.config.Enabled || int64(len(jsonData)) < cm.config.FileSizeThreshold {
		// Return uncompressed
		return jsonData, nil
	}

	// Apply gzip compression
	var compressed bytes.Buffer
	gz, err := gzip.NewWriterLevel(&compressed, cm.config.CompressionLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip writer: %w", err)
	}

	if _, err := gz.Write(jsonData); err != nil {
		gz.Close()
		return nil, fmt.Errorf("failed to write data to gzip: %w", err)
	}

	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Mark state as compressed before returning
	state.Compressed = true

	return compressed.Bytes(), nil
}

// DecompressState decompresses state from bytes (auto-detects JSON vs gzip)
func (cm *CompressionManager) DecompressState(data []byte) (*types.State, error) {
	// Try to detect if data is gzip compressed
	// Gzip files start with magic bytes: 0x1f 0x8b
	isGzip := len(data) > 1 && data[0] == 0x1f && data[1] == 0x8b

	if isGzip {
		// Decompress gzip
		gr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gr.Close()

		// Read decompressed data
		decompressed, err := io.ReadAll(gr)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress gzip data: %w", err)
		}

		data = decompressed
	}

	// Unmarshal JSON
	state := &types.State{}
	if err := json.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state JSON: %w", err)
	}

	return state, nil
}

// CompressFile compresses a state file
func (cm *CompressionManager) CompressFile(inputPath, outputPath string) error {
	// Read input file
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Decompress first if already compressed
	state, err := cm.DecompressState(data)
	if err != nil {
		// If decompression fails, try direct JSON unmarshal
		if err := json.Unmarshal(data, state); err != nil {
			return fmt.Errorf("failed to load state: %w", err)
		}
	}

	// Compress state
	compressed, err := cm.CompressState(state)
	if err != nil {
		return fmt.Errorf("failed to compress state: %w", err)
	}

	// Write output file
	if err := os.WriteFile(outputPath, compressed, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

// GetCompressionStats returns compression statistics for a state
type CompressionStats struct {
	UncompressedSize int64
	CompressedSize   int64
	CompressionRatio float64
	Savings          int64
}

// GetStats calculates compression statistics
func (cm *CompressionManager) GetStats(state *types.State) (*CompressionStats, error) {
	// Get uncompressed size
	jsonData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	uncompressedSize := int64(len(jsonData))

	// Get compressed size
	var compressed bytes.Buffer
	gz, err := gzip.NewWriterLevel(&compressed, cm.config.CompressionLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip writer: %w", err)
	}

	if _, err := gz.Write(jsonData); err != nil {
		gz.Close()
		return nil, fmt.Errorf("failed to write to gzip: %w", err)
	}

	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip: %w", err)
	}

	compressedSize := int64(compressed.Len())
	savings := uncompressedSize - compressedSize
	ratio := float64(compressedSize) / float64(uncompressedSize)

	return &CompressionStats{
		UncompressedSize: uncompressedSize,
		CompressedSize:   compressedSize,
		CompressionRatio: ratio,
		Savings:          savings,
	}, nil
}

// IsCompressed checks if data is gzip compressed
func IsCompressed(data []byte) bool {
	return len(data) > 1 && data[0] == 0x1f && data[1] == 0x8b
}
