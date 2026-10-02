// Session 13 screenshots (after the upstream merge).
// Usage: node shots-s13.mjs <base-url> <out-dir> <fa|en> <tag> <merge|main>
// Signs in as the seeded admin, fixes the browser clock at 2026-09-25
// 14:00 UTC (the seeded logs are from that day), turns motion off and
// captures full pages as <out-dir>/<nn>-<page>-<tag>.png.
//   merge: the pages the upstream merge touched (security with the new
//          access tokens and the edit dialog, usage logs, audit log and an
//          access token audit entry, users, wallet top-up, model details,
//          dashboard charts).
//   main:  the main pages, for the English pixel comparison.
// Every capture waits until the page is fully opaque and counts the colours
// of a sample; fewer than 16 is reported as BLANK.
import { chromium } from 'playwright'

const [base, out, lang, tag, mode = 'merge'] = process.argv.slice(2)
const PASSWORD = 'DemoPass-2026!'
const fa = lang === 'fa'

const browser = await chromium.launch({
  executablePath: '/opt/pw-browsers/chromium-1194/chrome-linux/chrome',
})
const context = await browser.newContext({
  viewport: { width: 1440, height: 900 },
  locale: fa ? 'fa-IR' : 'en-US',
  timezoneId: 'UTC',
  colorScheme: 'light',
  reducedMotion: 'reduce',
})
await context.addInitScript((l) => {
  localStorage.setItem('i18nextLng', l)
}, lang)
const page = await context.newPage()
await page.clock.setFixedTime(new Date('2026-09-25T14:00:00Z'))
const consoleErrors = []
page.on('console', (m) => m.type() === 'error' && consoleErrors.push(m.text()))
page.on('pageerror', (e) => consoleErrors.push(String(e)))

async function settle(ms = 1500) {
  await page.waitForLoadState('networkidle').catch(() => {})
  await page.evaluate(() => document.fonts.ready)
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

async function shot(name, fullPage = true) {
  await page.mouse.move(0, 0)
  const file = `${out}/${name}-${tag}.png`
  const buffer = await page.screenshot({ path: file, fullPage })
  const colours = await colourCount(buffer)
  console.log(
    `saved ${name}-${tag}.png (${colours} colours${colours < 16 ? ', BLANK' : ''})`
  )
}

async function open(path, wait = 2500) {
  await page.goto(`${base}${path}`)
  await settle(wait)
  await waitOpaque('main')
}

async function texts(selector, pattern) {
  return page.evaluate(
    ([sel, src]) => {
      const re = new RegExp(src)
      const root = document.querySelector(sel)
      if (!root) return []
      const out = new Set()
      const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT)
      while (walker.nextNode()) {
        const t = walker.currentNode.textContent?.trim()
        if (t && re.test(t)) out.add(t.slice(0, 120))
      }
      return [...out].slice(0, 12)
    },
    [selector, pattern.source]
  )
}

async function signIn() {
  await page.goto(`${base}/sign-in`)
  await settle()
  await page.locator('input[name="username"], input#username').first().fill('admin')
  await page.locator('input[type="password"]').first().fill(PASSWORD)
  const consent = page.getByRole('checkbox').first()
  if (await consent.count()) await consent.click()
  await page.locator('button[type="submit"]').first().click()
  await page.waitForURL((url) => !url.pathname.startsWith('/sign-in'), {
    timeout: 20000,
  })
  await settle()
}

console.log(`browser: Chromium ${browser.version()}, lang ${lang}, tag ${tag}, mode ${mode}`)
await signIn()

if (mode === 'main') {
  const pages = [
    ['600-dashboard', '/dashboard'],
    ['601-dashboard-models', '/dashboard/models'],
    ['602-usage-logs', '/usage-logs'],
    ['603-audit-log', '/usage-logs/audit'],
    ['604-pricing', '/pricing'],
    ['605-model-details', '/pricing/gpt-4o'],
    ['606-channels', '/channels'],
    ['607-models', '/models'],
    ['608-keys', '/keys'],
    ['609-profile', '/profile'],
    ['610-security', '/security'],
    ['611-wallet', '/wallet'],
    ['612-subscriptions', '/subscriptions'],
    ['613-users', '/users'],
    ['614-redemption-codes', '/redemption-codes'],
    ['615-system-settings', '/system-settings'],
  ]
  for (const [name, path] of pages) {
    await open(path)
    await shot(name)
  }
} else {
  // Security: the new scoped access tokens.
  await open('/security')
  console.log(`access tokens: ${JSON.stringify(await texts('main', fa ? /توکن|ساخته|انقضا|استفاده/ : /Created|Expires|Last used|Never/))}`)
  await shot('501-security-access-tokens')
  const menu = page.getByRole('button', { name: fa ? 'باز کردن منو' : 'Open menu' }).first()
  if (await menu.count()) {
    await menu.click()
    await settle(600)
    await page.getByRole('menuitem', { name: fa ? 'ویرایش' : 'Edit' }).first().click()
    await settle(1200)
    await waitOpaque('[role="dialog"]')
    await shot('502-access-token-edit', false)
    await page.keyboard.press('Escape')
    await settle(600)
  } else {
    console.log('no access token row menu found')
  }

  await open('/usage-logs')
  await shot('503-usage-logs')

  await open('/usage-logs/audit')
  console.log(`audit summaries: ${JSON.stringify(await texts('main', fa ? /توکن|←|→/ : /token|←|→/i))}`)
  await shot('504-audit-log')
  const tokenRow = page
    .locator('main table tbody tr')
    .filter({ hasText: fa ? /توکن دسترسی/ : /access token/i })
    .first()
  if (await tokenRow.count()) {
    await tokenRow.getByRole('button', { name: fa ? 'جزئیات' : 'Details' }).first().click()
    await settle(1500)
    await waitOpaque('[role="dialog"]')
    console.log(`audit entry: ${JSON.stringify(await texts('[role="dialog"]', fa ? /انقضا|مجوز|۱۴۰۵/ : /Expiration|permission|2026/))}`)
    await shot('505-audit-access-token-details', false)
    await page.keyboard.press('Escape')
  } else {
    console.log('no access token audit row found')
  }

  await open('/users')
  await shot('506-users')

  await open('/wallet')
  console.log(`wallet presets: ${JSON.stringify(await texts('main', fa ? /پرداخت|صرفه‌جویی/ : /Pay |Save /))}`)
  await shot('507-wallet-topup')

  await open('/pricing/gpt-4o')
  await shot('508-model-details')

  await open('/dashboard/models', 3500)
  await shot('509-dashboard-charts')
}

console.log(`console errors: ${JSON.stringify([...new Set(consoleErrors)].slice(0, 10))}`)
await browser.close()
