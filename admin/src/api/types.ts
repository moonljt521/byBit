export interface ApiResp<T = unknown> {
  code: number
  msg: string
  data: T
}

export interface AdminUser {
  id: number
  email: string
  username: string
  role: string
  status: number
  usdt_available: string
  created_at: string
}

export interface Stats {
  total_users: number
  new_users_today: number
  active_users: number
  usdt_available: string
  usdt_frozen: string
  ledger_today: number
}

export interface LedgerRow {
  id: number
  user_id: number
  username: string
  biz_type: string
  biz_id: string
  currency: string
  amount: string
  balance_after: string
  memo: string
  created_at: string
}

export interface PageData<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export interface LoginLog {
  id: number
  user_id: number | null
  username: string
  success: boolean
  reason: string
  ip: string
  user_agent: string
  created_at: string
}
