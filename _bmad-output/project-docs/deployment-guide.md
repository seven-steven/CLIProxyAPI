# 部署指南

**生成日期:** 2026-03-06

---

## 部署选项

CLIProxyAPI 支持多种部署方式：

1. **Docker** (推荐)
2. **二进制文件**
3. **Docker Compose**
4. **Kubernetes**

---

## Docker 部署

### 拉取镜像

```bash
docker pull eceasy/cli-proxy-api:latest
```

### 运行容器

```bash
# 基本运行
docker run -d \
  --name cliproxyapi \
  -p 8317:8317 \
  -v $(pwd)/config.yaml:/CLIProxyAPI/config.yaml \
  -v $(pwd)/auths:/CLIProxyAPI/auths \
  eceasy/cli-proxy-api:latest

# 带环境变量
docker run -d \
  --name cliproxyapi \
  -p 8317:8317 \
  -e MANAGEMENT_PASSWORD=your-password \
  -v $(pwd)/config.yaml:/CLIProxyAPI/config.yaml \
  -v $(pwd)/auths:/CLIProxyAPI/auths \
  eceasy/cli-proxy-api:latest
```

### 多架构支持

镜像支持以下架构：

- `linux/amd64`
- `linux/arm64`

```bash
# 指定架构
docker pull --platform linux/arm64 eceasy/cli-proxy-api:latest
```

---

## Docker Compose 部署

### docker-compose.yml

```yaml
version: "3.8"

services:
  cliproxyapi:
    image: eceasy/cli-proxy-api:latest
    container_name: cliproxyapi
    restart: unless-stopped
    ports:
      - "8317:8317"
    volumes:
      - ./config.yaml:/CLIProxyAPI/config.yaml
      - ./auths:/CLIProxyAPI/auths
      - ./logs:/CLIProxyAPI/logs
    environment:
      - TZ=Asia/Shanghai
      - MANAGEMENT_PASSWORD=${MANAGEMENT_PASSWORD}
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8317/"]
      interval: 30s
      timeout: 10s
      retries: 3
```

### 启动服务

```bash
# 创建 .env 文件
echo "MANAGEMENT_PASSWORD=your-password" > .env

# 启动
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止
docker-compose down
```

---

## 二进制文件部署

### 下载

