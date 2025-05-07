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

	"distribution-service/internal/api"
	"distribution-service/internal/config"
	"distribution-service/internal/domain/repositories"
	"distribution-service/internal/messaging"
	"distribution-service/internal/storage"
)

func main() {
	fmt.Println("分发服务启动中...")

	// 初始化日志
	logger := log.New(os.Stdout, "[DISTRIBUTION] ", log.LstdFlags)

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

	// 创建所需的存储库
	jobRepo := repositories.NewJobRepository(cfg.Database)
	videoRepo := repositories.NewVideoRepository(cfg.Database)

	// 创建Kafka客户端
	// 注意：由于没有合适的MessageHandler实现，这里暂时传nil
	// 实际需要创建一个实现了HandleMessage的消息处理器
	kafkaClient, err := messaging.NewKafkaClient(&cfg.Kafka, nil)
	if err != nil {
		logger.Printf("连接Kafka失败: %v, 将以无消息队列模式运行", err)
	}

	// 创建存储服务
	s3Config := storage.S3Config{
		AccessKey:     cfg.Storage.S3AccessKey,
		SecretKey:     cfg.Storage.S3SecretKey,
		Region:        cfg.Storage.S3Region,
		Bucket:        cfg.Storage.S3Bucket,
		TempDirectory: cfg.Platforms.TempDir,
	}
	storageService, err := storage.NewS3StorageService(s3Config)
	if err != nil {
		logger.Fatalf("创建存储服务失败: %v", err)
	}

	// 初始化API路由
	router := api.NewRouter(
		cfg,
		jobRepo,
		videoRepo,
		kafkaClient,
		storageService,
	)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    ":" + serverPort,
		Handler: router,
	}

	// 初始化并注册Nacos服务
	var nacosClient *nacos.Client
	if cfg.Nacos.Enable {
		// 打印出nacos配置信息（调试用）
		logger.Printf("Nacos配置: ServerAddr=%s, NamespaceID=%s, Group=%s, ServiceName=%s",
			cfg.Nacos.ServerAddr, cfg.Nacos.NamespaceID, cfg.Nacos.Group, cfg.Nacos.ServiceName)

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
			metadata := map[string]string{
				"version": "1.0.0",
				"env":     "dev",
			}
			if cfg.Nacos.Metadata != nil {
				metadata = cfg.Nacos.Metadata
			}
			success, err := nacosClient.RegisterService(
				cfg.Nacos.ServiceName,
				"", // 空字符串表示自动获取本机IP
				port,
				metadata,
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
		logger.Printf("分发服务已启动，端口: %s", serverPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("监听错误: %v", err)
		}
	}()

	// 等待中断信号优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("正在关闭分发服务...")

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

	logger.Println("分发服务已关闭")
}
