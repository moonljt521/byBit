import { useCallback, useEffect, useState } from 'react'
import { Button, Input, Modal, Space, Table, Tag, Typography, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { adjustFunds, setStatus, users } from '../api/admin'
import type { AdminUser } from '../api/types'
import { useAdminStore } from '../store/auth'

export default function Users() {
  const [list, setList] = useState<AdminUser[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [kw, setKw] = useState('')
  const [loading, setLoading] = useState(false)
  const me = useAdminStore((s) => s.username)

  const load = useCallback(
    async (p: number, keyword: string) => {
      setLoading(true)
      try {
        const d = await users(p, 20, keyword)
        setList(d.list)
        setTotal(d.total)
        setPage(p)
      } catch {
        /* 拦截器已提示 */
      } finally {
        setLoading(false)
      }
    },
    [],
  )
  useEffect(() => {
    void load(1, '')
  }, [load])

  async function toggleStatus(u: AdminUser) {
    const next = u.status === 1 ? 0 : 1
    Modal.confirm({
      title: next === 0 ? '禁用用户' : '启用用户',
      content:
        next === 0
          ? `禁用后 ${u.username} 将无法登录。确定？`
          : `恢复 ${u.username} 的登录权限。确定？`,
      okType: next === 0 ? 'danger' : 'primary',
      okText: '确定',
      cancelText: '取消',
      async onOk() {
        try {
          await setStatus(u.id, next)
          message.success('已更新')
          void load(page, kw)
        } catch {
          /* 拦截器已提示 */
        }
      },
    })
  }

  function grant(u: AdminUser) {
    let amount = ''
    let memo = ''
    Modal.confirm({
      title: `调拨虚拟资金 → ${u.username}`,
      content: (
        <div style={{ display: 'grid', gap: 12, marginTop: 12 }}>
          <Input
            placeholder="金额（正数调增 / 负数调减），如 1000 或 -500"
            onChange={(e) => (amount = e.target.value)}
          />
          <Input placeholder="备注（记入审计流水）" onChange={(e) => (memo = e.target.value)} />
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            当前 USDT 可用：{u.usdt_available}。调减后余额不能为负。
          </Typography.Text>
        </div>
      ),
      okText: '确认调拨',
      cancelText: '取消',
      async onOk() {
        const v = parseFloat(amount)
        if (!amount || isNaN(v) || v === 0) {
          message.warning('请输入有效金额')
          throw new Error('invalid')
        }
        try {
          await adjustFunds(u.id, String(v), memo || '管理员调拨')
          message.success('调拨成功')
          void load(page, kw)
        } catch {
          /* 拦截器已提示 */
        }
      },
    })
  }

  const columns: ColumnsType<AdminUser> = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '用户名', dataIndex: 'username' },
    { title: '邮箱', dataIndex: 'email' },
    {
      title: '角色',
      dataIndex: 'role',
      width: 90,
      render: (v: string) => <Tag color={v === 'admin' ? 'gold' : 'blue'}>{v}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (v: number) => <Tag color={v === 1 ? 'green' : 'red'}>{v === 1 ? '正常' : '禁用'}</Tag>,
    },
    {
      title: 'USDT 可用',
      dataIndex: 'usdt_available',
      align: 'right',
      render: (v: string) => <Typography.Text strong>{v}</Typography.Text>,
    },
    {
      title: '注册时间',
      dataIndex: 'created_at',
      render: (v: string) => new Date(v).toLocaleString('zh-CN'),
    },
    {
      title: '操作',
      key: 'op',
      width: 200,
      render: (_, u) => (
        <Space>
          <Button size="small" onClick={() => grant(u)}>
            调资金
          </Button>
          <Button size="small" danger={u.status === 1} disabled={u.username === me} onClick={() => void toggleStatus(u)}>
            {u.status === 1 ? '禁用' : '启用'}
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 12 }}>
        <h3 style={{ margin: 0 }}>用户管理</h3>
        <Space>
          <Input.Search
            placeholder="搜索用户名 / 邮箱"
            style={{ width: 260 }}
            onSearch={(v) => {
              setKw(v)
              void load(1, v)
            }}
            allowClear
          />
          <Button onClick={() => void load(page, kw)}>刷新</Button>
        </Space>
      </div>
      <Table
        rowKey="id"
        columns={columns}
        dataSource={list}
        loading={loading}
        pagination={{
          current: page,
          total,
          pageSize: 20,
          onChange: (p) => void load(p, kw),
          showTotal: (t) => `共 ${t} 个用户`,
        }}
      />
    </div>
  )
}
