package nacos

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// ServiceConfig 服务配置管理器
type ServiceConfig struct {
	configClient  *ConfigClient
	configManager *ConfigManager
	serviceName   string
	localConfig   *viper.Viper
	configPath    string
	logger        *log.Logger
	useNacos      bool
}

// NewServiceConfig 创建服务配置管理器
func NewServiceConfig(serviceName, configPath string, logger *log.Logger) (*ServiceConfig, error) {
	// 创建Viper实例
	v := viper.New()
	v.SetConfigFile(configPath)

	// 自动设置环境变量
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 读取本地配置
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 创建配置管理器
	sc := &ServiceConfig{
		serviceName: serviceName,
		localConfig: v,
		configPath:  configPath,
		logger:      logger,
		useNacos:    false,
	}

	// 检查是否启用Nacos
	var nacosConfig struct {
		ServerAddr  string `mapstructure:"server_addr"`
		NamespaceID string `mapstructure:"namespace_id"`
		Group       string `mapstructure:"group"`
		Enable      bool   `mapstructure:"enable"`
	}

	// 尝试获取Nacos配置
	if err := v.UnmarshalKey("nacos", &nacosConfig); err != nil {
		logger.Printf("获取Nacos配置失败: %v", err)
		return sc, nil
	}

	// 如果未启用Nacos，则返回
	if !nacosConfig.Enable {
		logger.Printf("Nacos服务未启用")
		return sc, nil
	}

	// 创建Nacos客户端
	config := &Config{
		ServerAddr:  nacosConfig.ServerAddr,
		NamespaceID: nacosConfig.NamespaceID,
		Group:       nacosConfig.Group,
		LogDir:      "/tmp/nacos/log",
		CacheDir:    "/tmp/nacos/cache",
	}

	configClient, err := NewConfigClient(config, logger)
	if err != nil {
		logger.Printf("创建Nacos配置客户端失败: %v", err)
		return sc, nil
	}

	// 创建配置管理器
	configManager := NewConfigManager(configClient, serviceName, logger)
	sc.configClient = configClient
	sc.configManager = configManager
	sc.useNacos = true

	logger.Printf("已启用Nacos配置中心: %s", config.ServerAddr)
	return sc, nil
}

// LoadConfig 加载配置到结构体
func (sc *ServiceConfig) LoadConfig(v interface{}) error {
	// 首先从本地配置文件加载
	if err := sc.localConfig.Unmarshal(v); err != nil {
		return fmt.Errorf("解析本地配置失败: %w", err)
	}

	// 如果启用了Nacos，尝试从Nacos获取配置
	if sc.useNacos {
		dataId := sc.serviceName + ".json"
		content, err := sc.configClient.GetConfig(dataId, "")
		if err == nil {
			// 从Nacos解析配置
			if err := json.Unmarshal([]byte(content), v); err != nil {
				sc.logger.Printf("解析Nacos配置失败: %v", err)
			} else {
				sc.logger.Printf("已从Nacos加载配置: %s", dataId)
			}
		} else {
			sc.logger.Printf("从Nacos获取配置[%s]失败: %v, 将使用本地配置", dataId, err)
		}
	}

	return nil
}

// LoadConfigSection 加载配置的指定部分到结构体
func (sc *ServiceConfig) LoadConfigSection(sectionName string, v interface{}) error {
	// 首先从本地配置文件加载
	if err := sc.localConfig.UnmarshalKey(sectionName, v); err != nil {
		return fmt.Errorf("解析本地配置[%s]失败: %w", sectionName, err)
	}

	// 如果启用了Nacos，尝试从Nacos获取配置
	if sc.useNacos {
		dataId := fmt.Sprintf("%s-%s.json", sc.serviceName, sectionName)
		content, err := sc.configClient.GetConfig(dataId, "")
		if err == nil {
			// 从Nacos解析配置
			if err := json.Unmarshal([]byte(content), v); err != nil {
				sc.logger.Printf("解析Nacos配置[%s]失败: %v", sectionName, err)
			} else {
				sc.logger.Printf("已从Nacos加载配置: %s", dataId)
			}
		} else {
			sc.logger.Printf("从Nacos获取配置[%s]失败: %v, 将使用本地配置", dataId, err)
		}
	}

	return nil
}

// WatchConfig 监听配置变更
func (sc *ServiceConfig) WatchConfig(v interface{}, onChange func()) error {
	// 监听本地配置文件变更
	sc.localConfig.WatchConfig()
	sc.localConfig.OnConfigChange(func(e fsnotify.Event) {
		sc.logger.Printf("本地配置文件已更改: %s", e.Name)
		if err := sc.localConfig.Unmarshal(v); err != nil {
			sc.logger.Printf("重新解析本地配置失败: %v", err)
		} else if onChange != nil {
			onChange()
		}
	})

	// 如果启用了Nacos，监听Nacos配置变更
	if sc.useNacos {
		dataId := sc.serviceName + ".json"
		err := sc.configClient.ListenConfig(dataId, "", func(dataId, group, content string) {
			sc.logger.Printf("Nacos配置已更改: %s", dataId)
			if err := json.Unmarshal([]byte(content), v); err != nil {
				sc.logger.Printf("解析Nacos配置失败: %v", err)
			} else if onChange != nil {
				onChange()
			}
		})
		if err != nil {
			return fmt.Errorf("监听Nacos配置失败: %w", err)
		}
	}

	return nil
}

