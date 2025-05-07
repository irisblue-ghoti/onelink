package nacos

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
)

// ServiceDiscovery 服务发现客户端
type ServiceDiscovery struct {
	client              *Client
	cachedInstances     map[string][]model.Instance
	cachedInstancesLock sync.RWMutex
	subscriptions       map[string]bool
	subscriptionsLock   sync.RWMutex
}

// NewServiceDiscovery 创建服务发现客户端
func NewServiceDiscovery(client *Client) *ServiceDiscovery {
	return &ServiceDiscovery{
		client:          client,
		cachedInstances: make(map[string][]model.Instance),
		subscriptions:   make(map[string]bool),
	}
}

// GetServiceURL 获取服务URL
func (sd *ServiceDiscovery) GetServiceURL(serviceName string) (string, error) {
	instance, err := sd.client.GetRandomServiceInstance(serviceName)
	if err != nil {
		return "", err
	}

	// 构建URL
	schema := "http"
	if _, ok := instance.Metadata["secure"]; ok {
		schema = "https"
	}

	return fmt.Sprintf("%s://%s:%d", schema, instance.Ip, instance.Port), nil
}

// GetServiceInstance 获取服务实例
func (sd *ServiceDiscovery) GetServiceInstance(serviceName string) (*model.Instance, error) {
	return sd.client.GetRandomServiceInstance(serviceName)
}

// GetAllServiceInstances 获取所有服务实例
func (sd *ServiceDiscovery) GetAllServiceInstances(serviceName string) ([]model.Instance, error) {
	// 尝试从缓存中获取
	sd.cachedInstancesLock.RLock()
	instances, ok := sd.cachedInstances[serviceName]
	sd.cachedInstancesLock.RUnlock()

	if ok {
		return instances, nil
	}

	// 从Nacos获取
	instances, err := sd.client.GetService(serviceName)
	if err != nil {
		return nil, err
	}

	// 缓存结果
	sd.cachedInstancesLock.Lock()
	sd.cachedInstances[serviceName] = instances
	sd.cachedInstancesLock.Unlock()

	return instances, nil
}

// SubscribeService 订阅服务变更
func (sd *ServiceDiscovery) SubscribeService(serviceName string) error {
	sd.subscriptionsLock.Lock()
	defer sd.subscriptionsLock.Unlock()

	// 如果已经订阅，则直接返回
	if _, ok := sd.subscriptions[serviceName]; ok {
		return nil
	}

	// 订阅服务变更
	err := sd.client.Subscribe(serviceName, func(instances []model.Instance) {
		// 更新缓存
		sd.cachedInstancesLock.Lock()
		sd.cachedInstances[serviceName] = instances
		sd.cachedInstancesLock.Unlock()

		log.Printf("服务[%s]实例列表已更新，共%d个实例", serviceName, len(instances))
	})

	if err != nil {
		return err
	}

	sd.subscriptions[serviceName] = true
	return nil
}

// UnsubscribeService 取消订阅服务变更
func (sd *ServiceDiscovery) UnsubscribeService(serviceName string) error {
	sd.subscriptionsLock.Lock()
	defer sd.subscriptionsLock.Unlock()

	// 如果没有订阅，则直接返回
	if _, ok := sd.subscriptions[serviceName]; !ok {
		return nil
	}

	// 取消订阅
	err := sd.client.Unsubscribe(serviceName)
	if err != nil {
		return err
	}

	delete(sd.subscriptions, serviceName)
	return nil
}

// CreateHTTPClient 创建带有服务发现功能的HTTP客户端
func (sd *ServiceDiscovery) CreateHTTPClient() *HTTPClient {
	return &HTTPClient{
		discovery:   sd,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		serviceURLs: make(map[string]string),
	}
}

// HTTPClient 带有服务发现功能的HTTP客户端
type HTTPClient struct {
	discovery   *ServiceDiscovery
	httpClient  *http.Client
	serviceURLs map[string]string
	mutex       sync.RWMutex
}

// GetServiceURL 获取服务URL，优先使用缓存
func (c *HTTPClient) GetServiceURL(serviceName string) (string, error) {
	// 尝试从缓存获取URL
	c.mutex.RLock()
	url, ok := c.serviceURLs[serviceName]
	c.mutex.RUnlock()

	if ok {
		return url, nil
	}

	// 从服务发现获取URL
	url, err := c.discovery.GetServiceURL(serviceName)
	if err != nil {
		return "", err
	}

	// 缓存URL
	c.mutex.Lock()
	c.serviceURLs[serviceName] = url
	c.mutex.Unlock()

	return url, nil
}

// Get 发送GET请求到指定服务
func (c *HTTPClient) Get(serviceName, path string) (*http.Response, error) {
	serviceURL, err := c.GetServiceURL(serviceName)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s%s", serviceURL, path)
	return c.httpClient.Get(url)
}

// Post 发送POST请求到指定服务
func (c *HTTPClient) Post(serviceName, path string, contentType string, body []byte) (*http.Response, error) {
	serviceURL, err := c.GetServiceURL(serviceName)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s%s", serviceURL, path)
	return c.httpClient.Post(url, contentType, nil)
}

// Do 执行HTTP请求
func (c *HTTPClient) Do(serviceName string, req *http.Request) (*http.Response, error) {
	serviceURL, err := c.GetServiceURL(serviceName)
	if err != nil {
		return nil, err
	}

	// 修改请求URL，替换为服务URL
	req.URL.Scheme = "http"
	req.URL.Host = serviceURL[7:] // 移除"http://"前缀

	return c.httpClient.Do(req)
}

// RefreshServiceURL 刷新服务URL缓存
func (c *HTTPClient) RefreshServiceURL(serviceName string) error {
	url, err := c.discovery.GetServiceURL(serviceName)
	if err != nil {
		return err
	}

	c.mutex.Lock()
	c.serviceURLs[serviceName] = url
	c.mutex.Unlock()

	return nil
}

// ClearServiceURLCache 清除服务URL缓存
func (c *HTTPClient) ClearServiceURLCache() {
	c.mutex.Lock()
	c.serviceURLs = make(map[string]string)
	c.mutex.Unlock()
}

// SetTimeout 设置HTTP客户端超时时间
func (c *HTTPClient) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}
