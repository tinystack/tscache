package tscache

import (
	"sync"
	"time"
)

// ShardStats holds statistics for a single cache shard
type ShardStats struct {
	mu        sync.RWMutex // Protects concurrent access to shard statistics
	Hits      int          // Number of successful cache hits in this shard
	Misses    int          // Number of cache misses in this shard
	Evictions int          // Number of items evicted in this shard
}

// ShardStatsSnapshot represents a snapshot of shard statistics at a point in time
type ShardStatsSnapshot struct {
	Hits         int // Number of successful cache hits in this shard
	Misses       int // Number of cache misses in this shard
	Evictions    int // Number of items evicted in this shard
	CurrentCount int // Current number of items in this shard
	CurrentSize  int // Current memory usage of this shard in bytes
}

// CacheShard represents a single shard of the cache, handling a subset of keys.
// Each shard maintains its own data storage, eviction list, and synchronization mechanisms.
// This design reduces lock contention by distributing cache operations across multiple shards.
type CacheShard struct {
	maxSize        int                   // Maximum memory usage for this shard in bytes
	evictionPolicy string                // Eviction policy: "LRU", "LFU", or "FIFO"
	data           map[string]*CacheItem // Hash map storing the actual cache data
	evictionList   EvictionList          // Eviction policy implementation for managing item priorities
	mu             sync.RWMutex          // Read-write mutex for thread-safe access
	stats          *ShardStats           // Shard-specific statistics
	currentSize    int                   // Current memory usage of this shard in bytes
	currentCount   int                   // Current number of items in this shard
	compressor     Compressor            // Compression algorithm
	compressSize   int                   // Compression size threshold
}

// CacheItem represents a single cached entry with metadata for eviction and expiration.
// Items store the actual value along with timing information and compression status.
type CacheItem struct {
	Key         string    `json:"key"`          // Cache key identifier
	Value       []byte    `json:"value"`        // Cached value (may be compressed)
	Size        int       `json:"size"`         // Memory size of the item in bytes
	ExpireAt    time.Time `json:"expire_at"`    // Expiration timestamp (zero value = no expiration)
	CreatedAt   time.Time `json:"created_at"`   // Creation timestamp
	AccessAt    time.Time `json:"access_at"`    // Last access timestamp (for LRU)
	AccessCount int       `json:"access_count"` // Access frequency counter (for LFU)
	Compressed  bool      `json:"compressed"`   // Whether the value is compressed
}

// NewCacheShard creates a new cache shard with specified limits and eviction policy.
//
// Parameters:
//   - maxSize: Maximum memory usage for this shard in bytes
//   - evictionPolicy: Eviction strategy ("LRU", "LFU", or "FIFO")
//   - compressor: Compression algorithm
//   - compressSize: Compression size threshold
//
// Returns:
//   - *CacheShard: A new initialized cache shard
//
// The shard initializes with the appropriate eviction list implementation based on the policy.
// Invalid policies default to LRU for consistent behavior.
func NewCacheShard(maxSize int, evictionPolicy string, compressor Compressor, compressSize int) *CacheShard {
	shard := &CacheShard{
		maxSize:        maxSize,
		evictionPolicy: evictionPolicy,
		data:           make(map[string]*CacheItem),
		stats:          &ShardStats{},
		compressor:     compressor,
		compressSize:   compressSize,
	}

	// Initialize the appropriate eviction list based on policy
	switch evictionPolicy {
	case EvictionLRU:
		shard.evictionList = NewLRUList()
	case EvictionLFU:
		shard.evictionList = NewLFUList()
	case EvictionFIFO:
		shard.evictionList = NewFIFOList()
	default:
		// Default to LRU for unknown policies
		shard.evictionList = NewLRUList()
		shard.evictionPolicy = EvictionLRU
	}

	return shard
}

