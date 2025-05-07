# Nacos配置中心

本工具用于将项目配置迁移到Nacos配置中心，并提供配置的导入、导出和管理功能。

## 功能特点

- 将YAML格式的本地配置文件导入到Nacos配置中心
- 支持导入完整配置或指定配置部分
- 从Nacos配置中心导出配置到本地文件
- 支持配置的热更新和动态监听
- 简单易用的命令行接口

## 使用方法

### 1. 导入配置

将`config`目录下的所有配置文件导入到Nacos：

```bash
go run cmd/nacos-config/main.go -action import -config-dir ./config
```

导入指定服务的配置：

```bash
go run cmd/nacos-config/main.go -action import -config-dir ./config -service nfc-service
```

导入服务的指定配置部分：

```bash
go run cmd/nacos-config/main.go -action import -config-dir ./config -service nfc-service -type database
```

### 2. 导出配置

从Nacos导出配置到本地文件：

```bash
go run cmd/nacos-config/main.go -action export -service nfc-service -out-dir ./exported_config
```

### 3. 删除配置

删除Nacos中的配置：

```bash
go run cmd/nacos-config/main.go -action delete -service nfc-service
```

删除指定配置部分：

```bash
go run cmd/nacos-config/main.go -action delete -service nfc-service -type database
```

## 在服务中集成Nacos配置中心

### 1. 在服务启动时加载配置

修改服务的配置加载逻辑，使其优先从Nacos加载配置，示例代码：

```go
package main

import (
    "log"
    "os"
    
    "github.com/nfc_card/shared/nacos"
    "your-service/internal/config"
)

func loadConfig() (*config.Config, error) {
    configPath := os.Getenv("CONFIG_PATH")
    if configPath == "" {
        configPath = "config/your-service.yaml"
    }
    
    logger := log.New(os.Stdout, "[配置] ", log.LstdFlags)
    
    // 创建服务配置管理器
    serviceConfig, err := nacos.NewServiceConfig("your-service", configPath, logger)
    if err != nil {
        logger.Printf("创建配置管理器失败: %v，将使用本地配置", err)
        return config.LoadConfig(configPath)
    }
    
    // 创建配置实例
    cfg := &config.Config{}
    
    // 加载配置
    if err := serviceConfig.LoadConfig(cfg); err != nil {
        logger.Printf("加载配置失败: %v，将使用本地配置", err)
        return config.LoadConfig(configPath)
    }
    
    // 设置配置变更监听
    serviceConfig.WatchConfig(cfg, func() {
        logger.Printf("配置已更新")
        // 这里可以添加配置变更后的回调处理
    })
    
    return cfg, nil
}
```

### 2. 添加配置热更新支持

对于需要动态更新的配置部分，可以单独监听：

```go
// 监听数据库配置变更
dbConfig := &config.DatabaseConfig{}
serviceConfig.LoadConfigSection("database", dbConfig)
serviceConfig.WatchConfigSection("database", dbConfig, func() {
    // 重新初始化数据库连接
    db.Reconnect(dbConfig)
})
```

### 3. 在Docker环境中使用

修改Docker Compose配置，确保Nacos服务地址正确：

```yaml
services:
  your-service:
    # ...其他配置...
    environment:
      - CONFIG_PATH=/app/config/your-service.yaml
      - NACOS_SERVER_ADDR=nacos:8848
      - NACOS_NAMESPACE_ID=public
    depends_on:
      - nacos
```

## 配置格式

在Nacos中，配置使用JSON格式存储，而本地使用YAML格式。转换是自动完成的。

配置ID的命名规则：
- 完整配置：`{服务名}.json`
- 配置部分：`{服务名}-{配置部分}.json`

例如：
- `nfc-service.json` - NFC服务的完整配置
- `nfc-service-database.json` - NFC服务的数据库配置部分

## 最佳实践

1. **启用配置热更新**：在开发和测试环境中启用配置热更新，便于调试和测试。

2. **配置优先级**：优先使用Nacos配置中心的配置，当获取失败时才使用本地配置。

3. **敏感信息加密**：对于密码等敏感信息，建议使用加密存储，并在应用程序中解密。

4. **分环境管理**：使用Nacos的命名空间功能，为不同环境（开发、测试、生产）创建独立的配置管理空间。

5. **版本管理**：利用Nacos的配置版本管理功能，记录配置变更历史。

6. **权限控制**：为不同角色设置适当的配置访问权限，避免误操作。

7. **使用内嵌数据库**：本项目使用Nacos的内嵌数据库（Derby）存储配置，无需额外的MySQL依赖，适合开发和小型部署环境。如需更好的性能和可靠性，可以在生产环境中配置外部MySQL。 