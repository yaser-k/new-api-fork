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
import { afterEach, describe, expect, test } from 'vitest'

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
