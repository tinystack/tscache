package tscache

import (
	"testing"
)

func TestGzipCompressor_CompressAndDecompress(t *testing.T) {
	compressor := NewGzipCompressor()
	data := []byte("hello world, this is a gzip test!")
	compressed, err := compressor.Compress(data)
	if err != nil {
		t.Fatalf("Compress error: %v", err)
	}
	if len(compressed) == 0 {
		t.Error("Compressed data should not be empty")
	}
	decompressed, err := compressor.Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompress error: %v", err)
	}
	if string(decompressed) != string(data) {
		t.Errorf("Decompressed data mismatch: got %s, want %s", string(decompressed), string(data))
	}
}
