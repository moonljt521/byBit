import { useCallback, useEffect, useState } from 'react'
import { Button, Table, Typography } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { me } from '../api/auth'
import type { Balance } from '../api/types'
import { useAuthStore } from '../store/auth'

const columns = [
  { title: '币种', dataIndex: 'currency', key: 'currency' },
  {
    title: '可用余额',
    dataIndex: 'available',
    key: 'available',
    align: 'right' as const,
    render: (v: string) => <Typography.Text strong>{v}</Typography.Text>,
  },
  { title: '冻结', dataIndex: 'frozen', key: 'frozen', align: 'right' as const },
]

export default function Assets() {
  const [balances, setBalances] = useState<Balance[]>([])
  const [loading, setLoading] = useState(false)
  const setUser = useAuthStore((s) => s.setUser)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const data = await me()
      setBalances(data.balances)
      setUser(data.user)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [setUser])

  useEffect(() => {
    void load()
  }, [load])

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
