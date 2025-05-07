package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/nfc_card/shared/nacos"
)

// 命令行参数
var (
	action      = flag.String("action", "import", "执行的动作: import, export, delete, list")
	configDir   = flag.String("config-dir", "./config", "配置文件目录")
	nacosAddr   = flag.String("nacos-addr", "nacos:8848", "Nacos服务地址")
	nacosNs     = flag.String("nacos-ns", "public", "Nacos命名空间")
	nacosGroup  = flag.String("nacos-group", "DEFAULT_GROUP", "Nacos分组")
	serviceName = flag.String("service", "", "服务名称")
	configType  = flag.String("type", "", "配置类型，如server, database等")
	outDir      = flag.String("out-dir", "./config_exported", "配置导出目录")
	verbose     = flag.Bool("verbose", false, "是否显示详细日志")
)

func main() {
	flag.Parse()

	// 创建日志记录器
	logger := log.New(os.Stdout, "[Nacos配置工具] ", log.LstdFlags)

	// 创建Nacos客户端
	config := &nacos.Config{
		ServerAddr:  *nacosAddr,
		NamespaceID: *nacosNs,
		Group:       *nacosGroup,
		LogDir:      "/tmp/nacos/log",
		CacheDir:    "/tmp/nacos/cache",
	}

	logger.Printf("连接到Nacos服务器: %s, 命名空间: %s, 分组: %s",
		config.ServerAddr, config.NamespaceID, config.Group)

	nacosClient, err := nacos.NewConfigClient(config, logger)
	if err != nil {
		logger.Fatalf("创建Nacos客户端失败: %v", err)
	}

	// 根据动作执行不同的操作
	converter := nacos.NewConfigConverter(*configDir, nacosClient, logger)

	switch *action {
	case "import":
		if *serviceName == "" {
			// 导入所有配置
			logger.Printf("导入目录[%s]中的所有配置...", *configDir)
			if err := converter.ImportAll(); err != nil {
				logger.Fatalf("导入所有配置失败: %v", err)
			}
			logger.Printf("成功导入所有配置")
		} else {
			// 导入指定服务的配置
			configPath := filepath.Join(*configDir, *serviceName+".yaml")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				configPath = filepath.Join(*configDir, *serviceName+".yml")
			}

			if *configType == "" {
				// 导入整个配置文件
				logger.Printf("导入配置: %s...", configPath)
				if err := converter.ImportConfig(*serviceName, configPath); err != nil {
					logger.Fatalf("导入配置失败: %v", err)
				}
				logger.Printf("成功导入配置: %s", *serviceName)
			} else {
				// 导入特定类型的配置
				logger.Printf("导入配置: %s, 类型: %s...", configPath, *configType)
				if err := converter.ImportConfigByType(*serviceName, configPath, *configType); err != nil {
					logger.Fatalf("导入配置失败: %v", err)
				}
				logger.Printf("成功导入配置: %s, 类型: %s", *serviceName, *configType)
			}
		}

	case "export":
		if *serviceName == "" {
			logger.Fatalf("导出配置时必须指定服务名称")
		}

		// 确保输出目录存在
		if err := os.MkdirAll(*outDir, 0755); err != nil {
			logger.Fatalf("创建输出目录失败: %v", err)
		}

		// 导出配置
		outPath := filepath.Join(*outDir, *serviceName+".yaml")
		logger.Printf("导出配置: %s -> %s...", *serviceName, outPath)
		if err := converter.ExportConfig(*serviceName, outPath); err != nil {
			logger.Fatalf("导出配置失败: %v", err)
		}
		logger.Printf("成功导出配置: %s", outPath)

	case "delete":
		if *serviceName == "" {
			logger.Fatalf("删除配置时必须指定服务名称")
		}

		// 构建DataID
		dataId := *serviceName
		if *configType != "" {
			dataId = fmt.Sprintf("%s-%s.json", *serviceName, *configType)
		} else {
			dataId = *serviceName + ".json"
		}

		// 删除配置
		logger.Printf("删除配置: %s...", dataId)
		success, err := nacosClient.DeleteConfig(dataId, "")
		if err != nil {
			logger.Fatalf("删除配置失败: %v", err)
		}
		if !success {
			logger.Fatalf("删除配置失败")
		}
		logger.Printf("成功删除配置: %s", dataId)

	case "list":
		logger.Printf("暂不支持列出配置")

	default:
		logger.Fatalf("未知的动作: %s", *action)
	}
}
