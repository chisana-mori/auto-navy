package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	fmt.Println("Kubernetes 集群资源报告生成器")
	fmt.Println("=============================")

	// 检查命令行参数
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("用法: go run *.go")
		fmt.Println("将根据template.html模板生成资源报告预览")
		return
	}

	// 运行预览生成器
	fmt.Println("正在生成资源报告预览...")
	fmt.Println("使用模板: ../template.html")

	// 调用预览生成器
	err := generatePreview()
	if err != nil {
		log.Fatalf("生成预览失败: %v", err)
	} else {
		fmt.Println("预览生成成功！")
	}
}
