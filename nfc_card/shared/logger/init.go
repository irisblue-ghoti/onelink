package logger

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

// 日志全局实例
var (
	// DefaultLogger 默认日志实例
	DefaultLogger Logger
)

// InitLogger 初始化日志系统
// 它会从环境变量或配置文件加载配置，并设置全局日志实例
func InitLogger(serviceName, configPath string) (Logger, error) {
	// 从环境变量读取日志级别，默认为info
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = LevelInfo
	}

	// 从环境变量读取文件路径
	filePath := os.Getenv("LOG_FILE_PATH")
	if filePath == "" {
		filePath = fmt.Sprintf("logs/%s.log", serviceName)
	}

	// 从环境变量读取是否输出到控制台
	consoleOutput := true
	if consoleEnv := os.Getenv("LOG_CONSOLE_OUTPUT"); consoleEnv != "" {
		var err error
		consoleOutput, err = strconv.ParseBool(consoleEnv)
		if err != nil {
			consoleOutput = true
		}
	}

	// 从环境变量读取是否使用JSON格式
	jsonFormat := true
	if jsonEnv := os.Getenv("LOG_JSON_FORMAT"); jsonEnv != "" {
		var err error
		jsonFormat, err = strconv.ParseBool(jsonEnv)
		if err != nil {
			jsonFormat = true
		}
	}

	// 从环境变量读取是否报告调用者
	reportCaller := true
	if callerEnv := os.Getenv("LOG_REPORT_CALLER"); callerEnv != "" {
		var err error
		reportCaller, err = strconv.ParseBool(callerEnv)
		if err != nil {
			reportCaller = true
		}
	}

	// 创建基本日志配置
	cfg := Config{
		Level:         level,
		ServiceName:   serviceName,
		FilePath:      filePath,
		ConsoleOutput: consoleOutput,
		JSONFormat:    jsonFormat,
		ReportCaller:  reportCaller,
	}

	// 创建基本日志器
	logger, err := NewLogger(cfg)
	if err != nil {
		return nil, fmt.Errorf("创建日志器失败: %w", err)
	}

	// 设置全局标准库日志输出
	log.SetOutput(logger.GetOutput())
	log.SetFlags(0) // 清除默认标志，我们自己格式化日志

	// 记录启动日志
	logger.Info("日志系统已初始化: 服务=%s, 级别=%s, 文件=%s", serviceName, level, filePath)

	// 检查是否启用Elasticsearch
	esEnabled := false
	if esEnv := os.Getenv("ELASTICSEARCH_ENABLED"); esEnv != "" {
		var err error
		esEnabled, err = strconv.ParseBool(esEnv)
		if err != nil {
			esEnabled = false
		}
	}

	// 如果启用Elasticsearch，创建Elasticsearch日志器
	if esEnabled {
		// 读取Elasticsearch配置
		esURL := os.Getenv("ELASTICSEARCH_URL")
		if esURL == "" {
			esURL = "http://elasticsearch:9200"
		}

		esIndexPrefix := os.Getenv("ELASTICSEARCH_INDEX_PREFIX")
		if esIndexPrefix == "" {
			esIndexPrefix = fmt.Sprintf("logs-%s", serviceName)
		}

		esUsername := os.Getenv("ELASTICSEARCH_USERNAME")
		esPassword := os.Getenv("ELASTICSEARCH_PASSWORD")

		esBatchSize := 100
		if batchEnv := os.Getenv("ELASTICSEARCH_BATCH_SIZE"); batchEnv != "" {
			if size, err := strconv.Atoi(batchEnv); err == nil && size > 0 {
				esBatchSize = size
			}
		}

		esFlushInterval := 5 * time.Second
		if flushEnv := os.Getenv("ELASTICSEARCH_FLUSH_INTERVAL"); flushEnv != "" {
			if seconds, err := strconv.Atoi(flushEnv); err == nil && seconds > 0 {
				esFlushInterval = time.Duration(seconds) * time.Second
			}
		}

		// 创建Elasticsearch配置
		esCfg := ElasticsearchConfig{
			URL:           esURL,
			IndexPrefix:   esIndexPrefix,
			Username:      esUsername,
			Password:      esPassword,
			Enabled:       true,
			BatchSize:     esBatchSize,
			FlushInterval: esFlushInterval,
		}

		// 创建Elasticsearch日志器
		esLogger := NewElasticsearchLogger(logger, esCfg)
		if esLogger != nil {
			logger = esLogger
			logger.Info("Elasticsearch日志收集已启用: URL=%s, 索引前缀=%s", esURL, esIndexPrefix)
		}
	}

	// 设置全局日志实例
	DefaultLogger = logger

	return logger, nil
}
