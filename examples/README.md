# TSCache Examples

This directory contains various usage examples for TSCache, demonstrating different features and capabilities of the cache.

## Example Files

- `basic_usage.go` - Basic usage examples including Set/Get operations, JSON serialization, TTL expiration, delete operations, etc.
- `eviction_policies.go` - Eviction policy examples demonstrating the behavior and performance comparison of LRU, LFU, and FIFO eviction policies
- `compression_demo.go` - Compression feature examples showcasing the usage and performance comparison of Gzip and Zstd compression algorithms
- `main.go` - Main program for running different examples

## Running Examples

### Prerequisites

Make sure you are in the TSCache project root directory and have Go 1.21 or higher installed.

### Running Individual Examples

```bash
# Run basic usage examples
go run examples/*.go basic

# Run eviction policy examples
go run examples/*.go eviction

# Run compression feature examples
go run examples/*.go compression
```

### Running All Examples

```bash
# Run all examples
go run examples/*.go all
```

### View Help

```bash
# View usage instructions
go run examples/*.go
```

## Example Descriptions

### Basic Usage Examples (basic_usage.go)

Demonstrates TSCache core functionality:

1. **Basic Set/Get Operations** - Store and retrieve string data
2. **JSON Data Serialization** - Store and retrieve structured data
3. **TTL Expiration Mechanism** - Set data time-to-live
4. **Delete Operations** - Remove specific cache items
5. **Statistics Information** - View cache performance statistics
6. **Clear Operations** - Clear all cached data

### Eviction Policy Examples (eviction_policies.go)

Compare the behavior of different eviction policies:

1. **LRU (Least Recently Used)** - Evict least recently accessed data
2. **LFU (Least Frequently Used)** - Evict least frequently accessed data
3. **FIFO (First In First Out)** - Evict earliest added data
4. **Performance Comparison** - Compare performance of different policies

### Compression Feature Examples (compression_demo.go)

Showcase compression functionality usage:

1. **Gzip Compression** - Use Gzip algorithm to compress data
2. **Zstd Compression** - Use Zstd algorithm to compress data
3. **No Compression** - Store data without compression
4. **Performance Comparison** - Compare performance and compression ratios of different algorithms
5. **Compression + TTL** - Combined usage of compression with TTL

## Sample Output

When running examples, you will see detailed output information including:

- Operation execution time
- Data integrity verification
- Cache statistics (hit rate, memory usage, eviction count, etc.)
- Compression ratios and performance comparisons

## Custom Examples

You can create your own test code based on these examples:

1. Copy relevant example functions
2. Modify parameters and test data
3. Add your own business logic
4. Run tests and observe results

## Notes

- Memory limits and TTL times used in examples are small for quick observation of effects
- In production environments, please adjust these parameters according to actual needs
- Compression functionality is suitable for larger data; small data may not show significant compression effects
- Different eviction policies are suitable for different usage scenarios; choose appropriate policies based on access patterns
