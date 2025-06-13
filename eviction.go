package tscache

import (
	"container/list"
	"time"
)

// EvictionList defines the interface for different cache eviction policies.
// Each eviction policy implements this interface to provide consistent behavior
// for managing cache item priorities and determining which items to evict.
type EvictionList interface {
	// Add inserts or updates an item in the eviction list
	Add(key string, item *CacheItem)
	// Remove deletes an item from the eviction list
	Remove(key string)
	// Update modifies an existing item's position/priority in the eviction list
	Update(key string, item *CacheItem)
	// RemoveLeast evicts and returns the key of the least valuable item
	RemoveLeast() string
	// Clear removes all items from the eviction list
	Clear()
}

// LRUNode represents a node in the LRU (Least Recently Used) doubly linked list.
// Each node maintains references to the cache item and its position in the access order.
type LRUNode struct {
	Key  string     // Cache key for this node
	Item *CacheItem // Reference to the actual cache item
}

// LRUList implements the Least Recently Used eviction policy using a doubly linked list.
// Items are ordered by access time, with the most recently accessed at the front
// and the least recently accessed at the back.
//
// Time Complexity:
//   - Add: O(1)
//   - Remove: O(1) with hash map lookup
//   - Update: O(1) with hash map lookup
//   - RemoveLeast: O(1) - always removes from back
//
// Note: This implementation is NOT thread-safe. Thread safety is handled at the shard level.
type LRUList struct {
	list    *list.List               // Doubly linked list maintaining access order
	nodeMap map[string]*list.Element // Hash map for O(1) key-to-node lookup
}

// NewLRUList creates a new LRU eviction list.
//
// Returns:
//   - *LRUList: A new LRU list ready for use
//
// The LRU list maintains items in access order, automatically moving
// accessed items to the front of the list.
func NewLRUList() *LRUList {
	return &LRUList{
		list:    list.New(),
		nodeMap: make(map[string]*list.Element),
	}
}

// Add inserts a new item or moves an existing item to the front of the LRU list.
//
// Parameters:
//   - key: Cache key identifier
//   - item: Cache item to add or update
//
// If the key already exists, the item is moved to the front (most recent position).
// If it's a new key, a new node is created at the front of the list.
func (lru *LRUList) Add(key string, item *CacheItem) {
	if element, exists := lru.nodeMap[key]; exists {
		// Update existing item and move to front
		node := element.Value.(*LRUNode)
		node.Item = item
		lru.list.MoveToFront(element)
	} else {
		// Create new node and add to front
		node := &LRUNode{
			Key:  key,
			Item: item,
		}
		element := lru.list.PushFront(node)
		lru.nodeMap[key] = element
	}
}

// Remove deletes an item from the LRU list.
//
// Parameters:
//   - key: Cache key to remove
//
// The item is removed from both the linked list and the key mapping.
// This operation is safe to call even if the key doesn't exist.
func (lru *LRUList) Remove(key string) {
	if element, exists := lru.nodeMap[key]; exists {
		lru.list.Remove(element)
		delete(lru.nodeMap, key)
	}
}

// Update moves an existing item to the front of the LRU list to mark it as recently used.
//
// Parameters:
//   - key: Cache key to update
//   - item: Updated cache item
//
// This method is called when an item is accessed to update its position
// in the access order. If the key doesn't exist, the operation is ignored.
func (lru *LRUList) Update(key string, item *CacheItem) {
	if element, exists := lru.nodeMap[key]; exists {
		// Update item data and move to front
		node := element.Value.(*LRUNode)
		node.Item = item
		lru.list.MoveToFront(element)
	}
}

// RemoveLeast evicts the least recently used item from the list.
//
// Returns:
//   - string: Key of the evicted item, empty string if list is empty
//
// This method removes the item from the back of the list (oldest access time)
// and returns its key for removal from the main cache storage.
func (lru *LRUList) RemoveLeast() string {
	// Get the least recently used item (back of list)
	element := lru.list.Back()
	if element == nil {
		return "" // List is empty
	}

	// Remove from both list and map
	node := element.Value.(*LRUNode)
	lru.list.Remove(element)
	delete(lru.nodeMap, node.Key)

	return node.Key
}

