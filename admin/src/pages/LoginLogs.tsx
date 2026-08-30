import { useCallback, useEffect, useState } from 'react'
import { Button, Table, Tag } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { loginLogs } from '../api/admin'
import type { LoginLog } from '../api/types'

export default function LoginLogs() {
  const [list, setList] = useState<LoginLog[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)

  const load = useCallback(async (p: number) => {
    setLoading(true)
    try {
      const d = await loginLogs(p, 20)
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

  const columns: ColumnsType<LoginLog> = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '用户名', dataIndex: 'username', width: 140 },
    {
      title: '结果',
      dataIndex: 'success',
      width: 90,
      render: (v: boolean) => <Tag color={v ? 'green' : 'red'}>{v ? '成功' : '失败'}</Tag>,
    },
    { title: '原因', dataIndex: 'reason', width: 140 },
    { title: 'IP', dataIndex: 'ip', width: 140 },
    { title: '设备（UA）', dataIndex: 'user_agent', ellipsis: true },
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
        <h3 style={{ margin: 0 }}>登录审计日志</h3>
        <Button onClick={() => void load(page)}>刷新</Button>
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
          onChange: (p) => void load(p),
          showTotal: (t) => `共 ${t} 条`,
        }}
      />
    </div>
  )
}
