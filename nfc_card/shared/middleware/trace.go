package middleware

import (
	"net/http"

	"github.com/nfc_card/shared/logger"
)

const (
	// TraceIDHeader 追踪ID的HTTP头
	TraceIDHeader = "X-Trace-ID"
)

// TraceMiddleware 为每个HTTP请求添加追踪ID
func TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查请求中是否已有追踪ID
		traceID := r.Header.Get(TraceIDHeader)

		// 如果没有，则生成一个新的
		if traceID == "" {
			traceID = logger.GenerateTraceID()
		}

		// 添加追踪ID到响应头
		w.Header().Set(TraceIDHeader, traceID)

		// 将追踪ID添加到请求上下文
		ctx := logger.WithTraceID(r.Context(), traceID)

		// 使用新的上下文继续处理请求
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
