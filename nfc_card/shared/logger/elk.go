package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// ElasticsearchConfig Elasticsearch配置
type ElasticsearchConfig struct {
	// Elasticsearch地址
	URL string
	// 索引名称前缀
	IndexPrefix string
	// 认证信息
	Username string
	Password string
	// 是否启用
	Enabled bool
	// 批量发送大小
	BatchSize int
	// 发送间隔
	FlushInterval time.Duration
}

// DefaultElasticsearchConfig 默认Elasticsearch配置
func DefaultElasticsearchConfig() ElasticsearchConfig {
	return ElasticsearchConfig{
		URL:           "http://elasticsearch:9200",
		IndexPrefix:   "logs",
		Enabled:       false,
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
	}
}

// ElasticsearchLogger Elasticsearch日志收集器
type ElasticsearchLogger struct {
	logger    Logger
	config    ElasticsearchConfig
	buffer    []map[string]interface{}
	client    *http.Client
	flushChan chan struct{}
	stopChan  chan struct{}
}

// 创建Elasticsearch日志收集器
func NewElasticsearchLogger(baseLogger Logger, cfg ElasticsearchConfig) *ElasticsearchLogger {
	if !cfg.Enabled {
		return nil
	}

	logger := &ElasticsearchLogger{
		logger:    baseLogger,
		config:    cfg,
		buffer:    make([]map[string]interface{}, 0, cfg.BatchSize),
		client:    &http.Client{Timeout: 5 * time.Second},
		flushChan: make(chan struct{}),
		stopChan:  make(chan struct{}),
	}

	// 启动定时刷新协程
	go logger.flushWorker()

	return logger
}

// 停止日志收集器
func (l *ElasticsearchLogger) Stop() {
	// 发送停止信号
	l.stopChan <- struct{}{}

	// 刷新剩余日志
	l.Flush()
}

// 日志收集工作协程
func (l *ElasticsearchLogger) flushWorker() {
	ticker := time.NewTicker(l.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.Flush()
		case <-l.flushChan:
			l.Flush()
		case <-l.stopChan:
			return
		}
	}
}

// Flush 刷新日志到Elasticsearch
func (l *ElasticsearchLogger) Flush() {
	if len(l.buffer) == 0 {
		return
	}

	// 复制当前缓冲区
	logs := make([]map[string]interface{}, len(l.buffer))
	copy(logs, l.buffer)

	// 清空缓冲区
	l.buffer = make([]map[string]interface{}, 0, l.config.BatchSize)

	// 异步发送日志
	go l.sendLogs(logs)
}

// 发送日志到Elasticsearch
func (l *ElasticsearchLogger) sendLogs(logs []map[string]interface{}) {
	// 检查日志数量
	if len(logs) == 0 {
		return
	}

	// 创建批量请求
	var bulkBody bytes.Buffer

	// 获取当前日期，用于索引名
	now := time.Now()
	indexName := fmt.Sprintf("%s-%s", l.config.IndexPrefix, now.Format("2006.01.02"))

	// 构建批量请求
	for _, log := range logs {
		// 添加索引信息
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": indexName,
			},
		}

		// 将日志添加到批量请求
		actionJSON, err := json.Marshal(action)
		if err != nil {
			l.logger.Error("序列化Elasticsearch索引信息失败: %v", err)
			continue
		}

		logJSON, err := json.Marshal(log)
		if err != nil {
			l.logger.Error("序列化日志失败: %v", err)
			continue
		}

		bulkBody.Write(actionJSON)
		bulkBody.WriteByte('\n')
		bulkBody.Write(logJSON)
		bulkBody.WriteByte('\n')
	}

	// 发送请求
	url := fmt.Sprintf("%s/_bulk", l.config.URL)
	req, err := http.NewRequest("POST", url, &bulkBody)
	if err != nil {
		l.logger.Error("创建Elasticsearch请求失败: %v", err)
		return
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/x-ndjson")

	// 添加认证
	if l.config.Username != "" && l.config.Password != "" {
		req.SetBasicAuth(l.config.Username, l.config.Password)
	}

	// 执行请求
	resp, err := l.client.Do(req)
	if err != nil {
		l.logger.Error("发送日志到Elasticsearch失败: %v", err)
		return
	}
	defer resp.Body.Close()

	// 检查响应
	if resp.StatusCode >= 400 {
		l.logger.Error("Elasticsearch响应错误: %s", resp.Status)
	}
}

