package tscache

import (
	"fmt"
	"reflect"
	"runtime"
	"unsafe"
)

// calculateSize estimates the memory size of a value in bytes.
//
// This function recursively calculates the memory footprint of Go values
// including basic types, slices, maps, pointers, and structs. It provides
// a reasonable approximation for cache memory accounting.
//
// Parameters:
//   - value: The value to measure (any Go type)
//
// Returns:
//   - int64: Estimated memory size in bytes
//
// Note: This is an approximation and may not account for all memory overhead
// such as GC metadata, alignment padding, or runtime-specific optimizations.
func calculateSize(value any) int64 {
	if value == nil {
		return 0
	}

	val := reflect.ValueOf(value)
	return calculateValueSize(val)
}

// calculateValueSize recursively calculates the size of a reflect.Value.
//
// This is the core implementation that handles different Go types:
// - Basic types: Use their known sizes
// - Pointers: Add size of pointed-to value
// - Slices: Calculate header + element sizes
// - Maps: Estimate based on key/value types and length
// - Structs: Sum all field sizes
// - Arrays: Element size * length
//
// Parameters:
//   - val: reflect.Value to measure
//
// Returns:
//   - int64: Estimated size in bytes
func calculateValueSize(val reflect.Value) int64 {
	if !val.IsValid() {
		return 0
	}

	switch val.Kind() {
	case reflect.Bool:
		return 1 // Boolean values are typically 1 byte

	case reflect.Int, reflect.Uint:
		return 8 // Platform-dependent, assume 64-bit architecture

	case reflect.Int8, reflect.Uint8:
		return 1

	case reflect.Int16, reflect.Uint16:
		return 2

	case reflect.Int32, reflect.Uint32, reflect.Float32:
		return 4

	case reflect.Int64, reflect.Uint64, reflect.Float64:
		return 8

	case reflect.Complex64:
		return 8 // Two 32-bit floats

	case reflect.Complex128:
		return 16 // Two 64-bit floats

	case reflect.String:
		// String header (16 bytes on 64-bit) + string data
		return 16 + int64(val.Len())

	case reflect.Slice:
		// Slice header (24 bytes on 64-bit) + elements
		headerSize := int64(24)
		if val.IsNil() {
			return headerSize
		}

		elementSize := calculateTypeSize(val.Type().Elem())
		elementsSize := elementSize * int64(val.Len())
		return headerSize + elementsSize

	case reflect.Array:
		// Fixed-size array - just the elements
		elementSize := calculateTypeSize(val.Type().Elem())
		return elementSize * int64(val.Len())

	case reflect.Map:
		// Map header + estimated bucket overhead + key/value pairs
		headerSize := int64(8) // Simplified map header
		if val.IsNil() || val.Len() == 0 {
			return headerSize
		}

		keySize := calculateTypeSize(val.Type().Key())
		valueSize := calculateTypeSize(val.Type().Elem())

		// Maps have overhead for hash buckets, estimate 1.5x the actual data
		pairSize := (keySize + valueSize) * int64(val.Len())
		bucketOverhead := pairSize / 2 // 50% overhead estimation

		return headerSize + pairSize + bucketOverhead

	case reflect.Ptr:
		// Pointer size + pointed-to value (if not nil)
		ptrSize := int64(8) // 64-bit pointer
		if val.IsNil() {
			return ptrSize
		}
		return ptrSize + calculateValueSize(val.Elem())

	case reflect.Interface:
		// Interface header + concrete value
		interfaceSize := int64(16) // Interface header on 64-bit
		if val.IsNil() {
			return interfaceSize
		}
		return interfaceSize + calculateValueSize(val.Elem())

	case reflect.Struct:
		// Sum of all field sizes
		var totalSize int64
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			totalSize += calculateValueSize(field)
		}
		return totalSize

	case reflect.Chan:
		// Channel header - simplified estimate
		return 96 // Approximate channel structure size

	case reflect.Func:
		// Function pointer
		return 8

	default:
		// Fallback for unknown types
		return 8
	}
}

// calculateTypeSize estimates the size of a type without an actual value.
//
// This is used for calculating slice and map element sizes when we need
// to estimate memory usage without examining every element.
//
// Parameters:
//   - t: reflect.Type to measure
//
// Returns:
//   - int64: Estimated size in bytes for values of this type
func calculateTypeSize(t reflect.Type) int64 {
	switch t.Kind() {
	case reflect.Bool:
		return 1

	case reflect.Int, reflect.Uint:
		return 8 // Assume 64-bit platform

	case reflect.Int8, reflect.Uint8:
		return 1

	case reflect.Int16, reflect.Uint16:
		return 2

	case reflect.Int32, reflect.Uint32, reflect.Float32:
		return 4

	case reflect.Int64, reflect.Uint64, reflect.Float64:
		return 8

	case reflect.Complex64:
		return 8

	case reflect.Complex128:
		return 16

	case reflect.String:
		return 24 // String header + average string length estimate

	case reflect.Slice:
		return 24 + calculateTypeSize(t.Elem())*4 // Header + 4 elements average

	case reflect.Array:
		return calculateTypeSize(t.Elem()) * int64(t.Len())

	case reflect.Map:
		keySize := calculateTypeSize(t.Key())
		valueSize := calculateTypeSize(t.Elem())
		return 8 + (keySize+valueSize)*4 // Header + 4 pairs average

	case reflect.Ptr, reflect.UnsafePointer:
		return 8

	case reflect.Interface:
		return 16

	case reflect.Struct:
		var size int64
		for i := 0; i < t.NumField(); i++ {
			size += calculateTypeSize(t.Field(i).Type)
		}
		return size

	case reflect.Chan:
		return 96

	case reflect.Func:
		return 8

	default:
		return 8
	}
}

