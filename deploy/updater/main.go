// devops-admin 在线升级执行器（updater sidecar，与 SoyDisk 同构组件）
//
// 同一二进制两种运行模式：
//
//	daemon —— 常驻服务（compose 服务 updater）：HTTP API 供 server 调用，
//	          负责升级包下载（断点续传）/ sha256 校验 / 解压，安装阶段经
//	          docker run 发起独立 job 容器（本程序 install 模式）执行
//	install —— 一次性安装 job：load 镜像 → 替换编排资产 → 更新 .env 版本 →
//	          docker compose up -d --force-recreate → 健康检查 → 状态落盘
//
// 安装阶段为什么 job 化：up -d 会重建 updater 自身容器，若在 updater 进程内直接
// 执行 compose，进程被杀会中断后续容器的重建编排；job 容器由 docker daemon 直接
// 管理、与 updater 容器生命周期解耦，能完整跑完安装并写终态，新 updater 起来后
// 从状态文件恢复展示。
//
// 零第三方依赖（纯标准库）；docker / compose 操作走 exec CLI，不引 docker SDK。
package main

import (
	"fmt"
	"os"
)

// 版本注入：发布构建经 ldflags 注入（与 server 镜像同机制，共用 APP_VERSION）
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	args := os.Args[1:]
	mode := "daemon"
	if len(args) > 0 {
		mode = args[0]
		args = args[1:]
	}
	switch mode {
	case "daemon":
		runDaemon()
	case "install":
		// install 模式由 daemon 经 docker run 发起；版本号定位已解压的升级包目录
		if err := runInstall(args); err != nil {
			fmt.Fprintf(os.Stderr, "[install] 失败: %v\n", err)
			os.Exit(1)
		}
	case "version":
		fmt.Printf("devops-admin-updater %s (build %s)\n", Version, BuildTime)
	default:
		fmt.Fprintf(os.Stderr, "未知模式: %s（可用：daemon / install / version）\n", mode)
		os.Exit(2)
	}
}
