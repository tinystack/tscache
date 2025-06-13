package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	fmt.Println("TSCache 示例程序")
	fmt.Println("================")

	if len(os.Args) < 2 {
		showUsage()
		return
	}

	switch os.Args[1] {
	case "basic":
		fmt.Println("运行基础使用示例...")
		DemonstrateBasicUsage()
	case "eviction":
		fmt.Println("运行淘汰策略示例...")
		DemonstrateEvictionPolicies()
	case "compression":
		fmt.Println("运行压缩功能示例...")
		DemonstrateCompression()
	case "all":
		fmt.Println("运行所有示例...")
		fmt.Println("\n" + strings.Repeat("=", 50))
		DemonstrateBasicUsage()
		fmt.Println("\n" + strings.Repeat("=", 50))
		DemonstrateEvictionPolicies()
		fmt.Println("\n" + strings.Repeat("=", 50))
		DemonstrateCompression()
	default:
		fmt.Printf("未知的示例类型: %s\n", os.Args[1])
		showUsage()
	}
}

func showUsage() {
	fmt.Println("使用方法:")
	fmt.Println("  go run examples/*.go basic      - 运行基础使用示例")
	fmt.Println("  go run examples/*.go eviction   - 运行淘汰策略示例")
	fmt.Println("  go run examples/*.go compression - 运行压缩功能示例")
	fmt.Println("  go run examples/*.go all        - 运行所有示例")
}
