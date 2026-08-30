import { Layout, Menu, Button, Dropdown } from 'antd'
import {
  DashboardOutlined,
  TeamOutlined,
  FileTextOutlined,
  SafetyOutlined,
  LogoutOutlined,
  UserOutlined,
} from '@ant-design/icons'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAdminStore } from '../store/auth'

const { Header, Sider, Content } = Layout

const menus = [
  { key: '/dashboard', icon: <DashboardOutlined />, label: '仪表盘' },
  { key: '/users', icon: <TeamOutlined />, label: '用户管理' },
  { key: '/ledgers', icon: <FileTextOutlined />, label: '资金流水' },
  { key: '/login-logs', icon: <SafetyOutlined />, label: '登录审计' },
]

export default function AdminLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { username, logout } = useAdminStore()

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider>
        <div
          style={{
            color: '#fff',
            fontWeight: 700,
            textAlign: 'center',
            padding: '18px 0',
            fontSize: 16,
          }}
        >
          CryptoSim Admin
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menus}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header
          style={{
            background: '#fff',
            display: 'flex',
            justifyContent: 'flex-end',
            alignItems: 'center',
            padding: '0 24px',
          }}
        >
          <Dropdown
            menu={{
              items: [
                {
                  key: 'logout',
                  icon: <LogoutOutlined />,
                  label: '退出登录',
                  onClick: () => {
                    logout()
                    navigate('/login')
                  },
                },
              ],
            }}
            trigger={['click']}
          >
            <Button type="text" icon={<UserOutlined />}>
              {username}
            </Button>
          </Dropdown>
        </Header>
        <Content style={{ margin: 16 }}>
          <div style={{ padding: 24, background: '#fff', borderRadius: 8, minHeight: 400 }}>
            <Outlet />
          </div>
        </Content>
      </Layout>
    </Layout>
  )
}
