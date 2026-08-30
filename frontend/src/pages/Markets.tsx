import { useEffect, useState } from 'react'
import { Table, Typography } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useNavigate } from 'react-router-dom'
import { fetchTickers } from '../api/market'
import { connectMarket } from '../api/ws'
import type { Ticker } from '../api/market'

const UP = '#0ecb81'
const DOWN = '#f6465d'

export const changeColor = (v: string) => (parseFloat(v) >= 0 ? UP : DOWN)

export default function Markets() {
  const [tickers, setTickers] = useState<Ticker[]>([])
  const [loading, setLoading] = useState(true)
  const navigate = useNavigate()

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
      dataSource={tickers}
      loading={loading}
      pagination={false}
      onRow={(r) => ({
        onClick: () => navigate(`/spot?symbol=${r.symbol}`),
        style: { cursor: 'pointer' },
      })}
    />
  )
}
