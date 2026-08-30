import { Button, Card, Form, Input, message } from 'antd'
import { Link, useNavigate } from 'react-router-dom'
import { login } from '../api/auth'
import { useAuthStore } from '../store/auth'

export default function Login() {
  const navigate = useNavigate()
  const setAuth = useAuthStore((s) => s.setAuth)

  async function onFinish(values: { account: string; password: string }) {
    try {
      const data = await login(values.account, values.password)
      setAuth(data.token, data.api_key, data.api_secret, data.user)
      message.success(`欢迎回来，${data.user.username}`)
      navigate('/markets')
    } catch {
      /* 错误已在拦截器中提示 */
    }
  }

  return (
    <div className="auth-page">
      <Card className="auth-card">
        <h2 className="auth-title">CryptoSim</h2>
        <div className="auth-subtitle">虚拟加密货币交易所 · 资金全虚拟 · 仅供学习</div>
        <Form layout="vertical" onFinish={onFinish}>
          <Form.Item name="account" label="邮箱 / 用户名" rules={[{ required: true, message: '请输入账号' }]}>
            <Input placeholder="邮箱或用户名" size="large" autoFocus />
          </Form.Item>
          <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password placeholder="密码" size="large" />
          </Form.Item>
          <Button type="primary" htmlType="submit" block size="large">
            登录
          </Button>
          <div style={{ marginTop: 16, textAlign: 'center' }}>
            还没有账号？<Link to="/register">立即注册</Link>（注册送 10,000 虚拟 USDT）
          </div>
        </Form>
      </Card>
    </div>
  )
}
