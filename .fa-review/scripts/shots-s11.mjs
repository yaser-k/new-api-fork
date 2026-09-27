// Session 11 screenshots.
// Usage:
//   node shots-s11.mjs <base-url> <out-dir> ups <before|after>
//     English home and sign-in page (upstream branch, font fix)
//   node shots-s11.mjs <base-url> <out-dir> fa
//     Persian: system info (page counter, percentages), retry chain,
//     wallet plan card, purchase dialog, home page (Vazirmatn)
// Every capture waits until the content is fully opaque, then counts the
// colours of a sample of the image; fewer than 16 colours is BLANK.
import { chromium } from 'playwright'

const [base, out, mode, label] = process.argv.slice(2)
const PASSWORD = 'DemoPass-2026!'
const lang = mode === 'fa' ? 'fa' : 'en'

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

async function shot(file, options = {}) {
  const buffer = await page.screenshot({ path: `${out}/${file}.png`, ...options })
  const colours = await colourCount(buffer)
  console.log(`saved ${file}.png (${colours} colours${colours < 16 ? ', BLANK' : ''})`)
}

async function open(path, root = 'main', wait = 1800) {
  await page.goto(`${base}${path}`)
  await settle(wait)
  await waitOpaque(root)
}

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

async function fontState() {
  return page.evaluate(() => ({
    html: `dir=${document.documentElement.dir} lang=${document.documentElement.lang}`,
    bodyFont: getComputedStyle(document.body).fontFamily,
    loadedFaces: [
      ...new Set(
        [...document.fonts]
          .filter((f) => f.status === 'loaded')
          .map((f) => f.family.replaceAll('"', ''))
      ),
    ],
  }))
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

console.log(`browser: Chromium ${browser.version()}`)

if (mode === 'ups') {
  const n = label === 'before' ? 0 : 1
  await open('/', 'body', 2500)
  console.log(`home ${label}: ${JSON.stringify(await fontState())}`)
  await shot(`${310 + n}-home-${label}-en`)
  await open('/sign-in', 'body', 2000)
  console.log(`sign-in ${label}: ${JSON.stringify(await fontState())}`)
  await shot(`${312 + n}-sign-in-${label}-en`)
} else {
  await signIn()

  // System info: task history page counter and the percentages.
  await open('/system-info', 'main', 2500)
  console.log(`page counter: ${JSON.stringify(await texts('main', /^[۰-۹0-9]+ \/ [۰-۹0-9]+$/))}`)
  console.log(`percentages: ${JSON.stringify(await texts('main', /[٪%]/))}`)
  await shot('320-system-info-counter-percent-rtl', { fullPage: true })

  // Usage log details: the admin retry chain.
  await open('/usage-logs', 'main', 2000)
  const row = page.locator('main table tbody tr').filter({ hasText: 'gpt-4o' }).first()
  await row.locator('button[title]').first().click()
  await settle(1200)
  await waitOpaque('[role="dialog"]')
  const chain = page.getByRole('dialog').getByText(/←|→/).first()
  if (await chain.count()) {
    await chain.scrollIntoViewIfNeeded()
    const box = await chain.evaluate((el) => {
      const r = el.getBoundingClientRect()
      return { text: el.textContent, dir: getComputedStyle(el).direction, x: Math.round(r.x), w: Math.round(r.width) }
    })
    console.log(`retry chain: ${JSON.stringify({ ...box, text: show(box.text) })}`)
    const order = await chain.evaluate((el) => {
      const range = document.createRange()
      const node = el.firstChild
      const xs = []
      for (const target of ['1', '2', '3']) {
        const i = node.textContent.indexOf(target)
        range.setStart(node, i)
        range.setEnd(node, i + 1)
        xs.push(`${target}@x${Math.round(range.getBoundingClientRect().x)}`)
      }
      return xs
    })
    console.log(`retry chain glyph positions: ${JSON.stringify(order)}`)
  } else {
    console.log('retry chain: not found')
  }
  await shot('321-retry-chain-rtl')
  await page.keyboard.press('Escape')

  // Wallet: plan card price, then the purchase dialog.
  await open('/wallet', 'main', 2500)
  const buy = page.getByRole('button', { name: 'خرید اشتراک' }).first()
  await buy.scrollIntoViewIfNeeded()
  await page.waitForTimeout(400)
  console.log(`plan card price: ${JSON.stringify(await texts('main', /^\$/))}`)
  await shot('322-wallet-plan-card-rtl')
  await buy.click()
  await settle(1000)
  await waitOpaque('[role="dialog"]')
  console.log(`purchase dialog price: ${JSON.stringify(await texts('[role="dialog"]', /^\$/))}`)
  await shot('323-purchase-dialog-rtl')
  await page.keyboard.press('Escape')

  // Home page in Persian: Vazirmatn still renders the text.
  await open('/', 'body', 2500)
  console.log(`home fa: ${JSON.stringify(await fontState())}`)
  await shot('324-home-vazirmatn-rtl')
}

console.log(`console errors: ${JSON.stringify([...new Set(consoleErrors)].slice(0, 10))}`)
await browser.close()
