import axios from 'axios'
import { message } from 'antd'
import type { ApiResp } from './types'
import { useAuthStore } from '../store/auth'
import { signRequest, sortedQuery } from '../utils/sign'

export const http = axios.create({ baseURL: '/api/v1', timeout: 10000 })

http.interceptors.request.use(async (config) => {
  const { token, apiKey, apiSecret } = useAuthStore.getState()
  if (token && apiKey && apiSecret) {
    // 签名覆盖完整请求路径（含 baseURL，如 /api/v1/account/me）
    const path = (config.baseURL || '') + (config.url || '')
    const body = config.data ? (typeof config.data === 'string' ? config.data : JSON.stringify(config.data)) : ''
    const sorted = sortedQuery(config.params as Record<string, unknown> | undefined)
    const { timestamp, signature } = await signRequest({
      method: (config.method || 'get').toUpperCase(),
      path,
      query: sorted,
      body,
      apiSecret,
    })
    config.params = sorted // 发送顺序与签名保持一致（字典序）
    config.headers['X-API-KEY'] = apiKey
    config.headers['X-API-TIMESTAMP'] = timestamp
    config.headers['X-API-SIGNATURE'] = signature
  }
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

http.interceptors.response.use(
  (resp) => {
    const body = resp.data as ApiResp
    if (body.code !== 0) {
      if (body.code === 10401) {
        useAuthStore.getState().logout()
        if (!location.pathname.startsWith('/login')) {
          message.warning(body.msg || '请先登录')
          location.href = '/login'
        }
      } else {
        message.error(body.msg || '请求失败')
      }
      return Promise.reject(new Error(body.msg))
    }
    return resp
  },
  (err) => {
    const status = err.response?.status
    const msg = err.response?.data?.msg
    if (status === 429) {
      message.warning(msg || '请求过于频繁，请稍后再试')
    } else if (status === 401) {
      useAuthStore.getState().logout()
      if (!location.pathname.startsWith('/login')) {
        message.warning(msg || '请重新登录')
        location.href = '/login'
      }
    } else {
      message.error(msg || err.message || '网络异常，请稍后重试')
    }
    return Promise.reject(err)
  },
)