// Set stores a key-value pair in this shard with optional TTL and automatic compression.
//
// Parameters:
//   - key: Cache key (must be non-empty)
//   - value: Value to cache (any type)
//   - ttl: Time to live (0 for no expiration)
//
// Returns:
//   - error: nil on success, error on failure
//
// The method handles:
// - Automatic compression for large values (>1KB)
// - Memory limit enforcement with eviction
// - TTL expiration setup
// - Eviction list management
// - Statistics updates
func (s *CacheShard) Set(key string, value []byte, ttl time.Duration) error {
	var (
		now        = time.Now()
		size       = len(value)
		finalValue = value
		compressed = false
	)
	if size > s.compressSize && s.compressor != nil {
		if compressedData, err := s.compressor.Compress(value); err == nil {
			compressedSize := len(compressedData)
			if compressedSize < size {
				finalValue = compressedData
				size = compressedSize
				compressed = true
			}
		}
	}

	var expireAt time.Time
	if ttl > 0 {
		expireAt = now.Add(ttl)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if oldItem, exists := s.data[key]; exists {
		oldSize := oldItem.Size
		s.currentSize -= oldSize
		s.evictionList.Remove(key)

		oldItem.Value = finalValue
		oldItem.Size = size
		oldItem.ExpireAt = expireAt
		oldItem.AccessAt = now
		oldItem.Compressed = compressed

		s.currentSize += size
		s.evictionList.Add(key, oldItem)
	} else {
		item := &CacheItem{
			Key:         key,
			Value:       finalValue,
			Size:        size,
			ExpireAt:    expireAt,
			CreatedAt:   now,
			AccessAt:    now,
			AccessCount: 0,
			Compressed:  compressed,
		}

		s.data[key] = item
		s.currentSize += size
		s.currentCount++
		s.evictionList.Add(key, item)
	}
	s.evictIfNeeded(0)

	return nil
}

// Get retrieves a value from the shard by key, handling expiration and access tracking.
//
// Parameters:
//   - key: Cache key to lookup
//
// Returns:
//   - []byte: The cached value (decompressed if necessary)
//   - error: nil if found, ErrKeyNotFound if not found or expired
//
// The method handles:
// - Expiration checking and cleanup
// - Automatic decompression
// - Access statistics updates
// - Eviction list updates for access tracking
func (s *CacheShard) Get(key string) ([]byte, error) {
	s.mu.RLock()
	item, exists := s.data[key]
	s.mu.RUnlock()

	if !exists {
		s.stats.mu.Lock()
		s.stats.Misses++
		s.stats.mu.Unlock()
		return nil, ErrKeyNotFound
	}

	// Check if the item has expired
	if !item.ExpireAt.IsZero() && time.Now().After(item.ExpireAt) {
		go s.Delete(key)

		s.stats.mu.Lock()
		s.stats.Misses++
		s.stats.mu.Unlock()
		return nil, ErrKeyNotFound
	}

	s.mu.Lock()
	item.AccessAt = time.Now()
	item.AccessCount++
	s.evictionList.Update(key, item)
	s.mu.Unlock()

	s.stats.mu.Lock()
	s.stats.Hits++
	s.stats.mu.Unlock()

	if item.Compressed {
		decompressedValue, err := s.compressor.Decompress(item.Value)
		return decompressedValue, err
	}

	return item.Value, nil
}

// Delete removes a key-value pair from the shard and updates all related structures.
//
// Parameters:
//   - key: Cache key to remove
//
// The method handles:
// - Removal from data storage
// - Eviction list cleanup
// - Memory accounting updates
// - Statistics updates
func (s *CacheShard) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find the item to delete
	item, exists := s.data[key]
	if !exists {
		return
	}

	// Remove from all data structures
	delete(s.data, key)        // Remove from hash map
	s.currentSize -= item.Size // Update memory accounting
	s.currentCount--           // Update item count
	s.evictionList.Remove(key) // Remove from eviction list
}

// Clear removes all items from the shard and resets its state.
//
// This operation is atomic and efficiently clears all shard data structures.
func (s *CacheShard) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear all data structures
	s.data = make(map[string]*CacheItem) // Create new empty map
	s.currentSize = 0                    // Reset memory accounting
	s.currentCount = 0                   // Reset item count
	s.evictionList.Clear()               // Clear eviction list

	// Reset shard statistics
	s.stats.mu.Lock()
	s.stats.Hits = 0
	s.stats.Misses = 0
	s.stats.Evictions = 0
	s.stats.mu.Unlock()
}

// evictIfNeeded checks if the shard exceeds memory limits and triggers eviction if necessary.
//
// Parameters:
//   - newItemSize: Size of a new item being added (for pre-eviction planning)
//
// This method enforces memory limits by repeatedly evicting items until the shard
// is within its memory budget. It only considers memory-based eviction currently.
func (s *CacheShard) evictIfNeeded(newItemSize int) {
	for s.maxSize > 0 && s.currentSize+newItemSize > s.maxSize {
		if !s.evictOne() {
			break
		}
	}
}

// evictOne removes a single item from the shard according to the eviction policy.
//
// Returns:
//   - bool: true if an item was evicted, false if no items to evict
//
// The method uses the eviction list to determine which item should be removed,
// then handles all cleanup including statistics updates.
func (s *CacheShard) evictOne() bool {
	keyToEvict := s.evictionList.RemoveLeast()
	if keyToEvict == "" {
		return false
	}

	if item, exists := s.data[keyToEvict]; exists {
		delete(s.data, keyToEvict)
		s.currentSize -= item.Size
		s.currentCount--

		s.stats.mu.Lock()
		s.stats.Evictions++
		s.stats.mu.Unlock()

		return true
	}

	return false
}

// getStats returns a snapshot of this shard's statistics including current count and size.
//
// Returns:
//   - ShardStatsSnapshot: Current shard statistics
//
// This method aggregates both the statistical counters and current state information.
func (s *CacheShard) getStats() ShardStatsSnapshot {
	s.stats.mu.RLock()
	hits := s.stats.Hits
	misses := s.stats.Misses
	evictions := s.stats.Evictions
	s.stats.mu.RUnlock()

	s.mu.RLock()
	currentCount := s.currentCount
	currentSize := s.currentSize
	s.mu.RUnlock()

	return ShardStatsSnapshot{
		Hits:         hits,
		Misses:       misses,
		Evictions:    evictions,
		CurrentCount: currentCount,
		CurrentSize:  currentSize,
	}
}