// fnv1a computes the FNV-1a hash of a string for consistent key distribution.
//
// FNV-1a is a fast, non-cryptographic hash function that provides good
// distribution properties for hash tables. It's used to distribute cache
// keys evenly across shards.
//
// Parameters:
//   - key: String to hash
//
// Returns:
//   - uint32: FNV-1a hash value
//
// The algorithm:
// 1. Start with FNV offset basis
// 2. For each byte: XOR with hash, then multiply by FNV prime
// 3. Return final hash value
func fnv1a(key string) uint32 {
	const (
		fnvOffsetBasis = 2166136261 // FNV-1a 32-bit offset basis
		fnvPrime       = 16777619   // FNV-1a 32-bit prime
	)

	hash := uint32(fnvOffsetBasis)

	// Process each byte of the string
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i]) // XOR with byte
		hash *= fnvPrime       // Multiply by FNV prime
	}

	return hash
}

// getOptimalShardCount determines the ideal number of cache shards based on system characteristics.
//
// The shard count affects concurrency performance by reducing lock contention.
// More shards allow better parallelism but increase memory overhead.
//
// Returns:
//   - int: Optimal number of shards (always a power of 2 for efficient modulo)
//
// The algorithm considers:
// - CPU core count (more cores benefit from more shards)
// - Memory overhead (each shard has its own structures)
// - Hash distribution efficiency (powers of 2 enable bitwise modulo)
func getOptimalShardCount() int {
	// Get the number of CPU cores
	numCPU := runtime.NumCPU()

	// Start with 2x the number of CPUs for good concurrency
	shardCount := numCPU * 2

	// Ensure minimum of 4 shards for reasonable distribution
	if shardCount < 4 {
		shardCount = 4
	}

	// Cap at 256 shards to avoid excessive overhead
	if shardCount > 256 {
		shardCount = 256
	}

	// Round to nearest power of 2 for efficient hash modulo operations
	// This allows using bitwise AND instead of modulo division
	return roundToPowerOfTwo(shardCount)
}

// roundToPowerOfTwo rounds a number up to the nearest power of 2.
//
// Powers of 2 are important for efficient hash distribution since
// hash % powerOf2 can be optimized to hash & (powerOf2 - 1).
//
// Parameters:
//   - n: Number to round up
//
// Returns:
//   - int: Nearest power of 2 >= n
//
// Algorithm uses bit manipulation to find the next power of 2:
// 1. Decrement n to handle exact powers of 2
// 2. Set all bits below the highest set bit
// 3. Increment to get the next power of 2
func roundToPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}

	// Handle exact powers of 2
	if n&(n-1) == 0 {
		return n
	}

	// Find the next power of 2
	power := 1
	for power < n {
		power <<= 1
	}

	return power
}

// formatBytes converts a byte count to a human-readable string with appropriate units.
//
// This utility function is used for displaying cache sizes in logs, statistics,
// and debugging output. It automatically selects the most appropriate unit.
//
// Parameters:
//   - bytes: Number of bytes to format
//
// Returns:
//   - string: Human-readable size string (e.g., "1.5 KB", "2.3 MB")
//
// Units used:
// - B: bytes (< 1024)
// - KB: kilobytes (1024 bytes)
// - MB: megabytes (1024 KB)
// - GB: gigabytes (1024 MB)
// - TB: terabytes (1024 GB)
func formatBytes(bytes int64) string {
	const unit = 1024

	// Handle negative and zero values
	if bytes < 0 {
		return fmt.Sprintf("-%s", formatBytes(-bytes))
	}
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	// Calculate the appropriate unit
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	// Format with 1 decimal place
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// getStringFromBytes converts a byte slice to a string without copying (unsafe operation).
//
// This is an optimization for performance-critical paths where we need to convert
// bytes to strings without the overhead of memory allocation and copying.
//
// Parameters:
//   - b: Byte slice to convert
//
// Returns:
//   - string: String representation of the bytes
//
// WARNING: This uses unsafe operations and should only be used when:
// 1. The byte slice won't be modified after conversion
// 2. The string won't outlive the byte slice
// 3. Performance is critical and the risk is understood
func getStringFromBytes(b []byte) string {
	if len(b) == 0 {
		return ""
	}

	// Use unsafe pointer conversion to avoid copying
	// This is safe as long as the byte slice is not modified
	return *(*string)(unsafe.Pointer(&b))
}

// getBytesFromString converts a string to a byte slice without copying (unsafe operation).
//
// This is the reverse of getStringFromBytes, used in performance-critical paths
// where allocation overhead needs to be minimized.
//
// Parameters:
//   - s: String to convert
//
// Returns:
//   - []byte: Byte slice representation of the string
//
// WARNING: This uses unsafe operations. The returned byte slice should NOT be modified
// as it shares memory with the original string, and strings are immutable in Go.
func getBytesFromString(s string) []byte {
	if len(s) == 0 {
		return nil
	}

	// Use unsafe pointer conversion to avoid copying
	// The returned slice must not be modified!
	return (*[0x7fffffff]byte)(unsafe.Pointer(
		(*reflect.StringHeader)(unsafe.Pointer(&s)).Data,
	))[:len(s):len(s)]
}

// getMemoryUsage 获取当前内存使用情况
func getMemoryUsage() (int64, int64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return int64(m.Alloc), int64(m.Sys)
}
