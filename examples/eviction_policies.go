package main

import (
	"fmt"
	"time"

	"github.com/tinystack/tscache"
)

// DemonstrateEvictionPolicies 演示不同的淘汰策略
func DemonstrateEvictionPolicies() {
	fmt.Println("=== TSCache 淘汰策略对比示例 ===")

	// 演示不同的淘汰策略
	demonstrateLRU()
	demonstrateLFU()
	demonstrateFIFO()
	compareEvictionPolicies()
}

func demonstrateLRU() {
	fmt.Println("\n=== LRU (最近最少使用) 策略 ===")

	// 创建小容量缓存来快速触发淘汰
	cache := tscache.NewCache(
		tscache.WithMaxSize(200), // 200字节限制
		tscache.WithEvictionPolicy("LRU"),
	)

	// 添加数据
	fmt.Println("添加数据:")
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("这是一个较长的值用于测试LRU策略_%d", i)
		cache.Set(key, []byte(value), 0)
		fmt.Printf("  设置 %s\n", key)
	}

	// 显示当前状态
	stats := cache.Stats()
	fmt.Printf("当前项目数: %d, 淘汰次数: %d\n", stats.CurrentCount, stats.Evictions)

	// 访问一些键，使它们成为"最近使用"
	fmt.Println("\n访问 key1 和 key2 (使它们成为最近使用):")
	cache.Get("key1")
	cache.Get("key2")

	// 添加新数据触发更多淘汰
	fmt.Println("\n添加更多数据触发淘汰:")
	for i := 6; i <= 8; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("这是一个较长的值用于测试LRU策略_%d", i)
		cache.Set(key, []byte(value), 0)
		fmt.Printf("  设置 %s\n", key)
	}

	// 检查哪些键还存在
	fmt.Println("\n检查剩余的键:")
	for i := 1; i <= 8; i++ {
		key := fmt.Sprintf("key%d", i)
		if _, err := cache.Get(key); err == nil {
			fmt.Printf("  %s: 存在\n", key)
		} else {
			fmt.Printf("  %s: 已被淘汰\n", key)
		}
	}

	finalStats := cache.Stats()
	fmt.Printf("最终统计: 项目数=%d, 淘汰次数=%d\n",
		finalStats.CurrentCount, finalStats.Evictions)
}

func demonstrateLFU() {
	fmt.Println("\n=== LFU (最少使用频率) 策略 ===")

	cache := tscache.NewCache(
		tscache.WithMaxSize(200),
		tscache.WithEvictionPolicy("LFU"),
	)

	// 添加数据
	fmt.Println("添加数据:")
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("这是一个较长的值用于测试LFU策略_%d", i)
		cache.Set(key, []byte(value), 0)
		fmt.Printf("  设置 %s\n", key)
	}

	// 多次访问某些键，增加它们的使用频率
	fmt.Println("\n多次访问 key1 和 key2 (增加使用频率):")
	for i := 0; i < 5; i++ {
		cache.Get("key1")
		cache.Get("key2")
	}
	fmt.Println("  key1 和 key2 各被访问 5 次")

	// 少量访问其他键
	cache.Get("key3")
	fmt.Println("  key3 被访问 1 次")

	// 添加新数据触发淘汰
	fmt.Println("\n添加更多数据触发淘汰:")
	for i := 6; i <= 8; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("这是一个较长的值用于测试LFU策略_%d", i)
		cache.Set(key, []byte(value), 0)
		fmt.Printf("  设置 %s\n", key)
	}

	// 检查哪些键还存在
	fmt.Println("\n检查剩余的键:")
	for i := 1; i <= 8; i++ {
		key := fmt.Sprintf("key%d", i)
		if _, err := cache.Get(key); err == nil {
			fmt.Printf("  %s: 存在\n", key)
		} else {
			fmt.Printf("  %s: 已被淘汰\n", key)
		}
	}

	finalStats := cache.Stats()
	fmt.Printf("最终统计: 项目数=%d, 淘汰次数=%d\n",
		finalStats.CurrentCount, finalStats.Evictions)
}

func demonstrateFIFO() {
	fmt.Println("\n=== FIFO (先进先出) 策略 ===")

	cache := tscache.NewCache(
		tscache.WithMaxSize(200),
		tscache.WithEvictionPolicy("FIFO"),
	)

	// 添加数据
	fmt.Println("添加数据:")
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("这是一个较长的值用于测试FIFO策略_%d", i)
		cache.Set(key, []byte(value), 0)
		fmt.Printf("  设置 %s (时间: %v)\n", key, time.Now().Format("15:04:05.000"))
		time.Sleep(10 * time.Millisecond) // 确保时间差异
	}

	// 访问所有键（FIFO不受访问影响）
	fmt.Println("\n访问所有键 (FIFO策略不受访问模式影响):")
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("key%d", i)
		cache.Get(key)
		fmt.Printf("  访问 %s\n", key)
	}

	// 添加新数据触发淘汰
	fmt.Println("\n添加更多数据触发淘汰:")
	for i := 6; i <= 8; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("这是一个较长的值用于测试FIFO策略_%d", i)
		cache.Set(key, []byte(value), 0)
		fmt.Printf("  设置 %s (时间: %v)\n", key, time.Now().Format("15:04:05.000"))
		time.Sleep(10 * time.Millisecond)
	}

	// 检查哪些键还存在
	fmt.Println("\n检查剩余的键:")
	for i := 1; i <= 8; i++ {
		key := fmt.Sprintf("key%d", i)
		if _, err := cache.Get(key); err == nil {
			fmt.Printf("  %s: 存在\n", key)
		} else {
			fmt.Printf("  %s: 已被淘汰 (最早添加)\n", key)
		}
	}

	finalStats := cache.Stats()
	fmt.Printf("最终统计: 项目数=%d, 淘汰次数=%d\n",
		finalStats.CurrentCount, finalStats.Evictions)
}

func compareEvictionPolicies() {
	fmt.Println("\n=== 淘汰策略性能对比 ===")

	policies := []string{"LRU", "LFU", "FIFO"}

	for _, policy := range policies {
		fmt.Printf("\n测试 %s 策略:\n", policy)

		cache := tscache.NewCache(
			tscache.WithMaxSize(1024*1024), // 1MB
			tscache.WithEvictionPolicy(policy),
		)

		start := time.Now()

		// 执行大量操作
		for i := 0; i < 10000; i++ {
			key := fmt.Sprintf("key_%d", i%1000) // 重复使用键
			value := fmt.Sprintf("value_%d", i)
			cache.Set(key, []byte(value), 0)

			if i%2 == 0 {
				cache.Get(key)
			}
		}

		duration := time.Since(start)
		stats := cache.Stats()

		fmt.Printf("  耗时: %v\n", duration)
		fmt.Printf("  命中率: %.2f%%\n",
			float64(stats.Hits)/float64(stats.Hits+stats.Misses)*100)
		fmt.Printf("  淘汰次数: %d\n", stats.Evictions)
		fmt.Printf("  最终项目数: %d\n", stats.CurrentCount)
	}
}
