// Which face renders var(--font-sans) once the page has settled.
// Usage: node font-probe.mjs <base-url> <path> <lang>
// Prints the registered faces and their status, the width of one sample
// line in var(--font-sans), generic sans-serif and the loaded face
// (measured last, after forcing it to load), the computed body font, and
// every woff2 file the page downloaded with its size.
import { chromium } from 'playwright'

const [base, path, lang = 'en'] = process.argv.slice(2)
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
const errors = []
page.on('console', (m) => m.type() === 'error' && errors.push(m.text()))
const fonts = []
page.on('response', async (res) => {
  if (!res.url().endsWith('.woff2')) return
  const body = await res.body().catch(() => Buffer.alloc(0))
  fonts.push(`${res.url().split('/').pop()} ${body.length} B`)
})

await page.goto(`${base}${path}`)
await page.waitForLoadState('networkidle').catch(() => {})
await page.evaluate(() => document.fonts.ready)
await page.waitForTimeout(1500)

const result = await page.evaluate(async () => {
  const sample = 'The quick brown fox jumps over the lazy dog 0123456789'
  const width = (family) => {
    const span = document.createElement('span')
    span.style.cssText = `font-family:${family};font-size:32px;font-weight:400;white-space:nowrap;position:absolute;visibility:hidden`
    span.textContent = sample
    document.body.append(span)
    const w = span.getBoundingClientRect().width
    span.remove()
    return Math.round(w * 100) / 100
  }
  const faces = () =>
    [...document.fonts]
      .filter((f) => /Public Sans|Vazirmatn/.test(f.family))
      .map((f) => `${f.family.replaceAll('"', '')} ${f.unicodeRange.slice(0, 13)}… ${f.status}`)
  const root = getComputedStyle(document.documentElement)
  const before = {
    html: `dir=${document.documentElement.dir} lang=${document.documentElement.lang} font=${document.documentElement.dataset.themeFont}`,
    fontSans: root.getPropertyValue('--font-sans').trim(),
    bodyFont: getComputedStyle(document.body).fontFamily,
    facesAfterSettle: faces(),
    loadedPublicSans: [...document.fonts].some((f) => /Public Sans/.test(f.family) && f.status === 'loaded'),
    widthFontSans: width('var(--font-sans)'),
    widthGenericSans: width('sans-serif'),
  }
  return before
})
const fontsBeforeForcedLoad = [...fonts]
result.widthPublicSansVariable = await page.evaluate(async () => {
  await document.fonts.load("32px 'Public Sans Variable'", 'The quick')
  const span = document.createElement('span')
  span.style.cssText =
    "font-family:'Public Sans Variable';font-size:32px;font-weight:400;white-space:nowrap;position:absolute;visibility:hidden"
  span.textContent = 'The quick brown fox jumps over the lazy dog 0123456789'
  document.body.append(span)
  const w = span.getBoundingClientRect().width
  span.remove()
  return Math.round(w * 100) / 100
})
console.log(`browser: Chromium ${browser.version()}`)
console.log(JSON.stringify(result, null, 2))
console.log(`woff2 downloaded by the page: ${JSON.stringify(fontsBeforeForcedLoad)}`)
console.log(`console errors: ${JSON.stringify([...new Set(errors)])}`)
await browser.close()
