import React from 'react'
import ReactDOM from 'react-dom/client'
import { ConfigProvider, theme as antdTheme } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import App from './App'
import { C } from './theme'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ConfigProvider
      locale={zhCN}
      theme={{
        algorithm: antdTheme.darkAlgorithm,
        token: {
          colorPrimary: C.accent,
          colorBgBase: C.bg,
          colorBgContainer: C.surface,
          colorBgElevated: C.surface2,
          colorBorder: C.border,
          colorBorderSecondary: C.border,
          colorText: C.text,
          colorTextSecondary: C.text2,
          colorLink: C.accent,
          borderRadius: 8,
          fontFamily: "-apple-system, 'PingFang SC', 'Microsoft YaHei', sans-serif",
        },
        components: {
          Layout: { siderBg: C.bg, headerBg: C.surface, bodyBg: C.bg },
          Menu: {
            darkItemBg: C.bg,
            darkItemSelectedBg: 'rgba(0,200,160,0.15)',
            darkItemSelectedColor: C.accent,
          },
          Table: { headerBg: C.surface2, rowHoverBg: C.surface2 },
        },
      }}
    >
      <App />
    </ConfigProvider>
  </React.StrictMode>,
)
