import { expect, type APIRequestContext, type Page } from '@playwright/test'

const mailpit = process.env.CESTIA_MAILPIT_URL ?? 'http://localhost:8025'
export const password = 'correct-horse-battery'

export function uniqueSuffix() {
  return `${Date.now()}-${Math.random().toString(16).slice(2, 8)}`
}

export async function waitForEmailToken(
  request: APIRequestContext,
  email: string,
  subject: string,
) {
  let token = ''
  await expect.poll(async () => {
    const list = await request.get(`${mailpit}/api/v1/messages`)
    if (!list.ok()) return ''
    const body = (await list.json()) as {
      messages?: Array<{ ID: string; Subject: string; To?: Array<{ Address: string }> }>
    }
    const message = body.messages?.find(
      (item) => item.Subject.includes(subject) && item.To?.some((to) => to.Address === email),
    )
    if (!message) return ''
    const detail = await request.get(`${mailpit}/api/v1/message/${message.ID}`)
    if (!detail.ok()) return ''
    const text = (await detail.json()) as { Text?: string }
    token = text.Text?.match(/token=([A-Za-z0-9_-]+)/)?.[1] ?? ''
    return token
  }, { timeout: 20_000 }).not.toBe('')
  return token
}

export async function registerAndVerify(
  page: Page,
  request: APIRequestContext,
  input: { name: string; email: string },
) {
  await page.goto('/sign-up')
  await page.getByLabel('Nombre').fill(input.name)
  await page.getByLabel('Correo').fill(input.email)
  await page.getByLabel('Contraseña').fill(password)
  await page.getByRole('button', { name: 'Crear cuenta' }).click()
  await expect(page).toHaveURL(/\/verify-email/)

  const token = await waitForEmailToken(request, input.email, 'Verifica')
  await page.goto(`/verify-email?token=${token}`)
  await expect(page.getByText('Correo verificado. Ya puedes iniciar sesión.')).toBeVisible()
}

export async function login(page: Page, email: string) {
  await page.goto('/sign-in')
  await page.getByLabel('Correo').fill(email)
  await page.getByLabel('Contraseña').fill(password)
  await page.getByRole('button', { name: 'Iniciar sesión' }).click()
  await expect(page).toHaveURL(/\/$/)
}

export async function createPersonalList(page: Page, name: string) {
  await page.goto('/personal')
  await page.getByRole('button', { name: 'Nueva lista' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('Nombre').fill(name)
  await dialog.getByRole('button', { name: 'Crear' }).click()
  await expect(page.getByRole('link', { name: new RegExp(name) })).toBeVisible()
  await page.getByRole('link', { name: new RegExp(name) }).click()
  await expect(page).toHaveURL(/\/lists\//)
}

export async function addItem(
  page: Page,
  input: { name: string; quantity: string; estimatedPrice: string },
) {
  await page.getByRole('button', { name: 'Agregar artículo' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('Nombre').fill(input.name)
  await dialog.getByLabel('Cantidad').fill(input.quantity)
  await dialog.getByLabel('Precio unitario estimado').fill(input.estimatedPrice)
  await dialog.getByRole('button', { name: 'Guardar' }).click()
  await expect(page.getByText(input.name, { exact: true })).toBeVisible()
}

export async function purchaseItem(
  page: Page,
  input: { price: string; payer?: string },
) {
  await page.getByRole('button', { name: 'Registrar compra' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('Precio unitario real').fill(input.price)
  if (input.payer) {
    await dialog.getByRole('button', { name: 'Pagador' }).click()
    await page.getByRole('option', { name: input.payer }).click()
  }
  await dialog.getByRole('button', { name: 'Registrar compra' }).click()
  await expect(dialog).toBeHidden()
  await expect(page.locator('span').filter({ hasText: /^Comprado$/ }).first()).toBeVisible()
}

export async function createGroup(page: Page, name: string) {
  await page.goto('/groups')
  await page.getByRole('button', { name: 'Nuevo grupo' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('Nombre').fill(name)
  await dialog.getByRole('button', { name: 'Crear' }).click()
  await expect(page.getByRole('link', { name: new RegExp(name) })).toBeVisible()
  await page.getByRole('link', { name: new RegExp(name) }).click()
  await expect(page).toHaveURL(/\/groups\//)
}

export async function createGroupList(page: Page, name: string) {
  await page.getByRole('button', { name: 'Nueva lista' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('Nombre').fill(name)
  await dialog.getByRole('button', { name: 'Crear' }).click()
  await expect(page.getByRole('link', { name: new RegExp(name) })).toBeVisible()
  await page.getByRole('link', { name: new RegExp(name) }).click()
  await expect(page).toHaveURL(/\/lists\//)
}