// 添加日志到缓冲区
func (l *ElasticsearchLogger) addLog(ctx context.Context, level, msg string, fields logrus.Fields) {
	// 创建日志记录
	logEntry := make(map[string]interface{})

	// 添加基本字段
	logEntry["timestamp"] = time.Now().Format(time.RFC3339)
	logEntry["level"] = level
	logEntry["message"] = msg

	// 添加追踪ID
	if ctx != nil {
		if traceID := GetTraceID(ctx); traceID != "" {
			logEntry["trace_id"] = traceID
		}
	}

	// 添加其他字段
	for k, v := range fields {
		logEntry[k] = v
	}

	// 添加到缓冲区
	l.buffer = append(l.buffer, logEntry)

	// 如果达到批量大小，触发刷新
	if len(l.buffer) >= l.config.BatchSize {
		l.flushChan <- struct{}{}
	}
}

// 实现Logger接口的方法
func (l *ElasticsearchLogger) Debug(format string, args ...interface{}) {
	l.logger.Debug(format, args...)
	l.addLog(nil, LevelDebug, fmt.Sprintf(format, args...), l.logger.(*logrusLogger).fields)
}

func (l *ElasticsearchLogger) Info(format string, args ...interface{}) {
	l.logger.Info(format, args...)
	l.addLog(nil, LevelInfo, fmt.Sprintf(format, args...), l.logger.(*logrusLogger).fields)
}

func (l *ElasticsearchLogger) Warn(format string, args ...interface{}) {
	l.logger.Warn(format, args...)
	l.addLog(nil, LevelWarn, fmt.Sprintf(format, args...), l.logger.(*logrusLogger).fields)
}

func (l *ElasticsearchLogger) Error(format string, args ...interface{}) {
	l.logger.Error(format, args...)
	l.addLog(nil, LevelError, fmt.Sprintf(format, args...), l.logger.(*logrusLogger).fields)
}

func (l *ElasticsearchLogger) Fatal(format string, args ...interface{}) {
	l.logger.Fatal(format, args...)
	// 不需要添加到Elasticsearch，因为Fatal会导致程序退出
}

func (l *ElasticsearchLogger) DebugContext(ctx context.Context, format string, args ...interface{}) {
	l.logger.DebugContext(ctx, format, args...)
	l.addLog(ctx, LevelDebug, fmt.Sprintf(format, args...), l.logger.(*logrusLogger).fields)
}

func (l *ElasticsearchLogger) InfoContext(ctx context.Context, format string, args ...interface{}) {
	l.logger.InfoContext(ctx, format, args...)
	l.addLog(ctx, LevelInfo, fmt.Sprintf(format, args...), l.logger.(*logrusLogger).fields)
}

func (l *ElasticsearchLogger) WarnContext(ctx context.Context, format string, args ...interface{}) {
	l.logger.WarnContext(ctx, format, args...)
	l.addLog(ctx, LevelWarn, fmt.Sprintf(format, args...), l.logger.(*logrusLogger).fields)
}

func (l *ElasticsearchLogger) ErrorContext(ctx context.Context, format string, args ...interface{}) {
	l.logger.ErrorContext(ctx, format, args...)
	l.addLog(ctx, LevelError, fmt.Sprintf(format, args...), l.logger.(*logrusLogger).fields)
}

func (l *ElasticsearchLogger) FatalContext(ctx context.Context, format string, args ...interface{}) {
	l.logger.FatalContext(ctx, format, args...)
	// 不需要添加到Elasticsearch，因为Fatal会导致程序退出
}

func (l *ElasticsearchLogger) WithField(key string, value interface{}) Logger {
	return &ElasticsearchLogger{
		logger:    l.logger.WithField(key, value),
		config:    l.config,
		buffer:    l.buffer,
		client:    l.client,
		flushChan: l.flushChan,
		stopChan:  l.stopChan,
	}
}

func (l *ElasticsearchLogger) WithFields(fields map[string]interface{}) Logger {
	return &ElasticsearchLogger{
		logger:    l.logger.WithFields(fields),
		config:    l.config,
		buffer:    l.buffer,
		client:    l.client,
		flushChan: l.flushChan,
		stopChan:  l.stopChan,
	}
}

func (l *ElasticsearchLogger) WithError(err error) Logger {
	return &ElasticsearchLogger{
		logger:    l.logger.WithError(err),
		config:    l.config,
		buffer:    l.buffer,
		client:    l.client,
		flushChan: l.flushChan,
		stopChan:  l.stopChan,
	}
}

func (l *ElasticsearchLogger) GetOutput() io.Writer {
	return l.logger.GetOutput()
}
