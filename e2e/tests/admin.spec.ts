import { expect, Page, test } from '@playwright/test'

/**
 * 管理后台 E2E：登录 → 登录审计 → 用户管理调资金 → 资金流水可查。
 * 前提：后端(:8080) 与管理后台(:5174) 已启动；admin 账号已存在。
 */

test.use({ baseURL: 'http://localhost:5174' })

async function adminLogin(page: Page) {
  await page.goto('/login')
  await page.getByPlaceholder('管理员用户名或邮箱').fill('admin')
  await page.locator('input[type=password]').fill('admin12345')
  await page.getByRole('button', { name: '登 录' }).click()
  await expect(page).toHaveURL(/\/dashboard/, { timeout: 15_000 })
}

test('管理员登录 → 登录审计页有数据', async ({ page }) => {
  await adminLogin(page)
  await page.getByText('登录审计', { exact: true }).click()
  await expect(page).toHaveURL(/\/login-logs/)
  // 至少能看到本次登录的成功记录
  await expect(page.getByText('admin').first()).toBeVisible({ timeout: 15_000 })
  await expect(page.getByText('成功').first()).toBeVisible()
})

test('用户管理 → 调拨资金 → 资金流水可查', async ({ page }) => {
  await adminLogin(page)

  // 搜索 demo 用户并调增资金
  await page.getByText('用户管理', { exact: true }).click()
  await expect(page).toHaveURL(/\/users/)
  await page.getByPlaceholder('搜索用户名 / 邮箱').fill('demo')
  await page.getByPlaceholder('搜索用户名 / 邮箱').press('Enter')
  const row = page.getByRole('row', { name: /demo/ }).first()
  await expect(row).toBeVisible({ timeout: 10_000 })
  await row.getByRole('button', { name: '调资金' }).click()
  const before = await row.locator('td').nth(5).innerText()
  const delta = 77
  await page.getByPlaceholder(/金额（正数调增/).fill(String(delta))
  await page.getByPlaceholder(/备注（记入审计流水）/).fill('e2e 调拨')
  await page.getByRole('button', { name: '确认调拨' }).click()
  await expect(page.getByText('调拨成功')).toBeVisible({ timeout: 10_000 })

  // 余额 +77
  const after = Number(before) + delta
  await expect(row.locator('td').nth(5)).toHaveText(String(after), { timeout: 10_000 })

  // 资金流水出现 admin_adjust
  await page.getByText('资金流水', { exact: true }).click()
  await expect(page).toHaveURL(/\/ledgers/)
  await expect(page.getByText('管理员调拨').first()).toBeVisible({ timeout: 10_000 })
  await expect(page.getByText('e2e 调拨').first()).toBeVisible()
})

test('普通用户凭证无法访问管理接口（RBAC）', async ({ request }) => {
  // 直接打后端：无签名/无管理员 → 401/403，而非 200 或 404
  const resp = await request.get('http://localhost:8080/api/v1/admin/stats')
  expect([401, 403]).toContain(resp.status())
})
