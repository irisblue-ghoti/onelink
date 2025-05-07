package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/nfc_card/shared/nacos"

	"content-service/internal/api"
	"content-service/internal/config"
	"content-service/internal/messaging"
	"content-service/internal/services"
	"content-service/internal/storage"

	"github.com/spf13/viper"
)

func main() {
	fmt.Println("内容服务启动中...")

	// 获取配置文件路径
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "/app/config/content-service.yaml"
	}

	fmt.Printf("使用配置文件: %s\n", configPath)

	// 检查文件是否存在
	_, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		fmt.Printf("错误: 配置文件不存在: %s\n", configPath)
		os.Exit(1)
	}

	// 打印目录内容以便调试
	dir := filepath.Dir(configPath)
	files, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("无法读取目录 %s: %v\n", dir, err)
	} else {
		fmt.Printf("目录 %s 内容:\n", dir)
		for _, file := range files {
			fmt.Printf("  - %s\n", file.Name())
		}
	}

	// 初始化Viper
	v := viper.New()
	v.SetConfigFile(configPath)

	// 读取配置
	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("配置文件加载成功，服务准备就绪")

	// 获取配置信息（但不打印敏感信息）
	dbHost := v.GetString("database.postgres.host")
	dbPort := v.GetInt("database.postgres.port")
	dbUser := v.GetString("database.postgres.user")
	dbPass := v.GetString("database.postgres.password")
	dbName := v.GetString("database.postgres.dbname")
	dbSSL := v.GetString("database.postgres.sslmode")

	// 初始化日志
	logger := log.New(os.Stdout, "[CONTENT] ", log.LstdFlags)

	// 获取服务端口
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = v.GetString("server.port")
	}

	// 初始化数据库
	db, err := storage.NewDBConnection(config.DatabaseConfig{
		Host:     dbHost,
		Port:     fmt.Sprintf("%d", dbPort),
		User:     dbUser,
		Password: dbPass,
		DBName:   dbName,
		SSLMode:  dbSSL,
	})
	if err != nil {
		logger.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// 初始化存储层
	repos := storage.NewRepositories(db)

	// 初始化消息服务
	kafkaClient, err := messaging.NewKafkaClient(config.KafkaConfig{
		Brokers: []string{v.GetString("kafka.addr")},
		Topic:   v.GetString("kafka.topic"),
	})
	if err != nil {
		logger.Printf("连接Kafka失败: %v, 将以无消息队列模式运行", err)
	} else {
		defer kafkaClient.Close()
	}

	// 创建完整的配置对象
	appConfig := &config.Config{
		Server: config.ServerConfig{
			Port: serverPort,
		},
		Database: config.DatabaseConfig{
			Host:     dbHost,
			Port:     fmt.Sprintf("%d", dbPort),
			User:     dbUser,
			Password: dbPass,
			DBName:   dbName,
			SSLMode:  dbSSL,
		},
		Kafka: config.KafkaConfig{
			Brokers: []string{v.GetString("kafka.addr")},
			Topic:   v.GetString("kafka.topic"),
		},
	}

	// 初始化服务层
	contentService := services.NewContentService(repos, kafkaClient, logger, appConfig)

	// 初始化API路由
	router := api.NewRouter(appConfig, contentService)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    ":" + serverPort,
		Handler: router,
	}

	// 初始化并注册Nacos服务
	var nacosClient *nacos.Client
	if v.GetBool("nacos.enable") {
		nacosConfig := &nacos.Config{
			ServerAddr:  v.GetString("nacos.server_addr"),
			NamespaceID: v.GetString("nacos.namespace_id"),
			Group:       v.GetString("nacos.group"),
			LogDir:      v.GetString("nacos.log_dir"),
			CacheDir:    v.GetString("nacos.cache_dir"),
		}

		nacosClient, err = nacos.NewClient(nacosConfig)
		if err != nil {
			logger.Printf("初始化Nacos客户端失败: %v", err)
		} else {
			// 获取本机IP并注册服务
			port, _ := strconv.Atoi(serverPort)
			metadata := make(map[string]string)
			if v.IsSet("nacos.metadata") {
				for k, val := range v.GetStringMapString("nacos.metadata") {
					metadata[k] = val
				}
			}
			success, err := nacosClient.RegisterService(
				v.GetString("nacos.service_name"),
				"", // 空字符串表示自动获取本机IP
				port,
				metadata,
			)
			if err != nil {
				logger.Printf("注册服务到Nacos失败: %v", err)
			} else if success {
				logger.Printf("已成功注册到Nacos，服务名: %s, 端口: %d", v.GetString("nacos.service_name"), port)

				// 启动健康检查
				nacosClient.StartHealthCheck(v.GetString("nacos.service_name"), "", port, 5*time.Second)
			}
		}
	}

	// 在goroutine中启动服务器，以便不阻塞信号处理
	go func() {
		logger.Printf("内容服务已启动，端口: %s", serverPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("监听错误: %v", err)
		}
	}()

	// 等待中断信号优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("正在关闭内容服务...")

	// 从Nacos注销服务
	if v.GetBool("nacos.enable") && nacosClient != nil {
		port, _ := strconv.Atoi(serverPort)
		_, err := nacosClient.DeregisterService(v.GetString("nacos.service_name"), "", port)
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

	logger.Println("内容服务已关闭")
}
