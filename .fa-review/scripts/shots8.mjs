// Session 8 screenshots: system settings sections in Persian, one settings
// dialog, the model edit dialog with the icon field, and the channels list in
// table view.
// Usage: node shots8.mjs <base-url> <out-dir> [lang] [only-prefix]
//   Saves 1NN-settings-<group>-<section>-rtl.png (full page) and
//   15N-*.png for the dialogs and the channels table. lang defaults to fa;
//   with en the files end in -en instead of -rtl.
//
// Every capture waits until the content is fully opaque, then counts the
// colours of a sample of the image; fewer than 16 colours is reported as
// BLANK (see shots7.mjs).
import { chromium } from 'playwright'

const [base, out, langArg, only] = process.argv.slice(2)
const PASSWORD = 'DemoPass-2026!'
const lang = langArg || 'fa'
const suffix = lang === 'fa' ? 'rtl' : lang
const browserLocales = { fa: 'fa-IR', zhCN: 'zh-CN', en: 'en-US' }

const browser = await chromium.launch({
  executablePath: '/opt/pw-browsers/chromium-1194/chrome-linux/chrome',
})
const context = await browser.newContext({
  viewport: { width: 1440, height: 900 },
  locale: browserLocales[lang] ?? 'en-US',
  timezoneId: 'UTC',
  colorScheme: 'light',
})
await context.addInitScript((l) => localStorage.setItem('i18nextLng', l), lang)
const page = await context.newPage()
const consoleErrors = []
page.on('console', (m) => m.type() === 'error' && consoleErrors.push(m.text()))
page.on('pageerror', (e) => consoleErrors.push(String(e)))

async function settle(ms = 800) {
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

// Untranslated text: visible text nodes with three or more Latin letters in a
// row, outside code, inputs and elements marked dir=ltr.
async function latinTexts() {
  return page.evaluate(() => {
    const found = new Set()
    const walker = document.createTreeWalker(document.querySelector('main') ?? document.body, NodeFilter.SHOW_TEXT)
    while (walker.nextNode()) {
      const n = walker.currentNode
      const el = n.parentElement
      if (!el || el.closest('code, pre, kbd, [dir="ltr"], script, style, .cm-editor')) continue
      if (!el.offsetParent && getComputedStyle(el).position !== 'fixed') continue
      const t = n.textContent.replace(/\s+/g, ' ').trim()
      if (/[A-Za-z]{3,}/.test(t) && !/[؀-ۿ]/.test(t)) found.add(t.slice(0, 80))
    }
    return [...found]
  })
}

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
}

const SECTIONS = [
  ['101', 'site', 'system-info'],
  ['102', 'site', 'notice'],
  ['103', 'site', 'header-navigation'],
  ['104', 'site', 'sidebar-modules'],
  ['105', 'billing', 'quota'],
  ['106', 'billing', 'currency'],
  ['107', 'billing', 'model-pricing'],
  ['108', 'billing', 'group-pricing'],
  ['109', 'billing', 'payment'],
  ['110', 'billing', 'checkin'],
  ['111', 'auth', 'basic-auth'],
  ['112', 'auth', 'oauth'],
  ['113', 'auth', 'passkey'],
  ['114', 'auth', 'bot-protection'],
  ['115', 'auth', 'custom-oauth'],
  ['116', 'content', 'dashboard'],
  ['117', 'content', 'announcements'],
  ['118', 'content', 'api-info'],
  ['119', 'content', 'faq'],
  ['120', 'content', 'uptime-kuma'],
  ['121', 'content', 'chat'],
  ['122', 'content', 'drawing'],
  ['123', 'operations', 'behavior'],
  ['124', 'operations', 'alerts'],
  ['125', 'operations', 'email'],
  ['126', 'operations', 'worker'],
  ['127', 'operations', 'logs'],
  ['128', 'operations', 'performance'],
  ['129', 'operations', 'update-checker'],
  ['130', 'security', 'rate-limit'],
  ['131', 'security', 'ssrf'],
  ['132', 'security', 'token-limits'],
  ['133', 'request-policies', 'filtering'],
  ['134', 'request-policies', 'routing'],
  ['135', 'request-policies', 'health'],
  ['136', 'models', 'global'],
  ['137', 'models', 'gemini'],
  ['138', 'models', 'claude'],
  ['139', 'models', 'grok'],
  ['140', 'models', 'model-deployment'],
]

