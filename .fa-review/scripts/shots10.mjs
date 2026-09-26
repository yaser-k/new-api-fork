// Session 10 screenshots: setup page (before the admin exists), the fixed
// left-to-right placeholders, the response-time column and the pages
// translated in batch 10.
// Usage: node shots10.mjs <base-url> <out-dir> setup|main [lang]
//   setup: 160-setup-rtl.png (run before /api/setup creates the admin)
//   main : 161..173-*.png in Persian, and with lang=en the custom OAuth
//          dialog in English (161-...-en.png)
// Every capture waits until the content is fully opaque, then counts the
// colours of a sample of the image; fewer than 16 colours is BLANK.
import { chromium } from 'playwright'

const [base, out, mode, langArg] = process.argv.slice(2)
const PASSWORD = 'DemoPass-2026!'
const lang = langArg || 'fa'
const suffix = lang === 'fa' ? 'rtl' : lang

const browser = await chromium.launch({
  executablePath: '/opt/pw-browsers/chromium-1194/chrome-linux/chrome',
})
const context = await browser.newContext({
  viewport: { width: 1440, height: 900 },
  locale: lang === 'fa' ? 'fa-IR' : 'en-US',
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

// Visible text with three or more Latin letters and no Persian letter,
// outside code, inputs and elements marked dir=ltr.
async function latinTexts(root = 'main') {
  return page.evaluate((sel) => {
    const found = new Set()
    const walker = document.createTreeWalker(document.querySelector(sel) ?? document.body, NodeFilter.SHOW_TEXT)
    while (walker.nextNode()) {
      const n = walker.currentNode
      const el = n.parentElement
      if (!el || el.closest('code, pre, kbd, [dir="ltr"], script, style, .cm-editor')) continue
      if (!el.offsetParent && getComputedStyle(el).position !== 'fixed') continue
      const t = n.textContent.replace(/\s+/g, ' ').trim()
      if (/[A-Za-z]{3,}/.test(t) && !/[؀-ۿ]/.test(t)) found.add(t.slice(0, 80))
    }
    return [...found]
  }, root)
}

async function page_(name, path, options = {}) {
  await page.goto(`${base}${path}`)
  await settle(options.wait ?? 1800)
  await waitOpaque(options.root ?? 'main')
  if (options.before) await options.before()
  await shot(name, { fullPage: options.fullPage ?? false })
  if (lang === 'fa') {
    const latin = await latinTexts(options.root ?? 'main')
    if (latin.length) console.log(`  latin: ${JSON.stringify(latin)}`)
  }
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

// Direction and visual order of the placeholder of each dir=ltr input.
async function placeholderReport(scope) {
  return page.evaluate((sel) =>
    [...document.querySelectorAll(`${sel} input[dir="ltr"], ${sel} textarea[dir="ltr"]`)]
      .filter((el) => el.placeholder && /[؀-ۿ]/.test(el.placeholder))
      .map((el) => ({
        placeholder: el.placeholder.replace(/[⁦-⁩]/g, (c) => `<U+${c.charCodeAt(0).toString(16)}>`),
        direction: getComputedStyle(el).direction,
      })),
    scope
  )
}

console.log(`browser: Chromium ${browser.version()}`)

if (mode === 'setup') {
  await page_('160-setup', '/setup', { root: 'body' })
  const info = await page.evaluate(() => ({ dir: document.documentElement.dir, lang: document.documentElement.lang }))
  console.log(`page: dir=${info.dir} lang=${info.lang}`)
} else {
  await signIn()
  const info = await page.evaluate(() => ({ dir: document.documentElement.dir, lang: document.documentElement.lang }))
  console.log(`page: dir=${info.dir} lang=${info.lang}`)

  // A. The custom OAuth provider dialog and the OAuth settings section.
  await page.goto(`${base}/system-settings/auth/custom-oauth`)
  await settle(1500)
  await page.locator('main button').filter({ hasText: /افزودن|Add/ }).first().click()
  await page.getByRole('dialog').first().waitFor()
  await settle(1200)
  await waitOpaque('[role="dialog"]')
  console.log(`dialog placeholders: ${JSON.stringify(await placeholderReport('[role="dialog"]'))}`)
  await shot('161-settings-custom-oauth-dialog')
  await page.keyboard.press('Escape')
  if (lang !== 'fa') {
    console.log(`console errors: ${JSON.stringify([...new Set(consoleErrors)].slice(0, 10))}`)
    await browser.close()
    process.exit(0)
  }
  await page.goto(`${base}/system-settings/auth/oauth`)
  await settle(1800)
  await waitOpaque('main')
  console.log(`oauth placeholders: ${JSON.stringify(await placeholderReport('main'))}`)
  await shot('162-settings-auth-oauth', { fullPage: true })

  // B. Channels table, scrolled to the response-time column.
  await page.goto(`${base}/channels`)
  await settle(1500)
  const tableToggle = page.getByRole('radio', { name: /جدول|Table/ }).first()
  if (await tableToggle.count()) await tableToggle.click()
  else await page.getByRole('button', { name: /جدول|Table/ }).first().click()
  await settle(1500)
  await waitOpaque('main table')
  const rt = await page.evaluate(() =>
    [...document.querySelectorAll('main table tbody td')]
      .filter((td) => /ms|ثانیه/.test(td.innerText) && td.innerText.length < 20)
      .map((td) => {
        const label = td.querySelector('span span, span') ?? td
        return {
          text: td.innerText.trim(),
          cell: Math.round(td.getBoundingClientRect().width),
          badge: Math.round((td.firstElementChild ?? td).getBoundingClientRect().width),
          truncated: [...td.querySelectorAll('*')].some((el) => el.scrollWidth > el.clientWidth + 1),
        }
      })
  )
  console.log(`response time cells: ${JSON.stringify(rt)}`)
  await page.evaluate(() => {
    const th = [...document.querySelectorAll('main table thead th')].find((el) => /پاسخ|Response/.test(el.textContent ?? ''))
    th?.scrollIntoView({ inline: 'center', block: 'nearest' })
  })
  await page.waitForTimeout(500)
  await shot('163-channels-table-response-time')

  // C. Pages translated in batch 10.
  await page_('164-usage-logs', '/usage-logs')
  await page_('165-dashboard', '/dashboard', { wait: 2500 })
  await page_('166-dashboard-flow', '/dashboard/flow', { wait: 2500 })
  await page_('167-playground', '/playground', { wait: 2000 })
  await page_('168-rankings', '/rankings', { wait: 2500, root: 'body', fullPage: true })
  await page_('169-api-keys', '/keys')
  await page_('170-system-info', '/system-info', { wait: 2500, fullPage: true })
  await page_('171-task-plugins', '/task-plugins', { wait: 2000 })
  // Task plugin detail sheet: open the first plugin row.
  const firstPlugin = page.locator('main table tbody tr').first()
  if (await firstPlugin.count()) {
    await firstPlugin.click()
    await settle(1500)
    const sheet = page.getByRole('dialog').first()
    if (await sheet.count()) {
      await waitOpaque('[role="dialog"]')
      await shot('172-task-plugin-detail')
      await page.keyboard.press('Escape')
    }
  }
  await page_('173-home', '/', { wait: 2000, root: 'body' })
}

console.log(`console errors: ${JSON.stringify([...new Set(consoleErrors)].slice(0, 10))}`)
await browser.close()
