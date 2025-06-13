package tscache

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// 辅助函数：将字符串转换为[]byte
func toBytes(s string) []byte {
	return []byte(s)
}

func TestNewCache(t *testing.T) {
	tests := []struct {
		name           string
		maxSize        int
		evictionPolicy string
		wantPolicy     string
	}{
		{
			name:           "LRU策略",
			maxSize:        1024 * 1024, // 1MB
			evictionPolicy: "LRU",
			wantPolicy:     "LRU",
		},
		{
			name:           "LFU策略",
			maxSize:        1024 * 1024,
			evictionPolicy: "LFU",
			wantPolicy:     "LFU",
		},
		{
			name:           "FIFO策略",
			maxSize:        1024 * 1024,
			evictionPolicy: "FIFO",
			wantPolicy:     "FIFO",
		},
		{
			name:           "默认策略",
			maxSize:        1024 * 1024,
			evictionPolicy: "INVALID",
			wantPolicy:     "LRU", // 默认使用LRU
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewCache(WithMaxSize(tt.maxSize), WithEvictionPolicy(tt.evictionPolicy))

			if cache == nil {
				t.Error("NewCache returned nil")
			}

			stats := cache.Stats()
			if stats.EvictionPolicy != tt.wantPolicy {
				t.Errorf("EvictionPolicy = %v, want %v", stats.EvictionPolicy, tt.wantPolicy)
			}

			if stats.MaxSize != tt.maxSize {
				t.Errorf("MaxSize = %v, want %v", stats.MaxSize, tt.maxSize)
			}
		})
	}
}

func TestNewCacheWithCompressSize(t *testing.T) {
	cache := NewCache(
		WithMaxSize(1024*1024),
		WithEvictionPolicy("LRU"),
		WithCompressor(NewGzipCompressor()),
		WithCompressSize(128), // 设置压缩阈值为128字节
	)

	// 小于阈值的数据不应被压缩，能正常存取
	small := toBytes("short data")
	if err := cache.Set("small", small, 0); err != nil {
		t.Errorf("Set small failed: %v", err)
	}
	val, err := cache.Get("small")
	if err != nil || string(val) != string(small) {
		t.Errorf("Get small failed: %v, got: %s", err, string(val))
	}

	// 大于阈值的数据应被压缩，能正常存取
	large := toBytes(strings.Repeat("compress me!", 20)) // >128字节
	if err := cache.Set("large", large, 0); err != nil {
		t.Errorf("Set large failed: %v", err)
	}
	val, err = cache.Get("large")
	if err != nil || string(val) != string(large) {
		t.Errorf("Get large failed: %v, got: %s", err, string(val))
	}
}

func TestCacheSetAndGet(t *testing.T) {
	cache := NewCache(WithMaxSize(1024*1024), WithEvictionPolicy("LRU"))

	// 测试基本的set和get
	err := cache.Set("key1", toBytes("value1"), 0)
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	value, err := cache.Get("key1")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}

	if string(value) != "value1" {
		t.Errorf("Get returned %v, want value1", string(value))
	}

	// 测试不存在的key
	_, err = cache.Get("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent key")
	}
}

func TestCacheExpiration(t *testing.T) {
	cache := NewCache(WithMaxSize(1024*1024), WithEvictionPolicy("LRU"))

	// 设置一个很短的TTL
	err := cache.Set("expiring_key", toBytes("expiring_value"), 100*time.Millisecond)
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	// 立即获取应该成功
	value, err := cache.Get("expiring_key")
	if err != nil {
		t.Errorf("Get failed immediately: %v", err)
	}
	if string(value) != "expiring_value" {
		t.Errorf("Get returned %v, want expiring_value", string(value))
	}

	// 等待过期
	time.Sleep(150 * time.Millisecond)

	// 过期后获取应该失败
	_, err = cache.Get("expiring_key")
	if err == nil {
		t.Error("Expected error for expired key")
	}
}

