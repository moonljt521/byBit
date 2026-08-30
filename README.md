# CryptoSim — 虚拟加密货币交易所（模拟盘）

一个**流程与真实交易所一致、资金全部虚拟**的模拟加密货币交易所（对标 Bybit 的产品形态）：真实行情 + 仿真撮合/合约风控 + 学习中心 + 管理后台 + 移动客户端。

> ⚠️ 本项目仅用于学习研究：所有资金均为虚拟资金，无任何真实充值、提币、转账通道，不构成任何投资建议。

## 项目组成（monorepo）

| 目录 | 说明 | 技术 |
|---|---|---|
| `backend/` | Go 后端：认证、行情、现货撮合、合约风控、账本、学习中心、管理接口 | Go + Gin + GORM + PostgreSQL + Redis |
| `frontend/` | 用户端网页：行情、现货/合约交易、资产、学习中心 | React 18 + TS + AntD + klinecharts |
| `admin/` | 管理后台：仪表盘、用户管理、资金调拨、审计流水 | React 18 + TS + AntD |
| `mobile/` | 移动客户端（Android/iOS） | Flutter |
| `content/` | 学习中心内容（百科/教程/词典 Markdown） | Markdown |
| `docs/` | 使用说明 | — |

## 功能总览

- **行情**：OKX → Binance → 内置模拟三级降级；K 线（1m~1d）、十档盘口、实时成交；Redis 缓存 + 上游熔断
- **现货**：限价单/市价单、撮合引擎（价格-时间优先、分批成交、尾单清扫）、资金冻结/解冻、maker/taker 手续费、最小下单金额 5 USDT
- **合约**：USDT 本位永续（逐仓）、1-20x 杠杆、多空双向、标记价格、每 8h 资金费率结算（多头付空头 0.01%）、维持保证金率 0.5% 触发的强平引擎
- **账户**：JWT 认证、注册送 10,000 虚拟 USDT、复式记账流水（每笔钱可追溯）、一键重置
- **学习中心**：8 篇币种百科、12 篇新手教程（含防骗与学习路线图）、46 条术语词典
- **管理后台**：RBAC（role=admin）、用户搜索/禁用/调资金、全局审计流水、运营看板
- **移动端**：Flutter 客户端，行情/交易/资产/我的四个 Tab，自绘蜡烛图

## 新机器从零部署

换电脑/重装系统后的完整装机手册（服务清单、安装命令、数据库初始化、密钥生成）：见 **[docs/SETUP.md](docs/SETUP.md)**。

## 本地开发快速开始

前置依赖：Go 1.22+、Node 18+、PostgreSQL、Redis、Flutter（仅移动端需要）。

```bash
# 1. 启动本地 PG / Redis
make services-start

# 2. 初始化数据库（建角色 + 建库，仅首次）
make db-init

# 3. 启动后端（:8080，自动迁移建表；国内网络请带代理变量访问 OKX）
cd backend && SIM_HTTP_PROXY=http://127.0.0.1:26002 go run ./cmd/server

# 4. 用户端网页（:5173）
make frontend-install && make frontend-dev

# 5. 管理后台（:5174，可选）
make admin-install && make admin-dev
```

内置账号：`demo / demo12345`（用户）、`admin / admin12345`（管理员）。

完整使用说明（小白向）见 **[docs/使用说明.md](docs/使用说明.md)**。

## 移动端

```bash
cd mobile
flutter pub get
flutter analyze          # 静态检查
flutter build apk --debug   # Android APK
flutter run              # 连接模拟器/真机运行
```

App 内「我的」页可配置服务器地址：安卓模拟器填 `http://10.0.2.2:8080/api/v1`，真机填电脑局域网 IP。

## Docker 部署（可选）

```bash
docker compose up -d --build   # postgres + redis + backend(:8080) + frontend(:3000) + admin(:3001)
```

## 技术选型依据（对标真实交易所）

- 真实交易所后端主力语言：**Bybit = Go**、币安 = Java + C++ 撮合；前端均为 React
- 公链客户端：Bitcoin Core = C++、以太坊 Geth = Go、波场 java-tron = Java
- 本项目用 Go 实现与真实交易所同构的撮合/账务/风控逻辑，学习价值等价

## 路线图（已完成）

- [x] M1 工程骨架 / 注册登录 / 账户体系 / 复式记账
- [x] M2 行情模块（真实行情 + 模拟兜底、K 线、盘口）
- [x] M3 现货交易（撮合引擎、分批成交、手续费）
- [x] M4 合约交易（逐仓保证金、资金费率、强平引擎）
- [x] M5 学习中心（百科 / 词典 / 教程）
- [x] 管理后台（RBAC / 看板 / 用户管理 / 审计流水 / 登录审计）
- [x] 移动客户端（Flutter：dio/go_router/Riverpod/加密存储，Android / iOS）
- [x] 安全加固（HMAC 验签 / AES-GCM 加密 / 限流 / 防重放 / 幂等 / 引擎单测）
- [x] WebSocket 实时推送（Web + Flutter，行情广播 + 私有成交通知，断线重连）
- [x] 条件单（止损/止盈）与 Post-Only（现货）
- [x] Playwright E2E 自动化（用户端 4 + 管理后台 3，共 7 用例）
- [x] Prometheus /metrics 监控（请求计数/时延）
- [x] OpenAPI 文档（docs/openapi.yaml）与中英双语（用户端 + Flutter）
- [x] CI（GitHub Actions：后端测试 / 双前端构建 / Flutter / E2E）
- [x] 性能压测（下单全链路 ≈8,700 单/秒，账本写路径 ≈9,000 ops/秒）
- [x] 生产部署模板（deploy/：TLS nginx + docker-compose.prod + 安全基线）
