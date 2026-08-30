import { expect, Page, test } from '@playwright/test'

/**
 * 核心链路 E2E：注册 → 资产 → 现货下单 → 当前委托 → 撤单 → 市价成交 → 学习中心。
 * 前提：后端(:8080) 与用户端(:5173) 已启动。
 * 每个用例注册独立账号（Playwright 重试会换 worker 重新求值模块，账号必须在用例内部生成）。
 */

const password = 'e2e-password-1'

async function registerAndLogin(page: Page): Promise<string> {
  const rand = Date.now() + Math.floor(Math.random() * 100000)
  const email = `e2e-${rand}@cryptosim.dev`
  const username = `e2e${rand}`.slice(0, 20)
  await page.goto('/register')
  await page.getByPlaceholder('you@example.com').fill(email)
  await page.getByPlaceholder('3-20 位字母、数字或下划线').fill(username)
  await page.getByPlaceholder('至少 8 位').fill(password)
  await page.getByPlaceholder('再次输入密码').fill(password)
  await page.getByRole('button', { name: '注册并领取虚拟资金' }).click()
  await expect(page).toHaveURL(/\/assets/, { timeout: 15_000 })
  return email
}

async function login(page: Page, email: string) {
  await page.goto('/login')
  await page.getByPlaceholder('邮箱或用户名').fill(email)
  await page.locator('input[type=password]').fill(password)
  await page.getByRole('button', { name: '登 录' }).click()
  await expect(page).toHaveURL(/\/markets/, { timeout: 15_000 })
}

test('注册并领取 10,000 虚拟 USDT', async ({ page }) => {
  await registerAndLogin(page)
  await expect(page.getByText('10000').first()).toBeVisible()
})

test('现货限价下单 → 出现在当前委托 → 撤单', async ({ page }) => {
  const email = await registerAndLogin(page)
  await login(page, email)

  await page.goto('/spot?symbol=BTCUSDT')
  await page.getByPlaceholder(/数量（BTC/).waitFor()

  // 挂一个远低于市价的买单（不会被成交，用于验证挂单/撤单）
  await page.getByRole('button', { name: '买入' }).click()
  await page.getByPlaceholder(/价格（最新/).fill('60000')
  await page.getByPlaceholder(/数量（BTC/).fill('0.001')
  await page.getByRole('button', { name: '买入 BTC' }).click()

  // 出现在当前委托
  await expect(page.getByRole('tab', { name: /当前委托/ })).toContainText('1', {
    timeout: 15_000,
  })
  await expect(page.getByText(/等待成交/).first()).toBeVisible()

  // 撤单：行内「撤销」链接 → 确认弹窗
  await page.getByText('撤销', { exact: true }).first().click()
  await page.getByRole('button', { name: /撤\s*销/ }).last().click()
  await expect(page.getByText('已撤销').first()).toBeVisible({ timeout: 15_000 })
})

test('市价买入立即成交', async ({ page }) => {
  const email = await registerAndLogin(page)
  await login(page, email)

  await page.goto('/spot?symbol=BTCUSDT')
  await page.getByRole('tab', { name: '市价单' }).click() // 切到市价单
  await page.getByPlaceholder(/数量（BTC/).waitFor()
  await page.getByPlaceholder(/数量（BTC/).fill('0.001')
  await page.getByRole('button', { name: '买入 BTC' }).click()

  // 功能性断言：成交后资产页出现 BTC 持仓
  await page.goto('/assets')
  await expect(page.getByText('BTC', { exact: true })).toBeVisible({ timeout: 15_000 })
})

test('学习中心内容可访问', async ({ page }) => {
  const email = await registerAndLogin(page)
  await login(page, email)

  await page.goto('/learn')
  await expect(page.getByRole('tab', { name: '币种百科' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'BTC 比特币（Bitcoin）' })).toBeVisible({
    timeout: 20_000,
  })
  await page.getByRole('tab', { name: '术语词典' }).click()
  await expect(page.getByText('稳定币').first()).toBeVisible({ timeout: 20_000 })
})
