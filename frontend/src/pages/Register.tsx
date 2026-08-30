import { Button, Card, Form, Input, message } from 'antd'
import { Link, useNavigate } from 'react-router-dom'
import { register } from '../api/auth'
import { useAuthStore } from '../store/auth'

export default function Register() {
  const navigate = useNavigate()
  const setAuth = useAuthStore((s) => s.setAuth)

  async function onFinish(values: {
    email: string
    username: string
    password: string
    confirm: string
  }) {
    try {
      const data = await register(values.email, values.username, values.password)
      setAuth(data.token, data.api_key, data.api_secret, data.user)
      message.success(`注册成功，已赠送 10,000 虚拟 USDT，欢迎 ${data.user.username}！`)
      navigate('/assets')
    } catch {
      /* 错误已在拦截器中提示 */
    }
  }

  return (
    <div className="auth-page">
      <Card className="auth-card">
        <h2 className="auth-title">注册 CryptoSim</h2>
        <div className="auth-subtitle">注册即赠送 10,000 虚拟 USDT · 资金全虚拟，仅供学习</div>
        <Form layout="vertical" onFinish={onFinish}>
          <Form.Item
            name="email"
            label="邮箱"
            rules={[
              { required: true, message: '请输入邮箱' },
              { type: 'email', message: '邮箱格式不正确' },
            ]}
          >
            <Input placeholder="you@example.com" size="large" />
          </Form.Item>
          <Form.Item
            name="username"
            label="用户名"
            rules={[
              { required: true, message: '请输入用户名' },
              { pattern: /^[a-zA-Z0-9_]{3,20}$/, message: '3-20 位字母、数字或下划线' },
            ]}
          >
            <Input placeholder="3-20 位字母、数字或下划线" size="large" />
          </Form.Item>
          <Form.Item
            name="password"
            label="密码"
            rules={[
              { required: true, message: '请输入密码' },
              { min: 8, message: '密码至少 8 位' },
            ]}
          >
            <Input.Password placeholder="至少 8 位" size="large" />
          </Form.Item>
          <Form.Item
            name="confirm"
            label="确认密码"
            dependencies={['password']}
            rules={[
              { required: true, message: '请再次输入密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('password') === value) return Promise.resolve()
                  return Promise.reject(new Error('两次输入的密码不一致'))
                },
              }),
            ]}
          >
            <Input.Password placeholder="再次输入密码" size="large" />
          </Form.Item>
          <Button type="primary" htmlType="submit" block size="large">
            注册并领取虚拟资金
          </Button>
          <div style={{ marginTop: 16, textAlign: 'center' }}>
            已有账号？<Link to="/login">直接登录</Link>
          </div>
        </Form>
      </Card>
    </div>
  )
}
