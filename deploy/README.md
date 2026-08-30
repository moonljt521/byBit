# 生产部署与安全基线

## 1. 证书

公网域名（推荐 Let's Encrypt）：

```bash
sudo certbot certonly --standalone -d your.domain.com
cp /etc/letsencrypt/live/your.domain.com/{fullchain.pem,privkey.pem} deploy/certs/
```

内网/本机测试（自签）：

```bash
openssl req -x509 -newkey rsa:2048 -nodes -days 365 \
  -keyout deploy/certs/privkey.pem -out deploy/certs/fullchain.pem \
  -subj "/CN=localhost"
```

## 2. 密钥清单（启动前必须全部替换，严禁使用代码默认值）

| 环境变量 | 用途 | 生成方式 |
|---|---|---|
| `SIM_JWT_SECRET` | JWT 签名 | `openssl rand -hex 32` |
| `SIM_ENC_KEY` | API Secret 落库加密（AES-256-GCM 主密钥） | `openssl rand -hex 32` |
| `POSTGRES_PASSWORD` | 数据库密码 | `openssl rand -hex 16` |
| `SIM_ALLOWED_ORIGINS` | CORS/WS 白名单 | `https://your.domain.com` |

## 3. 启动

```bash
cd deploy
cat > .env <<EOF
POSTGRES_PASSWORD=<...>
SIM_JWT_SECRET=<...>
SIM_ENC_KEY=<...>
SIM_ALLOWED_ORIGINS=https://your.domain.com
EOF
docker compose -f docker-compose.prod.yml up -d --build
```

对外只暴露 80/443（nginx TLS 终止）；postgres/redis/backend 仅内网互通。

## 4. 安全基线清单

已内置：

- [x] 密码 bcrypt 存储
- [x] JWT 身份 + HMAC-SHA256 请求验签（防篡改/防重放，±300s 时间窗）
- [x] API Secret AES-256-GCM 加密落库，明文仅签发时返回一次，支持轮换
- [x] 登录/注册按 IP 限流（默认 20 次/分钟，`SIM_AUTH_RATE_LIMIT` 可调）+ 全局限流
- [x] 登录审计日志（成功/失败、IP、UA）
- [x] 下单幂等（client_order_id）
- [x] 安全响应头（nosniff / DENY / Referrer-Policy；HTTPS 下 nginx 追加 HSTS/CSP）
- [x] 资金操作复式记账全量流水
- [x] TLS 1.2+/1.3、HTTP→HTTPS 跳转、HSTS

上线前自查：

- [ ] 替换全部默认密钥（第 2 节）
- [ ] 数据库定期备份：`pg_dump cryptosim > backup.sql`（建议 cron 每日）
- [ ] 如需公网开放，建议加 WAF/CDN 并开启 nginx 访问日志
- [ ] 教育用途声明保留在页面与 README（本平台不含任何真实资金通道）

## 5. 滚动升级

```bash
cd deploy && docker compose -f docker-compose.prod.yml build backend
docker compose -f docker-compose.prod.yml up -d backend   # 后端启动时自动执行 SQL 迁移
```
