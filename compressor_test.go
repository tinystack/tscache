package tscache

import (
	"testing"
)

func TestGzipCompressor(t *testing.T) {
	compressor := NewGzipCompressor()
	data := []byte("test gzip data")
	compressed, err := compressor.Compress(data)
	if err != nil {
		t.Fatalf("Gzip compression failed: %v", err)
	}
	decompressed, err := compressor.Decompress(compressed)
	if err != nil {
		t.Fatalf("Gzip decompression failed: %v", err)
	}
	if string(decompressed) != string(data) {
		t.Error("Decompressed data doesn't match original")
	}
}

func TestZstdCompressor(t *testing.T) {
	compressor, err := NewZstdCompressor()
	if err != nil {
		t.Fatalf("Failed to create Zstd compressor: %v", err)
	}
	defer compressor.Close()
	data := []byte("This is a test string for Zstd compression that should be large enough to see compression benefits.")
	compressed, err := compressor.Compress(data)
	if err != nil {
		t.Fatalf("Zstd compression failed: %v", err)
	}
	decompressed, err := compressor.Decompress(compressed)
	if err != nil {
		t.Fatalf("Zstd decompression failed: %v", err)
	}
	if string(decompressed) != string(data) {
		t.Errorf("Decompressed data doesn't match original: got %v, want %v", string(decompressed), string(data))
	}
}

func TestNoCompressor(t *testing.T) {
	compressor := NewNoCompressor()
	data := []byte("This is test data without compression")
	compressed, err := compressor.Compress(data)
	if err != nil {
		t.Fatalf("No-compression failed: %v", err)
	}
	decompressed, err := compressor.Decompress(compressed)
	if err != nil {
		t.Fatalf("No-decompression failed: %v", err)
	}
	if string(decompressed) != string(data) {
		t.Errorf("Data doesn't match original: got %v, want %v", string(decompressed), string(data))
	}
}

func TestCreateCompressor(t *testing.T) {
	testCases := []struct {
		name       string
		compressor Compressor
		needsClose bool
	}{
		{"Gzip", NewGzipCompressor(), false},
		{"None", NewNoCompressor(), false},
	}

	// Add Zstd test case if creation succeeds
	if zstdComp, err := NewZstdCompressor(); err == nil {
		testCases = append(testCases, struct {
			name       string
			compressor Compressor
			needsClose bool
		}{"Zstd", zstdComp, true})
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			compressor := tc.compressor
			if compressor == nil {
				t.Error("Compressor is nil")
				return
			}

			// Test basic functionality
			testData := []byte("test data")
			compressed, err := compressor.Compress(testData)
			if err != nil {
				t.Fatalf("Compression failed: %v", err)
			}

			decompressed, err := compressor.Decompress(compressed)
			if err != nil {
				t.Fatalf("Decompression failed: %v", err)
			}

			if string(decompressed) != string(testData) {
				t.Errorf("Data mismatch: got %v, want %v", string(decompressed), string(testData))
			}

			// Clean up if needed
			if tc.needsClose {
				if zstdComp, ok := compressor.(*ZstdCompressor); ok {
					zstdComp.Close()
				}
			}
		})
	}
}

func TestNewCacheWithDifferentCompressors(t *testing.T) {
	testCases := []struct {
		name        string
		compressor  Compressor
		needsClose  bool
		expectError bool
	}{
		{"Gzip", NewGzipCompressor(), false, false},
		{"None", NewNoCompressor(), false, false},
	}

	// Add Zstd test case if creation succeeds
	if zstdComp, err := NewZstdCompressor(); err == nil {
		testCases = append(testCases, struct {
			name        string
			compressor  Compressor
			needsClose  bool
			expectError bool
		}{"Zstd", zstdComp, true, false})
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache := NewCache(
				WithMaxSize(1024*1024),
				WithEvictionPolicy("LRU"),
				WithCompressor(tc.compressor),
			)

			if cache == nil {
				t.Error("NewCache returned nil")
				return
			}

			// Test basic cache operations
			err := cache.Set("test_key", []byte("test_value"), 0)
			if err != nil {
				t.Fatalf("Cache.Set failed: %v", err)
			}

			value, err := cache.Get("test_key")
			if err != nil {
				t.Fatalf("Cache.Get failed: %v", err)
			}

			if string(value) != "test_value" {
				t.Errorf("Cache value mismatch: got %v, want %v", string(value), "test_value")
			}

			// Clean up if needed
			if tc.needsClose {
				if zstdComp, ok := tc.compressor.(*ZstdCompressor); ok {
					zstdComp.Close()
				}
			}
		})
	}
}
