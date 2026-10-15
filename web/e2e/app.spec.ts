import { expect, test } from '@playwright/test'

test.describe('pages', () => {
  test('feature map shows areas, features and the review summary', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('heading', { level: 1, name: 'Acme Expense' })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Claims', exact: true })).toBeVisible()
    await expect(page.getByRole('link', { name: /Expense claims/ }).first()).toBeVisible()
    await expect(page.getByText('of 12 pages approved')).toBeVisible()

    await page.getByRole('button', { name: 'Approver', exact: true }).click()
    await expect(page).toHaveURL(/role=approver/)
    await expect(page.getByRole('heading', { name: 'Account', exact: true })).toHaveCount(1)
  })

  test('feature page lists its screens, flows, modules and terms', async ({ page }) => {
    await page.goto('/features/expense-claims')
    await expect(page.getByRole('heading', { level: 1, name: 'Expense claims' })).toBeVisible()
    const main = page.locator('main')
    await expect(main.getByRole('link', { name: /My claims/ })).toBeVisible()
    await expect(main.getByRole('link', { name: 'Submit an expense claim' })).toBeVisible()
    await expect(main.getByRole('link', { name: 'Claim pages' })).toBeVisible()
    await expect(main.getByRole('link', { name: 'Expense claim', exact: true })).toBeVisible()
    // The mermaid block in the body renders as an SVG diagram.
    await expect(main.locator('.md svg').first()).toBeVisible({ timeout: 15_000 })
  })

  test('screen page shows hotspots, variants and what is behind the screen', async ({ page }) => {
    await page.goto('/screens/claim-list')
    await expect(page.getByRole('heading', { level: 1, name: 'My claims' })).toBeVisible()
    await expect(page.getByRole('button', { name: '1. Status filter' })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Behind the screen' })).toBeVisible()

    await page.getByRole('tab', { name: 'No matching claims' }).click()
    await expect(page).toHaveURL(/v=empty/)
    await expect(page.getByRole('button', { name: '1. Empty state' })).toBeVisible()
  })

  test('#h<n> anchor highlights that hotspot in the list and on the image', async ({ page }) => {
    await page.goto('/screens/claim-list#h1')
    await expect(page.locator('[data-callout="1"]')).toHaveAttribute('data-active', 'true')
    await expect(page.getByRole('button', { name: '1. Status filter' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
  })

  test('flow page shows steps, sample values and a keyboard tour', async ({ page }) => {
    await page.goto('/flows/submit-claim')
    await expect(
      page.getByRole('heading', { level: 1, name: 'Submit an expense claim' }),
    ).toBeVisible()
    await expect(page.getByRole('button', { name: 'Copy Amount (¥)' })).toBeVisible()

    await page.getByRole('button', { name: 'Start tour' }).click()
    const tour = page.getByRole('dialog', { name: 'Start tour' })
    await expect(tour.getByText('1 / 4')).toBeVisible()
    await page.keyboard.press('ArrowRight')
    await expect(tour.getByText('2 / 4')).toBeVisible()
    await page.keyboard.press('ArrowLeft')
    await expect(tour.getByText('1 / 4')).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(tour).toBeHidden()
  })

  test('module page shows dependencies and users', async ({ page }) => {
    await page.goto('/modules/web-claims')
    await expect(page.getByRole('heading', { level: 1, name: 'Claim pages' })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Depends on' })).toBeVisible()
    await expect(page.locator('main').getByRole('link', { name: 'Data and session' })).toBeVisible()
  })

  test('glossary filters terms', async ({ page }) => {
    await page.goto('/glossary')
    await expect(page.getByRole('cell', { name: 'Receipt', exact: true })).toBeVisible()
    await page.getByLabel('Filter terms').fill('approver sends')
    await expect(page.getByRole('cell', { name: 'Receipt', exact: true })).toBeHidden()
    await expect(page.getByRole('cell', { name: 'Return', exact: true })).toBeVisible()
  })

  test('graph opens a page when a node is clicked', async ({ page }) => {
    await page.goto('/graph')
    await page.getByText('Module: Claim pages').click()
    await expect(page).toHaveURL(/\/modules\/web-claims$/)
  })

  test('Ctrl+K search opens a result', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('heading', { level: 1, name: 'Acme Expense' })).toBeVisible()
    await page.keyboard.press('Control+k')
    await page.getByRole('dialog', { name: 'Search' }).getByRole('searchbox').fill('waiting')
    await page.keyboard.press('Enter')
    await expect(page).toHaveURL(/\/screens\/approval-queue$/)
  })

  test('content language falls back with an untranslated marker', async ({ page }) => {
    await page.goto('/screens/claim-list')
    await page.getByLabel('Content language').selectOption('vi')
    await expect(page.getByRole('heading', { level: 1, name: 'Đề nghị của tôi' })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Mục đích' })).toBeVisible()
    await page.getByLabel('Content language').selectOption('ja')
    await expect(page.getByText(/Not translated; shown in EN/).first()).toBeAttached()
    await page.getByLabel('Content language').selectOption('en')
  })
})

test('clicking a hotspot with a target opens that screen', async ({ page }) => {
  await page.goto('/screens/claim-list')
  await page.getByRole('link', { name: /^2\. New claim/ }).click()
  await expect(page).toHaveURL(/\/screens\/claim-form$/)
  await expect(page.getByRole('heading', { level: 1, name: 'New claim' })).toBeVisible()
})

test('approving from the page updates the badge', async ({ page }) => {
  await page.goto('/modules/app-data')
  const badge = page.locator('header [data-testid="status-badge"]').first()
  await expect(badge).toHaveAttribute('data-state', 'draft')

  await page.getByRole('button', { name: 'Approve' }).click()
  const dialog = page.getByRole('dialog', { name: 'Who is reviewing?' })
  await dialog.getByLabel('Your name').fill('E2E Reviewer')
  await dialog.getByRole('button', { name: 'Continue' }).click()

  await expect(badge).toHaveAttribute('data-state', 'approved')
  await expect(page.getByText(/Approved by E2E Reviewer/)).toBeVisible()
  // The reviewer name is remembered and shown in the header.
  await expect(page.getByRole('button', { name: 'Reviewer: E2E Reviewer' })).toBeVisible()
})

test('a review recorded elsewhere reaches open pages through the live event', async ({
  page,
  request,
}) => {
  await page.goto('/review?state=all&kind=screen')
  const row = page.locator('[data-key="screen/sign-in"]')
  await expect(row.getByTestId('status-badge')).toHaveAttribute('data-state', 'draft')
  // Wait until the event stream is connected before changing the content.
  await page.waitForFunction(() =>
    performance.getEntriesByType('resource').some((e) => e.name.includes('/api/events')),
  )
  await page.evaluate(() => window.scrollTo(0, 120))
  const scrollBefore = await page.evaluate(() => window.scrollY)

  // Record the review outside this page, as `muninn review` or another browser would.
  const res = await request.post('/api/entities/screen/sign-in/review', {
    data: { status: 'approved', by: 'Someone else', note: '' },
  })
  expect(res.ok()).toBe(true)

  await expect(row.getByTestId('status-badge')).toHaveAttribute('data-state', 'approved')
  expect(await page.evaluate(() => window.scrollY)).toBe(scrollBefore)
})