func TestCacheDelete(t *testing.T) {
	cache := NewCache(WithMaxSize(1024*1024), WithEvictionPolicy("LRU"))

	// 设置然后删除
	cache.Set("delete_key", toBytes("delete_value"), 0)

	// 确认存在
	_, err := cache.Get("delete_key")
	if err != nil {
		t.Errorf("Get failed before delete: %v", err)
	}

	// 删除
	cache.Delete("delete_key")

	// 确认已删除
	_, err = cache.Get("delete_key")
	if err == nil {
		t.Error("Expected error for deleted key")
	}

	// 删除不存在的key（Delete方法不返回错误）
	cache.Delete("nonexistent")
}

func TestCacheClear(t *testing.T) {
	cache := NewCache(WithMaxSize(1024*1024), WithEvictionPolicy("LRU"))

	// 添加一些数据
	cache.Set("key1", toBytes("value1"), 0)
	cache.Set("key2", toBytes("value2"), 0)
	cache.Set("key3", toBytes("value3"), 0)

	// 确认数据存在
	stats := cache.Stats()
	if stats.CurrentCount != 3 {
		t.Errorf("CurrentCount = %v, want 3", stats.CurrentCount)
	}

	// 清空缓存
	cache.Clear()

	// 确认已清空
	stats = cache.Stats()
	if stats.CurrentCount != 0 {
		t.Errorf("CurrentCount after clear = %v, want 0", stats.CurrentCount)
	}
	if stats.CurrentSize != 0 {
		t.Errorf("CurrentSize after clear = %v, want 0", stats.CurrentSize)
	}
}

func TestCacheStats(t *testing.T) {
	cache := NewCache(WithMaxSize(1024*1024), WithEvictionPolicy("LRU"))

	// 初始统计
	stats := cache.Stats()
	if stats.Hits != 0 || stats.Misses != 0 || stats.CurrentCount != 0 {
		t.Error("Initial stats should be zero")
	}

	// 设置数据
	cache.Set("key1", toBytes("value1"), 0)

	// 命中统计
	cache.Get("key1")
	stats = cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("Hits = %v, want 1", stats.Hits)
	}

	// 未命中统计
	cache.Get("nonexistent")
	stats = cache.Stats()
	if stats.Misses != 1 {
		t.Errorf("Misses = %v, want 1", stats.Misses)
	}
}

func TestCacheEviction(t *testing.T) {
	// 创建一个小的缓存，由于去掉了MaxCount限制，只使用内存限制
	// 设置非常小的内存限制来触发淘汰
	cache := NewCache(WithMaxSize(150), WithEvictionPolicy("LRU"))

	// 添加足够多的数据来确保触发淘汰
	keys := make([]string, 20)
	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d_with_some_extra_data_to_make_it_larger", i)
		keys[i] = key
		err := cache.Set(key, toBytes(value), 0)
		if err != nil {
			t.Errorf("Set failed for %s: %v", key, err)
		}
	}

	// 等待一小段时间，让淘汰处理完成
	time.Sleep(10 * time.Millisecond)

	// 检查是否发生了淘汰
	stats := cache.Stats()
	if stats.Evictions == 0 {
		t.Logf("Current stats: Count=%d, Size=%d, MaxSize=%d",
			stats.CurrentCount, stats.CurrentSize, stats.MaxSize)
		t.Error("Expected eviction to occur")
	}

	// 检查当前内存使用是否符合限制
	if stats.CurrentSize > stats.MaxSize {
		t.Errorf("CurrentSize (%d) exceeds MaxSize (%d)", stats.CurrentSize, stats.MaxSize)
	}

	// 验证一些早期的键应该被淘汰了（LRU策略）
	evictedCount := 0
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		if _, err := cache.Get(key); err != nil {
			evictedCount++
		}
	}

	if evictedCount == 0 {
		t.Error("Expected some early keys to be evicted with LRU policy")
	}
}

