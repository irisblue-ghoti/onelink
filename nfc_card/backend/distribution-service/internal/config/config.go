package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 应用程序配置
type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	Kafka          KafkaConfig
	Platforms      PlatformsConfig
	TaskProcessing TaskProcessingConfig
	JWT            JWTConfig
	Nacos          NacosConfig

	// 兼容旧代码
	Adapters PlatformsConfig
	Storage  StorageConfig
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port                string `yaml:"port"`
	ReadTimeoutSeconds  int    `yaml:"readTimeoutSeconds"`
	WriteTimeoutSeconds int    `yaml:"writeTimeoutSeconds"`
	IdleTimeoutSeconds  int    `yaml:"idleTimeoutSeconds"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

// KafkaConfig Kafka配置
type KafkaConfig struct {
	Brokers        []string `yaml:"brokers"`
	Topic          string   `yaml:"topic"`
	ConsumerGroup  string   `yaml:"consumerGroup"`
	ConsumerTopics []string `yaml:"consumerTopics"`
	ProducerTopics []string `yaml:"producerTopics"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret      string `yaml:"secret"`
	ExpiryHours int    `yaml:"expiryHours"`
}

// NacosConfig Nacos配置
type NacosConfig struct {
	ServerAddr  string            `yaml:"server_addr"`  // Nacos服务地址
	NamespaceID string            `yaml:"namespace_id"` // 命名空间ID
	Group       string            `yaml:"group"`        // 分组
	ServiceName string            `yaml:"service_name"` // 服务名称
	Enable      bool              `yaml:"enable"`       // 是否启用服务发现
	Weight      int               `yaml:"weight"`       // 服务权重
	Metadata    map[string]string `yaml:"metadata"`     // 服务元数据
	LogDir      string            `yaml:"log_dir"`      // 日志目录
	CacheDir    string            `yaml:"cache_dir"`    // 缓存目录
}

// PlatformsConfig 平台配置
type PlatformsConfig struct {
	Douyin      DouyinConfig
	Kuaishou    KuaishouConfig
	Wechat      WechatConfig
	Xiaohongshu XiaohongshuConfig
	TempDir     string
}

// DouyinConfig 抖音配置
type DouyinConfig struct {
	ClientKey    string `yaml:"clientKey"`
	ClientSecret string `yaml:"clientSecret"`
	RedirectURI  string `yaml:"redirectURI"`
	AppID        string `yaml:"appId"`
	AppSecret    string `yaml:"appSecret"`
	APIHost      string `yaml:"apiHost"`
}

// KuaishouConfig 快手配置
type KuaishouConfig struct {
	AppID       string `yaml:"appId"`
	AppSecret   string `yaml:"appSecret"`
	CallbackURL string `yaml:"callbackURL"`
	APIHost     string `yaml:"apiHost"`
}

// WechatConfig 微信配置
type WechatConfig struct {
	AppID       string `yaml:"appId"`
	AppSecret   string `yaml:"appSecret"`
	Token       string `yaml:"token"`
	CallbackURL string `yaml:"callbackURL"`
	APIHost     string `yaml:"apiHost"`
}

// XiaohongshuConfig 小红书配置
type XiaohongshuConfig struct {
	AppID       string `yaml:"appId"`
	AppSecret   string `yaml:"appSecret"`
	CallbackURL string `yaml:"callbackURL"`
	APIHost     string `yaml:"apiHost"`
}

// TaskProcessingConfig 任务处理配置
type TaskProcessingConfig struct {
	RetryIntervalMinutes int  `yaml:"retryIntervalMinutes"`
	MaxRetries           int  `yaml:"maxRetries"`
	EnableDeadLetter     bool `yaml:"enableDeadLetter"`
	TaskTimeoutSeconds   int  `yaml:"taskTimeoutSeconds"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type        string // s3, oss, local
	S3Bucket    string
	S3Region    string
	S3AccessKey string
	S3SecretKey string
}

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件错误: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件错误: %w", err)
	}

	return &config, nil
}

// Load 从文件加载配置
func Load() (*Config, error) {
	configPath := getConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	if config.Server.Port == "" {
		config.Server.Port = "8082"
	}
	if config.Server.ReadTimeoutSeconds == 0 {
		config.Server.ReadTimeoutSeconds = 10
	}
	if config.Server.WriteTimeoutSeconds == 0 {
		config.Server.WriteTimeoutSeconds = 10
	}
	if config.Server.IdleTimeoutSeconds == 0 {
		config.Server.IdleTimeoutSeconds = 60
	}

	// 设置Kafka默认值
	if len(config.Kafka.Brokers) == 0 {
		config.Kafka.Brokers = []string{"kafka:9092"}
	}
	if config.Kafka.ConsumerGroup == "" {
		config.Kafka.ConsumerGroup = "distribution-service"
	}

	// 设置任务处理默认值
	if config.TaskProcessing.MaxRetries == 0 {
		config.TaskProcessing.MaxRetries = 3
	}
	if config.TaskProcessing.RetryIntervalMinutes == 0 {
		config.TaskProcessing.RetryIntervalMinutes = 5
	}

	// 设置JWT默认值
	if config.JWT.Secret == "" {
		config.JWT.Secret = "default-jwt-secret-key"
	}
	if config.JWT.ExpiryHours == 0 {
		config.JWT.ExpiryHours = 24
	}

	return &config, nil
}

// getConfigPath 获取配置文件路径
func getConfigPath() string {
	// 优先使用环境变量
	configPath := os.Getenv("CONFIG_PATH")
	if configPath != "" {
		return configPath
	}

	// 默认配置文件路径
	return "config.yaml"
}