// Clear removes all items from the LRU list and resets its state.
//
// This operation efficiently clears the entire eviction list by creating
// new empty data structures.
func (lru *LRUList) Clear() {
	lru.list = list.New()
	lru.nodeMap = make(map[string]*list.Element)
}

// LFUNode represents a node in the LFU (Least Frequently Used) data structure.
// Each node tracks access frequency and timing information for eviction decisions.
type LFUNode struct {
	Key       string     // Cache key for this node
	Item      *CacheItem // Reference to the actual cache item
	Frequency int        // Access frequency counter
	LastUsed  time.Time  // Timestamp of last access (for tie-breaking)
}

// LFUList implements the Least Frequently Used eviction policy.
// Items are organized by access frequency, with the least frequently accessed
// items being evicted first. For items with equal frequency, the least recently
// used item is evicted (LFU with LRU tie-breaking).
//
// Time Complexity:
//   - Add: O(1)
//   - Remove: O(1) for deletion, O(f) for frequency list cleanup where f is frequency count
//   - Update: O(1)
//   - RemoveLeast: O(n) where n is the number of items at minimum frequency
//
// Note: This implementation is NOT thread-safe. Thread safety is handled at the shard level.
type LFUList struct {
	nodes       map[string]*LFUNode // Hash map for O(1) key-to-node lookup
	frequencies map[int]*list.List  // Frequency buckets (frequency -> list of nodes)
	minFreq     int                 // Current minimum frequency for quick eviction
}

// NewLFUList creates a new LFU eviction list.
//
// Returns:
//   - *LFUList: A new LFU list ready for use
//
// The LFU list organizes items by access frequency, maintaining separate
// lists for each frequency level to enable efficient eviction.
func NewLFUList() *LFUList {
	return &LFUList{
		nodes:       make(map[string]*LFUNode),
		frequencies: make(map[int]*list.List),
		minFreq:     1,
	}
}

// Add inserts a new item or updates an existing item's frequency in the LFU list.
//
// Parameters:
//   - key: Cache key identifier
//   - item: Cache item to add or update
//
// New items start with frequency based on their access count. Existing items have
// their frequency updated and are moved to the appropriate frequency bucket.
func (lfu *LFUList) Add(key string, item *CacheItem) {
	now := time.Now()

	if node, exists := lfu.nodes[key]; exists {
		// Update existing node
		oldFreq := node.Frequency
		newFreq := item.AccessCount

		// Only move if frequency actually changed
		if oldFreq != newFreq {
			// Remove from old frequency bucket
			lfu.removeFromFrequency(node, oldFreq)

			// Update node data
			node.Frequency = newFreq
			node.LastUsed = now

			// Add to new frequency bucket
			lfu.addToFrequency(node, newFreq)

			// Update minimum frequency if necessary
			lfu.updateMinFreq()
		} else {
			// Just update the item and timestamp
			node.Item = item
			node.LastUsed = now
		}
	} else {
		// Create new node with frequency from item's access count
		frequency := item.AccessCount
		if frequency == 0 {
			frequency = 1 // Minimum frequency for new items
		}

		node := &LFUNode{
			Key:       key,
			Item:      item,
			Frequency: frequency,
			LastUsed:  now,
		}

		lfu.nodes[key] = node
		lfu.addToFrequency(node, frequency)

		// Update minimum frequency
		if frequency < lfu.minFreq || lfu.isEmpty(lfu.minFreq) {
			lfu.minFreq = frequency
		}
	}
}

// Remove deletes an item from the LFU list.
//
// Parameters:
//   - key: Cache key to remove
//
// The item is removed from both the node map and its frequency bucket.
func (lfu *LFUList) Remove(key string) {
	if node, exists := lfu.nodes[key]; exists {
		lfu.removeFromFrequency(node, node.Frequency)
		delete(lfu.nodes, key)
		lfu.updateMinFreq()
	}
}