func TestCacheMultipleDataTypes(t *testing.T) {
	cache := NewCache(WithMaxSize(1024*1024), WithEvictionPolicy("LRU"))

	// 测试不同类型的数据 - 现在只支持[]byte
	testCases := []struct {
		key   string
		value []byte
	}{
		{"string", toBytes("hello world")},
		{"json", toBytes(`{"name": "test", "value": 42}`)},
		{"binary", []byte{0x01, 0x02, 0x03, 0x04, 0x05}},
		{"empty", []byte{}},
		{"large", toBytes(strings.Repeat("large data test ", 100))},
	}

	// 设置所有数据
	for _, tc := range testCases {
		err := cache.Set(tc.key, tc.value, 0)
		if err != nil {
			t.Errorf("Set failed for %s: %v", tc.key, err)
		}
	}

	// 获取并验证所有数据
	for _, tc := range testCases {
		value, err := cache.Get(tc.key)
		if err != nil {
			t.Errorf("Get failed for %s: %v", tc.key, err)
			continue
		}

		// 验证数据一致性
		if len(value) != len(tc.value) {
			t.Errorf("Length mismatch for %s: got %d, want %d", tc.key, len(value), len(tc.value))
		}

		for i, b := range value {
			if b != tc.value[i] {
				t.Errorf("Data mismatch for %s at position %d: got %d, want %d", tc.key, i, b, tc.value[i])
				break
			}
		}
	}
}

// 并发测试
func TestCacheConcurrency(t *testing.T) {
	cache := NewCache(WithMaxSize(1024*1024), WithEvictionPolicy("LRU"))

	// 使用goroutine并发访问缓存
	done := make(chan bool, 10)

	// 并发写入
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				key := "key_" + string(rune(id)) + "_" + string(rune(j))
				value := "value_" + string(rune(id)) + "_" + string(rune(j))
				cache.Set(key, toBytes(value), 0)
			}
			done <- true
		}(i)
	}

	// 并发读取
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				key := "key_" + string(rune(id)) + "_" + string(rune(j))
				cache.Get(key)
			}
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 检查缓存状态
	stats := cache.Stats()
	if stats.CurrentCount == 0 {
		t.Error("Expected some items in cache after concurrent operations")
	}
}

