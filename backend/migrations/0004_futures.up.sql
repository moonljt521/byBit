-- 合约仓位（USDT 本位永续，逐仓）
CREATE TABLE futures_positions (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT         NOT NULL REFERENCES users (id),
    symbol        TEXT           NOT NULL,
    side          TEXT           NOT NULL, -- long / short
    leverage      INT            NOT NULL,
    size          NUMERIC(38,18) NOT NULL, -- 剩余仓位（base 币数量）
    entry_price   NUMERIC(38,18) NOT NULL,
    margin        NUMERIC(38,18) NOT NULL, -- 逐仓保证金（quote），随资金费率调整
    status        TEXT           NOT NULL DEFAULT 'open', -- open / closed / liquidated
    realized_pnl  NUMERIC(38,18) NOT NULL DEFAULT 0,
    fee           NUMERIC(38,18) NOT NULL DEFAULT 0,
    last_funding_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    opened_at     TIMESTAMPTZ    NOT NULL DEFAULT now(),
    closed_at     TIMESTAMPTZ,
    updated_at    TIMESTAMPTZ    NOT NULL DEFAULT now()
);
CREATE INDEX idx_futures_positions_user ON futures_positions (user_id, status);

-- 资金费率结算记录
CREATE TABLE funding_records (
    id          BIGSERIAL PRIMARY KEY,
    position_id BIGINT         NOT NULL REFERENCES futures_positions (id),
    user_id     BIGINT         NOT NULL,
    symbol      TEXT           NOT NULL,
    rate        NUMERIC(38,18) NOT NULL,
    amount      NUMERIC(38,18) NOT NULL, -- 正=收入（空头），负=支出（多头）
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT now()
);
CREATE INDEX idx_funding_user ON funding_records (user_id, id DESC);
