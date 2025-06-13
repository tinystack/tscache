// Package tscache provides a high-performance, thread-safe, in-memory cache library for Go.
//
// TSCache is designed for production use with features including:
// - Multiple eviction policies (LRU, LFU, FIFO)
// - Memory-based size limits with automatic eviction
// - TTL (Time To Live) support for cache entries
// - Data compression for large values
// - Sharded architecture for reduced lock contention
// - Thread-safe concurrent access
// - Comprehensive statistics and monitoring
//
// Example usage:
//
//	cache := tscache.NewCache(1024*1024, 0, "LRU") // 1MB cache with LRU eviction
//	err := cache.Set("key", "value", 10*time.Second) // Set with 10s TTL
//	value, err := cache.Get("key")
//	err = cache.Delete("key")
//	stats := cache.Stats()
package tscache

import (
	"time"
)

// Eviction policy constants
const (
	// EvictionLRU represents Least Recently Used eviction policy
	EvictionLRU = "LRU"
	// EvictionLFU represents Least Frequently Used eviction policy
	EvictionLFU = "LFU"
	// EvictionFIFO represents First In First Out eviction policy
	EvictionFIFO = "FIFO"
)

// Option defines a function type for configuring cache options
type Option func(*cacheOptions)

// cacheOptions holds the configuration options for creating a cache
type cacheOptions struct {
	maxSize        int        // Maximum memory usage in bytes
	evictionPolicy string     // Eviction policy
	compressor     Compressor // Compression algorithm
	compressSize   int        // Compression size threshold
}

// WithMaxSize sets the maximum memory size for the cache
func WithMaxSize(size int) Option {
	return func(opts *cacheOptions) {
		opts.maxSize = size
	}
}

// WithCompressSize sets the compression size threshold for the cache
func WithCompressSize(size int) Option {
	return func(opts *cacheOptions) {
		opts.compressSize = size
	}
}

// WithEvictionPolicy sets the eviction policy for the cache
func WithEvictionPolicy(policy string) Option {
	return func(opts *cacheOptions) {
		opts.evictionPolicy = policy
	}
}

// WithCompressor sets the compression algorithm for the cache
func WithCompressor(compressor Compressor) Option {
	return func(opts *cacheOptions) {
		opts.compressor = compressor
	}
}

// Cache represents a thread-safe, in-memory cache with configurable eviction policies.
// It uses a sharded architecture to reduce lock contention and improve concurrent performance.
// The cache supports memory-based size limits, TTL expiration, and automatic data compression.
type Cache struct {
	maxSize        int           // Maximum memory usage in bytes
	evictionPolicy string        // Eviction policy
	shards         []*CacheShard // Cache shards
	shardCount     int           // Number of cache shards
}

// Stats holds comprehensive statistics for cache performance monitoring and analysis.
// All fields are thread-safe and updated atomically across all cache operations.
type Stats struct {
	Hits           int    // Total number of successful cache hits
	Misses         int    // Total number of cache misses
	Evictions      int    // Total number of items evicted due to policies
	CurrentCount   int    // Current number of items in cache
	CurrentSize    int    // Current total memory usage in bytes
	MaxSize        int    // Maximum allowed memory size in bytes
	EvictionPolicy string // Current eviction policy name
	ShardCount     int    // Number of cache shards
}

// NewCache creates a new cache instance with configurable options.
//
// Parameters:
//   - opts: Variadic functional options to configure the cache
//
// Available options:
//   - WithMaxSize(size int64): Set maximum memory usage in bytes (default: 100MB)
//   - WithEvictionPolicy(policy string): Set eviction policy ("LRU", "LFU", or "FIFO") (default: "LRU")
//   - WithCompressor(compressor string): Set compression algorithm ("gzip", "zstd", "none") (default: "gzip")
//
// Returns:
//   - *Cache: A new cache instance ready for use
//
// Example usage:
//
//	cache := NewCache(WithMaxSize(100*1024*1024), WithEvictionPolicy("LRU"), WithCompressor("zstd"))
//
// The cache automatically determines the optimal number of shards based on the system's CPU count
// to maximize concurrent performance. Default values are used for any unspecified options.
func NewCache(opts ...Option) *Cache {
	// Apply default options
	options := &cacheOptions{
		maxSize:        1024 * 1024 * 100, // Default: 100MB
		evictionPolicy: EvictionLRU,       // Default: LRU
		compressor:     NewNoCompressor(), // Default: NoCompressor
		compressSize:   1024 * 1024,       // Default: 1MB
	}

	// Apply provided options
	for _, opt := range opts {
		opt(options)
	}

	// Validate and normalize eviction policy
	switch options.evictionPolicy {
	case EvictionLRU, EvictionLFU, EvictionFIFO:
		// Valid policies - keep as-is
	default:
		options.evictionPolicy = EvictionLRU // Default to LRU for invalid policies
	}

	// Calculate optimal shard count based on system characteristics
	shardCount := getOptimalShardCount()

	// Create cache instance
	cache := &Cache{
		maxSize:        options.maxSize,
		evictionPolicy: options.evictionPolicy,
		shardCount:     shardCount,
		shards:         make([]*CacheShard, shardCount),
	}

	// Initialize each shard with proportional memory limit
	shardMaxSize := options.maxSize / shardCount
	if shardMaxSize == 0 && options.maxSize > 0 {
		shardMaxSize = 1 // Ensure each shard has at least 1 byte limit
	}

	for i := 0; i < shardCount; i++ {
		cache.shards[i] = NewCacheShard(shardMaxSize, options.evictionPolicy, options.compressor, options.compressSize)
	}

	return cache
}

