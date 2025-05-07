package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// 日志级别常量
const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
	LevelFatal = "fatal"
)

// TraceIDKey 追踪ID的上下文键
type TraceIDKey string

const (
	// ContextKeyTraceID 追踪ID的上下文键名
	ContextKeyTraceID TraceIDKey = "trace_id"
)

// Logger 统一日志接口
type Logger interface {
	// 基本日志方法
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})

	// 带上下文的日志方法，支持传递追踪ID
	DebugContext(ctx context.Context, format string, args ...interface{})
	InfoContext(ctx context.Context, format string, args ...interface{})
	WarnContext(ctx context.Context, format string, args ...interface{})
	ErrorContext(ctx context.Context, format string, args ...interface{})
	FatalContext(ctx context.Context, format string, args ...interface{})

	// 设置额外字段
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
	WithError(err error) Logger

	// 获取输出Writer
	GetOutput() io.Writer
}

// logrusLogger logrus实现的日志器
type logrusLogger struct {
	logger *logrus.Logger
	fields logrus.Fields
}

// Config 日志配置
type Config struct {
	// 日志级别
	Level string
	// 服务名称
	ServiceName string
	// 日志文件路径
	FilePath string
	// 是否输出到控制台
	ConsoleOutput bool
	// 是否使用JSON格式
	JSONFormat bool
	// 是否包含调用文件和行号
	ReportCaller bool
}

// DefaultConfig 默认日志配置
func DefaultConfig() Config {
	return Config{
		Level:         LevelInfo,
		ServiceName:   "service",
		FilePath:      "logs/service.log",
		ConsoleOutput: true,
		JSONFormat:    true,
		ReportCaller:  true,
	}
}

// NewLogger 创建一个新的日志器
func NewLogger(cfg Config) (Logger, error) {
	// 创建logrus实例
	l := logrus.New()

	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	l.SetLevel(level)

	// 设置输出格式
	if cfg.JSONFormat {
		l.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyFunc:  "function",
				logrus.FieldKeyFile:  "file",
			},
		})
	} else {
		l.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: time.RFC3339,
			FullTimestamp:   true,
		})
	}

	// 设置是否报告调用位置
	l.SetReportCaller(cfg.ReportCaller)

	// 设置输出
	var writers []io.Writer

	// 添加控制台输出
	if cfg.ConsoleOutput {
		writers = append(writers, os.Stdout)
	}

	// 添加文件输出
	if cfg.FilePath != "" {
		// 确保日志目录存在
		logDir := filepath.Dir(cfg.FilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("创建日志目录失败: %w", err)
		}

		// 打开日志文件
		file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("打开日志文件失败: %w", err)
		}

		writers = append(writers, file)
	}

	// 如果存在多个输出，使用MultiWriter
	if len(writers) > 1 {
		l.SetOutput(io.MultiWriter(writers...))
	} else if len(writers) == 1 {
		l.SetOutput(writers[0])
	}

	return &logrusLogger{
		logger: l,
		fields: logrus.Fields{
			"service": cfg.ServiceName,
		},
	}, nil
}

// GenerateTraceID 生成追踪ID
func GenerateTraceID() string {
	return uuid.New().String()
}

// WithTraceID 向上下文添加追踪ID
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, ContextKeyTraceID, traceID)
}

// GetTraceID 从上下文获取追踪ID
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if traceID, ok := ctx.Value(ContextKeyTraceID).(string); ok {
		return traceID
	}
	return ""
}

// 提取调用者信息
func extractCaller() (function, file string, line int) {
	// 跳过两个调用帧：当前函数和调用日志的方法
	pc, file, line, ok := runtime.Caller(3)
	if !ok {
		return "unknown", "unknown", 0
	}

	// 获取函数名
	fn := runtime.FuncForPC(pc)
	function = "unknown"
	if fn != nil {
		function = fn.Name()
		// 只保留包名和函数名
		parts := strings.Split(function, ".")
		if len(parts) > 1 {
			function = parts[len(parts)-2] + "." + parts[len(parts)-1]
		}
	}

	// 简化文件路径，只保留最后两部分
	parts := strings.Split(file, "/")
	if len(parts) > 2 {
		file = filepath.Join(parts[len(parts)-2], parts[len(parts)-1])
	}

	return function, file, line
}