// 基准测试
func BenchmarkCacheSet(b *testing.B) {
	cache := NewCache(WithMaxSize(1024*1024*100), WithEvictionPolicy("LRU"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "benchmark_key_" + string(rune(i))
		value := "benchmark_value_" + string(rune(i))
		cache.Set(key, toBytes(value), 0)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	cache := NewCache(WithMaxSize(1024*1024*100), WithEvictionPolicy("LRU"))

	// 预填充数据
	for i := 0; i < 1000; i++ {
		key := "benchmark_key_" + string(rune(i))
		value := "benchmark_value_" + string(rune(i))
		cache.Set(key, toBytes(value), 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "benchmark_key_" + string(rune(i%1000))
		cache.Get(key)
	}
}

func BenchmarkCacheMixed(b *testing.B) {
	cache := NewCache(WithMaxSize(1024*1024*100), WithEvictionPolicy("LRU"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "benchmark_key_" + string(rune(i%1000))

		if i%2 == 0 {
			// 50% 写操作
			value := "benchmark_value_" + string(rune(i))
			cache.Set(key, toBytes(value), 0)
		} else {
			// 50% 读操作
			cache.Get(key)
		}
	}
}

// 添加更多全面的测试用例
func TestCacheAdvancedFeatures(t *testing.T) {
	// 测试不同的内存限制
	t.Run("different memory limits", func(t *testing.T) {
		cache := NewCache(WithMaxSize(512), WithEvictionPolicy("LRU"))

		// 添加大量数据来触发内存限制
		for i := 0; i < 100; i++ {
			key := "mem_test_" + string(rune(i))
			value := strings.Repeat("data", 50) // 大数据
			err := cache.Set(key, toBytes(value), 0)
			if err != nil {
				t.Errorf("Set failed: %v", err)
			}
		}

		stats := cache.Stats()
		if stats.CurrentSize > 512*2 { // 允许一些缓冲
			t.Errorf("Memory usage too high: %d bytes", stats.CurrentSize)
		}
	})

	// 测试压缩功能
	t.Run("compression", func(t *testing.T) {
		cache := NewCache(WithMaxSize(1024*1024), WithEvictionPolicy("LRU"), WithCompressor(NewGzipCompressor()))

		// 中等大小的数据应该被压缩
		largeData := strings.Repeat("This is test data for compression. ", 100)
		err := cache.Set("large", toBytes(largeData), 0)
		if err != nil {
			t.Errorf("Set large data failed: %v", err)
		}

		value, err := cache.Get("large")
		if err != nil {
			t.Errorf("Get large data failed: %v", err)
		}

		if string(value) != largeData {
			t.Error("Decompressed data should match original")
		}
	})

	// 测试TTL功能
	t.Run("TTL functionality", func(t *testing.T) {
		cache := NewCache(WithMaxSize(1024), WithEvictionPolicy("LRU"))

		// 设置短TTL
		err := cache.Set("ttl_key", toBytes("ttl_value"), 10*time.Millisecond)
		if err != nil {
			t.Errorf("Set with TTL failed: %v", err)
		}

		// 立即获取应该成功
		value, err := cache.Get("ttl_key")
		if err != nil {
			t.Errorf("Get immediately after set failed: %v", err)
		}
		if string(value) != "ttl_value" {
			t.Errorf("Expected ttl_value, got %v", string(value))
		}

		// 等待过期
		time.Sleep(15 * time.Millisecond)

		// 现在应该获取不到
		_, err = cache.Get("ttl_key")
		if err == nil {
			t.Error("Should not get expired key")
		}
	})
}

func TestCacheErrorCases(t *testing.T) {
	// 测试获取不存在的键
	t.Run("get non-existent key", func(t *testing.T) {
		cache := NewCache(WithMaxSize(1024), WithEvictionPolicy("LRU"))

		_, err := cache.Get("nonexistent")
		if err == nil {
			t.Error("Should return error for non-existent key")
		}
	})

	// 测试删除不存在的键（Delete方法不返回错误）
	t.Run("delete non-existent key", func(t *testing.T) {
		cache := NewCache(WithMaxSize(1024), WithEvictionPolicy("LRU"))

		// Delete方法不返回错误，直接调用
		cache.Delete("nonexistent")
	})
}

func TestCacheInvalidPolicy(t *testing.T) {
	// 测试无效的淘汰策略
	cache := NewCache(WithMaxSize(1024), WithEvictionPolicy("INVALID_POLICY"))

	stats := cache.Stats()
	if stats.EvictionPolicy != "LRU" {
		t.Errorf("Invalid policy should default to LRU, got %s", stats.EvictionPolicy)
	}
}

func TestCacheShardingBehavior(t *testing.T) {
	cache := NewCache(WithMaxSize(1024), WithEvictionPolicy("LRU"))

	// 测试键在不同分片中的分布
	keyShardMap := make(map[int][]string)

	for i := 0; i < 100; i++ {
		key := "shard_test_" + string(rune(i))
		shard := cache.getShard(key)

		// 找到分片索引
		shardIndex := -1
		for j, s := range cache.shards {
			if s == shard {
				shardIndex = j
				break
			}
		}

		if shardIndex == -1 {
			t.Error("Could not find shard index")
		}

		keyShardMap[shardIndex] = append(keyShardMap[shardIndex], key)

		// 设置值
		err := cache.Set(key, toBytes("value"), 0)
		if err != nil {
			t.Errorf("Set failed for key %s: %v", key, err)
		}
	}

	// 验证键分布在多个分片中
	nonEmptyShards := 0
	for _, keys := range keyShardMap {
		if len(keys) > 0 {
			nonEmptyShards++
		}
	}

	if nonEmptyShards < 2 {
		t.Error("Keys should be distributed across multiple shards")
	}
}

func TestCacheDataTypes(t *testing.T) {
	cache := NewCache(WithMaxSize(1024), WithEvictionPolicy("LRU"))

	// 测试基本数据类型
	t.Run("string", func(t *testing.T) {
		err := cache.Set("str_key", toBytes("string value"), 0)
		if err != nil {
			t.Error("Set string failed")
		}

		value, err := cache.Get("str_key")
		if err != nil {
			t.Error("Get string failed")
		}

		if string(value) != "string value" {
			t.Error("String value mismatch")
		}
	})

	t.Run("binary data", func(t *testing.T) {
		binaryData := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
		err := cache.Set("bin_key", binaryData, 0)
		if err != nil {
			t.Error("Set binary data failed")
		}

		value, err := cache.Get("bin_key")
		if err != nil {
			t.Error("Get binary data failed")
		}

		if len(value) != len(binaryData) {
			t.Error("Binary data length mismatch")
		}

		for i, b := range value {
			if b != binaryData[i] {
				t.Error("Binary data content mismatch")
			}
		}
	})
}

func TestCacheStressTest(t *testing.T) {
	cache := NewCache(WithMaxSize(1024*1024), WithEvictionPolicy("LRU"))

	// 压力测试：大量并发操作
	var wg sync.WaitGroup
	numGoroutines := 10
	opsPerGoroutine := 1000

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := fmt.Sprintf("stress_key_%d_%d", id, j)
				value := fmt.Sprintf("stress_value_%d_%d", id, j)

				// 设置值
				err := cache.Set(key, toBytes(value), 0)
				if err != nil {
					t.Errorf("Set failed in stress test: %v", err)
				}

				// 获取值
				retrieved, err := cache.Get(key)
				if err != nil {
					t.Errorf("Get failed in stress test: %v", err)
				}

				if string(retrieved) != value {
					t.Errorf("Value mismatch in stress test: got %s, want %s", string(retrieved), value)
				}
			}
		}(i)
	}

	wg.Wait()

	// 验证最终状态
	stats := cache.Stats()
	if stats.CurrentCount == 0 {
		t.Error("Cache should contain items after stress test")
	}
}

func TestCacheAdditionalCoverage(t *testing.T) {
	cache := NewCache(WithMaxSize(1024), WithEvictionPolicy("LRU"))

	// 测试空值
	t.Run("empty value", func(t *testing.T) {
		err := cache.Set("empty_key", []byte{}, 0)
		if err != nil {
			t.Error("Set empty value failed")
		}

		value, err := cache.Get("empty_key")
		if err != nil {
			t.Error("Get empty value failed")
		}

		if len(value) != 0 {
			t.Error("Empty value should have length 0")
		}
	})

	// 测试nil值
	t.Run("nil value", func(t *testing.T) {
		err := cache.Set("nil_key", nil, 0)
		if err != nil {
			t.Error("Set nil value failed")
		}

		value, err := cache.Get("nil_key")
		if err != nil {
			t.Error("Get nil value failed")
		}

		if value != nil {
			t.Error("Nil value should remain nil")
		}
	})
}

func TestInternalFunctionsCoverage(t *testing.T) {
	cache := NewCache(WithMaxSize(1024), WithEvictionPolicy("LRU"))

	// 测试内部函数
	t.Run("getShard", func(t *testing.T) {
		shard1 := cache.getShard("key1")
		shard2 := cache.getShard("key2")

		if shard1 == nil {
			t.Error("getShard should not return nil")
		}

		if shard2 == nil {
			t.Error("getShard should not return nil")
		}
	})
}

func TestEdgeCasesAndErrorPaths(t *testing.T) {
	cache := NewCache(WithMaxSize(1024*1024), WithEvictionPolicy("LRU"))

	// 测试边界情况
	t.Run("very large key", func(t *testing.T) {
		largeKey := strings.Repeat("a", 10000)
		err := cache.Set(largeKey, toBytes("value"), 0)
		if err != nil {
			t.Errorf("Set with large key failed: %v", err)
		}

		value, err := cache.Get(largeKey)
		if err != nil {
			t.Errorf("Get with large key failed: %v", err)
		}

		if string(value) != "value" {
			t.Error("Large key value mismatch")
		}
	})

	t.Run("very large value", func(t *testing.T) {
		largeValue := strings.Repeat("x", 5000)
		err := cache.Set("large_value_key", toBytes(largeValue), 0)
		if err != nil {
			t.Errorf("Set with large value failed: %v", err)
		}

		value, err := cache.Get("large_value_key")
		if err != nil {
			t.Errorf("Get with large value failed: %v", err)
		}

		if string(value) != largeValue {
			t.Error("Large value mismatch")
		}
	})
}