// Set stores a key-value pair in the cache with an optional TTL (Time To Live).
//
// Parameters:
//   - key: The cache key (must be non-empty string)
//   - value: The value to store (must be []byte)
//   - ttl: Time to live duration (0 for no expiration)
//
// Returns:
//   - error: nil on success, error if operation fails
//
// The value will be automatically compressed if it's large enough to benefit from compression.
// If the cache is full, old items may be evicted according to the configured eviction policy.
func (c *Cache) Set(key string, value []byte, ttl time.Duration) error {
	shard := c.getShard(key)
	return shard.Set(key, value, ttl)
}

// Get retrieves a value from the cache by key.
//
// Parameters:
//   - key: The cache key to lookup
//
// Returns:
//   - []byte: The cached value (nil if not found)
//   - error: nil if found, error if key doesn't exist or has expired
//
// This operation updates the access statistics for eviction policy decisions.
// Expired items are automatically removed from the cache during retrieval.
func (c *Cache) Get(key string) ([]byte, error) {
	shard := c.getShard(key)
	return shard.Get(key)
}

// Delete removes a key-value pair from the cache.
//
// Parameters:
//   - key: The cache key to remove
//
// This operation immediately frees the memory used by the cached item.
func (c *Cache) Delete(key string) {
	shard := c.getShard(key)
	shard.Delete(key)
}

// Clear removes all items from the cache across all shards.
//
// This is an atomic operation that will clear all shards.
func (c *Cache) Clear() {
	// Clear all shards
	for _, shard := range c.shards {
		shard.Clear()
	}
}

// Stats returns a snapshot of current cache statistics.
//
// Returns:
//   - Stats: Current cache performance and usage statistics
//
// The returned statistics provide insights into cache performance, hit rates,
// memory usage, and eviction patterns. All values represent the cumulative
// state across all cache shards.
func (c *Cache) Stats() Stats {
	var totalHits, totalMisses, totalEvictions int
	var totalCurrentCount, totalCurrentSize int

	// Aggregate statistics from all shards
	for _, shard := range c.shards {
		shardStats := shard.getStats()
		totalHits += shardStats.Hits
		totalMisses += shardStats.Misses
		totalEvictions += shardStats.Evictions
		totalCurrentCount += shardStats.CurrentCount
		totalCurrentSize += shardStats.CurrentSize
	}

	// Return aggregated statistics
	return Stats{
		Hits:           totalHits,
		Misses:         totalMisses,
		Evictions:      totalEvictions,
		CurrentCount:   totalCurrentCount,
		CurrentSize:    totalCurrentSize,
		MaxSize:        c.maxSize,
		EvictionPolicy: c.evictionPolicy,
		ShardCount:     c.shardCount,
	}
}

// getShard determines which shard should handle a given key using consistent hashing.
//
// Parameters:
//   - key: The cache key to hash
//
// Returns:
//   - *CacheShard: The appropriate shard for this key
//
// This method uses FNV-1a hashing to distribute keys evenly across shards,
// ensuring consistent shard assignment for the same key across operations.
func (c *Cache) getShard(key string) *CacheShard {
	// Use FNV-1a hash for good distribution properties
	hash := fnv1a(key)

	// Use bitwise AND for efficient modulo when shard count is power of 2
	// For non-power-of-2 shard counts, fall back to regular modulo
	shardIndex := int(hash) % c.shardCount

	return c.shards[shardIndex]
}
