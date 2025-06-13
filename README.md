# TSCache - High-Performance In-Memory Cache for Go

[🇨🇳 中文文档](README_CN.md) | [🇺🇸 English](README.md)

TSCache is a high-performance, thread-safe in-memory cache library for Go applications. It provides advanced features like memory management, multiple eviction policies, data compression, and automatic sharding.

[![Go Report Card](https://goreportcard.com/badge/github.com/tinystack/tscache)](https://goreportcard.com/report/github.com/tinystack/tscache)
![Go Version](https://img.shields.io/badge/go%20version-%3E=1.22.0-61CFDD.svg?style=flat-square)
[![PkgGoDev](https://pkg.go.dev/badge/mod/github.com/tinystack/tscache)](https://pkg.go.dev/mod/github.com/tinystack/tscache)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- **Memory Management**: Set maximum memory usage with automatic eviction when limits are exceeded
- **Data Expiration**: Support for TTL (Time To Live) with automatic cleanup of expired items
- **Multiple Eviction Policies**: LRU (Least Recently Used), LFU (Least Frequently Used), and FIFO (First In, First Out)
- **Thread-Safe**: Concurrent access support with optimized locking strategies
- **Sharded Cache**: Automatic sharding to reduce lock contention and improve performance
- **Memory Optimization**: Efficient data structures with O(1) or O(log n) complexity
- **Data Compression**: Automatic compression for large data to reduce memory usage
- **High-Performance Statistics**: Optimized per-shard statistics with lock-free aggregation for minimal performance impact
- **Memory-Only Eviction**: Eviction is based solely on memory usage, no item count limits

## Installation

```bash
go get github.com/tinystack/tscache
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"
    "github.com/tinystack/tscache"
)

func main() {
    // Create cache with 10MB max memory and LRU eviction using function options
    cache := tscache.NewCache(tscache.WithMaxSize(10*1024*1024), tscache.WithEvictionPolicy("LRU"))

    // Set a value without expiration
    cache.Set("user:1", []byte("Alice"), 0)

    // Set a value with 5-minute TTL
    cache.Set("session:abc", []byte("user_data"), 5*time.Minute)

    // Get a value
    if value, err := cache.Get("user:1"); err == nil {
        fmt.Printf("User: %s\n", value)
    }

    // Delete a value
    cache.Delete("user:1")

    // Get cache statistics
    stats := cache.Stats()
    fmt.Printf("Hits: %d, Misses: %d, Items: %d\n",
        stats.Hits, stats.Misses, stats.CurrentCount)

    // Clear all cache
    cache.Clear()
}
```

## API Reference

### Creating Cache

```go
func NewCache(opts ...Option) *Cache
```

Using function options pattern for flexible configuration:

```go
// With explicit options
cache := tscache.NewCache(
    tscache.WithMaxSize(100*1024*1024),    // 100MB max memory
    tscache.WithEvictionPolicy("LRU"),     // LRU eviction policy
)

// With default values (100MB, LRU)
cache := tscache.NewCache()

// With single option (using defaults for others)
cache := tscache.NewCache(tscache.WithMaxSize(50*1024*1024))
```

**Available Options:**

- `WithMaxSize(size int)`: Set maximum memory usage in bytes (default: 100MB)
- `WithEvictionPolicy(policy string)`: Set eviction strategy - "LRU", "LFU", or "FIFO" (default: "LRU")
- `WithCompressor(compressor Compressor)`: Set compression algorithm (default: NoCompressor)
- `WithCompressSize(size int)`: Set compression threshold in bytes (default: 1MB)

### Cache Operations

```go
// Set a cache item (value must be []byte)
func (c *Cache) Set(key string, value []byte, ttl time.Duration) error

// Get a cache item (returns []byte)
func (c *Cache) Get(key string) ([]byte, error)

// Delete a cache item
func (c *Cache) Delete(key string)

// Clear all cache items
func (c *Cache) Clear()

// Get cache statistics (aggregated from all shards)
func (c *Cache) Stats() Stats
```

### Stats Structure

```go
type Stats struct {
    Hits           int    // Total cache hit count (aggregated from all shards)
    Misses         int    // Total cache miss count (aggregated from all shards)
    Evictions      int    // Total eviction count (aggregated from all shards)
    CurrentSize    int    // Current memory usage in bytes (aggregated from all shards)
    CurrentCount   int    // Current item count (aggregated from all shards)
    EvictionPolicy string // Eviction policy
    MaxSize        int    // Maximum memory limit
    ShardCount     int    // Number of cache shards
}
```

## Eviction Policies

### LRU (Least Recently Used)

Evicts the least recently accessed items first. Best for applications with temporal locality.

```go
cache := tscache.NewCache(tscache.WithMaxSize(1024*1024), tscache.WithEvictionPolicy("LRU"))
```

### LFU (Least Frequently Used)

Evicts the least frequently accessed items first. Best for applications where some data is accessed much more often.

```go
cache := tscache.NewCache(tscache.WithMaxSize(1024*1024), tscache.WithEvictionPolicy("LFU"))
```

### FIFO (First In, First Out)

Evicts the oldest items first, regardless of access patterns. Simplest and most predictable.

```go
cache := tscache.NewCache(tscache.WithMaxSize(1024*1024), tscache.WithEvictionPolicy("FIFO"))
```

## Compression Options

TSCache supports multiple compression algorithms for optimal performance based on your needs:

### Gzip Compression (Default)

Good balance between compression ratio and CPU overhead, suitable for most applications.

```go
cache := tscache.NewCache(tscache.WithCompressor(tscache.NewGzipCompressor()))
```

### Zstandard (Zstd) Compression

Superior compression performance with better ratios and speed compared to gzip.

```go
cache := tscache.NewCache(tscache.WithCompressor(tscache.NewZstdCompressor()))
```

### No Compression

Pure storage without compression, fastest for small data or CPU-constrained environments.

```go
cache := tscache.NewCache(tscache.WithCompressor(tscache.NewNoCompressor()))
```

### Performance Comparison

Based on benchmarks with 100 map entries:

| Algorithm | Speed (ns/op) | Memory (B/op) | Best Use Case                         |
| --------- | ------------- | ------------- | ------------------------------------- |
| None      | 105,818       | 68,968        | Small data, CPU-limited               |
| Zstd      | 135,522       | 89,881        | Large data, balanced performance      |
| Gzip      | 334,110       | 961,036       | Memory-critical, legacy compatibility |

## Data Types Support

TSCache stores data as `[]byte` for optimal performance and memory efficiency:

```go
// String data
cache.Set("string", []byte("hello"), 0)

// JSON serialized data
import "encoding/json"

user := map[string]interface{}{
    "name": "Alice",
    "age":  30,
}
data, _ := json.Marshal(user)
cache.Set("user", data, 0)

// Retrieve and deserialize
if value, err := cache.Get("user"); err == nil {
    var user map[string]interface{}
    json.Unmarshal(value, &user)
    fmt.Printf("User: %+v\n", user)
}

// Binary data
binaryData := []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f}
cache.Set("binary", binaryData, 0)
```

## Performance Features

### Automatic Sharding

TSCache automatically shards data across multiple internal caches based on CPU cores to reduce lock contention:

- Shard count: 2 × CPU cores (minimum 4, maximum 64)
- Each shard has its own lock, eviction policy, and independent statistics
- Keys are distributed using FNV-1a hash algorithm
- Statistics are aggregated from all shards for global view

### Data Compression

Large data (>1MB by default) is automatically compressed using the configured algorithm:

```go
// Create cache with Zstd compression for better performance
cache := tscache.NewCache(
    tscache.WithCompressor(tscache.NewZstdCompressor()),
    tscache.WithCompressSize(1024), // Compress data larger than 1KB
)

// Large data is automatically compressed
largeData := []byte(strings.Repeat("Hello World! ", 1000))
cache.Set("large", largeData, 0)

// Transparently decompressed on retrieval
value, _ := cache.Get("large")
fmt.Println(string(value)) // Original data
```

### High-Performance Statistics

TSCache features an optimized statistics system designed for high-concurrency environments:

#### Architecture

- **Per-shard statistics**: Each shard maintains independent counters to eliminate global lock contention
- **Lock-free aggregation**: Statistics are aggregated only when `Stats()` is called, not during cache operations
- **Minimal overhead**: Statistics updates use shard-local locks, avoiding cross-shard synchronization
- **Real-time accuracy**: Provides accurate real-time cache performance metrics

#### Performance Benefits

- **Eliminates bottlenecks**: No global lock contention for statistics updates
- **Scales linearly**: Performance improves with more CPU cores and shards
- **Low latency**: Statistics updates don't block cache operations on other shards
- **High throughput**: Supports millions of operations per second with statistics enabled

#### Usage Example

```go
cache := tscache.NewCache(tscache.WithMaxSize(10*1024*1024))

// Perform cache operations
cache.Set("key1", []byte("value1"), 0)
cache.Get("key1")

// Get aggregated statistics (fast aggregation across all shards)
stats := cache.Stats()
fmt.Printf("Hit rate: %.2f%%, Items: %d, Memory: %d bytes\n",
    float64(stats.Hits)/float64(stats.Hits+stats.Misses)*100,
    stats.CurrentCount, stats.CurrentSize)
```

## Benchmarks

```bash
go test -bench=.
```

Typical performance on modern hardware:

- Set operations: ~2M ops/sec
- Get operations: ~5M ops/sec
- Mixed operations: ~3M ops/sec
- Stats access: ~1.4M calls/sec (716ns per call)
- Concurrent operations: ~2.3M ops/sec with stats monitoring

## Thread Safety

TSCache is fully thread-safe and optimized for concurrent access:

```go
cache := tscache.NewCache(tscache.WithMaxSize(1024*1024), tscache.WithEvictionPolicy("LRU"))

// Safe to use from multiple goroutines
go func() {
    for i := 0; i < 1000; i++ {
        data := []byte(fmt.Sprintf("value%d", i))
        cache.Set(fmt.Sprintf("key%d", i), data, 0)
    }
}()

go func() {
    for i := 0; i < 1000; i++ {
        cache.Get(fmt.Sprintf("key%d", i))
    }
}()

// Statistics are also thread-safe and optimized for concurrent access
go func() {
    for {
        stats := cache.Stats()
        fmt.Printf("Hit rate: %.2f%%\n",
            float64(stats.Hits)/float64(stats.Hits+stats.Misses)*100)
        time.Sleep(time.Second)
    }
}()
```

## Examples

📚 **[View Complete Examples →](examples/README.md)** | **[查看中文示例 →](examples/README_CN.md)**

The [examples](examples/) directory contains comprehensive, runnable examples demonstrating all TSCache features:

### Quick Start Examples

```bash
# Run basic usage examples (Set/Get, JSON, TTL, Delete, Stats, Clear)
go run examples/*.go basic

# Run eviction policy comparison (LRU, LFU, FIFO)
go run examples/*.go eviction

# Run compression examples (Gzip, Zstd, performance comparison)
go run examples/*.go compression

# Run all examples
go run examples/*.go all
```

### Example Categories

- **[Basic Operations](examples/basic_usage.go)** - Core functionality with []byte data, JSON serialization, TTL expiration, delete operations, statistics monitoring, and cache clearing
- **[Eviction Policies](examples/eviction_policies.go)** - Detailed comparison of LRU, LFU, and FIFO policies with performance benchmarks
- **[Compression Features](examples/compression_demo.go)** - Gzip and Zstd compression usage, performance comparison, and compression + TTL combinations
- **[Complete Documentation](examples/README.md)** - Detailed usage instructions and example descriptions

Each example includes detailed output, performance metrics, and practical usage patterns.

## Testing

```bash
# Run tests
go test

# Run tests with coverage
go test -cover

# Run benchmarks
go test -bench=.
```

## Code Documentation

TSCache provides comprehensive English documentation throughout the codebase. All source files include detailed comments following Go documentation standards.

### Documentation Features

#### 1. Complete API Documentation

- **Package-level comments**: Detailed description of TSCache's design goals and core features
- **Type documentation**: Every struct and interface includes purpose and design rationale
- **Function documentation**: Complete parameter, return value, and behavior descriptions
- **Field comments**: Important struct fields with role and constraint explanations

#### 2. Implementation Details

- **Algorithm explanations**: Step-by-step logic for key algorithms
- **Performance characteristics**: Time and space complexity analysis
- **Concurrency safety**: Thread safety guarantees and locking strategies
- **Edge cases**: Error handling and special condition behaviors

#### 3. Architecture Documentation

**Sharding Architecture**

```go
// Cache represents a thread-safe, in-memory cache with configurable eviction policies.
// It uses a sharded architecture to reduce lock contention and improve concurrent performance.
```

**Memory Management**

```go
// calculateSize estimates the memory size of a value in bytes.
// This function recursively calculates the memory footprint of Go values
// including basic types, slices, maps, pointers, and structs.
```

**Eviction Policies**

```go
// LRUList implements the Least Recently Used eviction policy using a doubly linked list.
// Items are ordered by access time, with the most recently accessed at the front
// and the least recently accessed at the back.
```

**Data Compression**

```go
// GzipCompressor implements the Compressor interface using gzip compression.
// It provides a good balance between compression ratio and CPU overhead,
// making it suitable for caching scenarios where memory is more valuable than CPU time.
```

### Source File Documentation

#### cache.go - Main Cache Interface

- Package-level description of TSCache design and features
- Cache struct explaining sharded architecture for reduced lock contention
- Stats struct for cache performance monitoring and analysis
- NewCache function with parameter explanation and optimal shard calculation
- Set/Get/Delete methods with complete parameter, return value, and behavior descriptions
- Internal helper functions like getShard for consistent hash distribution

#### shard.go - Cache Shard Implementation

- CacheShard struct for individual shard data storage and synchronization
- CacheItem struct with metadata for eviction and expiration
- Set method with automatic compression, memory limit enforcement, and TTL setup
- Get method with expiration checking, automatic decompression, and access statistics
- evictIfNeeded/evictOne methods for memory-based eviction policy implementation

#### eviction.go - Eviction Policy Implementation

- EvictionList interface for consistent behavior across different eviction policies
- LRUList: Least Recently Used policy with time complexity analysis
- LFUList: Least Frequently Used policy with LRU tie-breaking mechanism
- FIFOList: First In First Out policy with simplest predictable behavior
- Detailed algorithm descriptions for Add/Remove/Update/RemoveLeast methods

#### compression.go - Data Compression

- Compressor interface for pluggable compression algorithm definitions
- GzipCompressor implementation balancing compression ratio and CPU overhead
- Complete compression/decompression process: JSON serialization + gzip compression
- Global helper functions with safe type checking for compress/decompress operations

#### utils.go - Utility Functions

- calculateSize: Recursive memory size calculation supporting all Go types
- calculateValueSize/calculateTypeSize: Detailed type size estimation logic
- fnv1a: FNV-1a hash algorithm implementation for consistent key distribution
- getOptimalShardCount: System-based optimal shard count calculation
- formatBytes: Human-readable byte formatting with automatic unit selection
- Unsafe operation functions: Zero-copy string/byte slice conversion with safety warnings

### Documentation Quality

#### Standards Compliance

- **Go Documentation Format**: Follows official Go documentation conventions
- **Technical Accuracy**: Uses precise computer science terminology
- **Implementation Details**: Explains internal mechanisms and design decisions
- **Usage Guidance**: Provides suggestions for when to use specific features

#### Practical Value

- **Code Examples**: Usage examples for key APIs
- **Performance Tips**: Optimization suggestions and best practices
- **Safety Warnings**: Unsafe operation and potential risk warnings
- **Version Compatibility**: Current implementation limitations and future plans

#### Professional Quality

- **Enterprise Standards**: Meets open-source and enterprise-level documentation requirements
- **Complete API Reference**: Developers can understand all functionality through comments alone
- **Deep Technical Insights**: Explains design decisions and implementation details
- **Maintenance Friendly**: Reduces code maintenance and feature extension difficulty
- **Learning Value**: Serves as excellent reference for Go cache system implementation

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

If you have any questions or need help, please:

1. Check the [examples](example/main.go)
2. Read the documentation
3. Open an issue on GitHub

---

**TSCache** - High-performance caching made simple! 🚀
