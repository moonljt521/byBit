import { Button, Card, Form, Input, message } from 'antd'
import { useNavigate } from 'react-router-dom'
import { login } from '../api/admin'
import { useAdminStore } from '../store/auth'

export default function Login() {
  const navigate = useNavigate()
  const setAuth = useAdminStore((s) => s.setAuth)

  async function onFinish(values: { account: string; password: string }) {
    try {
      const d = await login(values.account, values.password)
      if (d.user.role !== 'admin') {
        message.error('该账号不是管理员')
        return
      }
      setAuth(d.token, d.api_key, d.api_secret, d.user.username)
      navigate('/dashboard')
    } catch {
      /* 拦截器已提示 */
    }
  }

  return (
    <div className="admin-login">
      <Card style={{ width: 380, boxShadow: '0 8px 40px rgba(0,0,0,.35)' }}>
        <h2 style={{ textAlign: 'center' }}>CryptoSim 管理后台</h2>
        <p style={{ textAlign: 'center', color: '#999', fontSize: 12 }}>
          仅限管理员账号 · 全部操作记入审计流水
        </p>
        <Form layout="vertical" onFinish={onFinish}>
          <Form.Item name="account" label="账号" rules={[{ required: true, message: '请输入账号' }]}>
            <Input placeholder="管理员用户名或邮箱" size="large" autoFocus />
          </Form.Item>
          <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password size="large" />
          </Form.Item>
          <Button type="primary" htmlType="submit" block size="large">
            登录
          </Button>
        </Form>
      </Card>
    </div>
  )
}