// Update increments an item's frequency and moves it to the appropriate bucket.
//
// Parameters:
//   - key: Cache key to update
//   - item: Updated cache item
//
// This method is called when an item is accessed to update its frequency count.
func (lfu *LFUList) Update(key string, item *CacheItem) {
	if node, exists := lfu.nodes[key]; exists {
		oldFreq := node.Frequency
		newFreq := item.AccessCount

		// Only move if frequency actually changed
		if oldFreq != newFreq {
			// Remove from old frequency bucket
			lfu.removeFromFrequency(node, oldFreq)

			// Update node
			node.Item = item
			node.Frequency = newFreq
			node.LastUsed = time.Now()

			// Add to new frequency bucket
			lfu.addToFrequency(node, newFreq)

			// Update minimum frequency
			lfu.updateMinFreq()
		} else {
			// Just update the item and timestamp
			node.Item = item
			node.LastUsed = time.Now()
		}
	}
}

// RemoveLeast evicts the least frequently used item from the list.
//
// Returns:
//   - string: Key of the evicted item, empty string if list is empty
//
// If multiple items have the same minimum frequency, the least recently
// used among them is evicted (LFU with LRU tie-breaking).
func (lfu *LFUList) RemoveLeast() string {
	if len(lfu.nodes) == 0 {
		return ""
	}

	// Find the frequency list with minimum frequency
	freqList, exists := lfu.frequencies[lfu.minFreq]
	if !exists || freqList.Len() == 0 {
		return "" // No items to evict
	}

	// Find the least recently used item among items with minimum frequency
	var oldestElement *list.Element
	var oldestTime time.Time = time.Now()

	for element := freqList.Front(); element != nil; element = element.Next() {
		node := element.Value.(*LFUNode)
		if oldestElement == nil || node.LastUsed.Before(oldestTime) {
			oldestElement = element
			oldestTime = node.LastUsed
		}
	}

	if oldestElement == nil {
		return ""
	}

	// Remove the selected node
	node := oldestElement.Value.(*LFUNode)
	freqList.Remove(oldestElement)
	delete(lfu.nodes, node.Key)

	// Update minimum frequency if this was the last item at minFreq
	lfu.updateMinFreq()

	return node.Key
}

// Clear removes all items from the LFU list and resets its state.
func (lfu *LFUList) Clear() {
	lfu.nodes = make(map[string]*LFUNode)
	lfu.frequencies = make(map[int]*list.List)
	lfu.minFreq = 1
}

// addToFrequency adds a node to the appropriate frequency bucket.
//
// Parameters:
//   - node: LFU node to add
//   - frequency: Frequency level for the bucket
func (lfu *LFUList) addToFrequency(node *LFUNode, frequency int) {
	if lfu.frequencies[frequency] == nil {
		lfu.frequencies[frequency] = list.New()
	}
	lfu.frequencies[frequency].PushBack(node)
}

// removeFromFrequency removes a node from its frequency bucket.
//
// Parameters:
//   - node: LFU node to remove
//   - frequency: Frequency level of the bucket
func (lfu *LFUList) removeFromFrequency(node *LFUNode, frequency int) {
	if freqList, exists := lfu.frequencies[frequency]; exists {
		// Find and remove the node from the frequency list
		for element := freqList.Front(); element != nil; element = element.Next() {
			if element.Value.(*LFUNode) == node {
				freqList.Remove(element)
				break
			}
		}
	}
}

// isEmpty checks if a frequency bucket is empty.
//
// Parameters:
//   - frequency: Frequency level to check
//
// Returns:
//   - bool: true if the bucket is empty or doesn't exist
func (lfu *LFUList) isEmpty(frequency int) bool {
	freqList, exists := lfu.frequencies[frequency]
	return !exists || freqList.Len() == 0
}

