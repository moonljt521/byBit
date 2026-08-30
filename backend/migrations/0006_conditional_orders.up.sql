-- 条件单（止损/止盈）与 Post-Only
-- trigger_price > 0 表示条件单：buy 在市场价 >= trigger 时触发，sell 在市场价 <= trigger 时触发，
-- 触发后按市价 ± 滑点成交。
ALTER TABLE spot_orders ADD COLUMN trigger_price NUMERIC(38,18) NOT NULL DEFAULT 0;
ALTER TABLE spot_orders ADD COLUMN post_only BOOLEAN NOT NULL DEFAULT FALSE;
