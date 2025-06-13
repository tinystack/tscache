package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinystack/tscache"
)

// DemonstrateCompression 演示压缩功能
func DemonstrateCompression() {
	fmt.Println("=== TSCache 压缩功能示例 ===")

	// 演示不同的压缩算法
	demonstrateGzipCompression()
	demonstrateZstdCompression()
	demonstrateNoCompression()
	compareCompressionPerformance()
}

func demonstrateGzipCompression() {
	fmt.Println("\n=== Gzip 压缩示例 ===")

	cache := tscache.NewCache(
		tscache.WithCompressor(tscache.NewGzipCompressor()),
		tscache.WithMaxSize(1024*1024), // 1MB
	)

	// 创建一个大的、重复性高的字符串（适合压缩）
	largeData := strings.Repeat("这是一个重复的字符串，用于测试压缩效果。", 100)
	originalSize := len([]byte(largeData))

	fmt.Printf("原始数据大小: %d 字节\n", originalSize)

	// 存储数据
	start := time.Now()
	cache.Set("large_data", []byte(largeData), 0)
	setDuration := time.Since(start)

	// 获取数据
	start = time.Now()
	retrieved, err := cache.Get("large_data")
	getDuration := time.Since(start)

	if err != nil {
		fmt.Printf("获取数据失败: %v\n", err)
		return
	}

	fmt.Printf("压缩存储耗时: %v\n", setDuration)
	fmt.Printf("解压获取耗时: %v\n", getDuration)
	fmt.Printf("数据完整性: %v\n", string(retrieved) == largeData)

	stats := cache.Stats()
	fmt.Printf("缓存统计: 项目数=%d, 总大小=%d 字节\n",
		stats.CurrentCount, stats.CurrentSize)
}

func demonstrateZstdCompression() {
	fmt.Println("\n=== Zstd 压缩示例 ===")

	// 创建Zstd压缩器
	zstdCompressor, err := tscache.NewZstdCompressor()
	if err != nil {
		fmt.Printf("创建Zstd压缩器失败: %v\n", err)
		return
	}

	cache := tscache.NewCache(
		tscache.WithCompressor(zstdCompressor),
		tscache.WithMaxSize(1024*1024),
	)

	// 测试不同类型的数据
	testData := map[string]string{
		"json_data":  `{"users":[{"id":1,"name":"张三","email":"zhangsan@example.com"},{"id":2,"name":"李四","email":"lisi@example.com"},{"id":3,"name":"王五","email":"wangwu@example.com"}]}`,
		"text_data":  strings.Repeat("Zstd是一个高效的压缩算法。", 50),
		"mixed_data": "数字123，英文ABC，符号!@#，中文测试" + strings.Repeat("混合内容", 30),
	}

	fmt.Println("测试不同类型数据的压缩:")
	for key, data := range testData {
		originalSize := len([]byte(data))

		start := time.Now()
		cache.Set(key, []byte(data), 0)
		setTime := time.Since(start)

		start = time.Now()
		retrieved, err := cache.Get(key)
		getTime := time.Since(start)

		if err != nil {
			fmt.Printf("  %s: 获取失败 - %v\n", key, err)
			continue
		}

		fmt.Printf("  %s:\n", key)
		fmt.Printf("    原始大小: %d 字节\n", originalSize)
		fmt.Printf("    存储耗时: %v\n", setTime)
		fmt.Printf("    获取耗时: %v\n", getTime)
		fmt.Printf("    数据正确: %v\n", string(retrieved) == data)
	}
}

func demonstrateNoCompression() {
	fmt.Println("\n=== 无压缩示例 ===")

	cache := tscache.NewCache(
		tscache.WithCompressor(tscache.NewNoCompressor()),
		tscache.WithMaxSize(1024*1024),
	)

	// 测试大量小数据的存储
	fmt.Println("批量存储小数据:")
	start := time.Now()

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("item_%d", i)
		value := fmt.Sprintf("这是第%d个测试项目，不使用压缩存储。", i)
		cache.Set(key, []byte(value), 0)
	}

	batchSetTime := time.Since(start)

	// 批量获取
	start = time.Now()
	successCount := 0

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("item_%d", i)
		if _, err := cache.Get(key); err == nil {
			successCount++
		}
	}

	batchGetTime := time.Since(start)

	stats := cache.Stats()
	fmt.Printf("批量存储1000项耗时: %v\n", batchSetTime)
	fmt.Printf("批量获取1000项耗时: %v\n", batchGetTime)
	fmt.Printf("成功获取: %d/1000\n", successCount)
	fmt.Printf("缓存统计: 项目数=%d, 命中率=%.2f%%\n",
		stats.CurrentCount,
		float64(stats.Hits)/float64(stats.Hits+stats.Misses)*100)
}

