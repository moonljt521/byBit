import { useEffect, useState } from 'react'
import { Table, Typography } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useNavigate } from 'react-router-dom'
import { fetchKlines, fetchTickers } from '../api/market'
import { connectMarket } from '../api/ws'
import type { Ticker } from '../api/market'
import { C, sparkPoints } from '../theme'

const UP = '#0ecb81'
const DOWN = '#f6465d'

export const changeColor = (v: string) => (parseFloat(v) >= 0 ? UP : DOWN)

/** 24h 价格在高低区间中的位置条（0-100%）。 */
function RangeBar({ t }: { t: Ticker }) {
  const lo = parseFloat(t.low24h)
  const hi = parseFloat(t.high24h)
  const last = parseFloat(t.last)
  if (!(hi > lo)) return <div style={{ height: 4 }} />
  const pct = Math.min(100, Math.max(0, ((last - lo) / (hi - lo)) * 100))
  return (
    <div
      title={`24h 区间位置 ${pct.toFixed(0)}%`}
      style={{
        position: 'relative',
        height: 4,
        borderRadius: 2,
        background: `linear-gradient(90deg, ${DOWN}, ${UP})`,
        opacity: 0.85,
        width: 120,
      }}
    >
      <div
        style={{
          position: 'absolute',
          left: `${pct}%`,
          top: -3,
          width: 2,
          height: 10,
          background: C.text,
          borderRadius: 1,
        }}
      />
    </div>
  )
}

/** 24h 迷你走势（首次加载拉 1h K 线 × 24）。 */
function Sparkline({ symbol }: { symbol: string }) {
  const [pts, setPts] = useState('')
  const [up, setUp] = useState(true)
  useEffect(() => {
    let stop = false
    fetchKlines(symbol, '1h', 24)
      .then((d) => {
        if (stop) return
        const closes = d.candles.map((x) => parseFloat(x.c))
        setPts(sparkPoints(closes, 120, 36))
        setUp(closes[closes.length - 1] >= closes[0])
      })
      .catch(() => {})
    return () => {
      stop = true
    }
  }, [symbol])
  if (!pts) return <div style={{ width: 120, height: 36 }} />
  return (
    <svg width={120} height={36}>
      <polyline points={pts} fill="none" stroke={up ? UP : DOWN} strokeWidth={1.6} />
    </svg>
  )
}

const starKey = 'cryptosim.stars'
function loadStars(): string[] {
  try {
    return JSON.parse(localStorage.getItem(starKey) || '[]')
  } catch {
    return []
  }
}

export default function Markets() {
  const [tickers, setTickers] = useState<Ticker[]>([])
  const [loading, setLoading] = useState(true)
  const [stars, setStars] = useState<string[]>(loadStars)
  const navigate = useNavigate()

  const toggleStar = (s: string) => {
    const next = stars.includes(s) ? stars.filter((x) => x !== s) : [...stars, s]
    setStars(next)
    localStorage.setItem(starKey, JSON.stringify(next))
  }

  useEffect(() => {
    let stop = false
    async function load() {
      try {
        const d = await fetchTickers()
        if (!stop) setTickers(d.tickers)
      } catch {
        /* 拦截器已提示 */
      } finally {
        if (!stop) setLoading(false)
      }
    }
    void load()
    // WebSocket 实时推送（主）+ 30 秒轮询兜底
    connectMarket({
      onTickers: (t) => {
        if (!stop) {
          setTickers(t)
          setLoading(false)
        }
      },
    })
    const timer = window.setInterval(load, 30000)
    return () => {
      stop = true
      window.clearInterval(timer)
    }
  }, [])

  const columns: ColumnsType<Ticker> = [
    {
      title: '',
      key: 'star',
      width: 36,
      render: (_, r) => (
        <span
          onClick={(e) => {
            e.stopPropagation()
            toggleStar(r.symbol)
          }}
          style={{ cursor: 'pointer', fontSize: 15, color: stars.includes(r.symbol) ? '#F0B90B' : C.text2 }}
        >
          {stars.includes(r.symbol) ? '★' : '☆'}
        </span>
      ),
    },
    {
      title: '交易对',
      dataIndex: 'symbol',
      render: (v: string) => (
        <Typography.Text strong>
          {v.replace(/USDT$/, '')}
          <span style={{ color: '#999' }}>/USDT</span>
        </Typography.Text>
      ),
    },
    {
      title: '最新价',
      dataIndex: 'last',
      align: 'right',
      render: (v: string) => <Typography.Text strong>{v}</Typography.Text>,
    },
    {
      title: '24h 涨跌',
      dataIndex: 'change_pct',
      align: 'right',
      render: (v: string) => (
        <span style={{ color: changeColor(v) }}>
          {parseFloat(v) >= 0 ? '+' : ''}
          {v}%
        </span>
      ),
    },
    {
      title: '24h 走势',
      key: 'spark',
      width: 140,
      render: (_, r) => <Sparkline symbol={r.symbol} />,
    },
    {
      title: '24h 区间',
      key: 'range',
      width: 140,
      render: (_, r) => <RangeBar t={r} />,
    },
    { title: '24h 最高', dataIndex: 'high24h', align: 'right' },
    { title: '24h 最低', dataIndex: 'low24h', align: 'right' },
    {
      title: '24h 成交量',
      dataIndex: 'vol24h',
      align: 'right',
      render: (v: string) => Number(v).toLocaleString('en-US', { maximumFractionDigits: 0 }),
    },
    { title: '', key: 'go', align: 'right', render: () => <Typography.Link>去交易</Typography.Link> },
  ]

  return (
    <Table
      rowKey="symbol"
      columns={columns}
      dataSource={[...tickers].sort(
        (a, b) => Number(stars.includes(b.symbol)) - Number(stars.includes(a.symbol)),
      )}
      loading={loading}
      pagination={false}
      onRow={(r) => ({
        onClick: () => navigate(`/spot?symbol=${r.symbol}`),
        style: { cursor: 'pointer' },
      })}
    />
  )
}
