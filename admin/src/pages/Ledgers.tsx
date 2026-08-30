import { useCallback, useEffect, useState } from 'react'
import { Button, Input, Space, Table, Tag, Typography } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { ledger } from '../api/admin'
import type { LedgerRow } from '../api/types'

const bizColor: Record<string, string> = {
  signup_grant: 'green',
  reset_grant: 'cyan',
  admin_adjust: 'gold',
  order_freeze: 'blue',
  order_unfreeze: 'geekblue',
  trade: 'purple',
  futures_margin: 'orange',
  futures_close: 'magenta',
}

const bizText: Record<string, string> = {
  signup_grant: '注册赠送',
  reset_grant: '重置赠送',
  admin_adjust: '管理员调拨',
  order_freeze: '下单冻结',
  order_unfreeze: '撤单解冻',
  trade: '现货成交',
  futures_margin: '合约保证金',
  futures_close: '合约平仓',
}

export default function Ledgers() {
  const [list, setList] = useState<LedgerRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [uidText, setUidText] = useState('')
  const [uid, setUid] = useState<number | undefined>()
  const [loading, setLoading] = useState(false)

  const load = useCallback(async (p: number, userId?: number) => {
    setLoading(true)
    try {
      const d = await ledger(p, 20, userId)
      setList(d.list)
      setTotal(d.total)
      setPage(p)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [])
  useEffect(() => {
    void load(1)
  }, [load])

  const columns: ColumnsType<LedgerRow> = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    {
      title: '用户',
      key: 'user',
      width: 160,
      render: (_, r) => (
        <Typography.Text copyable={{ text: String(r.user_id) }}>
          {r.username} #{r.user_id}
        </Typography.Text>
      ),
    },
    {
      title: '业务类型',
      dataIndex: 'biz_type',
      width: 120,
      render: (v: string) => <Tag color={bizColor[v] ?? 'default'}>{bizText[v] ?? v}</Tag>,
    },
    { title: '币种', dataIndex: 'currency', width: 80 },
    {
      title: '变动金额',
      dataIndex: 'amount',
      align: 'right',
      render: (v: string) => (
        <span style={{ color: parseFloat(v) >= 0 ? '#0ecb81' : '#f6465d' }}>{v}</span>
      ),
    },
    { title: '变动后余额', dataIndex: 'balance_after', align: 'right' },
    { title: '备注', dataIndex: 'memo', ellipsis: true },
    {
      title: '时间',
      dataIndex: 'created_at',
      width: 170,
      render: (v: string) => new Date(v).toLocaleString('zh-CN'),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 12 }}>
        <h3 style={{ margin: 0 }}>资金流水（全局审计）</h3>
        <Space style={{ display: 'flex' }}>
          <Input
            placeholder="按用户 ID 过滤"
            style={{ width: 180 }}
            value={uidText}
            onChange={(e) => setUidText(e.target.value)}
            allowClear
          />
          <Button
            onClick={() => {
              const v = parseInt(uidText, 10)
              const id = isNaN(v) ? undefined : v
              setUid(id)
              void load(1, id)
            }}
          >
            过滤
          </Button>
        </Space>
      </div>
      <Table
        rowKey="id"
        columns={columns}
        dataSource={list}
        loading={loading}
        size="small"
        pagination={{
          current: page,
          total,
          pageSize: 20,
          onChange: (p) => void load(p, uid),
          showTotal: (t) => `共 ${t} 条`,
        }}
      />
    </div>
  )
}