func compareCompressionPerformance() {
	fmt.Println("\n=== 压缩算法性能对比 ===")

	// 测试数据
	testData := strings.Repeat("性能测试数据，包含重复内容以便压缩。", 200)
	originalSize := len([]byte(testData))

	fmt.Printf("测试数据大小: %d 字节\n", originalSize)
	fmt.Println("算法对比结果:")

	// 测试无压缩
	fmt.Printf("\n无压缩:\n")
	testCompressionAlgorithm("none", tscache.NewNoCompressor(), testData, originalSize)

	// 测试Gzip压缩
	fmt.Printf("\nGZIP 压缩:\n")
	testCompressionAlgorithm("gzip", tscache.NewGzipCompressor(), testData, originalSize)

	// 测试Zstd压缩
	fmt.Printf("\nZSTD 压缩:\n")
	if zstdCompressor, err := tscache.NewZstdCompressor(); err == nil {
		testCompressionAlgorithm("zstd", zstdCompressor, testData, originalSize)
	} else {
		fmt.Printf("  Zstd压缩器创建失败: %v\n", err)
	}
}

func testCompressionAlgorithm(name string, compressor tscache.Compressor, testData string, originalSize int) {
	cache := tscache.NewCache(
		tscache.WithCompressor(compressor),
		tscache.WithMaxSize(1024*1024),
	)

	// 测试存储性能
	start := time.Now()
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("test_%d", i)
		cache.Set(key, []byte(testData), 0)
	}
	setTime := time.Since(start)

	// 测试获取性能
	start = time.Now()
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("test_%d", i)
		cache.Get(key)
	}
	getTime := time.Since(start)

	stats := cache.Stats()

	fmt.Printf("  存储100次耗时: %v (平均: %v)\n",
		setTime, setTime/100)
	fmt.Printf("  获取100次耗时: %v (平均: %v)\n",
		getTime, getTime/100)
	fmt.Printf("  缓存大小: %d 字节\n", stats.CurrentSize)

	if name != "none" && stats.CurrentSize > 0 {
		compressionRatio := float64(originalSize*100) / float64(stats.CurrentSize)
		fmt.Printf("  压缩比: %.2fx (原始: %d → 压缩后: %d)\n",
			compressionRatio, originalSize*100, stats.CurrentSize)
	}
}

// 演示压缩与TTL结合使用
func demonstrateCompressionWithTTL() {
	fmt.Println("\n=== 压缩 + TTL 示例 ===")

	cache := tscache.NewCache(
		tscache.WithCompressor(tscache.NewGzipCompressor()),
		tscache.WithMaxSize(1024*1024),
	)

	// 存储带TTL的压缩数据
	data := strings.Repeat("临时数据，将在5秒后过期。", 50)
	cache.Set("temp_data", []byte(data), 5*time.Second)

	fmt.Printf("存储临时数据 (5秒TTL): %d 字节\n", len([]byte(data)))

	// 立即获取
	if retrieved, err := cache.Get("temp_data"); err == nil {
		fmt.Printf("立即获取成功: %d 字节\n", len(retrieved))
	}

	// 等待3秒后获取
	fmt.Println("等待3秒...")
	time.Sleep(3 * time.Second)

	if retrieved, err := cache.Get("temp_data"); err == nil {
		fmt.Printf("3秒后获取成功: %d 字节\n", len(retrieved))
	}

	// 等待3秒后获取（总共6秒，应该过期）
	fmt.Println("再等待3秒...")
	time.Sleep(3 * time.Second)

	if _, err := cache.Get("temp_data"); err != nil {
		fmt.Printf("6秒后获取失败: %v (数据已过期)\n", err)
	}

	stats := cache.Stats()
	fmt.Printf("最终统计: 项目数=%d, 淘汰次数=%d\n",
		stats.CurrentCount, stats.Evictions)
}
