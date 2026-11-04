import { expect, test } from '@playwright/test'
import {
  login,
  registerAndVerify,
  uniqueSuffix,
} from './support'

test('collections paginate three at a time with stable card heights', async ({
  page,
  request,
}) => {
  test.setTimeout(120_000)

  const suffix = uniqueSuffix()
  const email = `e2e-pagination-${suffix}@example.com`
  await registerAndVerify(page, request, { name: 'Paginación E2E', email })
  await login(page, email)

  for (let index = 1; index <= 4; index += 1) {
    await page.goto('/personal')
    await page.getByRole('button', { name: 'Nueva lista' }).click()
    const dialog = page.getByRole('dialog')
    await dialog.getByLabel('Nombre').fill(`Pag ${suffix} ${index}`)
    await dialog.getByRole('button', { name: 'Crear' }).click()
    await expect(
      page.getByRole('link', { name: new RegExp(`Pag ${suffix} ${index}`) }),
    ).toBeVisible()
  }
  for (let index = 1; index <= 4; index += 1) {
    await page.goto('/groups')
    await page.getByRole('button', { name: 'Nuevo grupo' }).click()
    const dialog = page.getByRole('dialog')
    await dialog.getByLabel('Nombre').fill(`Grupo ${suffix} ${index}`)
    await dialog.getByRole('button', { name: 'Crear' }).click()
    await expect(dialog).toBeHidden()
  }

  await page.goto('/personal')
  const compactRows = page.locator('[class*=compactRow]')
  await expect(compactRows).toHaveCount(3)
  const height = await compactRows
    .first()
    .evaluate((node) => Math.round(node.getBoundingClientRect().height))
  expect(height).toBe(80)

  const listPagination = page.getByRole('navigation', {
    name: 'Paginación de mis listas',
  })
  const pageSizeTrigger = listPagination.getByRole('button', {
    name: 'Por página',
  })
  await listPagination.getByRole('button', { name: 'Siguiente' }).click()
  await expect(compactRows).toHaveCount(1)
  await listPagination.getByRole('button', { name: 'Anterior' }).click()
  await expect(compactRows).toHaveCount(3)

  // Changing the page size from a later page must return to the first page.
  await listPagination.getByRole('button', { name: 'Siguiente' }).click()
  await expect(compactRows).toHaveCount(1)
  await pageSizeTrigger.click()
  await page.getByRole('option', { name: '5', exact: true }).click()
  await expect(compactRows).toHaveCount(4)
  await expect(pageSizeTrigger).toHaveText('5')
  await expect(
    listPagination.getByRole('button', { name: 'Anterior' }),
  ).toBeDisabled()
  await expect(
    listPagination.getByRole('button', { name: 'Siguiente' }),
  ).toBeDisabled()

  await pageSizeTrigger.click()
  await page.getByRole('option', { name: '3', exact: true }).click()
  await expect(compactRows).toHaveCount(3)

  await page.goto('/groups')
  await expect(page.getByRole('link', { name: new RegExp(`Grupo ${suffix} 1\\b`) })).toBeVisible()
  const groupPagination = page.getByRole('navigation', {
    name: 'Paginación de grupos',
  })
  await groupPagination.getByRole('button', { name: 'Siguiente' }).click()
  await expect(page.getByRole('link', { name: new RegExp(`Grupo ${suffix} 4\\b`) })).toBeVisible()
  await groupPagination.getByRole('button', { name: 'Anterior' }).click()
  await expect(page.getByRole('link', { name: new RegExp(`Grupo ${suffix} 1\\b`) })).toBeVisible()

  await page.goto('/')
  await expect(page.getByRole('navigation', { name: 'Paginación de grupos' })).toBeVisible()
  await expect(
    page.getByRole('navigation', { name: 'Paginación de mis listas' }),
  ).toBeVisible()
  expect(
    await page.locator('[class*=compactRow]').first().evaluate((node) =>
      Math.round(node.getBoundingClientRect().height),
    ),
  ).toBe(80)

  await expect(
    page.getByRole('navigation', { name: 'Paginación de actividad reciente' }),
  ).toBeVisible()
  const activityItems = page.locator('[class*=timelineItem]')
  await expect(activityItems).toHaveCount(3)
  await page
    .getByRole('navigation', { name: 'Paginación de actividad reciente' })
    .getByRole('button', { name: 'Siguiente' })
    .click()
  await expect(activityItems).toHaveCount(1)
  await expect(
    page
      .getByRole('navigation', { name: 'Paginación de actividad reciente' })
      .getByRole('button', { name: 'Anterior' }),
  ).toBeEnabled()
})
