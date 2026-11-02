import { expect, test } from '@playwright/test'
import {
  addItem,
  createPersonalList,
  login,
  purchaseItem,
  registerAndVerify,
  uniqueSuffix,
} from './support'

test('personal purchase flow stores decimal expense and buyer/payer', async ({ page, request }, testInfo) => {
  const suffix = uniqueSuffix()
  const email = `e2e-personal-${suffix}@example.com`
  const listName = `Lista E2E ${suffix}`

  await registerAndVerify(page, request, { name: 'Compradora E2E', email })
  await login(page, email)

  await createPersonalList(page, listName)
  await addItem(page, { name: 'Leche', quantity: '3', estimatedPrice: '0.10' })
  await purchaseItem(page, { price: '0.10' })

  await expect(page.getByText('$0.30').first()).toBeVisible()

  if (testInfo.project.name === 'mobile') {
    const card = page.locator('[class*=itemRow]').first()
    const content = card.locator('[class*=rowMain]').first()
    const actions = card.locator('[class*=itemRowActions]').first()
    const [contentBox, actionBox, actionsDirection] = await Promise.all([
      content.boundingBox(),
      actions.boundingBox(),
      actions.evaluate((node) => getComputedStyle(node).flexDirection),
    ])
    expect(actionsDirection).toBe('row')
    expect(actionBox!.y).toBeGreaterThan(contentBox!.y + contentBox!.height)
  }

  await page.goto('/expenses')
  await expect(page.getByText('Gastado real').first()).toBeVisible()
  await expect(page.getByText('$0.30').first()).toBeVisible()
  await expect(page.getByText('No hay balances, deudas ni liquidaciones. Los importes son aportaciones reales.')).toBeVisible()

  const payerRow = page.locator('[class*=payerRow]').first()
  await expect(payerRow).toBeVisible()
  const [rowBox, barBox] = await Promise.all([
    payerRow.boundingBox(),
    payerRow.locator('[class*=bar]').first().boundingBox(),
  ])
  expect(Math.round(barBox!.width)).toBe(Math.round(rowBox!.width))

  await page.goto('/history')
  const historyRow = page.locator('[class*=historyRow]').first()
  await expect(historyRow.getByText('Compró Compradora E2E', { exact: true })).toBeVisible()
  await expect(historyRow.getByText('Pagó Compradora E2E', { exact: true })).toBeVisible()
  await expect(historyRow.getByText('Registró Compradora E2E', { exact: true })).toBeVisible()
  const [historyBox, contentBox, priceBox] = await Promise.all([
    historyRow.boundingBox(),
    historyRow.locator('[class*=rowMain]').first().boundingBox(),
    historyRow.locator('[class*=itemPrice]').first().boundingBox(),
  ])
  if (testInfo.project.name === 'mobile') {
    expect(priceBox!.y).toBeGreaterThanOrEqual(contentBox!.y + contentBox!.height)
    expect(Math.abs(priceBox!.x - contentBox!.x)).toBeLessThanOrEqual(1)
  } else {
    const centerDelta = Math.abs(
      (historyBox!.y + historyBox!.height / 2) - (priceBox!.y + priceBox!.height / 2),
    )
    expect(centerDelta).toBeLessThanOrEqual(2)
    expect(priceBox!.x).toBeGreaterThan(historyBox!.x + historyBox!.width / 2)
  }
})