从 [GitHub Releases](https://github.com/router-for-me/CLIProxyAPI/releases) 下载对应平台的二进制文件。

### Linux Systemd 服务

创建服务文件 `/etc/systemd/system/cliproxyapi.service`:

```ini
[Unit]
Description=CLIProxyAPI Service
After=network.target

[Service]
Type=simple
User=cliproxy
Group=cliproxy
WorkingDirectory=/opt/cliproxyapi
ExecStart=/opt/cliproxyapi/CLIProxyAPI -config /opt/cliproxyapi/config.yaml
Restart=on-failure
RestartSec=5s

# 环境变量
Environment="MANAGEMENT_PASSWORD=your-password"

# 安全设置
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/cliproxyapi

[Install]
WantedBy=multi-user.target
```

### 启用服务

```bash
# 重载 systemd
sudo systemctl daemon-reload

# 启用开机启动
sudo systemctl enable cliproxyapi

# 启动服务
sudo systemctl start cliproxyapi

# 查看状态
sudo systemctl status cliproxyapi

# 查看日志
sudo journalctl -u cliproxyapi -f
```

---

## Kubernetes 部署

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cliproxyapi
  labels:
    app: cliproxyapi
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cliproxyapi
  template:
    metadata:
      labels:
        app: cliproxyapi
    spec:
      containers:
        - name: cliproxyapi
          image: eceasy/cli-proxy-api:latest
          ports:
            - containerPort: 8317
          env:
            - name: MANAGEMENT_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: cliproxyapi-secret
                  key: password
          volumeMounts:
            - name: config
              mountPath: /CLIProxyAPI/config.yaml
              subPath: config.yaml
            - name: auths
              mountPath: /CLIProxyAPI/auths
          resources:
            requests:
              memory: "256Mi"
              cpu: "100m"
            limits:
              memory: "512Mi"
              cpu: "500m"
          livenessProbe:
            httpGet:
              path: /
              port: 8317
            initialDelaySeconds: 10
            periodSeconds: 30
          readinessProbe:
            httpGet:
              path: /
              port: 8317
            initialDelaySeconds: 5
            periodSeconds: 10
      volumes:
        - name: config
          configMap:
            name: cliproxyapi-config
        - name: auths
          persistentVolumeClaim:
            claimName: cliproxyapi-auths-pvc
```

### Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: cliproxyapi
spec:
  selector:
    app: cliproxyapi
  ports:
    - port: 8317
      targetPort: 8317
  type: ClusterIP
```

### Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: cliproxyapi-secret
type: Opaque
stringData:
  password: your-password
```

---

## TLS 配置

### 生成自签名证书

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem \
  -days 365 -nodes -subj "/CN=localhost"
```

### 配置 TLS

```yaml
# config.yaml
tls:
  enable: true
  cert: /path/to/cert.pem
  key: /path/to/key.pem
```

### 使用 Let's Encrypt

建议使用反向代理（如 Nginx、Caddy）处理 TLS 终止。

---

## 反向代理配置

### Nginx

```nginx
server {
    listen 80;
    server_name cliproxy.example.com;

    # 重定向到 HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name cliproxy.example.com;

    ssl_certificate /etc/letsencrypt/live/cliproxy.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/cliproxy.example.com/privkey.pem;

    # SSL 配置
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers off;

    location / {
        proxy_pass http://127.0.0.1:8317;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket 支持
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # 流式响应
        proxy_buffering off;
        proxy_cache off;
    }
}
```

### Caddy

```
cliproxy.example.com {
    reverse_proxy localhost:8317
}
```

---

## PostgreSQL 后端部署

### Docker Compose with PostgreSQL

```yaml
version: "3.8"

services:
  postgres:
    image: postgres:15
    container_name: cliproxy-db
    restart: unless-stopped
    environment:
      POSTGRES_USER: cliproxy
      POSTGRES_PASSWORD: db-password
      POSTGRES_DB: cliproxy
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U cliproxy"]
      interval: 10s
      timeout: 5s
      retries: 5

  cliproxyapi:
    image: eceasy/cli-proxy-api:latest
    container_name: cliproxyapi
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
    ports:
      - "8317:8317"
    environment:
      - PGSTORE_DSN=postgres://cliproxy:db-password@postgres:5432/cliproxy?sslmode=disable
      - MANAGEMENT_PASSWORD=${MANAGEMENT_PASSWORD}
    volumes:
      - ./config.yaml:/CLIProxyAPI/config.yaml

volumes:
  postgres-data:
```

---

## 环境变量参考

| 变量名                   | 描述                  | 示例                                |
| ------------------------ | --------------------- | ----------------------------------- |
| `DEPLOY`                 | 部署模式              | `cloud`                             |
| `MANAGEMENT_PASSWORD`    | 管理密码              | `your-password`                     |
| `PGSTORE_DSN`            | PostgreSQL 连接字符串 | `postgres://user:pass@host:5432/db` |
| `PGSTORE_SCHEMA`         | PostgreSQL schema     | `cliproxy`                          |
| `PGSTORE_LOCAL_PATH`     | 本地镜像路径          | `/var/lib/cliproxy`                 |
| `GITSTORE_GIT_URL`       | Git 仓库 URL          | `https://github.com/user/repo.git`  |
| `GITSTORE_GIT_USERNAME`  | Git 用户名            | `username`                          |
| `GITSTORE_GIT_TOKEN`     | Git token             | `ghp_xxx`                           |
| `GITSTORE_LOCAL_PATH`    | Git 本地路径          | `/var/lib/gitstore`                 |
| `OBJECTSTORE_ENDPOINT`   | 对象存储端点          | `https://s3.example.com`            |
| `OBJECTSTORE_ACCESS_KEY` | 访问密钥              | `access-key`                        |
| `OBJECTSTORE_SECRET_KEY` | 秘密密钥              | `secret-key`                        |
| `OBJECTSTORE_BUCKET`     | 存储桶                | `cliproxy`                          |
| `OBJECTSTORE_LOCAL_PATH` | 本地镜像路径          | `/var/lib/objectstore`              |

---

## 监控和日志

### 健康检查

```bash
# 基本健康检查
curl http://localhost:8317/

# 管理健康检查
curl -H "Authorization: Bearer $PASSWORD" \
  http://localhost:8317/v0/management/debug
```

### 日志访问

```bash
# 获取日志
curl -H "Authorization: Bearer $PASSWORD" \
  http://localhost:8317/v0/management/logs

# 获取错误日志
curl -H "Authorization: Bearer $PASSWORD" \
  http://localhost:8317/v0/management/request-error-logs
```

### 使用统计

```bash
curl -H "Authorization: Bearer $PASSWORD" \
  http://localhost:8317/v0/management/usage
```

---

## 升级指南

### Docker 升级

```bash
# 拉取最新镜像
docker pull eceasy/cli-proxy-api:latest

# 停止并删除旧容器
docker stop cliproxyapi
docker rm cliproxyapi

# 启动新容器
docker run -d \
  --name cliproxyapi \
  -p 8317:8317 \
  -v $(pwd)/config.yaml:/CLIProxyAPI/config.yaml \
  -v $(pwd)/auths:/CLIProxyAPI/auths \
  eceasy/cli-proxy-api:latest
```

### 二进制升级

```bash
# 下载新版本
wget https://github.com/router-for-me/CLIProxyAPI/releases/download/vX.X.X/CLIProxyAPI-linux-amd64.tar.gz

# 解压
tar -xzf CLIProxyAPI-linux-amd64.tar.gz

# 停止服务
sudo systemctl stop cliproxyapi

# 替换二进制文件
sudo mv CLIProxyAPI /opt/cliproxyapi/

# 启动服务
sudo systemctl start cliproxyapi
```
