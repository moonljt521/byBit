import { useCallback, useEffect, useState } from 'react'
import {
  Button,
  Card,
  Col,
  Empty,
  Input,
  message,
  Modal,
  Row,
  Select,
  Table,
  Tabs,
  Typography,
} from 'antd'
import { useSearchParams } from 'react-router-dom'
import CandleChart from '../components/CandleChart'
import { C } from '../theme'
import { changeColor } from './Markets'
import { fetchDepth, fetchTickers, fetchTrades } from '../api/market'
import { connectMarket } from '../api/ws'
import type { Balance } from '../api/types'
import type { Depth, MarketTrade, Ticker } from '../api/market'
import { me } from '../api/auth'
import { cancelOrder, myTrades, openOrders, orderHistory, placeOrder } from '../api/spot'
import type { SpotOrder, SpotTrade } from '../api/spot'

const UP = '#0ecb81'
const DOWN = '#f6465d'

const SYMBOLS = ['BTCUSDT', 'ETHUSDT', 'BNBUSDT', 'SOLUSDT', 'XRPUSDT', 'TRXUSDT', 'DOGEUSDT']

const sideText = (s: string) => (s === 'buy' ? '买入' : '卖出')
const sideColor = (s: string) => (s === 'buy' ? UP : DOWN)
const statusText: Record<string, string> = {
  pending: '等待成交',
  partial: '部分成交',
  filled: '已成交',
  canceled: '已撤销',
}

function OrderBook({ depth }: { depth: Depth | null }) {
  const asks = [...(depth?.asks ?? [])].slice(0, 10).reverse()
  const bids = (depth?.bids ?? []).slice(0, 10)
  const maxSize = Math.max(
    1e-12,
    ...(depth?.bids ?? []).slice(0, 10).map(([, s]) => Number(s)),
    ...(depth?.asks ?? []).slice(0, 10).map(([, s]) => Number(s)),
  )
  const row = ([p, s]: [string, string], color: string, key: string, side: 'bid' | 'ask') => {
    const ratio = Math.min(100, (Number(s) / maxSize) * 100)
    return (
      <div
        key={key}
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          padding: '1px 4px',
          position: 'relative',
          borderRadius: 2,
        }}
      >
        <div
          style={{
            position: 'absolute',
            left: 0,
            top: 0,
            bottom: 0,
            width: `${ratio}%`,
            background: side === 'bid' ? 'rgba(14,203,129,0.12)' : 'rgba(246,70,93,0.12)',
          }}
        />
        <span style={{ color, position: 'relative' }}>{p}</span>
        <span style={{ color: C.text2, position: 'relative' }}>{Number(s).toFixed(4)}</span>
      </div>
    )
  }
  return (
    <div style={{ fontSize: 12 }}>
      <Typography.Text type="secondary">卖盘</Typography.Text>
      <div>{asks.map(([p, s], i) => row([p, s], DOWN, `a${i}`, 'ask'))}</div>
      <div style={{ borderTop: `1px solid ${C.border}`, margin: '6px 0' }} />
      <Typography.Text type="secondary">买盘</Typography.Text>
      <div>{bids.map(([p, s], i) => row([p, s], UP, `b${i}`, 'bid'))}</div>
    </div>
  )
}

function TradeList({ trades }: { trades: MarketTrade[] }) {
  return (
    <div style={{ fontSize: 12, maxHeight: 260, overflowY: 'auto' }}>
      {trades.map((t, i) => (
        <div key={`${t.ts}-${i}`} style={{ display: 'flex', justifyContent: 'space-between', padding: '1px 0' }}>
          <span style={{ color: t.side === 'buy' ? UP : DOWN }}>{t.price}</span>
          <span style={{ color: C.text2 }}>{Number(t.size).toFixed(5)}</span>
        </div>
      ))}
    </div>
  )
}

interface OrderFormProps {
  symbol: string
  last: string
  baseAvailable: string
  usdtAvailable: string
  onDone: () => void
}

