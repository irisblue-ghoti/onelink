package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	JWT       JWTConfig       `yaml:"jwt"`
	Redis     RedisConfig     `yaml:"redis"`
	Kafka     KafkaConfig     `yaml:"kafka"`
	Adapters  AdaptersConfig  `yaml:"adapters"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
	Nacos     NacosConfig     `yaml:"nacos"`
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
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

// KafkaConfig Kafka配置
type KafkaConfig struct {
	Brokers        []string `yaml:"brokers"`
	ConsumerGroup  string   `yaml:"consumerGroup"`
	ConsumerTopics []string `yaml:"consumerTopics"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret      string `yaml:"secret"`
	ExpiryHours int    `yaml:"expiryHours"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
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

// AdaptersConfig 各平台适配器配置
type AdaptersConfig struct {
	Douyin      DouyinConfig      `yaml:"douyin"`
	Kuaishou    KuaishouConfig    `yaml:"kuaishou"`
	Xiaohongshu XiaohongshuConfig `yaml:"xiaohongshu"`
	Wechat      WechatConfig      `yaml:"wechat"`
}

// DouyinConfig 抖音平台配置
type DouyinConfig struct {
	AppID     string `yaml:"app_id"`
	AppSecret string `yaml:"app_secret"`
	BaseURL   string `yaml:"base_url"`
}

// KuaishouConfig 快手平台配置
type KuaishouConfig struct {
	AppID     string `yaml:"app_id"`
	AppSecret string `yaml:"app_secret"`
	BaseURL   string `yaml:"base_url"`
}

// XiaohongshuConfig 小红书平台配置
type XiaohongshuConfig struct {
	AppID     string `yaml:"app_id"`
	AppSecret string `yaml:"app_secret"`
	BaseURL   string `yaml:"base_url"`
}

// WechatConfig 微信平台配置
type WechatConfig struct {
	AppID     string `yaml:"app_id"`
	AppSecret string `yaml:"app_secret"`
	BaseURL   string `yaml:"base_url"`
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	RefreshPeriod int  `yaml:"refreshPeriod"` // 刷新周期，单位分钟
	BatchSize     int  `yaml:"batchSize"`     // 批处理大小
	Enabled       bool `yaml:"enabled"`       // 是否启用调度器
}

// DatabaseDSN 获取数据库连接字符串
func (c *DatabaseConfig) DatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
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
		config.Server.Port = "8083"
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
		config.Kafka.ConsumerGroup = "stats-service"
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

// LoadConfig 从文件加载配置
func LoadConfig(path string) (*Config, error) {
	config := &Config{}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	// 从环境变量覆盖配置
	if port := os.Getenv("PORT"); port != "" {
		config.Server.Port = port
	}

	return config, nil
}
