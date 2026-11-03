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
import i18next from 'i18next'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { INTERFACE_LANGUAGE_OPTIONS, toIntlLocale } from '@/i18n/languages'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
  type CurrencyConfig,
} from '@/stores/system-config-store'

import {
  formatBillingCurrencyFromUSD,
  formatCurrencyFromUSD,
  getCurrencyLabel,
} from '../currency'

const FSI = '⁨'
const PDI = '⁩'

function useCurrency(currency: Partial<CurrencyConfig>) {
  useSystemConfigStore.getState().setConfig({
    currency: { ...DEFAULT_CURRENCY_CONFIG, ...currency },
  })
}

afterEach(() => {
  useSystemConfigStore
    .getState()
    .setConfig({ currency: { ...DEFAULT_CURRENCY_CONFIG } })
})

describe('custom currency symbols in a right-to-left script', () => {
  test.each(['تومان', 'ر.س', 'ש"ח'])(
    'isolates %s from the number it precedes',
    (symbol) => {
      useCurrency({ quotaDisplayType: 'CUSTOM', customCurrencySymbol: symbol })

      expect(formatBillingCurrencyFromUSD(2016150, { locale: 'en-US' })).toBe(
        `${FSI}${symbol}${PDI} 2,016,150`
      )
      expect(getCurrencyLabel()).toBe(`${FSI}${symbol}${PDI}`)
    }
  )
})

describe('left-to-right currency symbols', () => {
  test.each<{
    name: string
    currency: Partial<CurrencyConfig>
    expected: string
    label: string
  }>([
    {
      name: 'USD keeps the dollar sign in front',
      currency: { quotaDisplayType: 'USD' },
      expected: '$1,234.5',
      label: 'USD',
    },
    {
      name: 'CNY keeps the yuan sign in front',
      currency: { quotaDisplayType: 'CNY', usdExchangeRate: 7 },
      expected: '¥8,641.5',
      label: 'CNY',
    },
    {
      name: 'a custom euro sign is not wrapped',
      currency: { quotaDisplayType: 'CUSTOM', customCurrencySymbol: '€' },
      expected: '€ 1,234.5',
      label: '€',
    },
    {
      name: 'a custom emoji symbol is not wrapped',
      currency: { quotaDisplayType: 'CUSTOM', customCurrencySymbol: '🐱' },
      expected: '🐱 1,234.5',
      label: '🐱',
    },
    {
      name: 'a custom shekel sign, which has no direction of its own, is not wrapped',
      currency: { quotaDisplayType: 'CUSTOM', customCurrencySymbol: '₪' },
      expected: '₪ 1,234.5',
      label: '₪',
    },
  ])('$name', (item) => {
    useCurrency(item.currency)

    expect(formatBillingCurrencyFromUSD(1234.5, { locale: 'en-US' })).toBe(
      item.expected
    )
    expect(formatCurrencyFromUSD(1234.5, { locale: 'en-US' })).toBe(
      item.expected
    )
    expect(getCurrencyLabel()).toBe(item.label)
  })

  test('TOKENS mode shows token counts and USD billing prices as before', () => {
    useCurrency({ quotaDisplayType: 'TOKENS', quotaPerUnit: 500000 })

    expect(formatCurrencyFromUSD(1234.5, { locale: 'en-US' })).toBe('617250k')
    expect(formatBillingCurrencyFromUSD(1234.5, { locale: 'en-US' })).toBe(
      '$1,234.5'
    )
    expect(getCurrencyLabel()).toBe('Tokens')
  })
})

