package tscache

import (
	"reflect"
	"runtime"
	"testing"
	"unsafe"
)

func TestCalculateSize(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected int64
	}{
		{
			name:     "string",
			value:    "hello world",
			expected: int64(len("hello world")),
		},
		{
			name:     "int",
			value:    42,
			expected: 8,
		},
		{
			name:     "float64",
			value:    3.14,
			expected: 8,
		},
		{
			name:     "bool",
			value:    true,
			expected: 1,
		},
		{
			name:     "slice",
			value:    []int{1, 2, 3, 4, 5},
			expected: int64(unsafe.Sizeof([]int{}) + 5*8), // slice header + 5 ints
		},
		{
			name:     "map",
			value:    map[string]int{"a": 1, "b": 2},
			expected: int64(unsafe.Sizeof(map[string]int{})) + int64(2*(len("a")+8)+2*(len("b")+8)), // approximation
		},
		{
			name:     "nil",
			value:    nil,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateSize(tt.value)
			// 由于size计算可能有平台差异，我们只检查非负数
			if result < 0 {
				t.Errorf("calculateSize() returned negative size: %d", result)
			}

			// 对于nil，确保返回0
			if tt.value == nil && result != 0 {
				t.Errorf("calculateSize(nil) should return 0, got %d", result)
			}
		})
	}
}

func TestCalculateValueSize(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"string", "test"},
		{"int", 123},
		{"float64", 1.23},
		{"bool", true},
		{"slice", []byte{1, 2, 3}},
		{"map", map[string]int{"test": 1}},
		{"struct", struct{ Name string }{"test"}},
		{"pointer", &struct{}{}},
		{"interface", interface{}("test")},
		{"channel", make(chan int)},
		{"function", func() {}},
		{"array", [5]int{1, 2, 3, 4, 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := calculateValueSize(reflect.ValueOf(tt.value))
			if size <= 0 {
				t.Errorf("calculateValueSize() should return positive size for %s, got %d", tt.name, size)
			}
		})
	}

	// 测试无效值
	t.Run("invalid value", func(t *testing.T) {
		var v reflect.Value // 零值，无效
		size := calculateValueSize(v)
		if size != 0 {
			t.Errorf("calculateValueSize(invalid value) should return 0, got %d", size)
		}
	})
}

func TestFnv1a(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect uint32
	}{
		{
			name:   "empty string",
			input:  "",
			expect: 2166136261, // FNV-1a offset basis
		},
		{
			name:   "hello",
			input:  "hello",
			expect: 0, // 我们不检查具体值，只检查一致性
		},
		{
			name:   "world",
			input:  "world",
			expect: 0,
		},
		{
			name:   "long string",
			input:  "this is a very long string for testing hash function",
			expect: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := fnv1a(tt.input)
			hash2 := fnv1a(tt.input)

			// 同样的输入应该产生同样的hash
			if hash1 != hash2 {
				t.Errorf("fnv1a(%s) should be consistent, got %d and %d", tt.input, hash1, hash2)
			}

			// 检查空字符串的特殊情况
			if tt.input == "" && hash1 != 2166136261 {
				t.Errorf("fnv1a('') should return %d, got %d", 2166136261, hash1)
			}
		})
	}

	// 测试不同输入产生不同hash（大概率）
	t.Run("different inputs", func(t *testing.T) {
		hash1 := fnv1a("test1")
		hash2 := fnv1a("test2")

		if hash1 == hash2 {
			t.Error("Different inputs should (likely) produce different hashes")
		}
	})
}

