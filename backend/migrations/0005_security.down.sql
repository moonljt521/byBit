DROP INDEX IF EXISTS idx_spot_orders_client;
ALTER TABLE spot_orders DROP COLUMN IF EXISTS client_order_id;
DROP TABLE IF EXISTS login_logs;
DROP TABLE IF EXISTS api_credentials;
