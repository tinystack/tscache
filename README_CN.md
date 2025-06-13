# TSCache - Go 语言高性能内存缓存库

[🇨🇳 中文文档](README_CN.md) | [🇺🇸 English](README.md)

TSCache 是一个为 Go 应用程序设计的高性能、线程安全的内存缓存库。它提供了内存管理、多种淘汰策略、数据压缩和自动分片等高级功能。

[![Go Report Card](https://goreportcard.com/badge/github.com/tinystack/tscache)](https://goreportcard.com/report/github.com/tinystack/tscache)
![Go Version](https://img.shields.io/badge/go%20version-%3E=1.22.0-61CFDD.svg?style=flat-square)
[![PkgGoDev](https://pkg.go.dev/badge/mod/github.com/tinystack/tscache)](https://pkg.go.dev/mod/github.com/tinystack/tscache)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 特性

- **内存管理**: 设置最大内存使用量，超出限制时自动触发淘汰机制
- **数据过期**: 支持 TTL（生存时间），自动清理过期数据
- **多种淘汰策略**: LRU（最近最少使用）、LFU（最少使用频率）和 FIFO（先进先出）
- **线程安全**: 支持并发访问，采用优化的锁定策略
- **分片缓存**: 自动分片以减少锁竞争，提高性能
- **内存优化**: 高效的数据结构，保证 O(1)或 O(log n)的查询复杂度
- **数据压缩**: 对大数据自动压缩以减少内存占用
- **统计信息**: 全面的缓存统计，包括命中率和淘汰次数
- **仅内存淘汰**: 淘汰机制仅基于内存使用量，不限制项目数量

## 安装

```bash
go get github.com/tinystack/tscache
```

## 快速开始

```go
package main

import (
    "fmt"
    "time"
    "github.com/tinystack/tscache"
)

func main() {
    // 创建缓存：最大内存10MB，项目数量无限制（maxCount已被忽略），LRU淘汰策略
    cache := tscache.NewCache(10*1024*1024, 0, "LRU")

    // 设置永不过期的值
    cache.Set("user:1", "Alice", 0)

    // 设置5分钟TTL的值
    cache.Set("session:abc", "user_data", 5*time.Minute)

    // 获取值
    if value, err := cache.Get("user:1"); err == nil {
        fmt.Printf("用户: %v\n", value)
    }

    // 删除值
    cache.Delete("user:1")

    // 获取缓存统计信息
    stats := cache.Stats()
    fmt.Printf("命中: %d, 未命中: %d, 项目数: %d\n",
        stats.Hits, stats.Misses, stats.CurrentCount)

    // 清空所有缓存
    cache.Clear()
}
```

## API 参考

### 创建缓存

```go
func NewCache(maxSize int64, maxCount int64, evictionPolicy string) *Cache
```

- `maxSize`: 最大内存使用量（字节）
- `maxCount`: **已废弃并被忽略** - 缓存不再限制项目数量，只限制内存使用
- `evictionPolicy`: 淘汰策略（"LRU"、"LFU" 或 "FIFO"）

### 缓存操作

```go
// 设置缓存项
func (c *Cache) Set(key string, value any, ttl time.Duration) error

// 获取缓存项
func (c *Cache) Get(key string) (any, error)

// 删除缓存项
func (c *Cache) Delete(key string) error

// 清空所有缓存项
func (c *Cache) Clear() error

// 获取缓存统计信息
func (c *Cache) Stats() Stats
```

### 统计信息结构

```go
type Stats struct {
    Hits           int64  // 缓存命中次数
    Misses         int64  // 缓存未命中次数
    Evictions      int64  // 淘汰次数
    CurrentSize    int64  // 当前内存使用量（字节）
    CurrentCount   int64  // 当前项目数量
    EvictionPolicy string // 淘汰策略
    MaxSize        int64  // 最大内存限制
    MaxCount       int64  // 总是为0（项目数量限制已移除）
}
```

## 淘汰策略

### LRU（最近最少使用）

优先淘汰最近最少访问的项目。适合具有时间局部性的应用程序。

```go
cache := tscache.NewCache(1024*1024, 100, "LRU")
```

### LFU（最少使用频率）

优先淘汰使用频率最低的项目。适合某些数据访问频率明显更高的应用程序。

```go
cache := tscache.NewCache(1024*1024, 100, "LFU")
```

### FIFO（先进先出）

优先淘汰最早的项目，不考虑访问模式。最简单和最可预测的策略。

```go
cache := tscache.NewCache(1024*1024, 100, "FIFO")
```

## 数据类型支持

TSCache 通过自动序列化支持所有 Go 数据类型：

```go
// 基本类型
cache.Set("string", "hello", 0)
cache.Set("int", 42, 0)
cache.Set("float", 3.14, 0)
cache.Set("bool", true, 0)

// 复杂类型
cache.Set("slice", []int{1, 2, 3}, 0)
cache.Set("map", map[string]int{"a": 1}, 0)

// 结构体
type User struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}
cache.Set("user", User{Name: "Alice", Age: 30}, 0)
```

## 性能特性

### 自动分片

TSCache 根据 CPU 核心数自动将数据分片到多个内部缓存中，以减少锁竞争：

- 分片数量：2 × CPU 核心数（最少 4 个，最多 64 个）
- 每个分片都有自己的锁和淘汰策略
- 使用 FNV-1a 哈希算法分布键

### 数据压缩

大数据（>1KB）会自动使用 gzip 压缩：

```go
// 大数据自动压缩
largeData := strings.Repeat("Hello World! ", 1000)
cache.Set("large", largeData, 0)

// 检索时透明解压
value, _ := cache.Get("large")
fmt.Println(value.(string)) // 原始数据
```

### 内存计算

准确计算所有 Go 数据类型的内存使用量，包括：

- 基本类型（int、string、bool 等）
- 复杂类型（切片、映射、结构体）
- 指针和接口
- 嵌套结构

## 基准测试

```bash
go test -bench=.
```

现代硬件上的典型性能：

- Set 操作：约 200 万次/秒
- Get 操作：约 500 万次/秒
- 混合操作：约 300 万次/秒

## 线程安全

TSCache 完全线程安全，针对并发访问进行了优化：

```go
cache := tscache.NewCache(1024*1024, 1000, "LRU")

// 可以安全地从多个goroutine使用
go func() {
    for i := 0; i < 1000; i++ {
        cache.Set(fmt.Sprintf("key%d", i), i, 0)
    }
}()

go func() {
    for i := 0; i < 1000; i++ {
        cache.Get(fmt.Sprintf("key%d", i))
    }
}()
```

## 示例

📚 **[查看完整示例 →](examples/README_CN.md)** | **[View English Examples →](examples/README.md)**

[examples](examples/) 目录包含了全面的可运行示例，演示 TSCache 的所有功能：

### 快速开始示例

```bash
# 运行基础使用示例（Set/Get、JSON、TTL、删除、统计、清空）
go run examples/*.go basic

# 运行淘汰策略对比（LRU、LFU、FIFO）
go run examples/*.go eviction

# 运行压缩功能示例（Gzip、Zstd、性能对比）
go run examples/*.go compression

# 运行所有示例
go run examples/*.go all
```

### 示例分类

- **[基础操作](examples/basic_usage.go)** - 核心功能包括[]byte 数据操作、JSON 序列化、TTL 过期、删除操作、统计监控和缓存清空
- **[淘汰策略](examples/eviction_policies.go)** - LRU、LFU、FIFO 策略的详细对比和性能基准测试
- **[压缩功能](examples/compression_demo.go)** - Gzip 和 Zstd 压缩使用、性能对比以及压缩+TTL 组合
- **[完整文档](examples/README_CN.md)** - 详细的使用说明和示例描述

每个示例都包含详细的输出、性能指标和实用的使用模式。

## 测试

```bash
# 运行测试
go test

# 运行覆盖率测试
go test -cover

# 运行基准测试
go test -bench=.
```

## 贡献

1. Fork 这个仓库
2. 创建您的功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交您的更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启一个 Pull Request

## 许可证

本项目基于 MIT 许可证 - 查看 LICENSE 文件了解详情。

## 支持

如果您有任何问题或需要帮助，请：

1. 查看 [示例](example/main.go)
2. 阅读文档
3. 在 GitHub 上开启 issue

## 架构设计

### 分片架构

```
Cache
├── Shard 0 (keys: hash % shardCount == 0)
│   ├── LRU/LFU/FIFO List
│   └── HashMap
├── Shard 1 (keys: hash % shardCount == 1)
│   ├── LRU/LFU/FIFO List
│   └── HashMap
└── ...
```

### 内存管理

- 每个缓存项都会计算准确的内存使用量
- 超出内存限制时自动触发淘汰
- 支持设置最大项目数量限制
- 压缩大数据以节省内存

### 并发控制

- 每个分片使用独立的读写锁
- 统计信息使用独立的锁
- 最小化锁竞争，提高并发性能

---

**TSCache** - 让高性能缓存变得简单！🚀
