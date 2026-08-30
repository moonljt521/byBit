import { http } from './http'
import type { AdminUser, ApiResp, LedgerRow, LoginLog, PageData, Stats } from './types'

export interface LoginData {
  token: string
  api_key: string
  api_secret: string
  user: { id: number; username: string; role: string }
}

export async function login(account: string, password: string) {
  const { data } = await http.post<ApiResp<LoginData>>('/auth/login', { account, password })
  return data.data
}

export async function loginLogs(page: number, pageSize: number) {
  const { data } = await http.get<ApiResp<PageData<LoginLog>>>('/admin/login-logs', {
    params: { page, page_size: pageSize },
  })
  return data.data
}

export async function stats() {
  const { data } = await http.get<ApiResp<Stats>>('/admin/stats')
  return data.data
}

export async function users(page: number, pageSize: number, keyword = '') {
  const { data } = await http.get<ApiResp<PageData<AdminUser>>>('/admin/users', {
    params: { page, page_size: pageSize, keyword: keyword || undefined },
  })
  return data.data
}

export async function setStatus(id: number, status: number) {
  await http.patch(`/admin/users/${id}/status`, { status })
}

export async function adjustFunds(id: number, amount: string, memo: string) {
  await http.post(`/admin/users/${id}/grant`, { amount, memo })
}

export async function ledger(page: number, pageSize: number, userId?: number) {
  const { data } = await http.get<ApiResp<PageData<LedgerRow>>>('/admin/ledger', {
    params: { page, page_size: pageSize, user_id: userId || undefined },
  })
  return data.data
}
