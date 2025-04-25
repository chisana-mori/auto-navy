package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	fmt.Println("Kubernetes 集群资源报告生成器")
	fmt.Println("=============================")

	// 定义命令行参数
	environmentPtr := flag.String("env", "prd", "环境类型: 'prd' 或 'test'。生产环境使用标准阈值，测试环境忽略低利用率并将高利用率阈值提高5%")
	helpPtr := flag.Bool("help", false, "显示帮助信息")

	// 解析命令行参数
	flag.Parse()

	// 检查帮助参数
	if *helpPtr {
		fmt.Println("用法: go run *.go [--env=prd|test]")
		fmt.Println("将根据template.html模板生成资源报告预览")
		fmt.Println("参数:")
		flag.PrintDefaults()
		return
	}

	// 验证环境参数
	if *environmentPtr != "prd" && *environmentPtr != "test" {
		fmt.Printf("警告: 无效的环境类型 '%s'，使用默认值 'prd'\n", *environmentPtr)
		*environmentPtr = "prd"
	}

	// 运行预览生成器
	fmt.Println("正在生成资源报告预览...")
	fmt.Printf("环境类型: %s\n", *environmentPtr)
	fmt.Println("使用模板: ../template.html")

	// 调用预览生成器
	err := generatePreview(*environmentPtr)
	if err != nil {
		log.Fatalf("生成预览失败: %v", err)
	} else {
		fmt.Println("预览生成成功！")
	}
}
