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

	"nfc-service/internal/api"
	"nfc-service/internal/config"
	"nfc-service/internal/services/cards"
	"nfc-service/internal/services/shortlinks"
	"nfc-service/internal/storage"
	"nfc-service/pkg/cloudflare"
)

func main() {
	fmt.Println("NFC服务启动中...")

	// 初始化日志
	logger := log.New(os.Stdout, "[NFC] ", log.LstdFlags)

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

	// 初始化Cloudflare客户端
	cfClient := cloudflare.NewClient(cfg.Cloudflare.APIToken, "", cfg.Cloudflare.ZoneID)

	// 初始化存储层
	repos := storage.NewRepositories(db)

	// 使用适配器创建领域仓库
	domainCardRepo := cards.NewCardRepositoryAdapter(repos.CardRepository)

	// 初始化服务层
	cardService := cards.NewCardService(domainCardRepo, logger)
	shortlinkService := shortlinks.NewShortlinkService(repos.ShortlinkRepository, cfClient, logger)

	// 初始化API路由
	router := api.NewRouter(cfg, cardService, shortlinkService)

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
		logger.Printf("NFC服务已启动，端口: %s", serverPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("监听错误: %v", err)
		}
	}()

	// 等待中断信号优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("正在关闭NFC服务...")

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

	logger.Println("NFC服务已关闭")
}
