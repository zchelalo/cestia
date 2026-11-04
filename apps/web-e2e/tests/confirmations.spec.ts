import { expect, test } from '@playwright/test'
import {
  addItem,
  createPersonalList,
  login,
  registerAndVerify,
  uniqueSuffix,
} from './support'

test('destructive actions use the custom confirmation modal', async ({
  page,
  request,
}) => {
  const suffix = uniqueSuffix()
  const email = `e2e-confirm-${suffix}@example.com`
  const itemName = `Producto ${suffix}`

  await registerAndVerify(page, request, { name: 'Confirmación E2E', email })
  await login(page, email)
  await createPersonalList(page, `Confirmación ${suffix}`)
  await addItem(page, {
    name: itemName,
    quantity: '1',
    estimatedPrice: '10',
  })

  let nativeDialog = false
  page.on('dialog', async (dialog) => {
    nativeDialog = true
    await dialog.dismiss()
  })

  await page.getByRole('button', { name: 'Eliminar' }).first().click()
  const dialog = page.getByRole('dialog', { name: 'Eliminar artículo' })
  await expect(dialog).toBeVisible()
  await expect(dialog).toContainText(itemName)

  await dialog.getByRole('button', { name: 'Cancelar' }).click()
  await expect(dialog).toBeHidden()
  await expect(page.getByText(itemName, { exact: true })).toBeVisible()

  await page.getByRole('button', { name: 'Eliminar' }).first().click()
  await expect(dialog).toBeVisible()
  await dialog.getByRole('button', { name: 'Eliminar' }).click()
  await expect(dialog).toBeHidden()
  await expect(page.getByText(itemName, { exact: true })).toBeHidden()
  expect(nativeDialog).toBe(false)
})
