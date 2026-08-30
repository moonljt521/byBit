import { useEffect, useState } from 'react'
import { Card, Col, Row, Statistic } from 'antd'
import { stats } from '../api/admin'
import type { Stats } from '../api/types'

export default function Dashboard() {
  const [st, setSt] = useState<Stats | null>(null)
  useEffect(() => {
    void stats().then(setSt).catch(() => {})
  }, [])
  return (
    <div>
      <h3>运营仪表盘</h3>
      <Row gutter={16}>
        <Col span={6}>
          <Card size="small">
            <Statistic title="用户总数" value={st?.total_users ?? '-'} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title="今日新增" value={st?.new_users_today ?? '-'} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title="正常状态用户" value={st?.active_users ?? '-'} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title="今日流水笔数" value={st?.ledger_today ?? '-'} />
          </Card>
        </Col>
        <Col span={6} style={{ marginTop: 16 }}>
          <Card size="small">
            <Statistic
              title="USDT 总余额（可用）"
              value={st?.usdt_available ?? '-'}
              precision={2}
              style={{ minWidth: 180 }}
            />
          </Card>
        </Col>
        <Col span={6} style={{ marginTop: 16 }}>
          <Card size="small">
            <Statistic title="USDT 冻结（挂单占用）" value={st?.usdt_frozen ?? '-'} precision={2} />
          </Card>
        </Col>
      </Row>
      <p style={{ marginTop: 24, color: '#999', fontSize: 12 }}>
        数据实时统计自 PostgreSQL。所有资金均为虚拟资金。
      </p>
    </div>
  )
}
