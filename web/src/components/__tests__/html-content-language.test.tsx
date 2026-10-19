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
import { render, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, test } from 'vitest'

import { HtmlContent } from '../html-content'

function isolatedWrapper(container: HTMLElement): HTMLElement {
  const host = container.firstElementChild as HTMLElement | null
  const wrapper = host?.shadowRoot?.lastElementChild as HTMLElement | null
  if (!wrapper) throw new Error('no isolated wrapper rendered')
  return wrapper
}

describe('isolated HtmlContent', () => {
  afterEach(() => {
    document.documentElement.removeAttribute('lang')
    document.documentElement.removeAttribute('dir')
    document.documentElement.classList.remove('dark')
  })

  test('the wrapper carries the page language and direction', () => {
    document.documentElement.setAttribute('lang', 'en')
    document.documentElement.setAttribute('dir', 'ltr')
    const { container } = render(
      <HtmlContent variant='isolated' content='<p>Hello</p>' />
    )
    const wrapper = isolatedWrapper(container)
    expect(wrapper.getAttribute('lang')).toBe('en')
    expect(wrapper.getAttribute('dir')).toBe('ltr')
  })

  test('the wrapper follows a later language change, and the theme', async () => {
    document.documentElement.setAttribute('lang', 'en')
    const { container } = render(
      <HtmlContent variant='isolated' content='<p>Hello</p>' />
    )
    const wrapper = isolatedWrapper(container)

    document.documentElement.setAttribute('lang', 'ar')
    document.documentElement.setAttribute('dir', 'rtl')
    document.documentElement.classList.add('dark')

    await waitFor(() => {
      expect(wrapper.getAttribute('lang')).toBe('ar')
      expect(wrapper.getAttribute('dir')).toBe('rtl')
      expect(wrapper.classList.contains('dark')).toBe(true)
    })
  })

  test('without a page language the wrapper has none', () => {
    const { container } = render(
      <HtmlContent variant='isolated' content='<p>Hello</p>' />
    )
    expect(isolatedWrapper(container).hasAttribute('lang')).toBe(false)
  })
})
