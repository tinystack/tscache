package tscache

import (
	"github.com/klauspost/compress/zstd"
)

// ZstdCompressor implements the Compressor interface using Zstandard compression.
// Zstd provides excellent compression ratios with high performance, making it ideal
// for high-throughput caching scenarios where both speed and compression efficiency matter.
type ZstdCompressor struct {
	encoder *zstd.Encoder
	decoder *zstd.Decoder
}

// NewZstdCompressor creates a new zstandard-based compressor instance.
//
// Returns:
//   - *ZstdCompressor: A new compressor ready for use
//   - error: nil on success, error if encoder/decoder creation fails
//
// The zstd compressor is thread-safe and provides better compression ratios
// and performance compared to gzip in most scenarios.
func NewZstdCompressor() (*ZstdCompressor, error) {
	// Create encoder with default compression level (level 3)
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return nil, err
	}

	// Create decoder for decompression
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		encoder.Close()
		return nil, err
	}

	return &ZstdCompressor{
		encoder: encoder,
		decoder: decoder,
	}, nil
}

// Close releases resources used by the compressor.
// This should be called when the compressor is no longer needed.
func (c *ZstdCompressor) Close() error {
	if c.encoder != nil {
		c.encoder.Close()
	}
	if c.decoder != nil {
		c.decoder.Close()
	}
	return nil
}

// Compress serializes the input data to JSON and compresses it using Zstandard.
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
// 2. Compress JSON bytes using Zstandard
// 3. Return compressed byte slice
//
// Zstd typically provides better compression ratios and performance than gzip.
func (c *ZstdCompressor) Compress(data []byte) ([]byte, error) {

	// Compress the JSON data using Zstandard
	compressed := c.encoder.EncodeAll(data, make([]byte, 0, len(data)))

	return compressed, nil
}

// Decompress decompresses Zstandard data and deserializes it back to the original structure.
//
// Parameters:
//   - data: Compressed byte slice (must be Zstandard-compressed JSON)
//
// Returns:
//   - any: Deserialized data in its original form
//   - error: nil on success, error if decompression or deserialization fails
//
// The decompression process:
// 1. Decompress Zstandard data to get JSON bytes
// 2. Deserialize JSON back to Go data structures
// 3. Return the reconstructed data
func (c *ZstdCompressor) Decompress(data []byte) ([]byte, error) {
	return c.decoder.DecodeAll(data, nil)
}