function OrderForm({ symbol, last, baseAvailable, usdtAvailable, onDone }: OrderFormProps) {
  const [type, setType] = useState<'limit' | 'market' | 'trigger'>('limit')
  const [side, setSide] = useState<'buy' | 'sell'>('buy')
  const [price, setPrice] = useState('')
  const [qty, setQty] = useState('')
  const [trigger, setTrigger] = useState('')
  const [postOnly, setPostOnly] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const base = symbol.replace(/USDT$/, '')

  async function submit() {
    if (!qty || parseFloat(qty) <= 0) {
      message.warning('请输入数量')
      return
    }
    if (type === 'limit' && (!price || parseFloat(price) <= 0)) {
      message.warning('请输入价格')
      return
    }
    if (type === 'trigger' && (!trigger || parseFloat(trigger) <= 0)) {
      message.warning('请输入触发价')
      return
    }
    setSubmitting(true)
    try {
      const order = await placeOrder({
        symbol,
        side,
        type: type === 'trigger' ? 'limit' : type,
        price: type === 'limit' ? price : undefined,
        amount: qty,
        triggerPrice: type === 'trigger' ? trigger : undefined,
        postOnly: type === 'limit' ? postOnly : false,
      })
      if (type === 'market') {
        message.success(`市价${sideText(side)}已成交，均价 ${order.avg_price}`)
      } else if (type === 'trigger') {
        message.success(`条件单已提交：市场价${side === 'buy' ? '涨破' : '跌破'} ${trigger} 时按市价${sideText(side)}`)
      } else {
        message.success(postOnly ? 'Post-Only 限价单已提交' : '限价单已提交，撮合引擎将按市场价自动撮合')
      }
      setQty('')
      onDone()
    } catch {
      /* 拦截器已提示 */
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Card size="small" title="下单" style={{ height: '100%' }}>
      <Tabs
        activeKey={type}
        onChange={(k) => setType(k as 'limit' | 'market' | 'trigger')}
        items={[
          { key: 'limit', label: '限价单' },
          { key: 'market', label: '市价单' },
          { key: 'trigger', label: '条件单' },
        ]}
        size="small"
      />
      <div style={{ display: 'flex', gap: 8, marginBottom: 8 }}>
        <Button
          style={{
            flex: 1,
            background: side === 'buy' ? UP : undefined,
            borderColor: UP,
            color: side === 'buy' ? '#fff' : UP,
          }}
          onClick={() => setSide('buy')}
        >
          买入
        </Button>
        <Button
          style={{
            flex: 1,
            background: side === 'sell' ? DOWN : undefined,
            borderColor: DOWN,
            color: side === 'sell' ? '#fff' : DOWN,
          }}
          onClick={() => setSide('sell')}
        >
          卖出
        </Button>
      </div>
      <div style={{ marginBottom: 8, fontSize: 12, color: C.text2 }}>
        可用：{side === 'buy' ? `${usdtAvailable} USDT` : `${baseAvailable} ${base}`}
      </div>
      {type === 'limit' && (
        <>
          <Input
            placeholder={`价格（最新 ${last || '-'}）`}
            value={price}
            onChange={(e) => setPrice(e.target.value)}
            style={{ marginBottom: 8 }}
          />
          <label style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8, fontSize: 12, color: '#666' }}>
            <input type="checkbox" checked={postOnly} onChange={(e) => setPostOnly(e.target.checked)} />
            Post-Only（只挂单不吃单，会立即成交则拒绝）
          </label>
        </>
      )}
      {type === 'trigger' && (
        <Input
          placeholder={`触发价（最新 ${last || '-'}）：买入涨破触发，卖出跌破触发`}
          value={trigger}
          onChange={(e) => setTrigger(e.target.value)}
          style={{ marginBottom: 8 }}
        />
      )}
      <Input
        placeholder={`数量（${base}）`}
        value={qty}
        onChange={(e) => setQty(e.target.value)}
        style={{ marginBottom: 6 }}
      />
      <div style={{ display: 'flex', gap: 6, marginBottom: 8 }}>
        {[25, 50, 75, 100].map((pct) => (
          <Button
            key={pct}
            size="small"
            style={{ flex: 1, fontSize: 12 }}
            onClick={() => {
              const px = parseFloat(type === 'limit' ? price || last : last) || 0
              if (!px) return
              const avail = side === 'buy' ? parseFloat(usdtAvailable) || 0 : parseFloat(baseAvailable) || 0
              const raw =
                side === 'buy'
                  ? (avail * (pct / 100)) / (px * 1.001) // 扣手续费预留
                  : avail * (pct / 100)
              setQty(raw.toFixed(6).replace(/0+$/, '').replace(/\.$/, ''))
            }}
          >
            {pct}%
          </Button>
        ))}
      </div>
      <div style={{ fontSize: 12, color: C.text2, marginBottom: 12 }}>
        预估金额：
        {(parseFloat(type === 'limit' ? price || '0' : last || '0') * (parseFloat(qty) || 0)).toFixed(2)} USDT
        （最小 5 USDT，taker 费率 0.1%）
      </div>
      <Button
        block
        size="large"
        loading={submitting}
        style={{ background: side === 'buy' ? UP : DOWN, borderColor: side === 'buy' ? UP : DOWN, color: '#fff' }}
        onClick={() => void submit()}
      >
        {sideText(side)} {base}
      </Button>
      <div style={{ fontSize: 12, color: C.text2, marginTop: 12 }}>
        限价单：市场价触及委托价时自动分批成交；市价单：按最新价 ± 0.05% 滑点立即成交
      </div>
    </Card>
  )
}

