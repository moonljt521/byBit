import { http } from './http'
import type { ApiResp } from './types'

export interface FuturesPosition {
  id: number
  symbol: string
  side: 'long' | 'short'
  leverage: number
  size: string
  entry_price: string
  margin: string
  status: 'open' | 'closed' | 'liquidated'
  realized_pnl: string
  fee: string
  opened_at: string
  closed_at: string | null
  // 服务端实时计算
  mark_price: string
  unrealized_pnl: string
  roi: string
  liquidation_price: string
}

export interface FundingRecord {
  id: number
  position_id: number
  symbol: string
  rate: string
  amount: string
  created_at: string
}

export async function openPosition(input: {
  symbol: string
  side: 'long' | 'short'
  leverage: number
  amount: string
}) {
  const { data } = await http.post<ApiResp<FuturesPosition>>('/futures/positions', input)
  return data.data
}

export async function closePosition(id: number, amount?: string) {
  await http.post(`/futures/positions/${id}/close`, amount ? { amount } : {})
}

export async function positions() {
  const { data } = await http.get<ApiResp<FuturesPosition[]>>('/futures/positions')
  return data.data
}

export async function positionHistory() {
  const { data } = await http.get<ApiResp<FuturesPosition[]>>('/futures/positions/history')
  return data.data
}

export async function fundingRecords() {
  const { data } = await http.get<ApiResp<FundingRecord[]>>('/futures/funding')
  return data.data
}