// getFields 从上下文获取字段，包括追踪ID
func (l *logrusLogger) getFields(ctx context.Context) logrus.Fields {
	fields := make(logrus.Fields)

	// 复制已有字段
	for k, v := range l.fields {
		fields[k] = v
	}

	// 添加上下文中的追踪ID
	if ctx != nil {
		if traceID := GetTraceID(ctx); traceID != "" {
			fields["trace_id"] = traceID
		}
	}

	return fields
}

// GetOutput 获取日志输出
func (l *logrusLogger) GetOutput() io.Writer {
	return l.logger.Out
}

// WithField 添加一个字段
func (l *logrusLogger) WithField(key string, value interface{}) Logger {
	newFields := make(logrus.Fields)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &logrusLogger{
		logger: l.logger,
		fields: newFields,
	}
}

// WithFields 添加多个字段
func (l *logrusLogger) WithFields(fields map[string]interface{}) Logger {
	newFields := make(logrus.Fields)
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &logrusLogger{
		logger: l.logger,
		fields: newFields,
	}
}

// WithError 添加错误字段
func (l *logrusLogger) WithError(err error) Logger {
	return l.WithField("error", err.Error())
}

// Debug 输出Debug级别日志
func (l *logrusLogger) Debug(format string, args ...interface{}) {
	entry := l.logger.WithFields(l.fields)
	entry.Debugf(format, args...)
}

// Info 输出Info级别日志
func (l *logrusLogger) Info(format string, args ...interface{}) {
	entry := l.logger.WithFields(l.fields)
	entry.Infof(format, args...)
}

// Warn 输出Warn级别日志
func (l *logrusLogger) Warn(format string, args ...interface{}) {
	entry := l.logger.WithFields(l.fields)
	entry.Warnf(format, args...)
}

// Error 输出Error级别日志
func (l *logrusLogger) Error(format string, args ...interface{}) {
	entry := l.logger.WithFields(l.fields)
	entry.Errorf(format, args...)
}

// Fatal 输出Fatal级别日志
func (l *logrusLogger) Fatal(format string, args ...interface{}) {
	entry := l.logger.WithFields(l.fields)
	entry.Fatalf(format, args...)
}

// DebugContext 使用上下文输出Debug级别日志
func (l *logrusLogger) DebugContext(ctx context.Context, format string, args ...interface{}) {
	entry := l.logger.WithFields(l.getFields(ctx))
	entry.Debugf(format, args...)
}

// InfoContext 使用上下文输出Info级别日志
func (l *logrusLogger) InfoContext(ctx context.Context, format string, args ...interface{}) {
	entry := l.logger.WithFields(l.getFields(ctx))
	entry.Infof(format, args...)
}

// WarnContext 使用上下文输出Warn级别日志
func (l *logrusLogger) WarnContext(ctx context.Context, format string, args ...interface{}) {
	entry := l.logger.WithFields(l.getFields(ctx))
	entry.Warnf(format, args...)
}

// ErrorContext 使用上下文输出Error级别日志
func (l *logrusLogger) ErrorContext(ctx context.Context, format string, args ...interface{}) {
	entry := l.logger.WithFields(l.getFields(ctx))
	entry.Errorf(format, args...)
}

// FatalContext 使用上下文输出Fatal级别日志
func (l *logrusLogger) FatalContext(ctx context.Context, format string, args ...interface{}) {
	entry := l.logger.WithFields(l.getFields(ctx))
	entry.Fatalf(format, args...)
}
