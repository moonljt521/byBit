import axios from 'axios'
import { message } from 'antd'
import type { ApiResp } from './types'
import { useAdminStore } from '../store/auth'
import { signRequest, sortedQuery } from '../utils/sign'

export const http = axios.create({ baseURL: '/api/v1', timeout: 10000 })

http.interceptors.request.use(async (config) => {
  const { token, apiKey, apiSecret } = useAdminStore.getState()
  if (token && apiKey && apiSecret) {
    const path = (config.baseURL || '') + (config.url || '')
    const body = config.data ? JSON.stringify(config.data) : ''
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
        useAdminStore.getState().logout()
        if (!location.pathname.startsWith('/login')) location.href = '/login'
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
      useAdminStore.getState().logout()
      if (!location.pathname.startsWith('/login')) location.href = '/login'
    } else if (status === 403) {
      message.error(msg || '需要管理员权限')
    } else {
      message.error(msg || err.message || '网络异常')
    }
    return Promise.reject(err)
  },
)
