import { expect, test } from '@playwright/test'
import {
  createGroup,
  createPersonalList,
  login,
  registerAndVerify,
  uniqueSuffix,
} from './support'

test('custom controls keep the dashboard, dialogs and date pickers consistent', async ({
  page,
  request,
}, testInfo) => {
  const suffix = uniqueSuffix()
  const email = `e2e-controls-${suffix}@example.com`

  await registerAndVerify(page, request, { name: 'Controles E2E', email })
  await login(page, email)
  await createPersonalList(page, `Controles ${suffix}`)

  await page.goto('/')
  const rows = page.locator('[class*=compactRow]')
  await expect(rows.first()).toBeVisible()
  const heights = await rows.evaluateAll((nodes) =>
    nodes.map((node) => Math.round(node.getBoundingClientRect().height)),
  )
  expect(new Set(heights).size).toBe(1)

  if (testInfo.project.name === 'mobile') {
    const actionBoxes = await page
      .locator('[class*=actionsGrid] > a')
      .evaluateAll((nodes) =>
        nodes.map((node) => {
          const box = node.getBoundingClientRect()
          return {
            left: box.left,
            right: box.right,
            width: box.width,
            decoration: getComputedStyle(node).textDecorationLine,
          }
        }),
      )
    expect(Math.round(actionBoxes[1].left - actionBoxes[0].right)).toBe(8)
    expect(Math.round(actionBoxes[0].width)).toBe(Math.round(actionBoxes[1].width))
    expect(actionBoxes.every((box) => box.decoration === 'none')).toBe(true)

    const mobileNav = page.locator('nav[class*=mobileNav] a')
    const navLinks = await mobileNav.evaluateAll((nodes) =>
      nodes.map((node) => ({
        label: node.getAttribute('aria-label'),
        text: node.textContent?.trim(),
        height: node.getBoundingClientRect().height,
      })),
    )
    expect(navLinks.map((link) => link.label)).toEqual([
      'Resumen',
      'Mis listas',
      'Grupos',
      'Gastos',
      'Historial',
    ])
    expect(navLinks.every((link) => link.text === '')).toBe(true)
    expect(navLinks.every((link) => link.height >= 44)).toBe(true)
  }

  await page.goto('/personal')
  await page.getByRole('link', { name: new RegExp(`Controles ${suffix}`) }).click()
  await page.getByRole('button', { name: 'Agregar artículo' }).click()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await expect(dialog.getByLabel('Asignado a')).toHaveCount(0)
  await expect(dialog.getByLabel('Precio unitario estimado')).toBeVisible()
  await page.locator('[class*=overlay]').click({ position: { x: 8, y: 8 } })
  await expect(dialog).toBeHidden()

  await page.goto('/expenses')
  const from = page.getByLabel('Desde')
  await from.fill('31102026')
  await expect(from).toHaveValue('31/10/2026')

  await page.getByRole('button', { name: 'Abrir calendario' }).first().click()
  const calendar = page.getByRole('dialog', { name: 'Calendario' })
  await expect(calendar).toBeVisible()

  await page.getByRole('button', { name: 'Mes', exact: true }).click()
  await page.getByRole('option', { name: 'diciembre' }).click()
  await expect(
    calendar.getByRole('button', { name: '25 de diciembre de 2026' }),
  ).toBeVisible()

  await page.getByRole('button', { name: 'Año', exact: true }).click()
  await page.getByRole('option', { name: '2028' }).click()
  await expect(
    calendar.getByRole('button', { name: '25 de diciembre de 2028' }),
  ).toBeVisible()

  await page.keyboard.press('Escape')
  await expect(calendar).toBeHidden()
})

test('group owner actions stack without overlap on mobile', async ({
  page,
  request,
}, testInfo) => {
  test.skip(testInfo.project.name !== 'mobile', 'Mobile layout check')

  const suffix = uniqueSuffix()
  const email = `e2e-group-actions-${suffix}@example.com`

  await registerAndVerify(page, request, { name: 'Grupo E2E', email })
  await login(page, email)
  await createGroup(page, `Acciones ${suffix}`)

  const actions = page.locator('[class*=actionsGrid] > *')
  await expect(actions).toHaveCount(3)
  const boxes = await actions.evaluateAll((nodes) =>
    nodes.map((node) => {
      const box = node.getBoundingClientRect()
      const button = node.tagName === 'BUTTON' ? node : node.querySelector('button')!
      const style = getComputedStyle(button)
      return {
        top: box.top,
        width: box.width,
        height: box.height,
        hasOverflow: node.scrollWidth > node.clientWidth,
        background: style.backgroundColor,
        borderWidth: style.borderTopWidth,
      }
    }),
  )

  expect(boxes[0].top).toBe(boxes[1].top)
  expect(Math.round(boxes[2].top - (boxes[0].top + boxes[0].height))).toBe(8)
  expect(
    Math.abs(boxes[2].width - (boxes[0].width + boxes[1].width + 8)),
  ).toBeLessThanOrEqual(1)
  expect(boxes.every((box) => box.height >= 44)).toBe(true)
  expect(boxes.every((box) => !box.hasOverflow)).toBe(true)
  expect(boxes.every((box) => box.background !== 'rgba(0, 0, 0, 0)')).toBe(true)
  expect(boxes.every((box) => box.borderWidth === '1px')).toBe(true)
})
