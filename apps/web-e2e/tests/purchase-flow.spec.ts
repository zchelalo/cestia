import { expect, test } from '@playwright/test'
import {
  addItem,
  createPersonalList,
  login,
  purchaseItem,
  registerAndVerify,
  uniqueSuffix,
} from './support'

test('personal purchase flow stores decimal expense and buyer/payer', async ({ page, request }) => {
  const suffix = uniqueSuffix()
  const email = `e2e-personal-${suffix}@example.com`
  const listName = `Lista E2E ${suffix}`

  await registerAndVerify(page, request, { name: 'Compradora E2E', email })
  await login(page, email)

  await createPersonalList(page, listName)
  await addItem(page, { name: 'Leche', quantity: '3', estimatedPrice: '0.10' })
  await purchaseItem(page, { price: '0.10' })

  await expect(page.getByText('$0.30').first()).toBeVisible()

  await page.goto('/expenses')
  await expect(page.getByText('Gastado real').first()).toBeVisible()
  await expect(page.getByText('$0.30').first()).toBeVisible()
  await expect(page.getByText('No hay balances, deudas ni liquidaciones. Los importes son aportaciones reales.')).toBeVisible()
})
