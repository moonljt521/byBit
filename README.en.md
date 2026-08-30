# CryptoSim — Virtual Crypto Exchange (Paper Trading)

[中文](README.md) | **English**

A paper-trading crypto exchange that **mirrors real exchange workflows with 100% virtual funds** (modeled after Bybit): real market data + simulated matching engine / derivatives risk engine + learning center + admin console + mobile clients.

> ⚠️ For learning and research only: all funds are virtual. There are no real deposit, withdrawal, or transfer channels, and nothing here constitutes investment advice.

## Monorepo Layout

| Directory | Description | Stack |
|---|---|---|
| `backend/` | Go backend: auth, market data, spot matching, derivatives risk engine, ledger, learning center, admin APIs | Go + Gin + GORM + PostgreSQL + Redis |
| `frontend/` | Web client: markets, spot/futures trading, assets, learning center | React 18 + TS + AntD + klinecharts |
| `admin/` | Admin console: dashboards, user management, fund adjustments, audit trails | React 18 + TS + AntD |
| `mobile/` | Mobile client (Android/iOS) | Flutter |
| `content/` | Learning center content (wiki/tutorials/glossary in Markdown) | Markdown |
| `docs/` | User guide / setup guide / OpenAPI | — |
| `deploy/` | Production deployment (TLS nginx + compose + security baseline) | — |
| `e2e/` | Playwright end-to-end tests | — |

## Features

- **Market data**: 3-level fallback (OKX → Binance → built-in simulator); candlesticks (1m–1d), 10-level order book, live trades; Redis caching + upstream circuit breaker
- **Spot**: limit/market orders, matching engine (price-time priority, partial fills, dust sweep), stop/trigger orders, Post-Only, fund freeze/unfreeze, maker/taker fees, 5 USDT min notional
- **Futures**: USDT-margined perpetuals (isolated), 1–20x leverage, long/short, mark price, funding rate settled every 8h (longs pay shorts 0.01%), liquidation engine at 0.5% maintenance margin
- **Accounts**: JWT auth, 10,000 virtual USDT on signup, double-entry ledger (every cent traceable), one-click reset
- **Learning center**: 8 coin wikis, 12 beginner tutorials (incl. scam awareness & learning roadmap), 46-term glossary
- **Admin console**: RBAC (role=admin), user search/disable/fund adjustment, global audit trail, login audit, ops dashboard
- **Mobile**: Flutter client (dio / go_router / Riverpod / encrypted storage), 5 tabs (markets/spot/futures/assets/learn), custom-painted candlesticks, LAN auto-discovery

## Fresh Machine Setup

Full provisioning guide for a new computer (service list, install commands, DB bootstrap, key generation): see **[docs/SETUP.md](docs/SETUP.md)** (Chinese).

## Quick Start (Local Dev)

Prerequisites: Go 1.22+, Node 18+, PostgreSQL, Redis, Flutter (mobile only).

```bash
# 1. Start local PG / Redis
make services-start

# 2. Bootstrap database (role + db, first time only)
make db-init

# 3. Start backend (:8080, auto-migrates schema; set proxy for OKX access in CN networks)
cd backend && SIM_HTTP_PROXY=http://127.0.0.1:26002 go run ./cmd/server

# 4. Web client (:5173)
make frontend-install && make frontend-dev

# 5. Admin console (:5174, optional)
make admin-install && make admin-dev
```

Built-in accounts: `demo / demo12345` (user), `admin / admin12345` (admin).

Full user manual (Chinese) is in **[docs/使用说明.md](docs/使用说明.md)**.

## Mobile

```bash
cd mobile
flutter pub get
flutter analyze             # static analysis
flutter build apk --debug   # Android APK
flutter run                 # run on emulator/device
```

Configure the server address in the app ("Me" tab, or the server icon on the login page). The login page supports **one-tap LAN server discovery**, and the app auto-reconnects when the host IP changes.

## Docker Deployment (Optional)

```bash
docker compose up -d --build   # postgres + redis + backend(:8080) + frontend(:3000) + admin(:3001)
```

Production (TLS + security baseline): see [deploy/README.md](deploy/README.md).

## Why This Stack (Benchmarked Against Real Exchanges)

- Real exchange backends: **Bybit = Go**, Binance = Java + C++ matching core; all major web frontends are React
- Public chain clients: Bitcoin Core = C++, Ethereum Geth = Go, Tron java-tron = Java
- This project implements exchange-equivalent matching/accounting/risk logic in Go — same architecture, same learning value

## Testing & Quality

- Backend unit tests: matching engine (10) + futures engine (liquidation/funding/close, 8) + auth/JWT
- Playwright E2E: 4 web-client cases + 3 admin cases
- Performance benchmarks: end-to-end order placement ≈ 8,700 orders/sec, ledger write path ≈ 9,000 ops/sec (in-memory DB)
- API docs: [docs/openapi.yaml](docs/openapi.yaml)

## Roadmap (Completed)

- [x] M1 Skeleton / auth / accounts / double-entry ledger
- [x] M2 Market data (real feeds + simulator fallback, candlesticks, order book)
- [x] M3 Spot trading (matching engine, partial fills, fees)
- [x] M4 Futures (isolated margin, funding rate, liquidation engine)
- [x] M5 Learning center (wiki / glossary / tutorials)
- [x] Admin console (RBAC / dashboard / user management / audit trails / login audit)
- [x] Mobile client (Flutter: dio/go_router/Riverpod/encrypted storage, Android & iOS)
- [x] Security hardening (HMAC request signing / AES-GCM encryption / rate limiting / anti-replay / idempotency / engine unit tests)
- [x] WebSocket real-time push (web + Flutter: market broadcast + private trade notifications, auto-reconnect)
- [x] Trigger orders (stop-loss/take-profit) & Post-Only (spot)
- [x] Playwright E2E (4 web + 3 admin cases)
- [x] Prometheus /metrics (request counts/latency)
- [x] OpenAPI docs (docs/openapi.yaml) & i18n (web + Flutter)
- [x] CI (GitHub Actions: backend tests / web builds / Flutter / E2E)
- [x] Performance benchmarks (≈8,700 orders/sec placement, ≈9,000 ledger ops/sec)
- [x] Production deployment templates (deploy/: TLS nginx + docker-compose.prod + security baseline)
