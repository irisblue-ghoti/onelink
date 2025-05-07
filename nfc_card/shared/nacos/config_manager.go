package nacos

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"sync"
)

// ConfigManager 配置管理器
type ConfigManager struct {
	client         *ConfigClient
	configs        map[string]interface{}
	configLock     sync.RWMutex
	serviceName    string
	changeHandlers map[string][]func(interface{})
	handlerLock    sync.RWMutex
	logger         *log.Logger
}

// NewConfigManager 创建配置管理器
func NewConfigManager(client *ConfigClient, serviceName string, logger *log.Logger) *ConfigManager {
	return &ConfigManager{
		client:         client,
		configs:        make(map[string]interface{}),
		serviceName:    serviceName,
		changeHandlers: make(map[string][]func(interface{})),
		logger:         logger,
	}
}

// LoadConfig 加载指定配置到结构体
func (m *ConfigManager) LoadConfig(configName string, v interface{}) error {
	// 构建DataID
	dataId := fmt.Sprintf("%s-%s.json", m.serviceName, configName)

	// 获取配置并解析
	err := m.client.GetConfigToStruct(dataId, "", v)
	if err != nil {
		return fmt.Errorf("加载配置[%s]失败: %w", configName, err)
	}

	// 缓存配置
	m.configLock.Lock()
	m.configs[configName] = v
	m.configLock.Unlock()

	m.logger.Printf("成功加载配置: %s", configName)
	return nil
}

// WatchConfig 监听配置变更
func (m *ConfigManager) WatchConfig(configName string, v interface{}, onChange func(interface{})) error {
	// 构建DataID
	dataId := fmt.Sprintf("%s-%s.json", m.serviceName, configName)

	// 监听配置变更
	err := m.client.ListenConfig(dataId, "", func(dataId, group, content string) {
		m.logger.Printf("配置[%s]已变更", configName)

		// 创建一个新的配置实例
		newConfig := cloneStruct(v)
		if newConfig == nil {
			m.logger.Printf("创建配置实例失败")
			return
		}

		// 解析新配置
		if err := json.Unmarshal([]byte(content), newConfig); err != nil {
			m.logger.Printf("解析变更配置失败: %v", err)
			return
		}

		// 更新缓存
		m.configLock.Lock()
		m.configs[configName] = newConfig
		m.configLock.Unlock()

		// 调用变更处理器
		m.handlerLock.RLock()
		handlers := m.changeHandlers[configName]
		m.handlerLock.RUnlock()

		for _, handler := range handlers {
			handler(newConfig)
		}

		// 调用特定的变更处理器
		if onChange != nil {
			onChange(newConfig)
		}
	})

	if err != nil {
		return fmt.Errorf("监听配置[%s]失败: %w", configName, err)
	}

	// 注册变更处理器
	if onChange != nil {
		m.handlerLock.Lock()
		m.changeHandlers[configName] = append(m.changeHandlers[configName], onChange)
		m.handlerLock.Unlock()
	}

	return nil
}

// GetConfig 获取配置
func (m *ConfigManager) GetConfig(configName string) interface{} {
	m.configLock.RLock()
	defer m.configLock.RUnlock()
	return m.configs[configName]
}

// GetTypedConfig 获取配置并转换为指定类型
func (m *ConfigManager) GetTypedConfig(configName string, v interface{}) bool {
	m.configLock.RLock()
	config, ok := m.configs[configName]
	m.configLock.RUnlock()

	if !ok {
		return false
	}

	// 序列化再反序列化，转换类型
	data, err := json.Marshal(config)
	if err != nil {
		m.logger.Printf("序列化配置[%s]失败: %v", configName, err)
		return false
	}

	if err := json.Unmarshal(data, v); err != nil {
		m.logger.Printf("反序列化配置[%s]失败: %v", configName, err)
		return false
	}

	return true
}

// SaveConfig 保存配置到Nacos
func (m *ConfigManager) SaveConfig(configName string, config interface{}) error {
	// 构建DataID
	dataId := fmt.Sprintf("%s-%s.json", m.serviceName, configName)

	// 发布配置
	success, err := m.client.PublishConfigFromStruct(dataId, "", config)
	if err != nil {
		return fmt.Errorf("保存配置[%s]失败: %w", configName, err)
	}

	if !success {
		return fmt.Errorf("保存配置[%s]失败", configName)
	}

	// 更新缓存
	m.configLock.Lock()
	m.configs[configName] = config
	m.configLock.Unlock()

	m.logger.Printf("成功保存配置: %s", configName)
	return nil
}

// RegisterChangeHandler 注册配置变更处理器
func (m *ConfigManager) RegisterChangeHandler(configName string, handler func(interface{})) {
	m.handlerLock.Lock()
	defer m.handlerLock.Unlock()
	m.changeHandlers[configName] = append(m.changeHandlers[configName], handler)
}

// Close 关闭配置管理器
func (m *ConfigManager) Close() {
	// 遍历所有配置，取消监听
	m.configLock.RLock()
	configNames := make([]string, 0, len(m.configs))
	for name := range m.configs {
		configNames = append(configNames, name)
	}
	m.configLock.RUnlock()

	for _, name := range configNames {
		dataId := fmt.Sprintf("%s-%s.json", m.serviceName, name)
		if err := m.client.CancelListenConfig(dataId, ""); err != nil {
			m.logger.Printf("取消监听配置[%s]失败: %v", name, err)
		}
	}
}

// cloneStruct 创建结构体的副本
func cloneStruct(src interface{}) interface{} {
	// 获取类型信息
	val := reflect.ValueOf(src)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// 只复制结构体
	if val.Kind() != reflect.Struct {
		return nil
	}

	// 创建一个新的同类型实例
	newVal := reflect.New(val.Type())

	// 序列化再反序列化
	data, err := json.Marshal(src)
	if err != nil {
		return nil
	}

	if err := json.Unmarshal(data, newVal.Interface()); err != nil {
		return nil
	}

	return newVal.Interface()
}
