package nacos

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// Config Nacos配置
type Config struct {
	ServerAddr  string `mapstructure:"server_addr"`  // Nacos服务地址，如localhost:8848
	NamespaceID string `mapstructure:"namespace_id"` // 命名空间ID，默认为public
	Group       string `mapstructure:"group"`        // 分组，默认为DEFAULT_GROUP
	DataID      string `mapstructure:"data_id"`      // 配置ID
	Username    string `mapstructure:"username"`     // 用户名
	Password    string `mapstructure:"password"`     // 密码
	LogDir      string `mapstructure:"log_dir"`      // 日志目录
	CacheDir    string `mapstructure:"cache_dir"`    // 缓存目录
}

// Client Nacos客户端
type Client struct {
	config       *Config
	namingClient naming_client.INamingClient
}

// NewClient 创建一个新的Nacos客户端
func NewClient(config *Config) (*Client, error) {
	// 设置默认值
	if config.NamespaceID == "" {
		config.NamespaceID = "public"
	}
	if config.Group == "" {
		config.Group = "DEFAULT_GROUP"
	}
	if config.LogDir == "" {
		config.LogDir = "/tmp/nacos/log"
	}
	if config.CacheDir == "" {
		config.CacheDir = "/tmp/nacos/cache"
	}

	// 解析服务器地址
	serverAddrs := strings.Split(config.ServerAddr, ",")
	serverConfigs := make([]constant.ServerConfig, 0, len(serverAddrs))

	for _, addr := range serverAddrs {
		parts := strings.Split(addr, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("无效的服务器地址格式: %s", addr)
		}

		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("无效的端口号: %s", parts[1])
		}

		serverConfigs = append(serverConfigs, constant.ServerConfig{
			IpAddr: parts[0],
			Port:   uint64(port),
		})
	}

	// 创建客户端配置
	clientConfig := constant.ClientConfig{
		NamespaceId:         config.NamespaceID,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              config.LogDir,
		CacheDir:            config.CacheDir,
		Username:            config.Username,
		Password:            config.Password,
		LogLevel:            "info",
	}

	// 创建命名服务客户端
	namingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("创建Nacos命名服务客户端失败: %w", err)
	}

	return &Client{
		config:       config,
		namingClient: namingClient,
	}, nil
}

// RegisterService 注册服务实例
func (c *Client) RegisterService(serviceName, ip string, port int, metadata map[string]string) (bool, error) {
	// 如果未指定IP，则尝试获取本机IP
	if ip == "" {
		localIP, err := c.getLocalIP()
		if err != nil {
			return false, fmt.Errorf("无法获取本机IP: %w", err)
		}
		ip = localIP
	}

	// 注册服务实例
	success, err := c.namingClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          ip,
		Port:        uint64(port),
		ServiceName: serviceName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    metadata,
		GroupName:   c.config.Group,
	})

	if err != nil {
		return false, fmt.Errorf("注册服务实例失败: %w", err)
	}

	return success, nil
}

// DeregisterService 注销服务实例
func (c *Client) DeregisterService(serviceName, ip string, port int) (bool, error) {
	success, err := c.namingClient.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          ip,
		Port:        uint64(port),
		ServiceName: serviceName,
		Ephemeral:   true,
		GroupName:   c.config.Group,
	})

	if err != nil {
		return false, fmt.Errorf("注销服务实例失败: %w", err)
	}

	return success, nil
}

// GetService 获取服务实例
func (c *Client) GetService(serviceName string) ([]model.Instance, error) {
	instances, err := c.namingClient.SelectInstances(vo.SelectInstancesParam{
		ServiceName: serviceName,
		GroupName:   c.config.Group,
		HealthyOnly: true,
	})

	if err != nil {
		return nil, fmt.Errorf("获取服务实例失败: %w", err)
	}

	return instances, nil
}

// GetRandomServiceInstance 随机获取一个服务实例
func (c *Client) GetRandomServiceInstance(serviceName string) (*model.Instance, error) {
	instance, err := c.namingClient.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		ServiceName: serviceName,
		GroupName:   c.config.Group,
	})

	if err != nil {
		return nil, fmt.Errorf("获取服务实例失败: %w", err)
	}

	return instance, nil
}

// StartHealthCheck 开始健康检查
func (c *Client) StartHealthCheck(serviceName, ip string, port int, checkInterval time.Duration) {
	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		for range ticker.C {
			_, err := c.namingClient.UpdateInstance(vo.UpdateInstanceParam{
				Ip:          ip,
				Port:        uint64(port),
				ServiceName: serviceName,
				Weight:      10,
				Enable:      true,
				Healthy:     true,
				Ephemeral:   true,
				GroupName:   c.config.Group,
			})

			if err != nil {
				log.Printf("更新服务实例状态失败: %v", err)
			}
		}
	}()
}

// Subscribe 订阅服务变更
func (c *Client) Subscribe(serviceName string, callback func(instances []model.Instance)) error {
	err := c.namingClient.Subscribe(&vo.SubscribeParam{
		ServiceName: serviceName,
		GroupName:   c.config.Group,
		SubscribeCallback: func(services []model.Instance, err error) {
			if err != nil {
				log.Printf("服务订阅回调错误: %v", err)
				return
			}

			callback(services)
		},
	})

	if err != nil {
		return fmt.Errorf("订阅服务失败: %w", err)
	}

	return nil
}

// Unsubscribe 取消订阅服务变更
func (c *Client) Unsubscribe(serviceName string) error {
	err := c.namingClient.Unsubscribe(&vo.SubscribeParam{
		ServiceName: serviceName,
		GroupName:   c.config.Group,
	})

	if err != nil {
		return fmt.Errorf("取消订阅服务失败: %w", err)
	}

	return nil
}

// getLocalIP 获取本机IP
func (c *Client) getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("无法获取本机IP地址")
}