function OrdersTable({
  orders,
  onCancel,
  loading,
}: {
  orders: SpotOrder[]
  onCancel?: (id: number) => void
  loading?: boolean
}) {
  return (
    <Table
      rowKey="id"
      size="small"
      loading={loading}
      locale={{ emptyText: <Empty description="暂无记录" image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
      pagination={false}
      dataSource={orders}
      columns={[
        { title: '时间', dataIndex: 'created_at', render: (v: string) => new Date(v).toLocaleTimeString('zh-CN') },
        {
          title: '交易对',
          dataIndex: 'symbol',
          render: (v: string) => v.replace(/USDT$/, '/USDT'),
        },
        {
          title: '方向',
          dataIndex: 'side',
          render: (v: string) => <span style={{ color: sideColor(v) }}>{sideText(v)}</span>,
        },
        { title: '类型', dataIndex: 'type', render: (v: string) => (v === 'limit' ? '限价' : '市价') },
        { title: '价格', dataIndex: 'price' },
        { title: '数量', dataIndex: 'amount' },
        { title: '已成交', dataIndex: 'filled' },
        { title: '均价', dataIndex: 'avg_price' },
        { title: '手续费', dataIndex: 'fee' },
        { title: '状态', dataIndex: 'status', render: (v: string) => statusText[v] ?? v },
        ...(onCancel
          ? [
              {
                title: '',
                key: 'op',
                render: (_: unknown, r: SpotOrder) =>
                  r.status === 'pending' || r.status === 'partial' ? (
                    <Typography.Link type="danger" onClick={() => onCancel(r.id)}>
                      撤销
                    </Typography.Link>
                  ) : null,
              },
            ]
          : []),
      ]}
    />
  )
}

function TradesTable({ trades }: { trades: SpotTrade[] }) {
  return (
    <Table
      rowKey="id"
      size="small"
      locale={{ emptyText: <Empty description="暂无成交" image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
      pagination={false}
      dataSource={trades}
      columns={[
        { title: '时间', dataIndex: 'created_at', render: (v: string) => new Date(v).toLocaleTimeString('zh-CN') },
        { title: '交易对', dataIndex: 'symbol', render: (v: string) => v.replace(/USDT$/, '/USDT') },
        {
          title: '方向',
          dataIndex: 'side',
          render: (v: string) => <span style={{ color: sideColor(v) }}>{sideText(v)}</span>,
        },
        { title: '价格', dataIndex: 'price' },
        { title: '数量', dataIndex: 'amount' },
        { title: '成交额', dataIndex: 'quote_amount' },
        { title: '手续费', dataIndex: 'fee' },
      ]}
    />
  )
}

export default function Spot() {
  const [params, setParams] = useSearchParams()
  const symbol = (
    params.get('symbol') && SYMBOLS.includes(params.get('symbol')!) ? params.get('symbol') : 'BTCUSDT'
  )!
  const [tickers, setTickers] = useState<Ticker[]>([])
  const [depth, setDepth] = useState<Depth | null>(null)
  const [marketTradeList, setMarketTradeList] = useState<MarketTrade[]>([])
  const [balances, setBalances] = useState<Balance[]>([])
  const [open, setOpen] = useState<SpotOrder[]>([])
  const [history, setHistory] = useState<SpotOrder[]>([])
  const [trades, setTrades] = useState<SpotTrade[]>([])
  const [reloadFlag, setReloadFlag] = useState(0)

  const ticker = tickers.find((t) => t.symbol === symbol)
  const base = symbol.replace(/USDT$/, '')
  const balOf = (cur: string) => balances.find((b) => b.currency === cur)?.available ?? '0'

  const refreshOrders = useCallback(() => {
    void openOrders().then(setOpen).catch(() => {})
    void orderHistory().then(setHistory).catch(() => {})
    void myTrades().then(setTrades).catch(() => {})
    void me()
      .then((d) => setBalances(d.balances))
      .catch(() => {})
  }, [])
  useEffect(refreshOrders, [refreshOrders, reloadFlag])

  useEffect(() => {
    let stop = false
    async function load() {
      try {
        const d = await fetchTickers()
        if (!stop) setTickers(d.tickers)
      } catch {
        /* 拦截器已提示 */
      }
    }
    void load()
    // WS 实时行情 + 私有成交通知；30 秒 REST 兜底
    connectMarket({
      onTickers: (t) => !stop && setTickers(t),
      onTrade: (e) => {
        if (stop) return
        message.success(
          `成交：${e.side === 'buy' ? '买入' : '卖出'} ${e.symbol} ${e.amount} @ ${e.price}`,
        )
        refreshOrders()
      },
    })
    const timer = window.setInterval(load, 30000)
    return () => {
      stop = true
      window.clearInterval(timer)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    let stop = false
    async function load() {
      try {
        const [d, t] = await Promise.all([fetchDepth(symbol, 10), fetchTrades(symbol, 20)])
        if (!stop) {
          setDepth(d)
          setMarketTradeList(t)
        }
      } catch {
        /* 拦截器已提示 */
      }
    }
    void load()
    const timer = window.setInterval(load, 4000)
    return () => {
      stop = true
      window.clearInterval(timer)
    }
  }, [symbol])

  async function onCancel(id: number) {
    Modal.confirm({
      title: '撤销委托',
      content: `撤单后将解冻未成交部分的资金，确定撤销订单 #${id}？`,
      okText: '撤销',
      cancelText: '取消',
      async onOk() {
        try {
          await cancelOrder(id)
          message.success('已撤销')
          refreshOrders()
        } catch {
          /* 拦截器已提示 */
        }
      },
    })
  }

  return (
    <div>
      <Row gutter={[8, 8]}>
        <Col span={4}>
          <Card size="small" title={`${symbol} 盘口`} style={{ marginBottom: 8 }}>
            <OrderBook depth={depth} />
          </Card>
          <Card size="small" title="实时成交">
            <TradeList trades={marketTradeList} />
          </Card>
        </Col>
        <Col span={14}>
          <Card size="small">
            <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 8 }}>
              <Select
                value={symbol}
                style={{ width: 140 }}
                onChange={(v) => setParams({ symbol: v })}
                options={SYMBOLS.map((s) => ({ value: s, label: s.replace(/USDT$/, '/USDT') }))}
              />
              <Typography.Text
                strong
                style={{ fontSize: 20, color: ticker ? changeColor(ticker.change_pct) : undefined }}
              >
                {ticker?.last ?? '-'}
              </Typography.Text>
              <Typography.Text style={{ color: ticker ? changeColor(ticker.change_pct) : undefined }}>
                {ticker ? `${parseFloat(ticker.change_pct) >= 0 ? '+' : ''}${ticker.change_pct}%` : ''}
              </Typography.Text>
            </div>
            <CandleChart symbol={symbol} height={440} />
          </Card>
        </Col>
        <Col span={6}>
          <OrderForm
            symbol={symbol}
            last={ticker?.last ?? ''}
            baseAvailable={balOf(base)}
            usdtAvailable={balOf('USDT')}
            onDone={() => setReloadFlag((n) => n + 1)}
          />
        </Col>
      </Row>
      <Card size="small" style={{ marginTop: 8 }}>
        <Tabs
          size="small"
          items={[
            {
              key: 'open',
              label: `当前委托 (${open.length})`,
              children: <OrdersTable orders={open} onCancel={onCancel} />,
            },
            { key: 'history', label: '历史委托', children: <OrdersTable orders={history} /> },
            { key: 'trades', label: '成交记录', children: <TradesTable trades={trades} /> },
          ]}
        />
      </Card>
    </div>
  )
}
