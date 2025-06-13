package tscache

import (
	"bytes"
	"compress/gzip"
	"io"
)

// GzipCompressor implements the Compressor interface using gzip compression.
// It provides a good balance between compression ratio and CPU overhead,
// making it suitable for caching scenarios where memory is more valuable than CPU time.
type GzipCompressor struct{}

// NewGzipCompressor creates a new gzip-based compressor instance.
//
// Returns:
//   - *GzipCompressor: A new compressor ready for use
//
// The gzip compressor is thread-safe and can be used concurrently.
func NewGzipCompressor() *GzipCompressor {
	return &GzipCompressor{}
}

// Compress serializes the input data to JSON and compresses it using gzip.
//
// Parameters:
//   - data: The data to compress (any JSON-serializable type)
//
// Returns:
//   - []byte: Compressed data as byte slice
//   - error: nil on success, error if serialization or compression fails
//
// The compression process:
// 1. Serialize data to JSON format
// 2. Compress JSON bytes using gzip
// 3. Return compressed byte slice
//
// Large or repetitive data structures benefit most from this compression.
func (c *GzipCompressor) Compress(data []byte) ([]byte, error) {

	// Create a buffer to hold compressed data
	var compressedBuffer bytes.Buffer

	// Create gzip writer with default compression level
	gzipWriter := gzip.NewWriter(&compressedBuffer)

	// Write JSON data to the gzip writer
	_, err := gzipWriter.Write(data)
	if err != nil {
		gzipWriter.Close()
		return nil, err
	}

	// Close the gzip writer to flush all data
	err = gzipWriter.Close()
	if err != nil {
		return nil, err
	}

	return compressedBuffer.Bytes(), nil
}

// Decompress decompresses gzip data and deserializes it back to the original structure.
//
// Parameters:
//   - data: Compressed byte slice (must be gzip-compressed JSON)
//
// Returns:
//   - any: Deserialized data in its original form
//   - error: nil on success, error if decompression or deserialization fails
//
// The decompression process:
// 1. Decompress gzip data to get JSON bytes
// 2. Deserialize JSON back to Go data structures
// 3. Return the reconstructed data
func (c *GzipCompressor) Decompress(data []byte) ([]byte, error) {
	// Create a reader from the compressed data
	compressedReader := bytes.NewReader(data)

	// Create gzip reader for decompression
	gzipReader, err := gzip.NewReader(compressedReader)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	return io.ReadAll(gzipReader)
}
