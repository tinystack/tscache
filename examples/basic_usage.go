package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/tinystack/tscache"
)

// User 示例用户结构体
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// DemonstrateBasicUsage 演示TSCache的基本使用方法
func DemonstrateBasicUsage() {
	fmt.Println("=== TSCache 基础使用示例 ===")

	// 创建缓存实例
	cache := tscache.NewCache(
		tscache.WithMaxSize(1024*1024),    // 1MB 最大内存
		tscache.WithEvictionPolicy("LRU"), // LRU 淘汰策略
	)

	// 1. 基本的 Set/Get 操作
	fmt.Println("\n1. 基本 Set/Get 操作:")

	// 存储字符串数据
	key := "greeting"
	value := "Hello, TSCache!"
	err := cache.Set(key, []byte(value), 0) // 0 表示永不过期
	if err != nil {
		fmt.Printf("存储失败: %v\n", err)
		return
	}
	fmt.Printf("存储: %s = %s\n", key, value)

	// 获取数据
	retrieved, err := cache.Get(key)
	if err != nil {
		fmt.Printf("获取失败: %v\n", err)
	} else {
		fmt.Printf("获取: %s = %s\n", key, string(retrieved))
	}

	// 2. JSON 数据序列化示例
	fmt.Println("\n2. JSON 数据序列化:")

	user := User{
		ID:    1,
		Name:  "张三",
		Email: "zhangsan@example.com",
	}

	// 序列化为JSON
	userJSON, err := json.Marshal(user)
	if err != nil {
		fmt.Printf("JSON序列化失败: %v\n", err)
		return
	}

	// 存储JSON数据
	userKey := "user:1"
	err = cache.Set(userKey, userJSON, 0)
	if err != nil {
		fmt.Printf("存储用户数据失败: %v\n", err)
		return
	}
	fmt.Printf("存储用户: %+v\n", user)

	// 获取并反序列化JSON数据
	retrievedJSON, err := cache.Get(userKey)
	if err != nil {
		fmt.Printf("获取用户数据失败: %v\n", err)
	} else {
		var retrievedUser User
		err = json.Unmarshal(retrievedJSON, &retrievedUser)
		if err != nil {
			fmt.Printf("JSON反序列化失败: %v\n", err)
		} else {
			fmt.Printf("获取用户: %+v\n", retrievedUser)
		}
	}

	// 3. TTL (生存时间) 示例
	fmt.Println("\n3. TTL 过期示例:")

	tempKey := "temp_data"
	tempValue := "这个数据将在3秒后过期"
	ttl := 3 * time.Second

	err = cache.Set(tempKey, []byte(tempValue), ttl)
	if err != nil {
		fmt.Printf("存储临时数据失败: %v\n", err)
		return
	}
	fmt.Printf("存储临时数据 (TTL: %v): %s\n", ttl, tempValue)

	// 立即获取
	if data, err := cache.Get(tempKey); err == nil {
		fmt.Printf("立即获取成功: %s\n", string(data))
	}

	// 等待2秒后获取
	fmt.Println("等待2秒...")
	time.Sleep(2 * time.Second)
	if data, err := cache.Get(tempKey); err == nil {
		fmt.Printf("2秒后获取成功: %s\n", string(data))
	}

	// 等待2秒后获取（总共4秒，应该过期）
	fmt.Println("再等待2秒...")
	time.Sleep(2 * time.Second)
	if _, err := cache.Get(tempKey); err != nil {
		fmt.Printf("4秒后获取失败: %v (数据已过期)\n", err)
	}

	// 4. Delete 操作示例
	fmt.Println("\n4. Delete 操作:")

	deleteKey := "to_be_deleted"
	cache.Set(deleteKey, []byte("这个数据将被删除"), 0)
	fmt.Printf("存储数据: %s\n", deleteKey)

	// 验证数据存在
	if _, err := cache.Get(deleteKey); err == nil {
		fmt.Printf("删除前: 数据存在\n")
	}

	// 删除数据
	cache.Delete(deleteKey)
	fmt.Printf("执行删除操作\n")

	// 验证数据已删除
	if _, err := cache.Get(deleteKey); err != nil {
		fmt.Printf("删除后: %v\n", err)
	}

	// 5. 统计信息示例
	fmt.Println("\n5. 缓存统计信息:")

	// 添加一些测试数据
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("test_key_%d", i)
		value := fmt.Sprintf("test_value_%d", i)
		cache.Set(key, []byte(value), 0)
	}

	// 执行一些Get操作来产生命中和未命中
	for i := 0; i < 15; i++ {
		key := fmt.Sprintf("test_key_%d", i)
		cache.Get(key) // 前10个会命中，后5个会未命中
	}

	// 获取统计信息
	stats := cache.Stats()
	fmt.Printf("缓存统计:\n")
	fmt.Printf("  命中次数: %d\n", stats.Hits)
	fmt.Printf("  未命中次数: %d\n", stats.Misses)
	fmt.Printf("  命中率: %.2f%%\n", float64(stats.Hits)/float64(stats.Hits+stats.Misses)*100)
	fmt.Printf("  当前项目数: %d\n", stats.CurrentCount)
	fmt.Printf("  当前内存使用: %d 字节\n", stats.CurrentSize)
	fmt.Printf("  最大内存限制: %d 字节\n", stats.MaxSize)
	fmt.Printf("  淘汰次数: %d\n", stats.Evictions)
	fmt.Printf("  淘汰策略: %s\n", stats.EvictionPolicy)
	fmt.Printf("  分片数量: %d\n", stats.ShardCount)

	// 6. Clear 操作示例
	fmt.Println("\n6. Clear 操作:")
	fmt.Printf("清空前项目数: %d\n", cache.Stats().CurrentCount)

	cache.Clear()
	fmt.Println("执行清空操作")

	finalStats := cache.Stats()
	fmt.Printf("清空后项目数: %d\n", finalStats.CurrentCount)
	fmt.Printf("清空后内存使用: %d 字节\n", finalStats.CurrentSize)
}
