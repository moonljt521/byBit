import { http } from './http'
import type { ApiResp } from './types'

export interface SpotOrder {
  id: number
  symbol: string
  side: 'buy' | 'sell'
  type: 'limit' | 'market'
  price: string
  amount: string
  filled: string
  avg_price: string
  fee: string
  status: 'pending' | 'partial' | 'filled' | 'canceled'
  created_at: string
}

export interface SpotTrade {
  id: number
  order_id: number
  symbol: string
  side: 'buy' | 'sell'
  price: string
  amount: string
  quote_amount: string
  fee: string
  created_at: string
}

export async function placeOrder(input: {
  symbol: string
  side: 'buy' | 'sell'
  type: 'limit' | 'market'
  price?: string
  amount: string
  triggerPrice?: string
  postOnly?: boolean
}) {
  const { data } = await http.post<ApiResp<SpotOrder>>('/spot/orders', {
    ...input,
    trigger_price: input.triggerPrice,
    post_only: input.postOnly,
  })
  return data.data
}

export async function cancelOrder(id: number) {
  await http.delete(`/spot/orders/${id}`)
}

export async function openOrders() {
  const { data } = await http.get<ApiResp<SpotOrder[]>>('/spot/orders/open')
  return data.data
}

export async function orderHistory() {
  const { data } = await http.get<ApiResp<SpotOrder[]>>('/spot/orders/history')
  return data.data
}

export async function myTrades() {
  const { data } = await http.get<ApiResp<SpotTrade[]>>('/spot/trades')
  return data.data
}
