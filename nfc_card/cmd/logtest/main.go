package main

import (
	"context"
	"os"
	"time"

	"github.com/nfc_card/shared/logger"
)

func main() {
	// 设置环境变量
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("LOG_FILE_PATH", "logs/test.log")
	os.Setenv("LOG_CONSOLE_OUTPUT", "true")
	os.Setenv("LOG_JSON_FORMAT", "true")

	// 初始化日志系统
	log, err := logger.InitLogger("log-test", "")
	if err != nil {
		panic("初始化日志系统失败: " + err.Error())
	}

	// 输出不同级别的日志
	log.Debug("这是一条调试日志")
	log.Info("这是一条信息日志")
	log.Warn("这是一条警告日志")
	log.Error("这是一条错误日志")

	// 使用字段
	log.WithField("module", "测试模块").Info("带有字段的日志")
	log.WithFields(map[string]interface{}{
		"user_id": 123,
		"action":  "login",
	}).Info("带有多个字段的日志")

	// 使用追踪ID
	ctx := context.Background()
	traceID := logger.GenerateTraceID()
	ctx = logger.WithTraceID(ctx, traceID)

	log.InfoContext(ctx, "这条日志包含追踪ID: %s", traceID)

	// 模拟请求处理
	processRequest(ctx, "GET", "/api/user/123")

	// 等待日志落盘
	time.Sleep(1 * time.Second)
}

func processRequest(ctx context.Context, method, path string) {
	log := logger.DefaultLogger.WithFields(map[string]interface{}{
		"method": method,
		"path":   path,
	})

	log.InfoContext(ctx, "开始处理请求")

	// 模拟处理过程
	time.Sleep(100 * time.Millisecond)

	// 记录处理结果
	log.InfoContext(ctx, "请求处理完成，耗时: 100ms")
}
