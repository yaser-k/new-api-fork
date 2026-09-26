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
import { describe, expect, it } from 'vitest'

import { toIntlLocale } from '@/i18n/languages'

import { formatResponseTime } from '../channel-utils'

const interpolate = (key: string, options?: { value?: number | string }) =>
  key.replace('{{value}}', String(options?.value))

describe('formatResponseTime milliseconds', () => {
  it('writes milliseconds with Persian digits when the interface language is Persian', () => {
    expect(formatResponseTime(456, interpolate, toIntlLocale('fa'))).toBe(
      '۴۵۶ms'
    )
  })

  it.each(['en', 'zhCN', 'zhTW', 'fr', 'ru', 'ja', 'vi'])(
    'keeps Latin digits for %s, unchanged from the raw value',
    (language) => {
      expect(formatResponseTime(456, interpolate, toIntlLocale(language))).toBe(
        '456ms'
      )
    }
  )

  it('falls back to Latin digits for an unknown interface language', () => {
    expect(
      formatResponseTime(456, interpolate, toIntlLocale('xx-invalid'))
    ).toBe('456ms')
  })

  it('keeps the seconds value unchanged', () => {
    expect(formatResponseTime(1234, interpolate, toIntlLocale('fa'))).toBe(
      '1.23s'
    )
  })
})
