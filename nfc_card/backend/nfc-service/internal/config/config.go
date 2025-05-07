package config

import (
	"encoding/json"
	"os"
	"strconv"

	"github.com/spf13/viper"
)

// Config 配置结构体
type Config struct {
	Server     ServerConfig     `json:"server" mapstructure:"server"`
	Database   DatabaseConfig   `json:"database" mapstructure:"database"`
	JWT        JWTConfig        `json:"jwt" mapstructure:"jwt"`
	Cloudflare CloudflareConfig `json:"cloudflare" mapstructure:"cloudflare"`
	Kafka      KafkaConfig      `json:"kafka" mapstructure:"kafka"`
	ShortLink  ShortLinkConfig  `json:"shortlink" mapstructure:"shortlink"`
	Nacos      NacosConfig      `json:"nacos" mapstructure:"nacos"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port                string `json:"port" mapstructure:"port"`
	ReadTimeoutSeconds  int    `json:"read_timeout_seconds" mapstructure:"read_timeout_seconds"`
	WriteTimeoutSeconds int    `json:"write_timeout_seconds" mapstructure:"write_timeout_seconds"`
	IdleTimeoutSeconds  int    `json:"idle_timeout_seconds" mapstructure:"idle_timeout_seconds"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string `json:"host" mapstructure:"host"`
	Port     int    `json:"port" mapstructure:"port"`
	User     string `json:"user" mapstructure:"user"`
	Password string `json:"password" mapstructure:"password"`
	Database string `json:"database" mapstructure:"dbname"`
	SSLMode  string `json:"ssl_mode" mapstructure:"sslmode"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret string `json:"secret" mapstructure:"secret"`
}

// CloudflareConfig Cloudflare配置
type CloudflareConfig struct {
	APIToken string `json:"api_token" mapstructure:"api_token"`
	ZoneID   string `json:"zone_id" mapstructure:"zone_id"`
	BaseURL  string `json:"base_url" mapstructure:"base_url"`
}

// ShortLinkConfig 短链接配置
type ShortLinkConfig struct {
	BaseURL string `json:"base_url" mapstructure:"base_url"`
}

// KafkaConfig Kafka配置
type KafkaConfig struct {
	Brokers        []string `json:"brokers" mapstructure:"brokers"`
	ConsumerGroup  string   `json:"consumer_group" mapstructure:"consumer_group"`
	ConsumerTopics []string `json:"consumer_topics" mapstructure:"consumer_topics"`
	ProducerTopics []string `json:"producer_topics" mapstructure:"producer_topics"`
}

// NacosConfig Nacos配置
type NacosConfig struct {
	ServerAddr  string            `json:"server_addr" mapstructure:"server_addr"`   // Nacos服务地址，如localhost:8848
	NamespaceID string            `json:"namespace_id" mapstructure:"namespace_id"` // 命名空间ID，默认为public
	Group       string            `json:"group" mapstructure:"group"`               // 分组，默认为DEFAULT_GROUP
	ServiceName string            `json:"service_name" mapstructure:"service_name"` // 服务名称
	Metadata    map[string]string `json:"metadata" mapstructure:"metadata"`         // 服务元数据
	Weight      float64           `json:"weight" mapstructure:"weight"`             // 服务权重
	Enable      bool              `json:"enable" mapstructure:"enable"`             // 是否启用
	LogDir      string            `json:"log_dir" mapstructure:"log_dir"`           // 日志目录
	CacheDir    string            `json:"cache_dir" mapstructure:"cache_dir"`       // 缓存目录
}

// Load 加载配置
func Load() (*Config, error) {
	configPath := getEnv("CONFIG_PATH", "config.json")

	// 如果配置文件存在，则从文件加载
	if _, err := os.Stat(configPath); err == nil {
		file, err := os.Open(configPath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		config := &Config{}
		err = decoder.Decode(config)
		if err != nil {
			return nil, err
		}

		return config, nil
	}

	// 否则从环境变量加载
	config := &Config{
		Server: ServerConfig{
			Port:                getEnv("SERVER_PORT", "8083"),
			ReadTimeoutSeconds:  getEnvAsInt("SERVER_READ_TIMEOUT", 10),
			WriteTimeoutSeconds: getEnvAsInt("SERVER_WRITE_TIMEOUT", 10),
			IdleTimeoutSeconds:  getEnvAsInt("SERVER_IDLE_TIMEOUT", 60),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "postgres"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Database: getEnv("DB_NAME", "nfc_card"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "your-secret-key"),
		},
		Cloudflare: CloudflareConfig{
			APIToken: getEnv("CLOUDFLARE_API_TOKEN", ""),
			ZoneID:   getEnv("CLOUDFLARE_ZONE_ID", ""),
			BaseURL:  getEnv("CLOUDFLARE_BASE_URL", "https://api.cloudflare.com/client/v4"),
		},
		ShortLink: ShortLinkConfig{
			BaseURL: getEnv("SHORTLINK_BASE_URL", "https://s.example.com"),
		},
		Kafka: KafkaConfig{
			Brokers:        getEnvAsStringSlice("KAFKA_BROKERS", []string{"kafka:9092"}),
			ConsumerGroup:  getEnv("KAFKA_CONSUMER_GROUP", "nfc-service"),
			ConsumerTopics: getEnvAsStringSlice("KAFKA_CONSUMER_TOPICS", []string{"card-events"}),
			ProducerTopics: getEnvAsStringSlice("KAFKA_PRODUCER_TOPICS", []string{"card-events"}),
		},
		Nacos: NacosConfig{
			ServerAddr:  getEnv("NACOS_SERVER_ADDR", "localhost:8848"),
			NamespaceID: getEnv("NACOS_NAMESPACE_ID", "public"),
			Group:       getEnv("NACOS_GROUP", "DEFAULT_GROUP"),
			ServiceName: getEnv("NACOS_SERVICE_NAME", "nfc-service"),
			Metadata:    getEnvAsMap("NACOS_METADATA", map[string]string{}),
			Weight:      getEnvAsFloat("NACOS_WEIGHT", 0.5),
			Enable:      getEnvAsBool("NACOS_ENABLE", true),
			LogDir:      getEnv("NACOS_LOG_DIR", "/tmp/nacos/log"),
			CacheDir:    getEnv("NACOS_CACHE_DIR", "/tmp/nacos/cache"),
		},
	}

	return config, nil
}

// 获取环境变量
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// 获取环境变量并转换为整数
func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsStringSlice 获取环境变量并转换为字符串切片
func getEnvAsStringSlice(key string, defaultValue []string) []string {
	if value, exists := os.LookupEnv(key); exists {
		var result []string
		if err := json.Unmarshal([]byte(value), &result); err == nil {
			return result
		}
	}
	return defaultValue
}

// getEnvAsMap 获取环境变量并转换为map
func getEnvAsMap(key string, defaultValue map[string]string) map[string]string {
	if value, exists := os.LookupEnv(key); exists {
		var result map[string]string
		if err := json.Unmarshal([]byte(value), &result); err == nil {
			return result
		}
	}
	return defaultValue
}

// getEnvAsFloat 获取环境变量并转换为float64
func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// getEnvAsBool 获取环境变量并转换为bool
func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// LoadConfig 从指定路径加载配置
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
