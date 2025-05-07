package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/nfc_card/shared/nacos"
)

// 服务名称常量
const (
	MerchantService     = "merchant-service"
	NFCService          = "nfc-service"
	ContentService      = "content-service"
	StatsService        = "stats-service"
	DistributionService = "distribution-service"
)

// Client 服务客户端
type Client struct {
	discovery    *nacos.ServiceDiscovery
	httpClient   *http.Client
	serviceURLs  map[string]string
	mutex        sync.RWMutex
	enableNacos  bool
	defaultPorts map[string]int
	logger       *log.Logger
}

// NewClient 创建服务客户端
func NewClient(nacosClient *nacos.Client, logger *log.Logger) *Client {
	client := &Client{
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		serviceURLs: make(map[string]string),
		enableNacos: nacosClient != nil,
		defaultPorts: map[string]int{
			MerchantService:     8082,
			NFCService:          8083,
			ContentService:      8081,
			StatsService:        8084,
			DistributionService: 8085,
		},
		logger: logger,
	}

	// 如果启用了Nacos，则创建服务发现客户端
	if client.enableNacos {
		client.discovery = nacos.NewServiceDiscovery(nacosClient)
	}

	return client
}

// GetServiceURL 获取服务URL
func (c *Client) GetServiceURL(serviceName string) string {
	// 尝试从缓存获取URL
	c.mutex.RLock()
	url, ok := c.serviceURLs[serviceName]
	c.mutex.RUnlock()

	if ok {
		return url
	}

	// 如果启用了Nacos，则从Nacos获取服务实例
	if c.enableNacos {
		instance, err := c.discovery.GetServiceInstance(serviceName)
		if err == nil {
			// 构建URL
			url = fmt.Sprintf("http://%s:%d", instance.Ip, instance.Port)

			// 缓存URL
			c.mutex.Lock()
			c.serviceURLs[serviceName] = url
			c.mutex.Unlock()

			return url
		}

		c.logger.Printf("从Nacos获取服务[%s]地址失败: %v，将使用默认地址", serviceName, err)
	}

	// 如果未启用Nacos或从Nacos获取失败，则使用默认地址
	port, ok := c.defaultPorts[serviceName]
	if !ok {
		c.logger.Printf("未知的服务名称: %s", serviceName)
		port = 8080
	}

	// 在开发环境中，使用服务名作为主机名（适用于Docker环境）
	url = fmt.Sprintf("http://%s:%d", serviceName, port)

	// 缓存URL
	c.mutex.Lock()
	c.serviceURLs[serviceName] = url
	c.mutex.Unlock()

	return url
}

// RefreshServiceURL 刷新服务URL
func (c *Client) RefreshServiceURL(serviceName string) {
	if !c.enableNacos {
		return
	}

	// 从Nacos获取服务实例
	instance, err := c.discovery.GetServiceInstance(serviceName)
	if err != nil {
		c.logger.Printf("刷新服务[%s]地址失败: %v", serviceName, err)
		return
	}

	// 构建URL
	url := fmt.Sprintf("http://%s:%d", instance.Ip, instance.Port)

	// 更新缓存
	c.mutex.Lock()
	c.serviceURLs[serviceName] = url
	c.mutex.Unlock()

	c.logger.Printf("已刷新服务[%s]地址: %s", serviceName, url)
}

// Get 发送GET请求
func (c *Client) Get(serviceName, path string) (*http.Response, error) {
	serviceURL := c.GetServiceURL(serviceName)
	url := fmt.Sprintf("%s%s", serviceURL, path)
	return c.httpClient.Get(url)
}

// GetJSON 发送GET请求并解析JSON响应
func (c *Client) GetJSON(serviceName, path string, result interface{}) error {
	resp, err := c.Get(serviceName, path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务[%s]返回非200状态码: %d，响应: %s", serviceName, resp.StatusCode, string(body))
	}

	// 解析JSON
	if err := json.Unmarshal(body, result); err != nil {
		return err
	}

	return nil
}

// Post 发送POST请求
func (c *Client) Post(serviceName, path string, contentType string, body io.Reader) (*http.Response, error) {
	serviceURL := c.GetServiceURL(serviceName)
	url := fmt.Sprintf("%s%s", serviceURL, path)
	return c.httpClient.Post(url, contentType, body)
}

// PostJSON 发送POST请求并解析JSON响应
func (c *Client) PostJSON(serviceName, path string, data, result interface{}) error {
	// 序列化请求数据
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// 发送请求
	resp, err := c.Post(serviceName, path, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 读取响应内容
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("服务[%s]返回非成功状态码: %d，响应: %s", serviceName, resp.StatusCode, string(respBody))
	}

	// 如果result不为nil，则解析JSON响应
	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return err
		}
	}

	return nil
}

// SetTimeout 设置HTTP客户端超时时间
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// SubscribeService 订阅服务变更
func (c *Client) SubscribeService(serviceName string) error {
	if !c.enableNacos {
		return nil
	}

	// 订阅服务变更
	return c.discovery.SubscribeService(serviceName)
}
