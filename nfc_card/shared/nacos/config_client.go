package nacos

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// ConfigClient Nacos配置中心客户端
type ConfigClient struct {
	client       config_client.IConfigClient
	config       *Config
	configCache  map[string]string
	cacheLock    sync.RWMutex
	listeners    map[string][]ConfigChangeListener
	listenerLock sync.RWMutex
	logger       *log.Logger
}

// ConfigChangeListener 配置变更监听器
type ConfigChangeListener func(dataId, group, content string)

// NewConfigClient 创建Nacos配置中心客户端
func NewConfigClient(config *Config, logger *log.Logger) (*ConfigClient, error) {
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
	serverAddrs := parseServerAddrs(config.ServerAddr)
	if len(serverAddrs) == 0 {
		return nil, fmt.Errorf("无效的服务器地址: %s", config.ServerAddr)
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

	// 创建配置中心客户端
	configClient, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverAddrs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("创建Nacos配置中心客户端失败: %w", err)
	}

	return &ConfigClient{
		client:      configClient,
		config:      config,
		configCache: make(map[string]string),
		listeners:   make(map[string][]ConfigChangeListener),
		logger:      logger,
	}, nil
}

// GetConfig 获取配置
func (c *ConfigClient) GetConfig(dataId, group string) (string, error) {
	// 如果未指定分组，使用默认分组
	if group == "" {
		group = c.config.Group
	}

	// 尝试从缓存获取配置
	cacheKey := fmt.Sprintf("%s:%s", group, dataId)
	c.cacheLock.RLock()
	if content, ok := c.configCache[cacheKey]; ok {
		c.cacheLock.RUnlock()
		return content, nil
	}
	c.cacheLock.RUnlock()

	// 从Nacos获取配置
	content, err := c.client.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})

	if err != nil {
		return "", fmt.Errorf("从Nacos获取配置失败: %w", err)
	}

	// 缓存配置
	c.cacheLock.Lock()
	c.configCache[cacheKey] = content
	c.cacheLock.Unlock()

	return content, nil
}

// PublishConfig 发布配置
func (c *ConfigClient) PublishConfig(dataId, group, content string) (bool, error) {
	// 如果未指定分组，使用默认分组
	if group == "" {
		group = c.config.Group
	}

	// 发布配置
	success, err := c.client.PublishConfig(vo.ConfigParam{
		DataId:  dataId,
		Group:   group,
		Content: content,
	})

	if err != nil {
		return false, fmt.Errorf("发布配置失败: %w", err)
	}

	// 更新缓存
	if success {
		cacheKey := fmt.Sprintf("%s:%s", group, dataId)
		c.cacheLock.Lock()
		c.configCache[cacheKey] = content
		c.cacheLock.Unlock()
	}

	return success, nil
}

// DeleteConfig 删除配置
func (c *ConfigClient) DeleteConfig(dataId, group string) (bool, error) {
	// 如果未指定分组，使用默认分组
	if group == "" {
		group = c.config.Group
	}

	// 删除配置
	success, err := c.client.DeleteConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})

	if err != nil {
		return false, fmt.Errorf("删除配置失败: %w", err)
	}

	// 更新缓存
	if success {
		cacheKey := fmt.Sprintf("%s:%s", group, dataId)
		c.cacheLock.Lock()
		delete(c.configCache, cacheKey)
		c.cacheLock.Unlock()
	}

	return success, nil
}

// ListenConfig 监听配置变更
func (c *ConfigClient) ListenConfig(dataId, group string, listener ConfigChangeListener) error {
	// 如果未指定分组，使用默认分组
	if group == "" {
		group = c.config.Group
	}

	// 注册监听器
	err := c.client.ListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
		OnChange: func(namespace, group, dataId, data string) {
			c.logger.Printf("配置变更: namespace=%s, group=%s, dataId=%s", namespace, group, dataId)

			// 更新缓存
			cacheKey := fmt.Sprintf("%s:%s", group, dataId)
			c.cacheLock.Lock()
			c.configCache[cacheKey] = data
			c.cacheLock.Unlock()

			// 调用监听器
			c.listenerLock.RLock()
			listenerKey := fmt.Sprintf("%s:%s", group, dataId)
			listeners := c.listeners[listenerKey]
			c.listenerLock.RUnlock()

			for _, l := range listeners {
				go l(dataId, group, data)
			}
		},
	})

	if err != nil {
		return fmt.Errorf("监听配置失败: %w", err)
	}

	// 记录监听器
	c.listenerLock.Lock()
	listenerKey := fmt.Sprintf("%s:%s", group, dataId)
	c.listeners[listenerKey] = append(c.listeners[listenerKey], listener)
	c.listenerLock.Unlock()

	return nil
}

// GetConfigToStruct 获取配置并解析为结构体
func (c *ConfigClient) GetConfigToStruct(dataId, group string, v interface{}) error {
	content, err := c.GetConfig(dataId, group)
	if err != nil {
		return err
	}

	// 解析JSON
	if err := json.Unmarshal([]byte(content), v); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	return nil
}

// PublishConfigFromStruct 将结构体发布为配置
func (c *ConfigClient) PublishConfigFromStruct(dataId, group string, v interface{}) (bool, error) {
	// 序列化结构体
	content, err := json.Marshal(v)
	if err != nil {
		return false, fmt.Errorf("序列化配置失败: %w", err)
	}

	// 发布配置
	return c.PublishConfig(dataId, group, string(content))
}

// CancelListenConfig 取消监听配置变更
func (c *ConfigClient) CancelListenConfig(dataId, group string) error {
	// 如果未指定分组，使用默认分组
	if group == "" {
		group = c.config.Group
	}

	// 取消监听
	err := c.client.CancelListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})

	if err != nil {
		return fmt.Errorf("取消监听配置失败: %w", err)
	}

	// 清除监听器
	c.listenerLock.Lock()
	listenerKey := fmt.Sprintf("%s:%s", group, dataId)
	delete(c.listeners, listenerKey)
	c.listenerLock.Unlock()

	return nil
}

// 解析服务器地址
func parseServerAddrs(serverAddr string) []constant.ServerConfig {
	addrs := splitAddr(serverAddr)
	serverConfigs := make([]constant.ServerConfig, 0, len(addrs))

	for _, addr := range addrs {
		ip, port, err := parseAddr(addr)
		if err != nil {
			log.Printf("解析地址失败: %v", err)
			continue
		}

		serverConfigs = append(serverConfigs, constant.ServerConfig{
			IpAddr: ip,
			Port:   uint64(port),
		})
	}

	return serverConfigs
}

// 分割地址字符串
func splitAddr(addrStr string) []string {
	return strings.Split(addrStr, ",")
}

// 解析IP和端口
func parseAddr(addr string) (string, int, error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("无效的地址格式: %s", addr)
	}

	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("无效的端口号: %s", parts[1])
	}

	return parts[0], port, nil
}