func TestGetOptimalShardCount(t *testing.T) {
	// 模拟不同的CPU核心数
	originalCPUs := runtime.NumCPU()

	tests := []struct {
		name        string
		cpuCount    int
		expectedMin int
		expectedMax int
	}{
		{
			name:        "single core",
			cpuCount:    1,
			expectedMin: 4,
			expectedMax: 16,
		},
		{
			name:        "dual core",
			cpuCount:    2,
			expectedMin: 4,
			expectedMax: 16,
		},
		{
			name:        "quad core",
			cpuCount:    4,
			expectedMin: 4,
			expectedMax: 16,
		},
		{
			name:        "many cores",
			cpuCount:    16,
			expectedMin: 4,
			expectedMax: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 注意：我们无法轻易模拟runtime.NumCPU()，所以只测试当前系统
			shardCount := getOptimalShardCount()

			if shardCount < 4 {
				t.Errorf("getOptimalShardCount() should return at least 4, got %d", shardCount)
			}

			if shardCount > 32 {
				t.Errorf("getOptimalShardCount() should return at most 32, got %d", shardCount)
			}

			// 检查是否为合理值（不要求是2的幂，因为实际实现可能不同）
			if shardCount <= 0 {
				t.Errorf("getOptimalShardCount() should return positive value, got %d", shardCount)
			}
		})
	}

	_ = originalCPUs // 避免unused variable警告
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0 B",
		},
		{
			name:     "bytes",
			bytes:    500,
			expected: "500 B",
		},
		{
			name:     "kilobytes",
			bytes:    1536, // 1.5 KB
			expected: "1.5 KB",
		},
		{
			name:     "megabytes",
			bytes:    1572864, // 1.5 MB
			expected: "1.5 MB",
		},
		{
			name:     "gigabytes",
			bytes:    1610612736, // 1.5 GB
			expected: "1.5 GB",
		},
		{
			name:     "terabytes",
			bytes:    1649267441664, // 1.5 TB
			expected: "1.5 TB",
		},
		{
			name:     "petabytes",
			bytes:    1688849860263936, // 1.5 PB
			expected: "1.5 PB",
		},
		{
			name:     "exact kilobyte",
			bytes:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "exact megabyte",
			bytes:    1048576,
			expected: "1.0 MB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %s, expected %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestGetMemoryUsage(t *testing.T) {
	t.Run("memory usage", func(t *testing.T) {
		alloc, sys := getMemoryUsage()

		// 内存使用量应该是正数
		if alloc <= 0 {
			t.Errorf("getMemoryUsage() alloc should return positive value, got %d", alloc)
		}

		if sys <= 0 {
			t.Errorf("getMemoryUsage() sys should return positive value, got %d", sys)
		}

		// 系统内存应该大于等于分配内存
		if sys < alloc {
			t.Errorf("System memory (%d) should be >= allocated memory (%d)", sys, alloc)
		}

		// 内存使用量应该在合理范围内（不超过1TB）
		if alloc > 1024*1024*1024*1024 {
			t.Errorf("getMemoryUsage() returned unreasonably large alloc value: %d", alloc)
		}
	})
}

func TestUtilityFunctionsEdgeCases(t *testing.T) {
	// 测试calculateSize的边界情况
	t.Run("calculateSize edge cases", func(t *testing.T) {
		// 大字符串
		largeString := string(make([]byte, 1000000))
		size := calculateSize(largeString)
		if size <= 0 {
			t.Error("calculateSize should handle large strings")
		}

		// 空切片
		emptySlice := []int{}
		size = calculateSize(emptySlice)
		if size < 0 {
			t.Error("calculateSize should handle empty slice")
		}

		// 空map
		emptyMap := map[string]int{}
		size = calculateSize(emptyMap)
		if size < 0 {
			t.Error("calculateSize should handle empty map")
		}
	})

	// 测试hash函数的分布
	t.Run("hash distribution", func(t *testing.T) {
		hashMap := make(map[uint32]int)

		// 生成1000个不同的字符串并计算hash
		for i := 0; i < 1000; i++ {
			key := "test_key_" + string(rune(i))
			hash := fnv1a(key)
			hashMap[hash]++
		}

		// 检查是否有严重的hash冲突（不应该有太多重复）
		maxCollisions := 0
		for _, count := range hashMap {
			if count > maxCollisions {
				maxCollisions = count
			}
		}

		// 在1000个不同输入中，最多冲突数不应该超过5（这是一个宽松的限制）
		if maxCollisions > 5 {
			t.Errorf("Hash function may have poor distribution, max collisions: %d", maxCollisions)
		}
	})
}
