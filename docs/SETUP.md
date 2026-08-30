# 新机器从零部署指南

换电脑/重装系统后，照着本文从零把整个项目跑起来。预计 30-60 分钟（多数时间在下载工具链）。

---

## 一、需要安装的服务与工具

| 工具 | 版本要求 | 用途 | 必装 |
|---|---|---|---|
| PostgreSQL | 14+（推荐 16） | 业务数据库 | ✅ |
| Redis | 7+ | 行情缓存 / 推送 | ✅ |
| Go | 1.22+ | 后端编译运行 | ✅ |
| Node.js | 18+（推荐 22） | 两个 Web 前端 | ✅ |
| Flutter SDK | 3.24+ | 仅移动端需要 | 手机端要跑才装 |
| Android Studio / SDK | — | 仅 Android 构建需要 | 同上 |
| Xcode（完整版） | 15+ | 仅 iOS 构建需要（Mac） | 同上 |

## 二、安装服务

### macOS（Homebrew）

```bash
brew install postgresql@16 redis go nodejs

# 启动数据库服务（开机自启用 brew services start）
brew services start postgresql@16
brew services start redis

# 注意：若本机曾有旧版 PostgreSQL（如 14），brew 升级会破坏其 ICU 依赖，
# 建议直接用新版 PG16 数据目录，旧库数据如有价值先 pg_dump 迁移
```

Flutter（含 Android 工具链）：

```bash
brew install --cask flutter
# 或手动：从 https://docs.flutter.dev/get-started/install 下载，解压后加入 PATH
flutter doctor   # 按提示补齐 Android/iOS 工具链
```

### Linux（Ubuntu/Debian）

```bash
sudo apt update
sudo apt install -y postgresql redis-server golang nodejs npm

sudo systemctl enable --now postgresql redis-server
```

Flutter 安装：[官方文档](https://docs.flutter.dev/get-started/install/linux)

## 三、初始化数据库

```bash
# 创建角色与库（仅首次）
psql postgres -c "CREATE ROLE cryptosim LOGIN PASSWORD 'cryptosim';"
createdb -O cryptosim cryptosim
```

后端启动时会**自动执行 SQL 迁移建表 + 写入种子数据**，无需手动建表。

## 四、启动项目

```bash
git clone git@github.com:moonljt521/byBit.git
cd byBit

# 后端（:8080）。国内网络访问 OKX 行情需要代理：
cd backend
SIM_HTTP_PROXY=http://127.0.0.1:<你的代理端口> go run ./cmd/server
# 不带代理也能跑：行情自动降级为内置模拟行情

# 用户端网页（:5173）—— 新终端
cd frontend && npm install && npm run dev

# 管理后台（:5174）—— 新终端（可选）
cd admin && npm install && npm run dev
```

浏览器打开：

- 用户端 http://localhost:5173 （注册即送 10,000 虚拟 USDT）
- 管理后台 http://localhost:5174 （账号见下）
- 内置账号：`demo / demo12345`（用户）、`admin / admin12345`（管理员）

> ⚠️ 生产环境必须先设置 `SIM_JWT_SECRET` / `SIM_ENC_KEY`（`openssl rand -hex 32`），见 [deploy/README.md](../deploy/README.md)。

## 五、移动端（Flutter）

```bash
cd mobile
flutter pub get
flutter analyze

# 连接的 Android 手机（USB 调试已授权）
flutter run
# 或只构建 APK
flutter build apk --debug   # 产物 build/app/outputs/flutter-apk/app-debug.apk
```

**手机连电脑后端**：App 登录页右上角服务器图标 →
- 「自动搜索局域网服务器」（推荐，自动找到电脑 IP）
- 或手填 `http://<电脑局域网IP>:8080/api/v1`

**建议**：在路由器给电脑绑定固定 IP（DHCP 静态分配），一劳永逸。IP 变了也不怕——App 请求失败时会自动扫描局域网重连。

## 六、生产部署

见 [deploy/README.md](../deploy/README.md)：TLS 证书、密钥清单、docker-compose 生产编排、安全基线自查表。

## 七、测试与验证

```bash
# 后端单测（含撮合引擎 8 例 + 合约引擎 8 例）
cd backend && go test ./... -race

# 前端构建
cd frontend && npm run build
cd admin && npm run build

# Flutter
cd mobile && flutter analyze && flutter test

# E2E（需后端 + 用户端 + 管理后台都在运行）
cd e2e && npm install && npx playwright install chromium && npx playwright test
```

## 八、常见问题

| 问题 | 处理 |
|---|---|
| 行情不动 / 行情源不可用 | 国内网络需给后端设 `SIM_HTTP_PROXY`；仍不行会自动降级为模拟行情 |
| 手机 App 连不上电脑 | 确认同 Wi-Fi、后端在跑；用登录页「自动搜索」；路由器绑定电脑固定 IP |
| 端口冲突 | 后端 8080 / 用户端 5173 / 管理后台 5174，配置项：`SIM_HTTP_ADDR`、两个 `vite.config.ts` |
| 登录报 401「缺少签名头」 | 客户端登录态是安全改造前的旧数据，重新登录即可 |
| 数据库想清空重来 | `psql postgresql://cryptosim:cryptosim@127.0.0.1:5432/cryptosim -c "TRUNCATE trades, spot_orders, funding_records, futures_positions, ledger_entries, balances, users RESTART IDENTITY CASCADE;"` |

## 九、项目结构

```
byBit/
├── backend/    Go 后端（Gin + GORM + PostgreSQL + Redis）
├── frontend/   用户端网页（React + TS + AntD + klinecharts）
├── admin/      管理后台（React + TS + AntD）
├── mobile/     Flutter 客户端（Android / iOS）
├── content/    学习中心内容（Markdown）
├── deploy/     生产部署（TLS nginx + compose + 安全基线）
├── docs/       使用说明 / 部署指南 / OpenAPI
└── e2e/        Playwright 端到端测试
```
