import { http } from './http'
import type { ApiResp } from './types'

export interface Ticker {
  symbol: string
  last: string
  open24h: string
  high24h: string
  low24h: string
  vol24h: string
  change_pct: string
}

export interface Candle {
  ts: number
  o: string
  h: string
  l: string
  c: string
  vol: string
}

export type DepthLevel = [string, string]

export interface Depth {
  bids: DepthLevel[]
  asks: DepthLevel[]
}

export interface MarketTrade {
  ts: number
  price: string
  size: string
  side: 'buy' | 'sell'
}

export async function fetchTickers() {
  const { data } = await http.get<ApiResp<{ symbols: string[]; tickers: Ticker[] }>>('/market/tickers')
  return data.data
}

export async function fetchKlines(symbol: string, bar: string, limit = 200) {
  const { data } = await http.get<ApiResp<{ symbol: string; bar: string; candles: Candle[] }>>(
    '/market/klines',
    { params: { symbol, bar, limit } },
  )
  return data.data
}

export async function fetchDepth(symbol: string, size = 15) {
  const { data } = await http.get<ApiResp<Depth>>('/market/depth', { params: { symbol, size } })
  return data.data
}

export async function fetchTrades(symbol: string, limit = 20) {
  const { data } = await http.get<ApiResp<MarketTrade[]>>('/market/trades', { params: { symbol, limit } })
  return data.data
}
