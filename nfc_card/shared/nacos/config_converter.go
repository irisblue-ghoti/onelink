package nacos

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigConverter YAML配置转换器
type ConfigConverter struct {
	configDir   string
	nacosClient *ConfigClient
	logger      *log.Logger
}

// NewConfigConverter 创建配置转换器
func NewConfigConverter(configDir string, nacosClient *ConfigClient, logger *log.Logger) *ConfigConverter {
	return &ConfigConverter{
		configDir:   configDir,
		nacosClient: nacosClient,
		logger:      logger,
	}
}

// ImportAll 导入配置目录中的所有配置
func (c *ConfigConverter) ImportAll() error {
	files, err := os.ReadDir(c.configDir)
	if err != nil {
		return fmt.Errorf("读取配置目录失败: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if strings.HasSuffix(file.Name(), ".yaml") || strings.HasSuffix(file.Name(), ".yml") {
			serviceName := strings.TrimSuffix(strings.TrimSuffix(file.Name(), ".yaml"), ".yml")
			c.logger.Printf("导入配置: %s", serviceName)

			if err := c.ImportConfig(serviceName, filepath.Join(c.configDir, file.Name())); err != nil {
				c.logger.Printf("导入配置[%s]失败: %v", serviceName, err)
			}
		}
	}

	return nil
}

// ImportConfig 导入单个配置文件
func (c *ConfigConverter) ImportConfig(serviceName, filePath string) error {
	// 读取YAML文件
	yamlData, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析YAML
	var yamlConfig map[string]interface{}
	if err := yaml.Unmarshal(yamlData, &yamlConfig); err != nil {
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 转换为JSON
	jsonData, err := json.Marshal(yamlConfig)
	if err != nil {
		return fmt.Errorf("转换为JSON失败: %w", err)
	}

	// 发布到Nacos
	dataId := serviceName + ".json"
	success, err := c.nacosClient.PublishConfig(dataId, "", string(jsonData))
	if err != nil {
		return fmt.Errorf("发布配置到Nacos失败: %w", err)
	}

	if !success {
		return fmt.Errorf("发布配置失败")
	}

	c.logger.Printf("成功导入配置: %s", dataId)
	return nil
}

// ImportConfigByType 根据配置类型导入配置
func (c *ConfigConverter) ImportConfigByType(serviceName, filePath string, configType string) error {
	// 读取YAML文件
	yamlData, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析YAML
	var yamlConfig map[string]interface{}
	if err := yaml.Unmarshal(yamlData, &yamlConfig); err != nil {
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 获取指定类型的配置
	configValue, ok := yamlConfig[configType]
	if !ok {
		return fmt.Errorf("配置[%s]中不存在[%s]配置", serviceName, configType)
	}

	// 转换为JSON
	jsonData, err := json.Marshal(configValue)
	if err != nil {
		return fmt.Errorf("转换为JSON失败: %w", err)
	}

	// 发布到Nacos
	dataId := fmt.Sprintf("%s-%s.json", serviceName, configType)
	success, err := c.nacosClient.PublishConfig(dataId, "", string(jsonData))
	if err != nil {
		return fmt.Errorf("发布配置到Nacos失败: %w", err)
	}

	if !success {
		return fmt.Errorf("发布配置失败")
	}

	c.logger.Printf("成功导入配置: %s", dataId)
	return nil
}

// ExportConfig 从Nacos导出配置到文件
func (c *ConfigConverter) ExportConfig(serviceName, filePath string) error {
	// 从Nacos获取配置
	dataId := serviceName + ".json"
	content, err := c.nacosClient.GetConfig(dataId, "")
	if err != nil {
		return fmt.Errorf("从Nacos获取配置失败: %w", err)
	}

	// 解析JSON
	var jsonConfig map[string]interface{}
	if err := json.Unmarshal([]byte(content), &jsonConfig); err != nil {
		return fmt.Errorf("解析JSON失败: %w", err)
	}

	// 转换为YAML
	yamlData, err := yaml.Marshal(jsonConfig)
	if err != nil {
		return fmt.Errorf("转换为YAML失败: %w", err)
	}

	// 保存到文件
	if err := os.WriteFile(filePath, yamlData, 0644); err != nil {
		return fmt.Errorf("保存配置到文件失败: %w", err)
	}

	c.logger.Printf("成功导出配置: %s -> %s", dataId, filePath)
	return nil
}
