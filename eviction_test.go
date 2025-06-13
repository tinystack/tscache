package tscache

import (
	"testing"
	"time"
)

func TestLRUList(t *testing.T) {
	lru := NewLRUList()

	// 测试添加项目
	t.Run("Add items", func(t *testing.T) {
		item1 := &CacheItem{Key: "key1", Value: []byte("value1"), CreatedAt: time.Now()}
		item2 := &CacheItem{Key: "key2", Value: []byte("value2"), CreatedAt: time.Now()}

		lru.Add("key1", item1)
		lru.Add("key2", item2)

		// 测试更新现有项目
		lru.Add("key1", item1) // 应该移动到前面
	})

	// 测试移除项目
	t.Run("Remove items", func(t *testing.T) {
		lru.Remove("key1")
		lru.Remove("nonexistent") // 不应该出错
	})

	// 测试更新项目
	t.Run("Update items", func(t *testing.T) {
		item2 := &CacheItem{Key: "key2", Value: []byte("updated_value2"), AccessAt: time.Now()}
		lru.Update("key2", item2)
		lru.Update("nonexistent", item2) // 不应该出错
	})

	// 测试移除最少使用的项目
	t.Run("RemoveLeast", func(t *testing.T) {
		// 添加一些测试项目
		for i := 1; i <= 5; i++ {
			item := &CacheItem{
				Key:       "test" + string(rune(i)),
				Value:     []byte("value" + string(rune(i))),
				CreatedAt: time.Now(),
			}
			lru.Add("test"+string(rune(i)), item)
		}

		// 移除最少使用的项目
		removedKey := lru.RemoveLeast()
		if removedKey == "" {
			t.Error("RemoveLeast should return a key")
		}

		// 再次移除
		removedKey2 := lru.RemoveLeast()
		if removedKey2 == "" {
			t.Error("RemoveLeast should return another key")
		}
	})

	// 测试清空
	t.Run("Clear", func(t *testing.T) {
		lru.Clear()

		// 清空后移除应该返回空字符串
		removedKey := lru.RemoveLeast()
		if removedKey != "" {
			t.Error("RemoveLeast after clear should return empty string")
		}
	})
}

func TestLFUList(t *testing.T) {
	lfu := NewLFUList()

	// 测试添加项目
	t.Run("Add items", func(t *testing.T) {
		item1 := &CacheItem{Key: "key1", Value: []byte("value1"), AccessCount: 1}
		item2 := &CacheItem{Key: "key2", Value: []byte("value2"), AccessCount: 1}

		lfu.Add("key1", item1)
		lfu.Add("key2", item2)

		// 测试更新现有项目（增加频率）
		item1.AccessCount = 2
		lfu.Add("key1", item1)
	})

	// 测试移除项目
	t.Run("Remove items", func(t *testing.T) {
		lfu.Remove("key1")
		lfu.Remove("nonexistent") // 不应该出错
	})

	// 测试更新项目
	t.Run("Update items", func(t *testing.T) {
		item2 := &CacheItem{Key: "key2", Value: []byte("updated_value2"), AccessCount: 3}
		lfu.Update("key2", item2)
		lfu.Update("nonexistent", item2) // 不应该出错
	})

	// 测试移除最少使用的项目
	t.Run("RemoveLeast", func(t *testing.T) {
		// 添加一些测试项目
		for i := 1; i <= 5; i++ {
			item := &CacheItem{
				Key:         "test" + string(rune(i)),
				Value:       []byte("value" + string(rune(i))),
				AccessCount: i, // 不同的访问频率
			}
			lfu.Add("test"+string(rune(i)), item)
		}

		// 移除最少使用的项目（应该是频率最低的）
		removedKey := lfu.RemoveLeast()
		if removedKey == "" {
			t.Error("RemoveLeast should return a key")
		}
	})

	// 测试清空
	t.Run("Clear", func(t *testing.T) {
		lfu.Clear()

		// 清空后移除应该返回空字符串
		removedKey := lfu.RemoveLeast()
		if removedKey != "" {
			t.Error("RemoveLeast after clear should return empty string")
		}
	})
}