// WatchConfigSection 监听配置部分变更
func (sc *ServiceConfig) WatchConfigSection(sectionName string, v interface{}, onChange func()) error {
	// 监听本地配置文件变更
	sc.localConfig.WatchConfig()
	sc.localConfig.OnConfigChange(func(e fsnotify.Event) {
		sc.logger.Printf("本地配置文件已更改: %s", e.Name)
		if err := sc.localConfig.UnmarshalKey(sectionName, v); err != nil {
			sc.logger.Printf("重新解析本地配置[%s]失败: %v", sectionName, err)
		} else if onChange != nil {
			onChange()
		}
	})

	// 如果启用了Nacos，监听Nacos配置变更
	if sc.useNacos {
		dataId := fmt.Sprintf("%s-%s.json", sc.serviceName, sectionName)
		err := sc.configClient.ListenConfig(dataId, "", func(dataId, group, content string) {
			sc.logger.Printf("Nacos配置已更改: %s", dataId)
			if err := json.Unmarshal([]byte(content), v); err != nil {
				sc.logger.Printf("解析Nacos配置失败: %v", err)
			} else if onChange != nil {
				onChange()
			}
		})
		if err != nil {
			return fmt.Errorf("监听Nacos配置失败: %w", err)
		}
	}

	return nil
}

// SaveConfig 保存配置到Nacos
func (sc *ServiceConfig) SaveConfig(v interface{}) error {
	if !sc.useNacos {
		return fmt.Errorf("Nacos未启用，无法保存配置")
	}

	// 保存完整配置
	dataId := sc.serviceName + ".json"
	success, err := sc.configClient.PublishConfigFromStruct(dataId, "", v)
	if err != nil {
		return fmt.Errorf("保存配置到Nacos失败: %w", err)
	}

	if !success {
		return fmt.Errorf("保存配置到Nacos失败")
	}

	sc.logger.Printf("成功保存配置到Nacos: %s", dataId)
	return nil
}

// SaveConfigSection 保存配置部分到Nacos
func (sc *ServiceConfig) SaveConfigSection(sectionName string, v interface{}) error {
	if !sc.useNacos {
		return fmt.Errorf("Nacos未启用，无法保存配置")
	}

	// 保存配置部分
	dataId := fmt.Sprintf("%s-%s.json", sc.serviceName, sectionName)
	success, err := sc.configClient.PublishConfigFromStruct(dataId, "", v)
	if err != nil {
		return fmt.Errorf("保存配置[%s]到Nacos失败: %w", sectionName, err)
	}

	if !success {
		return fmt.Errorf("保存配置[%s]到Nacos失败", sectionName)
	}

	sc.logger.Printf("成功保存配置[%s]到Nacos: %s", sectionName, dataId)
	return nil
}

// MigrateToNacos 将本地配置迁移到Nacos
func (sc *ServiceConfig) MigrateToNacos() error {
	if !sc.useNacos {
		return fmt.Errorf("Nacos未启用，无法迁移配置")
	}

	// 读取原始配置内容
	content, err := os.ReadFile(sc.configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config map[string]interface{}

	// 解析配置，支持YAML和JSON
	if strings.HasSuffix(sc.configPath, ".yaml") || strings.HasSuffix(sc.configPath, ".yml") {
		if err := yaml.Unmarshal(content, &config); err != nil {
			return fmt.Errorf("解析YAML配置失败: %w", err)
		}
	} else if strings.HasSuffix(sc.configPath, ".json") {
		if err := json.Unmarshal(content, &config); err != nil {
			return fmt.Errorf("解析JSON配置失败: %w", err)
		}
	} else {
		return fmt.Errorf("不支持的配置文件格式: %s", sc.configPath)
	}

	// 发布完整配置
	dataId := sc.serviceName + ".json"
	jsonData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("转换为JSON失败: %w", err)
	}

	success, err := sc.configClient.PublishConfig(dataId, "", string(jsonData))
	if err != nil {
		return fmt.Errorf("发布配置到Nacos失败: %w", err)
	}

	if !success {
		return fmt.Errorf("发布配置到Nacos失败")
	}

	sc.logger.Printf("成功将配置迁移到Nacos: %s", dataId)

	// 迁移每个配置部分
	for section, value := range config {
		sectionDataId := fmt.Sprintf("%s-%s.json", sc.serviceName, section)
		sectionData, err := json.Marshal(value)
		if err != nil {
			sc.logger.Printf("转换配置部分[%s]为JSON失败: %v", section, err)
			continue
		}

		success, err := sc.configClient.PublishConfig(sectionDataId, "", string(sectionData))
		if err != nil {
			sc.logger.Printf("发布配置部分[%s]到Nacos失败: %v", section, err)
			continue
		}

		if !success {
			sc.logger.Printf("发布配置部分[%s]到Nacos失败", section)
			continue
		}

		sc.logger.Printf("成功将配置部分[%s]迁移到Nacos: %s", section, sectionDataId)
	}

	return nil
}

// SaveLocalConfig 保存到本地配置文件
func (sc *ServiceConfig) SaveLocalConfig(v interface{}) error {
	// 将配置转换为YAML
	yamlData, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("转换为YAML失败: %w", err)
	}

	// 保存到文件
	if err := os.WriteFile(sc.configPath, yamlData, 0644); err != nil {
		return fmt.Errorf("保存配置到文件失败: %w", err)
	}

	sc.logger.Printf("成功保存配置到本地文件: %s", sc.configPath)
	return nil
}

// Close 关闭配置管理器
func (sc *ServiceConfig) Close() {
	if sc.useNacos && sc.configManager != nil {
		sc.configManager.Close()
	}
}
