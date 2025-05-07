package config

import (
	"os"
	"strconv"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
	Nacos    NacosConfig    `mapstructure:"nacos"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `mapstructure:"port"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpiryHours int    `mapstructure:"expiry_hours"`
}

// KafkaConfig Kafka配置
type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`
	Topic   string   `mapstructure:"topic"`
}

// NacosConfig Nacos配置
type NacosConfig struct {
	ServerAddr  string            `mapstructure:"server_addr"`  // Nacos服务地址，如localhost:8848
	NamespaceID string            `mapstructure:"namespace_id"` // 命名空间ID，默认为public
	Group       string            `mapstructure:"group"`        // 分组，默认为DEFAULT_GROUP
	ServiceName string            `mapstructure:"service_name"` // 服务名称
	Metadata    map[string]string `mapstructure:"metadata"`     // 服务元数据
	Weight      float64           `mapstructure:"weight"`       // 服务权重
	Enable      bool              `mapstructure:"enable"`       // 是否启用
	LogDir      string            `mapstructure:"log_dir"`      // 日志目录
	CacheDir    string            `mapstructure:"cache_dir"`    // 缓存目录
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// Load 从配置文件和环境变量加载配置
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// 从环境变量加载配置
	viper.AutomaticEnv()

	var config Config

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	// 绑定配置到结构体
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// 从环境变量覆盖配置
	if port := os.Getenv("PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err == nil {
			config.Server.Port = strconv.Itoa(p)
		}
	}

	// 设置默认值
	if config.Server.Port == "" {
		config.Server.Port = "3002"
	}

	return &config, nil
}
