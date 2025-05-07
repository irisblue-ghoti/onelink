package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config 应用程序配置
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Kafka    KafkaConfig
	Storage  StorageConfig
	JWT      JWTConfig
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// KafkaConfig Kafka配置
type KafkaConfig struct {
	Brokers []string
	Topic   string
}

// StorageConfig MinIO存储配置
type StorageConfig struct {
	Endpoint   string
	BucketName string
	AccessKey  string
	SecretKey  string
	UseSSL     bool
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret      string
	ExpiryHours int
}

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件错误: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件错误: %w", err)
	}

	// 设置JWT默认值
	if config.JWT.Secret == "" {
		config.JWT.Secret = "default-jwt-secret-key"
	}

	if config.JWT.ExpiryHours <= 0 {
		config.JWT.ExpiryHours = 24
	}

	return &config, nil
}
