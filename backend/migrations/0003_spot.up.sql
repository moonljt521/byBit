-- 现货订单
CREATE TABLE spot_orders (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT         NOT NULL REFERENCES users (id),
    symbol       TEXT           NOT NULL, -- BTCUSDT
    side         TEXT           NOT NULL, -- buy / sell
    type         TEXT           NOT NULL, -- limit / market
    price        NUMERIC(38,18) NOT NULL DEFAULT 0, -- 市价单为 0
    amount       NUMERIC(38,18) NOT NULL,           -- base 币数量
    filled       NUMERIC(38,18) NOT NULL DEFAULT 0,
    avg_price    NUMERIC(38,18) NOT NULL DEFAULT 0,
    fee          NUMERIC(38,18) NOT NULL DEFAULT 0, -- 累计手续费（quote 计）
    frozen_quote NUMERIC(38,18) NOT NULL DEFAULT 0, -- 买单冻结的 USDT（剩余）
    status       TEXT           NOT NULL DEFAULT 'pending', -- pending/partial/filled/canceled
    created_at   TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ    NOT NULL DEFAULT now()
);
CREATE INDEX idx_spot_orders_user ON spot_orders (user_id, status);
CREATE INDEX idx_spot_orders_open ON spot_orders (status) WHERE status IN ('pending', 'partial');

-- 成交记录
CREATE TABLE trades (
    id           BIGSERIAL PRIMARY KEY,
    order_id     BIGINT         NOT NULL REFERENCES spot_orders (id),
    user_id      BIGINT         NOT NULL REFERENCES users (id),
    symbol       TEXT           NOT NULL,
    side         TEXT           NOT NULL,
    price        NUMERIC(38,18) NOT NULL,
    amount       NUMERIC(38,18) NOT NULL,
    quote_amount NUMERIC(38,18) NOT NULL,
    fee          NUMERIC(38,18) NOT NULL,
    created_at   TIMESTAMPTZ    NOT NULL DEFAULT now()
);
CREATE INDEX idx_trades_user ON trades (user_id, id DESC);
