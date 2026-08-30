-- RBAC：用户角色（user / admin）
ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'user';

-- 后台流水查询索引
CREATE INDEX idx_ledger_created ON ledger_entries (created_at DESC);
