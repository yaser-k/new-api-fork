// This session's screenshots (the final cleanup pass).
// Usage: node shots-final.mjs <base-url> <out-dir> setup|main|rt [lang]
//   setup: 200-setup-steps-rtl.png (run before /api/setup creates the admin)
//   main : 201..212-*-rtl.png in Persian
//   rt   : 201-channels-response-time-<lang>.png only (en, fr)
// Every capture waits until the content is fully opaque, then counts the
// colours of a sample of the image; fewer than 16 colours is BLANK.
import { chromium } from 'playwright'

const [base, out, mode, langArg] = process.argv.slice(2)
const PASSWORD = 'DemoPass-2026!'
const lang = langArg || 'fa'
const suffix = lang === 'fa' ? 'rtl' : lang
const LOCALES = { fa: 'fa-IR', en: 'en-US', fr: 'fr-FR' }

const browser = await chromium.launch({
  executablePath: '/opt/pw-browsers/chromium-1194/chrome-linux/chrome',
})
const context = await browser.newContext({
  viewport: { width: 1440, height: 900 },
  locale: LOCALES[lang] ?? 'en-US',
  timezoneId: 'UTC',
  colorScheme: 'light',
})
await context.addInitScript((l) => localStorage.setItem('i18nextLng', l), lang)
const page = await context.newPage()
const consoleErrors = []
page.on('console', (m) => m.type() === 'error' && consoleErrors.push(m.text()))
page.on('pageerror', (e) => consoleErrors.push(String(e)))

async function settle(ms = 1200) {
  await page.waitForLoadState('networkidle').catch(() => {})
  await page.waitForTimeout(ms)
}

async function waitOpaque(selector) {
  await page.waitForFunction(
    (sel) => {
      let el = document.querySelector(sel)
      if (!el) return false
      while (el) {
        if (getComputedStyle(el).opacity !== '1') return false
        el = el.parentElement
      }
      return true
    },
    selector,
    { timeout: 30000 }
  )
}

async function colourCount(buffer) {
  return page.evaluate(async (b64) => {
    const blob = await (await fetch(`data:image/png;base64,${b64}`)).blob()
    const bitmap = await createImageBitmap(blob)
    const canvas = new OffscreenCanvas(bitmap.width, bitmap.height)
    const ctx = canvas.getContext('2d')
    ctx.drawImage(bitmap, 0, 0)
    const data = ctx.getImageData(0, 0, bitmap.width, bitmap.height).data
    const colours = new Set()
    for (let i = 0; i < data.length; i += 16) {
      colours.add((data[i] << 16) | (data[i + 1] << 8) | data[i + 2])
    }
    return colours.size
  }, buffer.toString('base64'))
}

async function shot(name, options = {}) {
  const file = `${name}-${suffix}`
  const buffer = await page.screenshot({ path: `${out}/${file}.png`, ...options })
  const colours = await colourCount(buffer)
  console.log(`saved ${file}.png (${colours} colours${colours < 16 ? ', BLANK' : ''})`)
}

async function open(path, root = 'main', wait = 1800) {
  await page.goto(`${base}${path}`)
  await settle(wait)
  await waitOpaque(root)
}

// Visible texts matching a pattern inside a root, for the log.
async function texts(root, pattern) {
  return page.evaluate(
    ([sel, source]) => {
      const re = new RegExp(source)
      const found = new Set()
      const scope = document.querySelector(sel) ?? document.body
      const walker = document.createTreeWalker(scope, NodeFilter.SHOW_TEXT)
      while (walker.nextNode()) {
        const el = walker.currentNode.parentElement
        if (!el || (!el.offsetParent && getComputedStyle(el).position !== 'fixed')) continue
        const t = el.innerText?.replace(/\s+/g, ' ').trim() ?? ''
        if (t && t.length < 120 && re.test(t)) found.add(t)
      }
      return [...found].slice(0, 12)
    },
    [root, pattern.source]
  )
}

const show = (s) => s.replace(/[⁦-⁩‎]/g, (c) => `<U+${c.charCodeAt(0).toString(16)}>`)

async function signIn() {
  await page.goto(`${base}/sign-in`)
  await settle()
  await page.locator('input[name="username"], input#username').first().fill('admin')
  await page.locator('input[type="password"]').first().fill(PASSWORD)
  const consent = page.getByRole('checkbox').first()
  if (await consent.count()) await consent.click()
  await page.locator('button[type="submit"]').first().click()
  await page.waitForURL((url) => !url.pathname.startsWith('/sign-in'), { timeout: 20000 })
  await settle()
  const info = await page.evaluate(() => ({ dir: document.documentElement.dir, lang: document.documentElement.lang }))
  console.log(`page: dir=${info.dir} lang=${info.lang}`)
}

async function responseTimeColumn() {
  await open('/channels', 'main', 1500)
  const tableToggle = page.getByRole('radio', { name: /جدول|Table|tableau/ }).first()
  if (await tableToggle.count()) await tableToggle.click()
  else await page.getByRole('button', { name: /جدول|Table|tableau/ }).first().click()
  await settle(1500)
  await waitOpaque('main table')
  const cells = await page.evaluate(() =>
    [...document.querySelectorAll('main table tbody td')]
      .map((td) => td.innerText.trim())
      .filter((t) => /(ms|ثانیه|\d ?s)$/.test(t) && t.length < 20)
  )
  console.log(`response time cells: ${JSON.stringify(cells)}`)
  await page.evaluate(() => {
    const th = [...document.querySelectorAll('main table thead th')].find((el) =>
      /پاسخ|Response|réponse/i.test(el.textContent ?? '')
    )
    th?.scrollIntoView({ inline: 'center', block: 'nearest' })
  })
  await page.waitForTimeout(500)
  await shot('201-channels-response-time')
}

