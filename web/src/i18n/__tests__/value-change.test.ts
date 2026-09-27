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

import { formatValueChange, INTERFACE_LANGUAGE_OPTIONS } from '../languages'

const FSI = '⁨'
const PDI = '⁩'

describe('formatValueChange', () => {
  it.each(
    INTERFACE_LANGUAGE_OPTIONS.filter((option) => option.code !== 'fa').map(
      (option) => option.code
    )
  )('for %s, keeps the left-to-right arrow and the plain values', (code) => {
    expect(formatValueChange('v1.0.0', 'v1.2.0', code)).toBe('v1.0.0 → v1.2.0')
  })

  it('for Persian, isolates each value and points the arrow to the left', () => {
    expect(formatValueChange('$۱٫۰۰', '$۲٫۰۰', 'fa')).toBe(
      `${FSI}$۱٫۰۰${PDI} ← ${FSI}$۲٫۰۰${PDI}`
    )
  })

  it('accepts the Persian Intl locale as well as the interface code', () => {
    expect(formatValueChange('a', 'b', 'fa-u-nu-latn')).toBe(
      `${FSI}a${PDI} ← ${FSI}b${PDI}`
    )
  })

  it('falls back to the left-to-right arrow for an unknown language', () => {
    expect(formatValueChange('a', 'b', 'xx-invalid')).toBe('a → b')
    expect(formatValueChange('a', 'b', undefined)).toBe('a → b')
  })
})