func TestFIFOList(t *testing.T) {
	fifo := NewFIFOList()

	// 测试添加项目
	t.Run("Add items", func(t *testing.T) {
		item1 := &CacheItem{Key: "key1", Value: []byte("value1"), CreatedAt: time.Now()}
		item2 := &CacheItem{Key: "key2", Value: []byte("value2"), CreatedAt: time.Now()}

		fifo.Add("key1", item1)
		fifo.Add("key2", item2)

		// 测试更新现有项目（FIFO不改变位置）
		fifo.Add("key1", item1)
	})

	// 测试移除项目
	t.Run("Remove items", func(t *testing.T) {
		fifo.Remove("key1")
		fifo.Remove("nonexistent") // 不应该出错
	})

	// 测试更新项目
	t.Run("Update items", func(t *testing.T) {
		item2 := &CacheItem{Key: "key2", Value: []byte("updated_value2")}
		fifo.Update("key2", item2)
		fifo.Update("nonexistent", item2) // 不应该出错
	})

	// 测试移除最先进入的项目
	t.Run("RemoveLeast", func(t *testing.T) {
		// 添加一些测试项目
		for i := 1; i <= 5; i++ {
			item := &CacheItem{
				Key:       "test" + string(rune(i)),
				Value:     []byte("value" + string(rune(i))),
				CreatedAt: time.Now(),
			}
			fifo.Add("test"+string(rune(i)), item)
			time.Sleep(1 * time.Millisecond) // 确保时间戳不同
		}

		// 移除最先进入的项目
		removedKey := fifo.RemoveLeast()
		if removedKey == "" {
			t.Error("RemoveLeast should return a key")
		}
	})

	// 测试清空
	t.Run("Clear", func(t *testing.T) {
		fifo.Clear()

		// 清空后移除应该返回空字符串
		removedKey := fifo.RemoveLeast()
		if removedKey != "" {
			t.Error("RemoveLeast after clear should return empty string")
		}
	})
}

func TestEvictionPolicyIntegration(t *testing.T) {
	policies := []string{"LRU", "LFU", "FIFO"}

	for _, policy := range policies {
		t.Run(policy+" integration", func(t *testing.T) {
			cache := NewCache(WithMaxSize(1024), WithEvictionPolicy(policy))

			// 添加一些数据
			for i := 0; i < 10; i++ {
				key := "key" + string(rune(i))
				value := "value" + string(rune(i))
				err := cache.Set(key, []byte(value), 0)
				if err != nil {
					t.Errorf("Failed to set %s: %v", key, err)
				}
			}

			// 根据策略访问一些数据
			switch policy {
			case "LRU":
				// 访问前几个键，使它们成为最近使用的
				for i := 0; i < 3; i++ {
					key := "key" + string(rune(i))
					cache.Get(key)
				}
			case "LFU":
				// 多次访问前几个键，增加它们的使用频率
				for i := 0; i < 3; i++ {
					key := "key" + string(rune(i))
					for j := 0; j < 5; j++ {
						cache.Get(key)
					}
				}
			case "FIFO":
				// FIFO不需要特殊访问
			}

			// 验证统计信息
			stats := cache.Stats()
			if stats.CurrentCount == 0 {
				t.Error("Cache should contain items")
			}

			if stats.EvictionPolicy != policy {
				t.Errorf("Expected policy %s, got %s", policy, stats.EvictionPolicy)
			}
		})
	}
}

func TestLFUSpecificFunctionality(t *testing.T) {
	lfu := NewLFUList()

	// 测试频率管理
	t.Run("frequency management", func(t *testing.T) {
		// 添加具有不同频率的项目
		item1 := &CacheItem{Key: "low", Value: []byte("value1"), AccessCount: 1}
		item2 := &CacheItem{Key: "high", Value: []byte("value2"), AccessCount: 10}

		lfu.Add("low", item1)
		lfu.Add("high", item2)

		// 移除最少使用的（应该是low）
		removed := lfu.RemoveLeast()
		if removed != "low" {
			t.Errorf("Expected to remove 'low', got '%s'", removed)
		}

		// 再次移除应该得到high
		removed = lfu.RemoveLeast()
		if removed != "high" {
			t.Errorf("Expected to remove 'high', got '%s'", removed)
		}
	})
}
