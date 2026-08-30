import { useCallback, useEffect, useState } from 'react'
import {
  Alert,
  Button,
  Card,
  Col,
  Empty,
  Input,
  message,
  Modal,
  Row,
  Select,
  Slider,
  Table,
  Tabs,
  Typography,
} from 'antd'
import CandleChart from '../components/CandleChart'
import { fetchTickers } from '../api/market'
import type { Ticker } from '../api/market'
import { me } from '../api/auth'
import { closePosition, fundingRecords, openPosition, positionHistory, positions } from '../api/futures'
import type { FuturesPosition, FundingRecord } from '../api/futures'

const UP = '#0ecb81'
const DOWN = '#f6465d'

const FUT_SYMBOLS = ['BTCUSDT', 'ETHUSDT']

const statusText: Record<string, string> = { open: '持仓中', closed: '已平仓', liquidated: '已强平' }

interface PosTableProps {
  list: FuturesPosition[]
  live: boolean
  onClose?: (id: number) => void
}

function PosTable({ list, live, onClose }: PosTableProps) {
  return (
    <Table
      rowKey="id"
      size="small"
      pagination={false}
      locale={{ emptyText: <Empty description="暂无仓位" image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
      dataSource={list}
      columns={[
        { title: '交易对', dataIndex: 'symbol', render: (v: string) => v.replace(/USDT$/, '/USDT') },
        {
          title: '方向',
          dataIndex: 'side',
          render: (v: string) => (
            <span style={{ color: v === 'long' ? UP : DOWN }}>{v === 'long' ? '做多' : '做空'}</span>
          ),
        },
        { title: '杠杆', dataIndex: 'leverage', render: (v: number) => `${v}x` },
        { title: '数量', dataIndex: 'size' },
        { title: '开仓价', dataIndex: 'entry_price' },
        ...(live
          ? [{ title: '标记价', dataIndex: 'mark_price' }]
          : [
              { title: '平仓价', dataIndex: 'mark_price' },
              { title: '状态', dataIndex: 'status', render: (v: string) => statusText[v] ?? v },
            ]),
        {
          title: live ? '未实现盈亏' : '已实现盈亏',
          key: 'pnl',
          render: (_: unknown, r: FuturesPosition) => {
            const pnl = live ? r.unrealized_pnl : r.realized_pnl
            return <span style={{ color: parseFloat(pnl) >= 0 ? UP : DOWN }}>{pnl}</span>
          },
        },
        { title: live ? '收益率' : '手续费', key: 'x', render: (_: unknown, r: FuturesPosition) => (live ? `${r.roi}%` : r.fee) },
        ...(live
          ? [
              { title: '强平价', dataIndex: 'liquidation_price' },
              {
                title: '',
                key: 'op',
                render: (_: unknown, r: FuturesPosition) => (
                  <Typography.Link type="danger" onClick={() => onClose?.(r.id)}>
                    平仓
                  </Typography.Link>
                ),
              },
            ]
          : []),
      ]}
    />
  )
}

export default function Futures() {
  const [symbol, setSymbol] = useState('BTCUSDT')
  const [side, setSide] = useState<'long' | 'short'>('long')
  const [leverage, setLeverage] = useState(5)
  const [amount, setAmount] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [tickers, setTickers] = useState<Ticker[]>([])
  const [usdt, setUsdt] = useState('0')
  const [open, setOpen] = useState<FuturesPosition[]>([])
  const [history, setHistory] = useState<FuturesPosition[]>([])
  const [funding, setFunding] = useState<FundingRecord[]>([])
  const [reloadFlag, setReloadFlag] = useState(0)

  const ticker = tickers.find((t) => t.symbol === symbol)
  const base = symbol.replace(/USDT$/, '')

  const refresh = useCallback(() => {
    void positions().then(setOpen).catch(() => {})
    void positionHistory().then(setHistory).catch(() => {})
    void fundingRecords().then(setFunding).catch(() => {})
    void me()
      .then((d) => {
        const b = d.balances.find((x) => x.currency === 'USDT')
        setUsdt(b?.available ?? '0')
      })
      .catch(() => {})
  }, [])
  useEffect(refresh, [refresh, reloadFlag])

  // 行情 + 仓位实时刷新
  useEffect(() => {
    let stop = false
    async function load() {
      try {
        const d = await fetchTickers()
        if (!stop) setTickers(d.tickers)
      } catch {
        /* 忽略 */
      }
    }
    void load()
    const t1 = window.setInterval(load, 5000)
    const t2 = window.setInterval(() => void positions().then((l) => !stop && setOpen(l)).catch(() => {}), 4000)
    return () => {
      stop = true
      window.clearInterval(t1)
      window.clearInterval(t2)
    }
  }, [])

  const notional = (parseFloat(ticker?.last ?? '0') || 0) * (parseFloat(amount) || 0)
  const margin = notional / leverage
  const estFee = notional * 0.0005

  async function submit() {
    if (!amount || parseFloat(amount) <= 0) {
      message.warning('请输入数量')
      return
    }
    setSubmitting(true)
    try {
      await openPosition({ symbol, side, leverage, amount })
      message.success(`开${side === 'long' ? '多' : '空'}成功，当前标记价 ${ticker?.last ?? '-'}`)
      setAmount('')
      setReloadFlag((n) => n + 1)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setSubmitting(false)
    }
  }

  function onClose(id: number) {
    Modal.confirm({
      title: '平仓',
      content: '将按当前标记价全部平仓并结算盈亏，确定？',
      okText: '平仓',
      okType: 'danger',
      cancelText: '取消',
      async onOk() {
        try {
          await closePosition(id)
          message.success('已平仓')
          setReloadFlag((n) => n + 1)
        } catch {
          /* 拦截器已提示 */
        }
      },
    })
  }

  return (
    <div>
      <Alert
        style={{ marginBottom: 8 }}
        type="warning"
        showIcon
        message="模拟合约规则（逐仓）：taker 0.05% · 维持保证金率 0.5% · 资金费率每 8 小时结算 0.01%（多头付空头）· 触及强平价将损失全部保证金"
      />
      <Row gutter={[8, 8]}>
        <Col span={16}>
          <Card size="small">
            <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 8 }}>
              <Select
                value={symbol}
                style={{ width: 180 }}
                onChange={setSymbol}
                options={FUT_SYMBOLS.map((s) => ({ value: s, label: `${s.replace(/USDT$/, '')}USDT 永续` }))}
              />
              <Typography.Text
                strong
                style={{ fontSize: 20, color: ticker ? (parseFloat(ticker.change_pct) >= 0 ? UP : DOWN) : undefined }}
              >
                {ticker?.last ?? '-'}
              </Typography.Text>
            </div>
            <CandleChart symbol={symbol} height={400} />
          </Card>
          <Card size="small" style={{ marginTop: 8 }}>
            <Tabs
              size="small"
              items={[
                {
                  key: 'pos',
                  label: `当前仓位 (${open.length})`,
                  children: <PosTable list={open} live onClose={onClose} />,
                },
                { key: 'hist', label: '历史仓位', children: <PosTable list={history} live={false} /> },
                {
                  key: 'funding',
                  label: '资金费率记录',
                  children: (
                    <Table
                      rowKey="id"
                      size="small"
                      pagination={false}
                      locale={{ emptyText: <Empty description="暂无记录（每 8 小时结算一次）" image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
                      dataSource={funding}
                      columns={[
                        { title: '时间', dataIndex: 'created_at', render: (v: string) => new Date(v).toLocaleString('zh-CN') },
                        { title: '交易对', dataIndex: 'symbol', render: (v: string) => v.replace(/USDT$/, '/USDT') },
                        { title: '费率', dataIndex: 'rate' },
                        {
                          title: '金额（正=收）',
                          dataIndex: 'amount',
                          render: (v: string) => (
                            <span style={{ color: parseFloat(v) >= 0 ? UP : DOWN }}>{v}</span>
                          ),
                        },
                      ]}
                    />
                  ),
                },
              ]}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small" title="开仓">
            <div style={{ display: 'flex', gap: 8, marginBottom: 16 }}>
              <Button
                block
                style={{
                  background: side === 'long' ? UP : undefined,
                  borderColor: UP,
                  color: side === 'long' ? '#fff' : UP,
                }}
                onClick={() => setSide('long')}
              >
                开多（看涨）
              </Button>
              <Button
                block
                style={{
                  background: side === 'short' ? DOWN : undefined,
                  borderColor: DOWN,
                  color: side === 'short' ? '#fff' : DOWN,
                }}
                onClick={() => setSide('short')}
              >
                开空（看跌）
              </Button>
            </div>
            <div style={{ fontSize: 12, color: '#666', marginBottom: 4 }}>
              杠杆：<b>{leverage}x</b>（强平价距开仓价约 {Math.round(100 / leverage - 0.5)}%）
            </div>
            <Slider min={1} max={20} value={leverage} onChange={setLeverage} marks={{ 1: '1x', 5: '5x', 10: '10x', 20: '20x' }} />
            <div style={{ fontSize: 12, color: '#666', margin: '12px 0 8px' }}>
              可用 USDT：<b>{usdt}</b>
            </div>
            <Input
              placeholder={`数量（${base}）`}
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              style={{ marginBottom: 8 }}
            />
            <div style={{ fontSize: 12, color: '#999', marginBottom: 12 }}>
              名义价值 {notional.toFixed(2)} USDT · 保证金 {margin.toFixed(2)} · 预估手续费 {estFee.toFixed(4)}
              <br />
              最小名义价值 5 USDT；做多强平价 ≈ 开仓价×(1-1/杠杆+0.5%)
            </div>
            <Button
              block
              size="large"
              loading={submitting}
              style={{ background: side === 'long' ? UP : DOWN, borderColor: side === 'long' ? UP : DOWN, color: '#fff' }}
              onClick={() => void submit()}
            >
              {side === 'long' ? '开多' : '开空'} {base} 永续
            </Button>
          </Card>
        </Col>
      </Row>
    </div>
  )
}
