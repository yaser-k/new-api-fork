// Session 12 pixel comparison of two screenshot sets.
// Usage: node compare-s12.mjs <dir-a> <tag-a> <dir-b> <tag-b>
// For every <name>-<tag-a>.png in dir-a, compares it with
// <name>-<tag-b>.png in dir-b: image sizes, the number of pixels whose RGBA
// differs at all, and the bounding box of those pixels. Decoding and
// comparison run in Chromium (canvas getImageData, no colour management
// on PNG screenshots), so no image library is needed.
import fs from 'node:fs'
import path from 'node:path'

import { chromium } from 'playwright'

const [dirA, tagA, dirB, tagB] = process.argv.slice(2)
const browser = await chromium.launch({
  executablePath: '/opt/pw-browsers/chromium-1194/chrome-linux/chrome',
})
const page = await browser.newPage()

const suffix = `-${tagA}.png`
const names = fs
  .readdirSync(dirA)
  .filter((f) => f.endsWith(suffix))
  .map((f) => f.slice(0, -suffix.length))
  .sort()

let total = 0
for (const name of names) {
  const fileB = path.join(dirB, `${name}-${tagB}.png`)
  if (!fs.existsSync(fileB)) {
    console.log(`${name}: missing in ${dirB}`)
    continue
  }
  const a = fs.readFileSync(path.join(dirA, `${name}${suffix}`)).toString('base64')
  const b = fs.readFileSync(fileB).toString('base64')
  const result = await page.evaluate(
    async ([a64, b64]) => {
      const decode = async (b) => {
        const blob = await (await fetch(`data:image/png;base64,${b}`)).blob()
        const bitmap = await createImageBitmap(blob, {
          colorSpaceConversion: 'none',
          premultiplyAlpha: 'none',
        })
        const canvas = new OffscreenCanvas(bitmap.width, bitmap.height)
        const ctx = canvas.getContext('2d')
        ctx.drawImage(bitmap, 0, 0)
        return ctx.getImageData(0, 0, bitmap.width, bitmap.height)
      }
      const [ia, ib] = await Promise.all([decode(a64), decode(b64)])
      if (ia.width !== ib.width || ia.height !== ib.height) {
        return { size: `${ia.width}x${ia.height} vs ${ib.width}x${ib.height}` }
      }
      let diff = 0
      let x0 = Infinity
      let y0 = Infinity
      let x1 = -1
      let y1 = -1
      for (let i = 0; i < ia.data.length; i += 4) {
        if (
          ia.data[i] !== ib.data[i] ||
          ia.data[i + 1] !== ib.data[i + 1] ||
          ia.data[i + 2] !== ib.data[i + 2] ||
          ia.data[i + 3] !== ib.data[i + 3]
        ) {
          diff++
          const p = i / 4
          const x = p % ia.width
          const y = Math.floor(p / ia.width)
          x0 = Math.min(x0, x)
          y0 = Math.min(y0, y)
          x1 = Math.max(x1, x)
          y1 = Math.max(y1, y)
        }
      }
      return {
        size: `${ia.width}x${ia.height}`,
        diff,
        box: diff ? `x ${x0}-${x1}, y ${y0}-${y1}` : null,
      }
    },
    [a, b]
  )
  if (result.diff) total += result.diff
  console.log(
    `${name}: ${result.size}, ${result.diff ?? 'n/a'} differing pixels${
      result.box ? ` (${result.box})` : ''
    }`
  )
}
console.log(`total differing pixels: ${total}`)
await browser.close()