describe('amounts in the interface language', () => {
  const TOMAN = `${FSI}تومان${PDI}`
  const useToman = () =>
    useCurrency({ quotaDisplayType: 'CUSTOM', customCurrencySymbol: 'تومان' })

  beforeEach(() => {
    // The browser of a Persian reader may still be set to English; pin the
    // runtime locale so only the interface language can bring Persian digits.
    const NativeNumberFormat = Intl.NumberFormat
    vi.spyOn(Intl, 'NumberFormat').mockImplementation(function (
      locales?: Intl.LocalesArgument,
      options?: Intl.NumberFormatOptions
    ) {
      return new NativeNumberFormat(locales ?? 'en-US', options)
    } as typeof Intl.NumberFormat)
  })

  afterEach(async () => {
    await i18next.changeLanguage('en')
  })

  test('an English interface keeps the Persian currency word before the number', async () => {
    await i18next.changeLanguage('en')
    useToman()

    expect(formatBillingCurrencyFromUSD(203700)).toBe(`${TOMAN} 203,700`)
  })

  test.each<{
    name: string
    currency: Partial<CurrencyConfig>
    persian: string
    english: string
  }>([
    {
      name: 'the dollar sign',
      currency: { quotaDisplayType: 'USD' },
      persian: '\u200e$۲۰۳٬۷۰۰',
      english: '$203,700',
    },
    {
      name: 'the yuan sign',
      currency: { quotaDisplayType: 'CNY', usdExchangeRate: 7 },
      persian: '\u200e¥۱٬۴۲۵٬۹۰۰',
      english: '¥1,425,900',
    },
    {
      name: 'a custom dollar sign',
      currency: { quotaDisplayType: 'CUSTOM', customCurrencySymbol: '$' },
      persian: '$ ۲۰۳٬۷۰۰',
      english: '$ 203,700',
    },
    {
      name: 'a custom euro sign',
      currency: { quotaDisplayType: 'CUSTOM', customCurrencySymbol: '€' },
      persian: '€ ۲۰۳٬۷۰۰',
      english: '€ 203,700',
    },
  ])(
    '$name stays before the number in a Persian and an English interface',
    async (item) => {
      useCurrency(item.currency)

      await i18next.changeLanguage('fa')
      expect(formatBillingCurrencyFromUSD(203700)).toBe(item.persian)
      await i18next.changeLanguage('en')
      expect(formatBillingCurrencyFromUSD(203700)).toBe(item.english)
    }
  )

  test('an explicit locale wins over a Persian interface', async () => {
    await i18next.changeLanguage('fa')
    useToman()

    expect(formatBillingCurrencyFromUSD(203700, { locale: 'en-US' })).toBe(
      `${TOMAN} 203,700`
    )
  })

  test('an explicit Persian locale wins over an English interface', async () => {
    await i18next.changeLanguage('en')
    useCurrency({ quotaDisplayType: 'USD' })

    expect(formatBillingCurrencyFromUSD(203700, { locale: 'fa' })).toBe(
      '\u200e$۲۰۳٬۷۰۰'
    )
  })

  test('an explicitly undefined locale keeps the runtime locale and the symbol first', async () => {
    await i18next.changeLanguage('fa')
    useToman()

    expect(formatBillingCurrencyFromUSD(203700, { locale: undefined })).toBe(
      `${TOMAN} 203,700`
    )
  })

  test.each(
    INTERFACE_LANGUAGE_OPTIONS.filter(
      (option) => (option.code as string) !== 'fa'
    ).map((option) => option.code)
  )(
    'the %s interface formats in its own locale with the currency word first',
    async (code) => {
      await i18next.changeLanguage(code)
      useToman()

      const formatted = formatBillingCurrencyFromUSD(1234.5)
      expect(formatted).toBe(
        formatBillingCurrencyFromUSD(1234.5, { locale: toIntlLocale(code) })
      )
      expect(formatted.startsWith(`${TOMAN} `)).toBe(true)
    }
  )

  test('an interface language that is not a valid locale falls back to the runtime locale', async () => {
    await i18next.changeLanguage('not a language')
    useToman()

    expect(formatBillingCurrencyFromUSD(1234.5)).toBe(`${TOMAN} 1,234.5`)
  })
})