// updateMinFreq recalculates the minimum frequency across all buckets.
//
// This method finds the lowest frequency that still contains items,
// which is needed for efficient eviction operations.
func (lfu *LFUList) updateMinFreq() {
	// If current minFreq bucket is empty, find the next non-empty bucket
	if lfu.isEmpty(lfu.minFreq) {
		lfu.minFreq = 1 // Reset to minimum possible frequency

		// Find the actual minimum frequency with items
		for freq, freqList := range lfu.frequencies {
			if freqList.Len() > 0 {
				if freq < lfu.minFreq || lfu.minFreq == 1 {
					lfu.minFreq = freq
				}
			}
		}
	}
}

// FIFONode represents a node in the FIFO (First In First Out) queue.
// Each node maintains creation time information for eviction ordering.
type FIFONode struct {
	Key       string     // Cache key for this node
	Item      *CacheItem // Reference to the actual cache item
	CreatedAt time.Time  // Creation timestamp for FIFO ordering
}

// FIFOList implements the First In First Out eviction policy using a simple queue.
// Items are evicted in the order they were added, regardless of access patterns.
// This is the simplest eviction policy with predictable behavior.
//
// Time Complexity:
//   - Add: O(1)
//   - Remove: O(1) with hash map lookup
//   - Update: O(1) - no reordering needed
//   - RemoveLeast: O(1) - always removes from front
//
// Note: This implementation is NOT thread-safe. Thread safety is handled at the shard level.
type FIFOList struct {
	list    *list.List               // Queue maintaining insertion order
	nodeMap map[string]*list.Element // Hash map for O(1) key-to-node lookup
}

// NewFIFOList creates a new FIFO eviction list.
//
// Returns:
//   - *FIFOList: A new FIFO list ready for use
//
// The FIFO list maintains items in insertion order, with the oldest
// items at the front and newest at the back.
func NewFIFOList() *FIFOList {
	return &FIFOList{
		list:    list.New(),
		nodeMap: make(map[string]*list.Element),
	}
}

// Add inserts a new item at the back of the FIFO queue.
//
// Parameters:
//   - key: Cache key identifier
//   - item: Cache item to add
//
// For FIFO policy, adding an existing key doesn't change its position
// in the queue, only updates the item data.
func (fifo *FIFOList) Add(key string, item *CacheItem) {
	if element, exists := fifo.nodeMap[key]; exists {
		// Update existing item without changing position
		node := element.Value.(*FIFONode)
		node.Item = item
	} else {
		// Add new item to the back of the queue
		node := &FIFONode{
			Key:       key,
			Item:      item,
			CreatedAt: time.Now(),
		}
		element := fifo.list.PushBack(node)
		fifo.nodeMap[key] = element
	}
}

// Remove deletes an item from the FIFO queue.
//
// Parameters:
//   - key: Cache key to remove
//
// The item is removed from both the queue and the key mapping.
func (fifo *FIFOList) Remove(key string) {
	if element, exists := fifo.nodeMap[key]; exists {
		fifo.list.Remove(element)
		delete(fifo.nodeMap, key)
	}
}

// Update modifies an existing item's data without changing its queue position.
//
// Parameters:
//   - key: Cache key to update
//   - item: Updated cache item
//
// In FIFO policy, updates don't affect the eviction order since it's
// based solely on insertion time, not access patterns.
func (fifo *FIFOList) Update(key string, item *CacheItem) {
	if element, exists := fifo.nodeMap[key]; exists {
		// Update item data without changing position
		node := element.Value.(*FIFONode)
		node.Item = item
	}
}

// RemoveLeast evicts the oldest item from the FIFO queue.
//
// Returns:
//   - string: Key of the evicted item, empty string if queue is empty
//
// This method always removes the item from the front of the queue,
// which is the oldest item by insertion time.
func (fifo *FIFOList) RemoveLeast() string {
	// Get the first item (oldest)
	element := fifo.list.Front()
	if element == nil {
		return "" // Queue is empty
	}

	// Remove from both queue and map
	node := element.Value.(*FIFONode)
	fifo.list.Remove(element)
	delete(fifo.nodeMap, node.Key)

	return node.Key
}

// Clear removes all items from the FIFO queue and resets its state.
func (fifo *FIFOList) Clear() {
	fifo.list = list.New()
	fifo.nodeMap = make(map[string]*list.Element)
}
