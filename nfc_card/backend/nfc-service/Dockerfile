FROM golang:1.24.2-alpine AS builder

WORKDIR /app

# 设置Go环境变量
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct

# 安装必要的构建工具
RUN apk add --no-cache git gcc musl-dev

# 创建项目结构
RUN mkdir -p /app/backend/nfc-service
RUN mkdir -p /app/shared

# 复制服务代码
COPY backend/nfc-service/ /app/backend/nfc-service/

# 复制shared目录到正确的相对路径
COPY shared/ /app/shared/

# 设置工作目录到服务目录
WORKDIR /app/backend/nfc-service

# 修复依赖问题
RUN go get github.com/spf13/viper && \
    go get github.com/gin-gonic/gin/binding@v1.10.0 && \
    go get google.golang.org/protobuf/proto && \
    go mod tidy

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -o nfc-service ./cmd/server/main.go

# 创建最终镜像
FROM alpine:latest

WORKDIR /app

# 复制可执行文件
COPY --from=builder /app/backend/nfc-service/nfc-service .
COPY backend/nfc-service/config.yaml .

# 设置时区
RUN apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 创建日志目录
RUN mkdir -p /app/logs

# 设置环境变量
ENV PORT=8083

# 暴露端口
EXPOSE 8083

# 启动应用
CMD ["./nfc-service"] 