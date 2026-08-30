// 后端统一响应封装 {"code":0,"msg":"ok","data":...}
export interface ApiResp<T = unknown> {
  code: number
  msg: string
  data: T
}

export interface User {
  id: number
  email: string
  username: string
  status: number
  created_at: string
}

export interface Balance {
  currency: string
  available: string
  frozen: string
}

export interface AuthData {
  token: string
  api_key: string
  api_secret: string
  user: User
}

export interface MeData {
  user: User
  balances: Balance[]
}
