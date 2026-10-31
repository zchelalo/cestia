import { expect, test } from '@playwright/test'
import {
  addItem,
  createGroup,
  createGroupList,
  login,
  purchaseItem,
  registerAndVerify,
  uniqueSuffix,
  waitForEmailToken,
} from './support'

test('group invitation attributes purchase to buyer and payer', async ({ browser, request }, testInfo) => {
  test.skip(testInfo.project.name === 'mobile', 'Multi-user group flow runs on desktop.')
  test.setTimeout(120_000)

  const suffix = uniqueSuffix()
  const groupName = `Alacena E2E ${suffix}`
  const listName = `Súper E2E ${suffix}`
  const owner = { name: `Owner E2E ${suffix}`, email: `e2e-owner-${suffix}@example.com` }
  const member = { name: `Member E2E ${suffix}`, email: `e2e-member-${suffix}@example.com` }

  const ownerContext = await browser.newContext()
  const ownerPage = await ownerContext.newPage()
  await registerAndVerify(ownerPage, request, owner)
  await login(ownerPage, owner.email)

  await createGroup(ownerPage, groupName)
  const groupUrl = ownerPage.url()
  await createGroupList(ownerPage, listName)
  await addItem(ownerPage, { name: 'Arroz', quantity: '2', estimatedPrice: '30' })

  await ownerPage.goto(groupUrl)
  await ownerPage.getByRole('link', { name: 'Integrantes' }).click()
  await ownerPage.getByLabel('Correo de la persona').fill(member.email)
  await ownerPage.getByRole('button', { name: 'Invitar por correo' }).click()
  await expect(ownerPage.getByText('Invitación encolada. Se enviará por correo.')).toBeVisible()

  const memberContext = await browser.newContext()
  const memberPage = await memberContext.newPage()
  await registerAndVerify(memberPage, request, member)
  await login(memberPage, member.email)

  const invitationToken = await waitForEmailToken(request, member.email, 'Te invitaron')
  await memberPage.goto(`/invitations/accept?token=${invitationToken}`)
  await expect(memberPage.getByText(groupName)).toBeVisible()
  await memberPage.getByRole('button', { name: 'Aceptar invitación' }).click()
  await expect(memberPage).toHaveURL(new RegExp(`/groups/`))

  await memberPage.getByRole('link', { name: new RegExp(listName) }).click()
  await purchaseItem(memberPage, { price: '30', payer: owner.name })
  await expect(memberPage.getByText('$60.00').first()).toBeVisible()

  await memberPage.goto('/expenses')
  await memberPage.getByRole('button', { name: 'Espacio' }).click()
  await memberPage.getByRole('option', { name: groupName }).click()
  await expect(memberPage.getByText('$60.00').first()).toBeVisible()
  await expect(memberPage.getByText(owner.name)).toBeVisible()

  await memberPage.goto('/history')
  await memberPage.getByRole('button', { name: 'Espacio' }).click()
  await memberPage.getByRole('option', { name: groupName }).click()
  await expect(memberPage.getByText(new RegExp(`Compró ${member.name}.*pagó ${owner.name}`))).toBeVisible()

  await ownerContext.close()
  await memberContext.close()
})
