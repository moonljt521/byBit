// WebSocket 实时行情客户端：自动重连 + 频道订阅。
// 服务端消息格式：{"channel":"tickers","data":[...],"ts":...}
//                {"channel":"private","data":{"type":"trade",...},"ts":...}

import type { Ticker } from './market'

type TickersHandler = (t: Ticker[]) => void
type TradeHandler = (e: Record<string, unknown>) => void

let ws: WebSocket | null = null
let retry = 0
let tickersHandler: TickersHandler | null = null
let tradeHandler: TradeHandler | null = null
let closed = false

function getToken(): string {
  try {
    const saved = JSON.parse(localStorage.getItem('cryptosim.auth') || 'null')
    return saved?.token || ''
  } catch {
    return ''
  }
}

/** 建立连接（幂等）。页面调用一次即可，断线自动重连。 */
export function connectMarket(h: { onTickers?: TickersHandler; onTrade?: TradeHandler }) {
  tickersHandler = h.onTickers ?? tickersHandler
  tradeHandler = h.onTrade ?? tradeHandler
  if (ws) return
  closed = false
  _connect()
}

function _connect() {
  if (closed) return
  const token = getToken()
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  // 开发环境直连后端（vite 代理 WS 会劫持 keep-alive 连接导致后续 HTTP 请求挂起）；
  // 生产环境走同源 nginx（/ws/ 反代）
  const wsHost =
    location.port === '5173' || location.port === '5174' ? 'localhost:8080' : location.host
  const channels = token ? 'tickers,private' : 'tickers'
  ws = new WebSocket(
    `${proto}://${wsHost}/api/v1/ws/market?channels=${channels}${token ? `&token=${encodeURIComponent(token)}` : ''}`,
  )
  ws.onopen = () => {
    retry = 0
  }
  ws.onmessage = (ev) => {
    try {
      const msg = JSON.parse(ev.data as string)
      if (msg.channel === 'tickers' && tickersHandler) tickersHandler(msg.data as Ticker[])
      if (msg.channel === 'private' && tradeHandler) tradeHandler(msg.data as Record<string, unknown>)
    } catch {
      /* 忽略非法帧 */
    }
  }
  ws.onclose = () => {
    ws = null
    if (closed) return
    retry = Math.min(retry + 1, 5)
    setTimeout(_connect, 1000 * 2 ** retry) // 指数退避重连
  }
  ws.onerror = () => ws?.close()
}

export function closeMarket() {
  closed = true
  ws?.close()
  ws = null
}
