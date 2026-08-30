import { useState } from 'react'
import { Layout, Menu, Dropdown, Button, Modal, message, theme } from 'antd'
import {
  BarChartOutlined,
  SwapOutlined,
  RiseOutlined,
  WalletOutlined,
  BookOutlined,
  UserOutlined,
  ReloadOutlined,
  KeyOutlined,
  LogoutOutlined,
} from '@ant-design/icons'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAuthStore } from '../store/auth'
import { resetAccount, resetCredentials } from '../api/auth'
import { t, useLang } from '../i18n'

const { Header, Sider, Content } = Layout

// 菜单文案随语言切换（组件内用 useLang 计算）


export default function AppLayout() {
  const [collapsed, setCollapsed] = useState(false)
  const navigate = useNavigate()
  const location = useLocation()
  const { user, logout, setCredentials } = useAuthStore()
  const { lang, toggle } = useLang()
  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken()

  function onResetCredentials() {
    Modal.confirm({
      title: '重置 API 凭证',
      content: '将轮换 HMAC 签名密钥对，旧凭证立即失效（其他已登录设备需重新登录）。确定？',
      okText: '确定轮换',
      cancelText: '取消',
      async onOk() {
        try {
          const d = await resetCredentials()
          setCredentials(d.api_key, d.api_secret)
          message.success('API 凭证已轮换')
        } catch {
          /* 错误已在拦截器中提示 */
        }
      },
    })
  }

  async function onReset() {
    Modal.confirm({
      title: '重置账户',
      content: '将清空全部虚拟持仓与余额，恢复为初始 10,000 USDT。历史流水保留。确定继续？',
      okText: '确定重置',
      okType: 'danger',
      cancelText: '取消',
      async onOk() {
        try {
          await resetAccount()
          message.success('账户已重置为初始状态')
          navigate('/assets')
        } catch {
          /* 错误已在拦截器中提示 */
        }
      },
    })
  }

  const userMenu = {
    items: [
      { key: 'username', icon: <UserOutlined />, label: user?.username, disabled: true },
      { type: 'divider' as const },
      { key: 'resetCred', icon: <KeyOutlined />, label: t('resetCred', lang), onClick: onResetCredentials },
      { key: 'reset', icon: <ReloadOutlined />, label: t('resetAccount', lang), onClick: onReset },
      {
        key: 'logout',
        icon: <LogoutOutlined />,
        label: t('logout', lang),
        onClick: () => {
          logout()
          navigate('/login')
        },
      },
    ],
  }

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider collapsible collapsed={collapsed} onCollapse={setCollapsed}>
        <div
          style={{
            color: '#00c8a0',
            fontWeight: 700,
            fontSize: collapsed ? 14 : 18,
            textAlign: 'center',
            padding: '18px 0',
            cursor: 'pointer',
          }}
          onClick={() => navigate('/markets')}
        >
          {collapsed ? 'CS' : 'CryptoSim'}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={[
            { key: '/markets', icon: <BarChartOutlined />, label: t('nav', lang) },
            { key: '/spot', icon: <SwapOutlined />, label: t('navSpot', lang) },
            { key: '/futures', icon: <RiseOutlined />, label: t('navFutures', lang) },
            { key: '/assets', icon: <WalletOutlined />, label: t('navAssets', lang) },
            { key: '/learn', icon: <BookOutlined />, label: t('navLearn', lang) },
          ]}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header
          style={{
            padding: '0 24px',
            background: colorBgContainer,
            display: 'flex',
            justifyContent: 'flex-end',
            alignItems: 'center',
          }}
        >
          <Button type="text" onClick={toggle} style={{ marginRight: 8 }}>
            {lang === 'zh' ? 'EN' : '中'}
          </Button>
          <Dropdown menu={userMenu} trigger={['click']}>
            <Button type="text" icon={<UserOutlined />}>
              {user?.username}
            </Button>
          </Dropdown>
        </Header>
        <Content style={{ margin: 16 }}>
          <div
            style={{
              padding: 24,
              minHeight: 360,
              background: colorBgContainer,
              borderRadius: borderRadiusLG,
              border: '1px solid #252930',
            }}
          >
            <Outlet />
          </div>
        </Content>
      </Layout>
    </Layout>
  )
}
