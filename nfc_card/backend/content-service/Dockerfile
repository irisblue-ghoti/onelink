FROM golang:1.24.2-alpine AS builder

WORKDIR /app

# 设置Go环境变量
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct

# 安装必要的构建工具
RUN apk add --no-cache git gcc musl-dev

# 创建项目结构
RUN mkdir -p /app/backend/content-service
RUN mkdir -p /app/shared

# 复制服务代码
COPY backend/content-service/ /app/backend/content-service/

# 复制shared目录到正确的相对路径
COPY shared/ /app/shared/

# 设置工作目录到服务目录
WORKDIR /app/backend/content-service

# 添加replace指令
RUN echo "replace github.com/nfc_card/shared => ../../shared" >> go.mod

# 修复依赖问题
RUN go get github.com/spf13/viper && \
    go get github.com/gin-gonic/gin/binding@v1.10.0 && \
    go get google.golang.org/protobuf/proto && \
    go mod tidy

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -o content-service ./cmd/server/main.go

# 创建最终镜像
FROM alpine:latest

WORKDIR /app

# 复制可执行文件
COPY --from=builder /app/backend/content-service/content-service .

# 复制配置文件目录
RUN mkdir -p /app/config
COPY config/ /app/config/

# 设置时区
RUN apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 创建日志目录
RUN mkdir -p /app/logs

# 设置环境变量
ENV PORT=8081
ENV CONFIG_PATH=/app/config/content-service.yaml

# 暴露端口
EXPOSE 8081

# 启动应用
CMD ["./content-service"] 