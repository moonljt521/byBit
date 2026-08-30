import { http } from './http'
import type { ApiResp, AuthData, MeData } from './types'

export async function register(email: string, username: string, password: string) {
  const { data } = await http.post<ApiResp<AuthData>>('/auth/register', {
    email,
    username,
    password,
  })
  return data.data
}

export async function login(account: string, password: string) {
  const { data } = await http.post<ApiResp<AuthData>>('/auth/login', { account, password })
  return data.data
}

export async function resetCredentials() {
  const { data } = await http.post<ApiResp<{ api_key: string; api_secret: string }>>(
    '/auth/credentials/reset',
  )
  return data.data
}

export async function me() {
  const { data } = await http.get<ApiResp<MeData>>('/account/me')
  return data.data
}

export async function resetAccount() {
  await http.post('/account/reset')
}
