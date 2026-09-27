// Carousel harness screenshot and the Public Sans check.
// Usage: node carousel-shot.mjs <harness-url> <out.png> <app-base-url>
// The harness page is carousel-harness.tsx bundled with `bun build` (a
// tsconfig mapping @/* to web/src/*, node_modules linked to web/node_modules)
// and an index.html that loads the app's built CSS, served over HTTP.
// It logs the button positions per direction, saves the capture (fewer than
// 16 colours is BLANK), then reads the font faces of the running app.
import { chromium } from 'playwright'
const [htmlPath, out, base] = process.argv.slice(2)
const browser = await chromium.launch({ executablePath: '/opt/pw-browsers/chromium-1194/chrome-linux/chrome' })
const page = await browser.newPage({ viewport: { width: 900, height: 420 } })
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
await page.goto(htmlPath)
await page.waitForTimeout(1500)
const info = await page.evaluate(() => [...document.querySelectorAll('section')].map((s) => {
  const prev = s.querySelector('[data-slot=carousel-previous]').getBoundingClientRect()
  const next = s.querySelector('[data-slot=carousel-next]').getBoundingClientRect()
  const icon = getComputedStyle(s.querySelector('[data-slot=carousel-previous] svg')).transform
  return { dir: s.dir, previousX: Math.round(prev.x), nextX: Math.round(next.x), previousIcon: icon, firstSlideX: Math.round(s.querySelector('[data-slot=carousel-item]').getBoundingClientRect().x) }
}))
console.log(JSON.stringify(info))
const buffer = await page.screenshot({ path: out })
const colours = await page.evaluate(async (b64) => {
  const bitmap = await createImageBitmap(await (await fetch(`data:image/png;base64,${b64}`)).blob())
  const c = new OffscreenCanvas(bitmap.width, bitmap.height).getContext('2d'); c.drawImage(bitmap, 0, 0)
  const d = c.getImageData(0, 0, bitmap.width, bitmap.height).data; const set = new Set()
  for (let i = 0; i < d.length; i += 16) set.add((d[i] << 16) | (d[i + 1] << 8) | d[i + 2])
  return set.size
}, buffer.toString('base64'))
console.log(`saved ${out} (${colours} colours${colours < 16 ? ', BLANK' : ''}); errors ${JSON.stringify(errors)}`)
// Public Sans check in the running app (English).
const app = await browser.newPage()
await app.goto(`${base}/sign-in`)
await app.waitForTimeout(2500)
console.log(JSON.stringify(await app.evaluate(async () => {
  await document.fonts.ready
  const faces = [...document.fonts].map((f) => `${f.family}:${f.status}`).filter((f) => /Public/.test(f))
  return {
    bodyFont: getComputedStyle(document.body).fontFamily,
    fontSansToken: getComputedStyle(document.documentElement).getPropertyValue('--font-sans').trim(),
    publicSansFaces: [...new Set(faces)],
    checkPublicSans: document.fonts.check('16px "Public Sans"'),
    checkPublicSansVariable: document.fonts.check('16px "Public Sans Variable"'),
  }
})))
await browser.close()
