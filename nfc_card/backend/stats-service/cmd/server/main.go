package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/nfc_card/shared/nacos"

	"stats-service/internal/adapters/douyin"
	"stats-service/internal/adapters/kuaishou"
	"stats-service/internal/adapters/xiaohongshu"
	"stats-service/internal/api"
	"stats-service/internal/config"
	"stats-service/internal/domain/repositories"
	"stats-service/internal/services"
	"stats-service/internal/storage"
)

func main() {
	fmt.Println("统计服务启动中...")

	// 初始化日志
	logger := log.New(os.Stdout, "[STATS] ", log.LstdFlags)

	// 获取配置文件路径
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config.yaml"
	}

	// 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Fatalf("加载配置失败: %v", err)
	}

	// 获取服务端口
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = cfg.Server.Port
	}

	// 初始化数据库
	db, err := storage.NewDBConnection(cfg.Database)
	if err != nil {
		logger.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// 初始化存储层
	repos := storage.NewRepositories(db)

	// 创建领域层仓库 (当调度器启用时取消注释)
	domainRepo, err := repositories.NewStatsRepository(cfg.Database)
	if err != nil {
		logger.Fatalf("创建统计数据仓库失败: %v", err)
	}

	// 创建仓库适配器
	repoAdapter := storage.NewRepoAdapter(repos, domainRepo)

	// 初始化服务层
	statsService := services.NewStatsService(repoAdapter, logger)

	// 注册平台适配器
	if cfg.Adapters.Douyin.AppID != "" && cfg.Adapters.Douyin.AppSecret != "" {
		douyinAdapter := douyin.NewDouyinAdapter(config.DouyinConfig{
			AppID:     cfg.Adapters.Douyin.AppID,
			AppSecret: cfg.Adapters.Douyin.AppSecret,
			BaseURL:   cfg.Adapters.Douyin.BaseURL,
		})
		statsService.RegisterAdapter(douyinAdapter)
		logger.Printf("已注册抖音平台适配器")
	}

	if cfg.Adapters.Kuaishou.AppID != "" && cfg.Adapters.Kuaishou.AppSecret != "" {
		kuaishouAdapter := kuaishou.NewKuaishouAdapter(config.KuaishouConfig{
			AppID:     cfg.Adapters.Kuaishou.AppID,
			AppSecret: cfg.Adapters.Kuaishou.AppSecret,
			BaseURL:   cfg.Adapters.Kuaishou.BaseURL,
		})
		statsService.RegisterAdapter(kuaishouAdapter)
		logger.Printf("已注册快手平台适配器")
	}

	if cfg.Adapters.Xiaohongshu.AppID != "" && cfg.Adapters.Xiaohongshu.AppSecret != "" {
		xiaohongshuAdapter := xiaohongshu.NewXiaohongshuAdapter(
			cfg.Adapters.Xiaohongshu.AppID,
			cfg.Adapters.Xiaohongshu.AppSecret,
		)
		statsService.RegisterAdapter(xiaohongshuAdapter)
		logger.Printf("已注册小红书平台适配器")
	}

	// 注意：调度器功能暂时不启用，需要进一步处理类型兼容问题
	logger.Printf("统计数据调度功能暂未启用，稍后将通过配置文件支持")

	// 如果需要启用调度器，可以取消下面的注释
	// statsScheduler := services.NewStatsScheduler(statsService, domainRepo, logger)
	// statsScheduler.Start()
	// defer statsScheduler.Stop()

	// 初始化API路由
	router := api.NewRouter(cfg, statsService)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    ":" + serverPort,
		Handler: router,
	}

	// 初始化并注册Nacos服务
	var nacosClient *nacos.Client
	if cfg.Nacos.Enable {
		nacosConfig := &nacos.Config{
			ServerAddr:  cfg.Nacos.ServerAddr,
			NamespaceID: cfg.Nacos.NamespaceID,
			Group:       cfg.Nacos.Group,
			LogDir:      cfg.Nacos.LogDir,
			CacheDir:    cfg.Nacos.CacheDir,
		}

		nacosClient, err = nacos.NewClient(nacosConfig)
		if err != nil {
			logger.Printf("初始化Nacos客户端失败: %v", err)
		} else {
			// 获取本机IP并注册服务
			port, _ := strconv.Atoi(serverPort)
			success, err := nacosClient.RegisterService(
				cfg.Nacos.ServiceName,
				"", // 空字符串表示自动获取本机IP
				port,
				cfg.Nacos.Metadata,
			)
			if err != nil {
				logger.Printf("注册服务到Nacos失败: %v", err)
			} else if success {
				logger.Printf("已成功注册到Nacos，服务名: %s, 端口: %d", cfg.Nacos.ServiceName, port)

				// 启动健康检查
				nacosClient.StartHealthCheck(cfg.Nacos.ServiceName, "", port, 5*time.Second)
			}
		}
	}

	// 在goroutine中启动服务器，以便不阻塞信号处理
	go func() {
		logger.Printf("统计服务已启动，端口: %s", serverPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("监听错误: %v", err)
		}
	}()

	// 等待中断信号优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("正在关闭统计服务...")

	// 从Nacos注销服务
	if cfg.Nacos.Enable && nacosClient != nil {
		port, _ := strconv.Atoi(serverPort)
		_, err := nacosClient.DeregisterService(cfg.Nacos.ServiceName, "", port)
		if err != nil {
			logger.Printf("从Nacos注销服务失败: %v", err)
		} else {
			logger.Println("已从Nacos注销服务")
		}
	}

	// 创建一个5秒的超时上下文，用于优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 优雅关闭服务器
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("服务器关闭错误: %v", err)
	}

	logger.Println("统计服务已关闭")
}
