// Session 12 screenshots (right-to-left sweep).
// Usage: node shots-s12.mjs <base-url> <out-dir> <fa|en> <tag>
// Signs in as the seeded admin, fixes the browser clock at
// 2026-09-25 14:00 UTC (the seeded logs are from that day), turns motion
// off, and captures every page whose layout the sweep changed, full page,
// as <out-dir>/<nn>-<page>-<tag>.png. Every capture waits until the page is
// fully opaque and counts the colours of a sample; fewer than 16 is BLANK.
// On the dashboard it also logs where the parts fixed in part A sit.
import { chromium } from 'playwright'

const [base, out, lang, tag] = process.argv.slice(2)
const PASSWORD = 'DemoPass-2026!'

const browser = await chromium.launch({
  executablePath: '/opt/pw-browsers/chromium-1194/chrome-linux/chrome',
})
const context = await browser.newContext({
  viewport: { width: 1440, height: 900 },
  locale: lang === 'fa' ? 'fa-IR' : 'en-US',
  timezoneId: 'UTC',
  colorScheme: 'light',
  reducedMotion: 'reduce',
})
await context.addInitScript((l) => {
  localStorage.setItem('i18nextLng', l)
  localStorage.setItem('dashboard_overview_setup_guide_expanded', 'expanded')
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

async function shot(name) {
  await page.mouse.move(0, 0)
  const file = `${out}/${name}-${tag}.png`
  const buffer = await page.screenshot({ path: file, fullPage: true })
  const colours = await colourCount(buffer)
  console.log(
    `saved ${name}-${tag}.png (${colours} colours${colours < 16 ? ', BLANK' : ''})`
  )
}

async function open(path, wait = 2000) {
  await page.goto(`${base}${path}`)
  await settle(wait)
  await waitOpaque('main')
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

// Positions of the part A fixes on the dashboard overview.
async function dashboardLayout() {
  return page.evaluate(() => {
    const box = (el) => {
      if (!el) return null
      const r = el.getBoundingClientRect()
      return { l: Math.round(r.left), r: Math.round(r.right), w: Math.round(r.width) }
    }
    const code = document.querySelector('[data-slot="setup-guide-backdrop-code"]')
      ?? [...document.querySelectorAll('main pre')].find((p) =>
        p.textContent?.includes('client.responses.create')
      )
    const card = code?.closest('section, [class*="rounded"]')
    const heading = card?.querySelector('h1, h2, h3')
    const step = document.querySelector('main ol li, main ul li')
    const connector = step?.querySelector('span[aria-hidden="true"].bg-border')
    const circle = connector?.nextElementSibling
    const action = [...document.querySelectorAll('main a')].find(
      (a) => a.querySelector('span.bg-muted.size-9, span[class*="size-9"]')
    )
    const actionIcon = action?.querySelector('span[class*="size-9"]')
    const actionTitle = action?.querySelector('span.truncate')
    const titleRange = document.createRange()
    if (actionTitle?.firstChild) titleRange.selectNodeContents(actionTitle)
    return {
      dir: document.documentElement.dir,
      guideHeading: box(heading),
      codeBackdrop: box(code),
      codeTextAlign: code ? getComputedStyle(code).textAlign : null,
      guideCard: box(card),
      stepConnectorX: connector ? Math.round(connector.getBoundingClientRect().left) : null,
      stepCircle: box(circle),
      quickActionIcon: box(actionIcon),
      quickActionTitleText: actionTitle?.firstChild ? box(titleRange) : null,
    }
  })
}

console.log(`browser: Chromium ${browser.version()}, lang ${lang}, tag ${tag}`)
await signIn()

const pages = [
  ['410-dashboard-setup-guide', '/dashboard'],
  ['411-usage-logs', '/usage-logs'],
  ['412-pricing', '/pricing'],
  ['413-model-details', '/pricing/gpt-4o'],
  ['414-channels', '/channels'],
  ['415-models', '/models'],
  ['416-profile', '/profile'],
  ['417-wallet', '/wallet'],
  ['418-subscriptions', '/subscriptions'],
]
for (const [name, path] of pages) {
  await open(path, 2500)
  if (name.includes('dashboard')) {
    console.log(`dashboard layout: ${JSON.stringify(await dashboardLayout())}`)
  }
  await shot(name)
}

// A shared dropdown menu: the channel row actions (items, shortcuts).
await open('/channels', 2000)
const trigger = page
  .getByRole('button', { name: lang === 'fa' ? 'باز کردن منو' : 'Open menu' })
  .first()
if (await trigger.count()) {
  await trigger.click()
  await settle(800)
  await shot('419-channels-row-menu')
  await page.keyboard.press('Escape')
}

console.log(`console errors: ${JSON.stringify([...new Set(consoleErrors)].slice(0, 10))}`)
await browser.close()
