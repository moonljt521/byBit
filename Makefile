SHELL := /bin/bash

.PHONY: help db-init services-start services-stop backend-run backend-build backend-test \
        frontend-install frontend-dev frontend-build admin-install admin-dev admin-build \
        mobile-analyze mobile-apk

help:
	@echo "db-init          创建本地数据库角色与库（仅首次）"
	@echo "services-start   启动本地 postgres/redis（brew services run）"
	@echo "services-stop    停止本地 postgres/redis"
	@echo "backend-run      运行 Go 后端（:8080）"
	@echo "backend-build    编译后端二进制"
	@echo "backend-test     运行后端单元测试"
	@echo "frontend-install 安装用户端网页依赖"
	@echo "frontend-dev     启动用户端网页（:5173）"
	@echo "frontend-build   构建用户端网页"
	@echo "admin-install    安装管理后台依赖"
	@echo "admin-dev        启动管理后台（:5174）"
	@echo "admin-build      构建管理后台"
	@echo "mobile-analyze   Flutter 客户端静态检查"
	@echo "mobile-apk       构建 Android APK（debug）"

services-start:
	brew services run postgresql; brew services run redis

services-stop:
	brew services stop postgresql; brew services stop redis

db-init:
	psql postgres -tAc "SELECT 1 FROM pg_roles WHERE rolname='cryptosim'" | grep -q 1 || psql postgres -c "CREATE ROLE cryptosim LOGIN PASSWORD 'cryptosim';"
	psql postgres -tAc "SELECT 1 FROM pg_database WHERE datname='cryptosim'" | grep -q 1 || createdb -O cryptosim cryptosim

backend-run:
	cd backend && go run ./cmd/server

backend-build:
	cd backend && go build -o bin/server ./cmd/server

backend-test:
	cd backend && go test ./... -race -count=1

frontend-install:
	cd frontend && npm install

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build

admin-install:
	cd admin && npm install

admin-dev:
	cd admin && npm run dev

admin-build:
	cd admin && npm run build

mobile-analyze:
	cd mobile && flutter analyze && flutter test

mobile-apk:
	cd mobile && flutter build apk --debug
