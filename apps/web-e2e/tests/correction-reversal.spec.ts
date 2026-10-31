import { expect, test } from '@playwright/test'
import {
  addItem,
  createPersonalList,
  login,
  purchaseItem,
  registerAndVerify,
  uniqueSuffix,
} from './support'

test('correction, unknown payer and reversal preserve history', async ({ page, request }, testInfo) => {
  test.skip(testInfo.project.name === 'mobile', 'Desktop covers the full correction workflow.')
  test.setTimeout(90_000)

  const suffix = uniqueSuffix()
  const email = `e2e-correction-${suffix}@example.com`

  await registerAndVerify(page, request, { name: 'Correctora E2E', email })
  await login(page, email)
  await createPersonalList(page, `Correcciones ${suffix}`)
  const listUrl = page.url()
  await addItem(page, { name: 'Arroz', quantity: '1', estimatedPrice: '10' })
  await purchaseItem(page, { price: '10' })
  await expect(page.getByText('$10.00').first()).toBeVisible()

  await page.getByRole('button', { name: 'Corregir compra' }).click()
  let dialog = page.getByRole('dialog')
  await dialog.getByLabel('Precio unitario real').fill('20')
  await dialog.getByLabel('Motivo de la corrección').fill('Precio corregido E2E')
  await dialog.getByRole('button', { name: 'Guardar' }).click()
  await expect(dialog).toBeHidden()
  await expect(page.getByText('$20.00').first()).toBeVisible()

  await page.getByRole('button', { name: 'Corregir compra' }).click()
  dialog = page.getByRole('dialog')
  await dialog.getByLabel('Pagador').selectOption({ label: 'No identificado' })
  await dialog.getByLabel('Motivo de la corrección').fill('Pagador no identificado E2E')
  await dialog.getByRole('button', { name: 'Guardar' }).click()
  await expect(dialog).toBeHidden()

  await page.goto('/expenses')
  await expect(page.getByText('Sin pagador identificado').first()).toBeVisible()
  await expect(page.getByText('$20.00').first()).toBeVisible()

  await page.goto(listUrl)
  await page.getByRole('button', { name: 'Revertir compra' }).click()
  dialog = page.getByRole('dialog')
  await dialog.getByLabel('Motivo de la reversión').fill('Reversión E2E')
  await dialog.getByRole('button', { name: 'Revertir compra' }).click()
  await expect(dialog).toBeHidden()
  await expect(page.locator('span').filter({ hasText: /^Pendiente$/ }).first()).toBeVisible()

  await page.goto('/history')
  await expect(page.getByText('Revertida')).toBeVisible()
  await expect(page.getByText('Motivo: Reversión E2E')).toBeVisible()
  await page.getByRole('button', { name: 'Actividad' }).click()
  await expect(page.getByText('Compra corregida').first()).toBeVisible()
})
