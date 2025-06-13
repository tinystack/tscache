package tscache

// Compressor defines the interface for data compression implementations.
// Different compression algorithms can be plugged in by implementing this interface.
type Compressor interface {
	// Compress serializes and compresses the given data
	Compress(data []byte) ([]byte, error)
	// Decompress decompresses and deserializes data back to its original form
	Decompress(data []byte) ([]byte, error)
}

// NoCompressor implements the Compressor interface with no actual compression.
// It only handles serialization without compression, useful for small data or testing.
type NoCompressor struct{}

// NewNoCompressor creates a new no-compression compressor instance.
func NewNoCompressor() *NoCompressor {
	return &NoCompressor{}
}

// Compress only serializes the data without compression.
func (c *NoCompressor) Compress(data []byte) ([]byte, error) {
	return data, nil
}

// Decompress only deserializes the data without decompression.
func (c *NoCompressor) Decompress(data []byte) ([]byte, error) {
	return data, nil
}
