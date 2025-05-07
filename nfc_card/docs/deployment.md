# NFC 碰一碰分发系统部署文档

本文档提供了 NFC 碰一碰分发系统的部署指南，包括开发环境的单商户部署和生产环境的多租户部署。

## 目录

- [环境要求](#环境要求)
- [本地开发环境](#本地开发环境)
- [单商户部署](#单商户部署)
- [多租户 Kubernetes 部署](#多租户-kubernetes-部署)
- [配置说明](#配置说明)
- [监控和维护](#监控和维护)

## 环境要求

### 开发环境

- Docker 20.10+
- Docker Compose 2.0+
- Node.js 18+
- Go 1.18+
- PostgreSQL 14+
- Redis 7+
- Kafka 3.0+

### 生产环境

- Kubernetes 1.22+
- Helm 3.5+
- NGINX Ingress Controller
- cert-manager 1.8+
- 一个域名 (用于访问系统)

## 本地开发环境

### 1. 克隆仓库

```bash
git clone https://github.com/your-org/nfc-card.git
cd nfc-card
```

### 2. 安装依赖

```bash
# 安装 NestJS 服务依赖
cd merchant-service && npm install
cd ../content-service && npm install
cd ../stats-service && npm install

# 安装前端依赖
cd ../frontend && npm install

# 安装 Go 依赖
cd ../distribution-service && go mod tidy
cd ../nfc-service && go mod tidy
cd ..
```

### 3. 启动开发环境

```bash
# 启动所需的基础服务
docker-compose -f docker-compose.dev.yml up -d

# 运行服务 (每个服务在单独的终端窗口中)
cd merchant-service && npm run start:dev
cd content-service && npm run start:dev
cd stats-service && npm run start:dev
cd distribution-service && go run cmd/server/main.go
cd nfc-service && go run cmd/server/main.go
cd frontend && npm run dev
```

### 4. 访问开发环境

- 前端: http://localhost:3000
- API: http://localhost:9080

## 单商户部署

对于小规模使用或者测试部署，可以使用 Docker Compose 快速部署单商户版本：

### 1. 准备环境

```bash
# 克隆代码库
git clone https://github.com/your-org/nfc-card.git
cd nfc-card

# 初始化环境变量
cp .env.example .env
```

### 2. 配置环境变量

编辑 `.env` 文件，设置以下关键配置：

```
# 基础配置
NODE_ENV=production
SINGLE_TENANT_MODE=true
DEFAULT_TENANT_ID=00000000-0000-0000-0000-000000000001

# 数据库配置
DB_HOST=postgres
DB_PORT=5432
DB_USERNAME=postgres
DB_PASSWORD=your_secure_password
DB_NAME=nfc_card

# 认证配置
JWT_SECRET=your_jwt_secret_key
JWT_EXPIRES_IN=1d

# 对象存储配置
STORAGE_TYPE=s3     # 可选: local, s3, oss
S3_BUCKET=your-nfc-videos
S3_REGION=your-region
S3_ACCESS_KEY=your-access-key
S3_SECRET_KEY=your-secret-key

# 平台配置 (示例：抖音)
DOUYIN_CLIENT_KEY=your_client_key
DOUYIN_CLIENT_SECRET=your_client_secret
```

### 3. 启动服务

```bash
# 启动所有服务
docker-compose up -d

# 确认所有服务都已启动
docker-compose ps
```

### 4. 初始化数据库

```bash
# 运行数据库迁移和种子数据
docker-compose exec merchant-service npm run db:migrate
docker-compose exec merchant-service npm run db:seed
```

### 5. 创建管理员账户

```bash
# 创建平台管理员
docker-compose exec merchant-service npm run create-admin -- --email=admin@example.com --password=admin_password
```

### 6. 访问系统

- 前端: http://your-server-ip:3000
- API: http://your-server-ip:9080

## 多租户 Kubernetes 部署

对于生产环境，推荐使用 Kubernetes 进行多租户部署：

### 1. 前置准备

- 已配置好的 Kubernetes 集群
- kubectl 命令行工具
- Helm 3.5+
- 已配置的域名，指向 Kubernetes 集群入口 IP

### 2. 添加必要的 Helm 仓库

```bash
# 添加需要的 Helm 仓库
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add apisix https://charts.apiseven.com
helm repo add cert-manager https://charts.jetstack.io
helm repo update
```

### 3. 安装证书管理器

```bash
# 安装 cert-manager (用于 HTTPS 证书)
helm install cert-manager cert-manager/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set installCRDs=true
```

### 4. 创建命名空间和密钥

```bash
# 创建命名空间
kubectl create namespace nfc-card

# 创建数据库密钥
kubectl create secret generic postgres-credentials \
  --namespace nfc-card \
  --from-literal=postgres-password=YOUR_SECURE_PASSWORD

# 创建应用密钥
kubectl create secret generic app-secrets \
  --namespace nfc-card \
  --from-literal=jwt-secret=YOUR_JWT_SECRET \
  --from-literal=s3-access-key=YOUR_S3_ACCESS_KEY \
  --from-literal=s3-secret-key=YOUR_S3_SECRET_KEY \
  --from-literal=douyin-client-key=YOUR_DOUYIN_KEY \
  --from-literal=douyin-client-secret=YOUR_DOUYIN_SECRET
```

### 5. 安装依赖服务

```bash
# 安装 PostgreSQL
helm install postgres bitnami/postgresql \
  --namespace nfc-card \
  --set global.postgresql.auth.postgresPassword=YOUR_SECURE_PASSWORD \
  --set global.postgresql.auth.database=nfc_card

# 安装 Redis
helm install redis bitnami/redis \
  --namespace nfc-card \
  --set auth.password=YOUR_REDIS_PASSWORD

# 安装 Kafka
helm install kafka bitnami/kafka \
  --namespace nfc-card \
  --set replicaCount=3
```

### 6. 安装 APISIX 网关

```bash
# 安装 APISIX
helm install apisix apisix/apisix \
  --namespace nfc-card \
  --set gateway.type=LoadBalancer \
  --set ingress-controller.enabled=true
```

### 7. 配置 TLS 证书签发器

```bash
# 创建 ClusterIssuer (用于自动获取 HTTPS 证书)
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: your-email@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
```

### 8. 部署应用

```bash
# 使用 Helm 部署应用
helm install nfc-card ./helm \
  --namespace nfc-card \
  --values ./helm/values.prod.yaml \
  --set global.environment=production \
  --set global.domain=your-domain.com \
  --set postgres.existingSecret=postgres-credentials
```

### 9. 验证部署

```bash
# 检查 Pod 状态
kubectl get pods -n nfc-card

# 检查服务
kubectl get svc -n nfc-card
```

### 10. 访问系统

- 前端: https://your-domain.com
- API: https://api.your-domain.com

## 配置说明

### 核心配置文件

- `.env` - 开发环境配置
- `helm/values.yaml` - Helm 默认配置
- `helm/values.prod.yaml` - 生产环境配置
- `helm/values.dev.yaml` - 开发环境配置

### 重要配置项

| 配置项 | 说明 | 默认值 |
|--------|------|---------|
| `global.environment` | 环境标识 | `development` |
| `global.domain` | 系统域名 | `nfc-card.example.com` |
| `postgres.host` | PostgreSQL 主机 | `postgres` |
| `redis.host` | Redis 主机 | `redis` |
| `kafka.brokers` | Kafka 代理列表 | `kafka-0.kafka-headless:9092` |
| `services.*.replicaCount` | 服务副本数 | `2` |
| `services.*.resources` | 资源限制 | 视服务而定 |

## 监控和维护

### 日志收集

系统使用 Fluentd 收集日志，并发送到 Elasticsearch：

```bash
# 安装 Elasticsearch 和 Kibana
helm install elasticsearch bitnami/elasticsearch \
  --namespace nfc-card-monitoring \
  --create-namespace

helm install kibana bitnami/kibana \
  --namespace nfc-card-monitoring \
  --set elasticsearch.hosts[0]=elasticsearch-coordinating-only \
  --set elasticsearch.port=9200

# 安装 Fluentd
helm install fluentd bitnami/fluentd \
  --namespace nfc-card-monitoring \
  --set forwarder.configMap=fluentd-forwarder \
  --set aggregator.configMap=fluentd-aggregator
```

### 性能监控

系统使用 Prometheus 和 Grafana 进行监控：

```bash
# 安装 Prometheus
helm install prometheus bitnami/kube-prometheus \
  --namespace nfc-card-monitoring

# 安装 Grafana
helm install grafana bitnami/grafana \
  --namespace nfc-card-monitoring \
  --set datasources.secretName=grafana-datasources
```

### 备份策略

数据库备份使用 PostgreSQL 的连续归档（WAL）和定期备份：

```bash
# 设置定期备份任务
kubectl apply -f k8s/cronjobs/db-backup.yaml
```

### 故障恢复

系统设计为高可用，但如需手动恢复：

1. 从备份恢复数据库：

```bash
kubectl exec -it postgres-0 -n nfc-card -- bash -c 'pg_restore -d nfc_card /backups/nfc_card_backup.dump'
```

2. 重启服务：

```bash
kubectl rollout restart deployment -n nfc-card
``` 