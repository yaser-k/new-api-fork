/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import fs from 'node:fs'
import path from 'node:path'

import { compile } from '@tailwindcss/node'
import { afterAll, afterEach, beforeAll, describe, expect, it } from 'vitest'

// Code, keyboard keys and sample output are left-to-right text. On a
// right-to-left page they inherit `direction: rtl`, so a path such as
// `/api/ratio` or a list of regexes has its slashes and punctuation moved to
// the other end. The stylesheet gives them their own direction there, and
// leaves left-to-right pages untouched.

const STYLESHEET = path.resolve(import.meta.dirname, '../index.css')

let style: HTMLStyleElement
let host: HTMLElement

beforeAll(async () => {
  const compiler = await compile(fs.readFileSync(STYLESHEET, 'utf8'), {
    base: path.dirname(STYLESHEET),
    onDependency: () => {},
  })
  style = document.createElement('style')
  style.textContent = compiler.build([])
  document.head.append(style)
}, 60_000)

afterEach(() => {
  host?.remove()
})

afterAll(() => {
  style.remove()
})

function renderInside(dir: 'rtl' | 'ltr', html: string): HTMLElement {
  host = document.createElement('div')
  host.setAttribute('dir', dir)
  host.innerHTML = html
  document.body.append(host)
  return host
}

function styleOf(root: HTMLElement, selector: string): CSSStyleDeclaration {
  const element = root.querySelector(selector)
  if (!element) throw new Error(`missing ${selector}`)
  return getComputedStyle(element)
}

describe('code direction', () => {
  it('isolates inline code as left-to-right text on a right-to-left page', () => {
    const root = renderInside('rtl', '<p>از طریق <code>/api/ratio</code></p>')

    const code = styleOf(root, 'code')
    expect(code.direction).toBe('ltr')
    expect(code.unicodeBidi).toBe('isolate')
  })

  it('applies to keyboard keys and sample output too', () => {
    const root = renderInside('rtl', '<kbd>Ctrl+K</kbd><samp>200 OK</samp>')

    expect(styleOf(root, 'kbd').direction).toBe('ltr')
    expect(styleOf(root, 'samp').direction).toBe('ltr')
  })

  it('respects an explicit dir attribute on the element', () => {
    const root = renderInside('rtl', '<code dir="rtl">مقدار</code>')

    expect(styleOf(root, 'code').direction).toBe('rtl')
  })

  it('adds nothing on a left-to-right page', () => {
    const root = renderInside('ltr', '<p>via <code>/api/ratio</code></p>')

    expect(styleOf(root, 'code').unicodeBidi).not.toBe('isolate')
  })
})
