package main

import (
	"myflowhub/manager/internal/api"
	"myflowhub/manager/internal/client"
	"myflowhub/pkg/config"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// 设置日志格式
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	log.Info().Msg("启动 Manager 服务...")

	// 加载配置
	config.LoadConfig()

	// 创建Hub客户端
	hubAddr := config.AppConfig.Hub.Address
	if hubAddr == "" {
		hubAddr = "ws://localhost:8080/ws" // 默认地址
	}
	managerToken := config.AppConfig.Server.ManagerToken
	hubClient := client.NewHubClient(hubAddr, managerToken)

	// 连接到Hub
	if err := hubClient.Connect(); err != nil {
		log.Fatal().Err(err).Msg("连接到Hub失败")
	}
	defer hubClient.Close()

	// 创建管理API
	managerAPI := api.NewManagerAPI(hubClient)

	// 设置HTTP路由
	mux := http.NewServeMux()
	managerAPI.RegisterRoutes(mux)

	// 健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok", "service": "manager"}`))
	})

	// 启动HTTP服务器
	server := &http.Server{
		Addr:    ":8090",
		Handler: mux,
	}

	go func() {
		log.Info().Str("addr", ":8090").Msg("Manager API 服务启动")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP服务器启动失败")
		}
	}()

	// 等待中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info().Msg("收到关闭信号，正在优雅关闭...")

	// 关闭HTTP服务器
	if err := server.Close(); err != nil {
		log.Error().Err(err).Msg("关闭HTTP服务器失败")
	}

	log.Info().Msg("Manager 服务已停止")
}
