-- 用户表
CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT        NOT NULL UNIQUE,
    username      TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    status        SMALLINT    NOT NULL DEFAULT 1, -- 1 正常 0 禁用
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 余额表：available 可用，frozen 冻结（挂单锁定）
CREATE TABLE balances (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    currency   TEXT           NOT NULL,
    available  NUMERIC(38,18) NOT NULL DEFAULT 0,
    frozen     NUMERIC(38,18) NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ    NOT NULL DEFAULT now(),
    UNIQUE (user_id, currency)
);
CREATE INDEX idx_balances_user ON balances (user_id);

-- 资金流水（复式记账简化版：单向流水 + 余额快照，M3 升级为严格复式借贷分录）
CREATE TABLE ledger_entries (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    biz_type      TEXT           NOT NULL, -- signup_grant / trade / fee / reset ...
    biz_id        TEXT           NOT NULL DEFAULT '',
    currency      TEXT           NOT NULL,
    amount        NUMERIC(38,18) NOT NULL, -- 正为入账，负为出账
    balance_after NUMERIC(38,18) NOT NULL,
    memo          TEXT           NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ    NOT NULL DEFAULT now()
);
CREATE INDEX idx_ledger_user_time ON ledger_entries (user_id, currency, id DESC);

-- 币种目录
CREATE TABLE coins (
    id           SERIAL PRIMARY KEY,
    symbol       TEXT        NOT NULL UNIQUE, -- BTC / ETH ...
    display_name TEXT        NOT NULL,
    sort         INT         NOT NULL DEFAULT 100,
    enabled      BOOLEAN     NOT NULL DEFAULT TRUE,
    meta         JSONB       NOT NULL DEFAULT '{}'::jsonb, -- M5 学习中心：简介/共识机制/总量等
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 交易对（现货与合约共用，pair_type 区分）
CREATE TABLE trading_pairs (
    id             SERIAL PRIMARY KEY,
    symbol         TEXT        NOT NULL, -- BTCUSDT
    base_currency  TEXT        NOT NULL,
    quote_currency TEXT        NOT NULL,
    pair_type      TEXT        NOT NULL DEFAULT 'spot', -- spot / futures
    enabled        BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (symbol, pair_type)
);
