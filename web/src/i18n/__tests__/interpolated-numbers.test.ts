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
import { createInstance } from 'i18next'
import { describe, expect, it } from 'vitest'

import en from '../locales/en.json'
import fa from '../locales/fa.json'
import { localeNumberFormatter } from '../number-interpolation'

const RESEND = 'Resend ({{seconds}}s)'

async function translatorFor(lng: string) {
  const i18n = createInstance()
  await i18n.use(localeNumberFormatter).init({
    lng,
    fallbackLng: 'en',
    nsSeparator: false,
    resources: { en: { translation: { [RESEND]: RESEND } } },
    interpolation: { escapeValue: false },
  })
  return i18n
}

describe('numbers interpolated into translations', () => {
  it.each(['en', 'zhCN', 'zhTW', 'fr', 'ru', 'ja', 'vi'])(
    'keeps Latin digits unchanged in the %s interface language',
    async (lng) => {
      const i18n = await translatorFor(lng)
      expect(i18n.t(RESEND, { seconds: 23 })).toBe('Resend (23s)')
      expect(i18n.t('{{value}} left', { value: 0.0001 })).toBe('0.0001 left')
    }
  )

  it('writes the resend countdowns in Persian digits in the Persian interface language', async () => {
    const i18n = createInstance()
    await i18n.use(localeNumberFormatter).init({
      lng: 'fa',
      fallbackLng: 'en',
      nsSeparator: false,
      resources: { en, fa },
      interpolation: { escapeValue: false },
    })
    expect(i18n.t(RESEND, { seconds: 23 })).toBe('ارسال دوباره (۲۳ ثانیه)')
    expect(i18n.t('Resend in {{seconds}}s', { seconds: 23 })).toBe(
      'ارسال دوباره پس از ۲۳ ثانیه'
    )
  })

  it('falls back to the runtime digits for an invalid language code', async () => {
    const i18n = await translatorFor('not a language')
    expect(i18n.t(RESEND, { seconds: 23 })).toBe('Resend (23s)')
  })

  it('writes the digits of a language whose numbering system is not Latin', async () => {
    const i18n = await translatorFor('en-u-nu-arabext')
    expect(i18n.t(RESEND, { seconds: 23 })).toBe('Resend (۲۳s)')
    expect(i18n.t('{{value}} left', { value: 1234.0001 })).toBe(
      '۱۲۳۴٫۰۰۰۱ left'
    )
  })

  it('leaves interpolated strings and non-finite numbers as they are', async () => {
    const i18n = await translatorFor('en-u-nu-arabext')
    expect(i18n.t('Key {{id}}', { id: '23' })).toBe('Key 23')
    expect(i18n.t('{{value}} left', { value: Number.NaN })).toBe('NaN left')
  })

  it('keeps named formats working', async () => {
    const i18n = await translatorFor('de')
    expect(i18n.t('{{value, number}}', { value: 1234.5 })).toBe('1.234,5')
  })

  it('follows a language change', async () => {
    const i18n = await translatorFor('en')
    expect(i18n.t(RESEND, { seconds: 9 })).toBe('Resend (9s)')
    await i18n.changeLanguage('en-u-nu-arabext')
    expect(i18n.t(RESEND, { seconds: 9 })).toBe('Resend (۹s)')
  })
})
