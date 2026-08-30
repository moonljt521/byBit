import { useEffect, useMemo, useState } from 'react'
import { Button, Card, Col, Progress, Row, Statistic, Table, Typography } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { ReloadOutlined } from '@ant-design/icons'
import { me } from '../api/auth'
import { fetchTickers } from '../api/market'
import type { Balance } from '../api/types'
import { useAuthStore } from '../store/auth'

const columns: ColumnsType<Balance & { value: number; share: number }> = [
  {
    title: '币种',
    dataIndex: 'currency',
    key: 'currency',
    render: (v: string) => <Typography.Text strong>{v}</Typography.Text>,
  },
  {
    title: '可用余额',
    dataIndex: 'available',
    key: 'available',
    align: 'right' as const,
    render: (v: string) => <Typography.Text strong>{v}</Typography.Text>,
  },
  { title: '冻结', dataIndex: 'frozen', key: 'frozen', align: 'right' as const },
  {
    title: '价值 (USDT)',
    key: 'value',
    align: 'right' as const,
    render: (r: Balance & { value: number; share: number }) =>
      r.currency === 'USDT' ? (
        <Typography.Text>{r.available}</Typography.Text>
      ) : (
        <Typography.Text type="secondary">≈ {r.value.toFixed(2)}</Typography.Text>
      ),
  },
  {
    title: '占比',
    key: 'share',
    width: 220,
    render: (r: Balance & { value: number; share: number }) => (
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <Progress
          percent={r.share}
          showInfo={false}
          strokeColor="#00C8A0"
          trailColor="#252930"
          size={{ height: 6 }}
        />
        <Typography.Text type="secondary" style={{ fontSize: 12, width: 48, textAlign: 'right' }}>
          {r.share.toFixed(1)}%
        </Typography.Text>
      </div>
    ),
  },
]

export default function Assets() {
  const [balances, setBalances] = useState<(Balance & { value: number; share: number })[]>([])
  const [loading, setLoading] = useState(false)
  const setUser = useAuthStore((s) => s.setUser)

  const load = useMemo(
    () => async () => {
      setLoading(true)
      try {
        const [meData, tk] = await Promise.all([me(), fetchTickers()])
        const priceOf = (cur: string) =>
          cur === 'USDT' ? 1 : parseFloat(tk.tickers.find((t) => t.symbol === cur + 'USDT')?.last ?? '0')
        const rows = meData.balances.map((b) => ({
          ...b,
          value: parseFloat(b.available) * priceOf(b.currency),
        }))
        const total = rows.reduce((s, r) => s + r.value, 0) || 1
        setBalances(rows.map((r) => ({ ...r, share: (r.value / total) * 100 })))
        setUser(meData.user)
      } catch {
        /* 拦截器已提示 */
      } finally {
        setLoading(false)
      }
    },
    [setUser],
  )

  useEffect(() => {
    void load()
  }, [load])

  const total = balances.reduce((s, r) => s + r.value, 0)

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Typography.Title level={4} style={{ margin: 0 }}>
          我的资产（虚拟）
        </Typography.Title>
        <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void load()}>
          刷新
        </Button>
      </div>
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card size="small">
            <Statistic title="总权益估值" value={total.toFixed(2)} suffix="USDT" precision={2} />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic
              title="持仓币种"
              value={balances.filter((b) => parseFloat(b.available) > 0).length}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic title="冻结资产" value={balances.reduce((s, b) => s + parseFloat(b.frozen), 0).toFixed(4)} suffix="USDT" />
          </Card>
        </Col>
      </Row>
      <Table
        rowKey="currency"
        columns={columns}
        dataSource={balances}
        loading={loading}
        pagination={false}
        locale={{ emptyText: '暂无资产' }}
      />
    </div>
  )
}