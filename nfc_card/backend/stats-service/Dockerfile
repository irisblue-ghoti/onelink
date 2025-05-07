FROM golang:1.24.2-alpine AS builder

WORKDIR /app

# 设置Go环境变量
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct

# 安装必要的构建工具
RUN apk add --no-cache git gcc musl-dev

# 创建项目结构
RUN mkdir -p /app/backend/stats-service
RUN mkdir -p /app/shared

# 复制服务代码
COPY backend/stats-service/ /app/backend/stats-service/

# 复制shared目录到正确的相对路径
COPY shared/ /app/shared/

# 设置工作目录到服务目录
WORKDIR /app/backend/stats-service

# 添加replace指令
RUN echo "replace github.com/nfc_card/shared => ../../shared" >> go.mod

# 修复依赖问题
RUN go get github.com/spf13/viper && \
    go get github.com/gin-gonic/gin/binding@v1.10.0 && \
    go get google.golang.org/protobuf/proto && \
    go mod tidy

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -o stats-service ./cmd/server/main.go

# 创建最终镜像
FROM alpine:latest

WORKDIR /app

# 复制可执行文件
COPY --from=builder /app/backend/stats-service/stats-service .
COPY backend/stats-service/config.yaml .

# 设置时区
RUN apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 创建日志目录
RUN mkdir -p /app/logs

# 设置环境变量
ENV PORT=8084

# 暴露端口
EXPOSE 8084

# 启动应用
CMD ["./stats-service"] 