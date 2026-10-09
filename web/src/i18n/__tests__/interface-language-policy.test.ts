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
  getOfferedInterfaceLanguages,
  isInterfaceLanguageOffered,
  readInterfaceLanguageSettings,
  resolveInterfaceLanguage,
} from '../interface-language-policy'

const unset = readInterfaceLanguageSettings({})

describe('readInterfaceLanguageSettings', () => {
  test('an empty status is the unset setting', () => {
    expect(unset).toEqual({
      languages: [],
      defaultLanguage: '',
      currencySymbols: {},
    })
    expect(readInterfaceLanguageSettings(null)).toEqual(unset)
  })

  test('keeps shipped languages in order, once, and drops unknown codes', () => {
    const settings = readInterfaceLanguageSettings({
      interface_languages: ['ja', 'en', 'xx', 'ja', 42],
      default_interface_language: 'ja',
      custom_currency_symbols: { en: ' EUR ', xx: 'X', ja: '' },
    })
    expect(settings.languages).toEqual(['ja', 'en'])
    expect(settings.defaultLanguage).toBe('ja')
    expect(settings.currencySymbols).toEqual({ en: 'EUR' })
  })

  test('an unknown default is no default, not English', () => {
    const settings = readInterfaceLanguageSettings({
      default_interface_language: 'xx',
    })
    expect(settings.defaultLanguage).toBe('')
  })
})

describe('getOfferedInterfaceLanguages', () => {
  test('unset offers every shipped language', () => {
    const codes = getOfferedInterfaceLanguages(unset).map((l) => l.code)
    expect(codes).toContain('en')
    expect(codes).toContain('fr')
    expect(codes.length).toBeGreaterThan(2)
  })

  test('a configured list is offered in its own order', () => {
    const settings = readInterfaceLanguageSettings({
      interface_languages: ['fr', 'en'],
    })
    expect(getOfferedInterfaceLanguages(settings).map((l) => l.code)).toEqual([
      'fr',
      'en',
    ])
    expect(isInterfaceLanguageOffered(settings, 'fr')).toBe(true)
    expect(isInterfaceLanguageOffered(settings, 'ja')).toBe(false)
  })
})

describe('resolveInterfaceLanguage', () => {
  const frFirst = readInterfaceLanguageSettings({
    interface_languages: ['fr', 'en'],
    default_interface_language: 'fr',
  })

  test('unset never switches', () => {
    expect(resolveInterfaceLanguage('ja', unset, null)).toBeNull()
    expect(resolveInterfaceLanguage('en', unset, 'ja')).toBeNull()
  })

  test('without a choice, the default applies', () => {
    expect(resolveInterfaceLanguage('en', frFirst, null)).toBe('fr')
    expect(resolveInterfaceLanguage('fr', frFirst, null)).toBeNull()
  })

  test('a choice that is offered is kept, and restored', () => {
    expect(resolveInterfaceLanguage('en', frFirst, 'en')).toBeNull()
    expect(resolveInterfaceLanguage('fr', frFirst, 'en')).toBe('en')
  })

  test('a choice that is no longer offered gives way to the default', () => {
    expect(resolveInterfaceLanguage('ja', frFirst, 'ja')).toBe('fr')
  })

  test('without a default, only a language that is not offered changes', () => {
    const list = readInterfaceLanguageSettings({
      interface_languages: ['en', 'fr'],
    })
    expect(resolveInterfaceLanguage('fr', list, null)).toBeNull()
    expect(resolveInterfaceLanguage('ja', list, null)).toBe('en')
  })

  test('a default that is not offered falls back to the first offered', () => {
    const settings = readInterfaceLanguageSettings({
      interface_languages: ['en'],
      default_interface_language: 'ja',
    })
    expect(resolveInterfaceLanguage('fr', settings, null)).toBe('en')
  })
})
