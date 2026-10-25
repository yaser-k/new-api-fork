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
import { describe, expect, test } from 'vitest'

import {
  parseLocalizedTextString,
  resolveLocalizedTextString,
} from '../localized-text-string'

const BOTH = JSON.stringify({ en: 'Fast model', fr: 'Modèle rapide' })

describe('parseLocalizedTextString', () => {
  test('returns null for plain text', () => {
    expect(parseLocalizedTextString('Fast model')).toBeNull()
  })

  test('returns null for empty and missing values', () => {
    expect(parseLocalizedTextString('')).toBeNull()
    expect(parseLocalizedTextString(null)).toBeNull()
    expect(parseLocalizedTextString(undefined)).toBeNull()
  })

  test('returns null for text that only looks like JSON', () => {
    expect(parseLocalizedTextString('{not json}')).toBeNull()
    expect(parseLocalizedTextString('{} and more')).toBeNull()
  })

  test('returns null for JSON that is not an object of strings', () => {
    expect(parseLocalizedTextString('["en", "fr"]')).toBeNull()
    expect(parseLocalizedTextString('{"en": 1}')).toBeNull()
    expect(parseLocalizedTextString('{"en": {"text": "x"}}')).toBeNull()
    expect(parseLocalizedTextString('{}')).toBeNull()
    expect(parseLocalizedTextString('{"en": "  "}')).toBeNull()
  })

  test('returns the map for an object of texts', () => {
    expect(parseLocalizedTextString(`  ${BOTH}\n`)).toEqual({
      en: 'Fast model',
      fr: 'Modèle rapide',
    })
  })
})

describe('resolveLocalizedTextString', () => {
  test('keeps plain text unchanged in every language', () => {
    expect(resolveLocalizedTextString('Fast model', 'fr')).toBe('Fast model')
    expect(resolveLocalizedTextString('{not json}', 'en')).toBe('{not json}')
  })

  test('returns an empty string for missing values', () => {
    expect(resolveLocalizedTextString(null, 'en')).toBe('')
    expect(resolveLocalizedTextString(undefined, 'en')).toBe('')
  })

  test('picks the text for the interface language', () => {
    expect(resolveLocalizedTextString(BOTH, 'fr')).toBe('Modèle rapide')
    expect(resolveLocalizedTextString(BOTH, 'en')).toBe('Fast model')
  })

  test('falls back to English, then to the first language', () => {
    expect(resolveLocalizedTextString(BOTH, 'ja')).toBe('Fast model')
    expect(
      resolveLocalizedTextString(JSON.stringify({ fr: 'Rapide' }), 'ja')
    ).toBe('Rapide')
  })

  test('accepts region tags and i18next codes', () => {
    const value = JSON.stringify({ en: 'Fast', 'zh-TW': '快速' })
    expect(resolveLocalizedTextString(value, 'zhTW')).toBe('快速')
    expect(resolveLocalizedTextString(value, 'en-US')).toBe('Fast')
  })
})
