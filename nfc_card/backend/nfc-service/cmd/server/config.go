package main

import (
	"log"
	"os"

	"nfc-service/internal/config"

	"github.com/nfc_card/shared/nacos"
)

// 加载配置
func loadConfig() (*config.Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/nfc-service.yaml"
	}

	logger := log.New(os.Stdout, "[配置加载] ", log.LstdFlags)
	logger.Printf("加载配置文件: %s", configPath)

	// 1. 尝试使用Nacos配置中心加载
	serviceConfig, err := nacos.NewServiceConfig("nfc-service", configPath, logger)
	if err != nil {
		logger.Printf("创建配置管理器失败: %v，将直接使用本地配置", err)
		return config.LoadConfig(configPath)
	}

	// 2. 创建配置实例
	cfg := &config.Config{}

	// 3. 加载配置
	if err := serviceConfig.LoadConfig(cfg); err != nil {
		logger.Printf("加载配置失败: %v，将直接使用本地配置", err)
		return config.LoadConfig(configPath)
	}

	// 4. 设置配置变更监听（配置热更新）
	if err := serviceConfig.WatchConfig(cfg, func() {
		logger.Printf("配置已更新")
		// 这里可以添加配置变更后的回调处理
	}); err != nil {
		logger.Printf("设置配置变更监听失败: %v", err)
	}

	// 5. 如果是首次运行，将配置迁移到Nacos
	// 注意：这个逻辑应该根据实际需求决定是否执行
	if os.Getenv("MIGRATE_CONFIG") == "true" {
		logger.Printf("开始迁移配置到Nacos...")
		if err := serviceConfig.MigrateToNacos(); err != nil {
			logger.Printf("迁移配置失败: %v", err)
		} else {
			logger.Printf("配置迁移成功")
		}
	}

	logger.Printf("配置加载完成")
	return cfg, nil
}
