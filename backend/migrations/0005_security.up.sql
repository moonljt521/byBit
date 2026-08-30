-- API 凭证：HMAC 验签密钥对（secret 加密落库，明文仅签发时返回一次）
CREATE TABLE api_credentials (
    user_id          BIGINT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    api_key          TEXT        NOT NULL UNIQUE,
    secret_encrypted TEXT        NOT NULL, -- AES-256-GCM 密文
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 登录审计日志
CREATE TABLE login_logs (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT REFERENCES users (id) ON DELETE SET NULL,
    username   TEXT        NOT NULL DEFAULT '',
    success    BOOLEAN     NOT NULL,
    reason     TEXT        NOT NULL DEFAULT '',
    ip         TEXT        NOT NULL DEFAULT '',
    user_agent TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_login_logs_time ON login_logs (created_at DESC);
CREATE INDEX idx_login_logs_user ON login_logs (user_id, id DESC);

-- 下单幂等：客户端订单号（同用户内唯一）
ALTER TABLE spot_orders ADD COLUMN client_order_id TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX idx_spot_orders_client ON spot_orders (user_id, client_order_id) WHERE client_order_id <> '';