console.log(`browser: Chromium ${browser.version()}`)

if (mode === 'setup') {
  await open('/setup', 'body')
  const badges = await page.evaluate(() =>
    [...document.querySelectorAll('span.size-6')].map((el) => el.textContent)
  )
  console.log(`setup step badges: ${JSON.stringify(badges)}`)
  await shot('200-setup-steps')
} else if (mode === 'rt') {
  await signIn()
  await responseTimeColumn()
} else {
  await signIn()
  await responseTimeColumn()

  // System info: percentages, bytes, refresh interval, Solar Hijri titles.
  await open('/system-info', 'main', 2500)
  console.log(`system info numbers: ${JSON.stringify(await texts('main', /%|ثانیه/))}`)
  const titles = await page.evaluate(() =>
    [...document.querySelectorAll('main td[title]')].map((td) => td.title).filter((t) => /[۰-۹]{4}\//.test(t))
  )
  console.log(`system info date titles: ${JSON.stringify(titles.slice(0, 4))}`)
  const disk = page.locator('main table tbody tr').first().locator('td').nth(5).locator('button').first()
  if (await disk.count()) {
    await disk.hover()
    await page.waitForTimeout(800)
    console.log(`disk tooltip: ${JSON.stringify(await texts('body', /B$/))}`)
  }
  await shot('202-system-info', { fullPage: true })

  // Dashboard flow: Top N tabs, filter values; hover a link for the tooltip.
  await open('/dashboard/flow', 'main', 2500)
  console.log(`flow tabs: ${JSON.stringify(await texts('main', /برتر/))}`)
  await shot('203-dashboard-flow')

  // Task plugins: model counts and priority.
  await open('/task-plugins', 'main', 2000)
  await shot('204-task-plugins')

  // Quota audit summary and user.manage entries in the audit log.
  await open('/usage-logs/audit', 'main', 2000)
  console.log(`audit summaries: ${JSON.stringify((await texts('main', /←|→|شناسه/)).map(show))}`)
  await shot('205-audit-log')
  const manageRow = page.locator('main table tbody tr').filter({ hasText: /ارتقا به مدیر/ }).first()
  if (await manageRow.count()) {
    await manageRow.getByRole('button', { name: 'جزئیات' }).first().click()
    await settle(1200)
    await waitOpaque('[role="dialog"]')
    console.log(`audit entry: ${JSON.stringify((await texts('[role="dialog"]', /شناسه|ارتقا/)).map(show))}`)
    await shot('206-audit-entry-action')
    await page.keyboard.press('Escape')
  } else {
    console.log('audit entry: no user.manage row found')
  }
  const quotaRow = page.locator('main table tbody tr').filter({ hasText: /←/ }).first()
  if (await quotaRow.count()) {
    await quotaRow.getByRole('button', { name: 'جزئیات' }).first().click()
    await settle(1200)
    await waitOpaque('[role="dialog"]')
    await shot('207-quota-audit-summary')
    await page.keyboard.press('Escape')
  }

  // User quota preview: edit demo-user, Adjust Quota, Override, amount 5.
  await open('/users', 'main', 1800)
  const userRow = page.locator('main table tbody tr').filter({ hasText: 'demo-user' }).first()
  await userRow.getByRole('button', { name: 'ویرایش' }).first().click()
  await settle(1200)
  await page.getByRole('button', { name: 'تنظیم سهمیه' }).first().click()
  await settle(800)
  const quotaDialog = page.getByRole('dialog').filter({ hasText: 'تنظیم سهمیه' }).last()
  await quotaDialog.getByRole('button', { name: 'بازنویسی' }).click()
  await quotaDialog.getByRole('spinbutton').fill('5')
  await page.waitForTimeout(500)
  const preview = await quotaDialog.locator('div.text-sm').first().innerText()
  console.log(`quota preview: ${JSON.stringify(show(preview))}`)
  await shot('208-user-quota-preview')
  await page.keyboard.press('Escape')
  await page.keyboard.press('Escape')

  // Pricing model details, API tab.
  await open('/pricing/gpt-4o', 'body', 2500)
  const apiTab = page.getByRole('tab', { name: 'API' }).first()
  if (await apiTab.count()) {
    await apiTab.click()
    await settle(1000)
    const rateLimits = page.getByRole('columnheader', { name: 'RPM' }).first()
    if (await rateLimits.count()) await rateLimits.scrollIntoViewIfNeeded()
    await page.waitForTimeout(400)
  }
  console.log(`rate limit headers: ${JSON.stringify(await page.evaluate(() =>
    [...document.querySelectorAll('th')].filter((th) => /^(RPM|TPM|RPD)$/.test(th.textContent.trim()))
      .map((th) => `${th.textContent.trim()} ${getComputedStyle(th).textAlign}`)))}`)
  await shot('209-pricing-api-tab', { fullPage: true })

  // Dashboard models: header total, then the chart preferences dialog.
  await open('/dashboard/models', 'main', 2500)
  console.log(`chart header total: ${JSON.stringify((await texts('main', /\$/)).map(show).slice(0, 3))}`)
  await shot('210-dashboard-models')
  await page.getByRole('button', { name: 'ترجیحات' }).first().click()
  await settle(800)
  await waitOpaque('[role="dialog"]')
  await shot('211-chart-preferences')
  await page.keyboard.press('Escape')

  // Subscriptions list.
  await open('/subscriptions', 'main', 1800)
  console.log(`subscriptions row: ${JSON.stringify(await texts('main table tbody', /[۰-۹]/))}`)
  await shot('212-subscriptions')
}

console.log(`console errors: ${JSON.stringify([...new Set(consoleErrors)].slice(0, 10))}`)
await browser.close()