console.log(`browser: Chromium ${browser.version()}`)
await signIn()
const info = await page.evaluate(() => ({ dir: document.documentElement.dir, lang: document.documentElement.lang }))
console.log(`page: dir=${info.dir} lang=${info.lang}`)

for (const [n, group, section] of SECTIONS) {
  const name = `${n}-settings-${group}-${section}`
  if (only && !name.startsWith(only)) continue
  await page.goto(`${base}/system-settings/${group}/${section}`)
  await settle(1500)
  await waitOpaque('main')
  await shot(name, { fullPage: true })
  const latin = await latinTexts()
  if (latin.length) console.log(`  latin: ${JSON.stringify(latin)}`)
}

if (!only || only === '15') {
  // One settings dialog: the custom OAuth provider form.
  await page.goto(`${base}/system-settings/auth/custom-oauth`)
  await settle(1500)
  await page.locator('main button').filter({ hasText: /افزودن|Add/ }).first().click()
  await page.getByRole('dialog').first().waitFor()
  await settle(1200)
  await waitOpaque('[role="dialog"]')
  await shot('150-settings-custom-oauth-dialog')

  // Model edit dialog with the icon field.
  await page.goto(`${base}/models`)
  await settle(1500)
  const row = page.locator('table tbody tr', { hasText: 'gpt-4o-mini' }).first()
  await row.getByRole('button', { name: lang === 'fa' ? 'ویرایش' : 'Edit' }).click()
  await page.getByRole('dialog').first().waitFor()
  await settle(1500)
  await waitOpaque('[role="dialog"]')
  const icon = page.getByRole('dialog').getByText(lang === 'fa' ? 'آیکون' : 'icon', { exact: false }).first()
  if (await icon.count()) await icon.scrollIntoViewIfNeeded()
  await shot('151-model-edit-icon-field')
  await page.keyboard.press('Escape')

  // Channels list in table view: the view is remembered in localStorage.
  await page.goto(`${base}/channels`)
  await settle(1500)
  const tableToggle = page.getByRole('radio', { name: /جدول|Table/ }).first()
  if (await tableToggle.count()) await tableToggle.click()
  else await page.getByRole('button', { name: /جدول|Table/ }).first().click()
  await settle(1500)
  await waitOpaque('main table')
  const widths = await page.evaluate(() =>
    [...document.querySelectorAll('main table thead th')].map((th) => `${th.innerText.trim()}:${Math.round(th.getBoundingClientRect().width)}`)
  )
  console.log(`channels header: ${JSON.stringify(widths)}`)
  const rt = await page.evaluate(() =>
    [...document.querySelectorAll('main table tbody td')]
      .filter((td) => /میلی‌ثانیه|ms\b/.test(td.innerText))
      .map((td) => ({ text: td.innerText.trim(), cell: Math.round(td.getBoundingClientRect().width), content: Math.round((td.firstElementChild ?? td).scrollWidth) }))
  )
  console.log(`response time cells: ${JSON.stringify(rt)}`)
  await shot('152-channels-table')
  // The table scrolls sideways; bring the response-time column into view.
  await page.evaluate(() => {
    const th = [...document.querySelectorAll('main table thead th')].find((el) =>
      /پاسخ|Response/.test(el.textContent ?? '')
    )
    th?.scrollIntoView({ inline: 'center', block: 'nearest' })
  })
  await page.waitForTimeout(500)
  await shot('153-channels-table-response-time')
}

console.log(`console errors: ${JSON.stringify([...new Set(consoleErrors)].slice(0, 10))}`)
await browser.close()
