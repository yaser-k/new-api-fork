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
import { afterEach, beforeEach, describe, expect, test } from 'vitest'

import {
  formatBillingCurrencyFromUSD,
  formatCurrencyFromUSD,
  formatLocalCurrencyAmount,
  formatQuotaWithCurrency,
  getCurrencyFractionDigits,
  getCurrencyLabel,
} from '@/lib/currency'
import { useAuthStore } from '@/stores/auth-store'
import { useDisplayCurrencyStore } from '@/stores/display-currency-store'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
  type CurrencyConfig,
  type CurrencyDisplayType,
} from '@/stores/system-config-store'

const SITES: Record<CurrencyDisplayType, CurrencyConfig> = {
  CUSTOM: {
    ...DEFAULT_CURRENCY_CONFIG,
    quotaDisplayType: 'CUSTOM',
    customCurrencySymbol: 'L',
    customCurrencyExchangeRate: 250,
  },
  CNY: {
    ...DEFAULT_CURRENCY_CONFIG,
    quotaDisplayType: 'CNY',
    usdExchangeRate: 7,
  },
  USD: { ...DEFAULT_CURRENCY_CONFIG, quotaDisplayType: 'USD' },
  TOKENS: { ...DEFAULT_CURRENCY_CONFIG, quotaDisplayType: 'TOKENS' },
}

/** What every formatter gives on the tag for each site, before any choice. */
const SITE_FIGURES: Record<CurrencyDisplayType, Record<string, string>> = {
  CUSTOM: {
    balance: 'L 375',
    usd: 'L 375',
    billing: 'L 375',
    label: 'L',
    local: 'L 50',
    digits: '4',
  },
  CNY: {
    balance: '¥10.5',
    usd: '¥10.5',
    billing: '¥10.5',
    label: 'CNY',
    local: '¥50',
    digits: '4',
  },
  USD: {
    balance: '$1.5',
    usd: '$1.5',
    billing: '$1.5',
    label: 'USD',
    local: '$50',
    digits: '4',
  },
  TOKENS: {
    balance: '750k',
    usd: '750k',
    billing: '$1.5',
    label: 'Tokens',
    local: '$50',
    digits: '4',
  },
}

const USD_FIGURES = SITE_FIGURES.USD

function figures(): Record<string, string> {
  return {
    balance: formatQuotaWithCurrency(750_000),
    usd: formatCurrencyFromUSD(1.5),
    billing: formatBillingCurrencyFromUSD(1.5),
    label: getCurrencyLabel(),
    local: formatLocalCurrencyAmount(50),
    digits: String(getCurrencyFractionDigits(0.5)),
  }
}

function showSite(site: CurrencyConfig): void {
  useSystemConfigStore.getState().setConfig({ currency: { ...site } })
}

function writeBrowserChoice(value: unknown): void {
  localStorage.setItem(
    'display-currency',
    JSON.stringify({ state: { displayCurrency: value }, version: 0 })
  )
}

function resetStores(): void {
  localStorage.clear()
  useAuthStore.getState().auth.setUser(null)
  useDisplayCurrencyStore.setState(
    useDisplayCurrencyStore.getInitialState(),
    true
  )
  useSystemConfigStore.setState(useSystemConfigStore.getInitialState(), true)
}

beforeEach(resetStores)
afterEach(resetStores)

describe('display currency with no choice made', () => {
  test.each(Object.keys(SITES) as CurrencyDisplayType[])(
    'a %s site shows every figure exactly as before',
    (type) => {
      showSite(SITES[type])

      expect(figures()).toEqual(SITE_FIGURES[type])
    }
  )
})

describe('display currency with US dollars chosen', () => {
  test.each(['CUSTOM', 'CNY'] as const)(
    'a %s site shows balances and prices in $ at rate 1',
    (type) => {
      showSite(SITES[type])

      useDisplayCurrencyStore.getState().setDisplayCurrency('USD')

      expect(figures()).toEqual({
        ...USD_FIGURES,
        // An amount already in local currency keeps the site's symbol.
        local: SITE_FIGURES[type].local,
      })
      expect(useSystemConfigStore.getState().config.currency).toMatchObject({
        quotaDisplayType: 'USD',
      })
    }
  )

  test.each(['USD', 'TOKENS'] as const)(
    'a %s site shows every figure exactly as before',
    (type) => {
      showSite(SITES[type])

      useDisplayCurrencyStore.getState().setDisplayCurrency('USD')

      expect(figures()).toEqual(SITE_FIGURES[type])
    }
  )

  test('choosing the site currency again restores the site figures', () => {
    showSite(SITES.CUSTOM)
    useDisplayCurrencyStore.getState().setDisplayCurrency('USD')

    useDisplayCurrencyStore.getState().setDisplayCurrency('site')

    expect(figures()).toEqual(SITE_FIGURES.CUSTOM)
  })

  test('keeps the admin values reachable unchanged', () => {
    showSite(SITES.CUSTOM)

    useDisplayCurrencyStore.getState().setDisplayCurrency('USD')

    expect(useSystemConfigStore.getState().config.siteCurrency).toEqual(
      SITES.CUSTOM
    )
  })

  test('a writer that copies the displayed config back does not change the site type', () => {
    showSite(SITES.CUSTOM)
    useDisplayCurrencyStore.getState().setDisplayCurrency('USD')
    const shown = useSystemConfigStore.getState().config.currency

    useSystemConfigStore
      .getState()
      .setConfig({ currency: { ...shown, customCurrencySymbol: 'K' } })

    expect(useSystemConfigStore.getState().config.siteCurrency).toEqual({
      ...SITES.CUSTOM,
      customCurrencySymbol: 'K',
    })
    useDisplayCurrencyStore.getState().setDisplayCurrency('site')
    expect(formatCurrencyFromUSD(1.5)).toBe('K 375')
  })

  test('a site switched to US dollars ends the choice', () => {
    showSite(SITES.CUSTOM)
    useDisplayCurrencyStore.getState().setDisplayCurrency('USD')

    useSystemConfigStore
      .getState()
      .setConfig({ siteCurrency: { ...SITES.USD } })

    expect(useSystemConfigStore.getState().config.siteCurrency).toEqual(
      SITES.USD
    )
    useDisplayCurrencyStore.getState().setDisplayCurrency('site')
    expect(figures()).toEqual(SITE_FIGURES.USD)
  })
})

describe('display currency with a config saved before this change', () => {
  test('reads the saved currency as the site values', async () => {
    useDisplayCurrencyStore.getState().setDisplayCurrency('USD')
    localStorage.setItem(
      'system-config-storage',
      JSON.stringify({ state: { config: { currency: SITES.CUSTOM } } })
    )

    await useSystemConfigStore.persist.rehydrate()

    expect(useSystemConfigStore.getState().config.siteCurrency).toEqual(
      SITES.CUSTOM
    )
    expect(figures()).toEqual({
      ...USD_FIGURES,
      local: SITE_FIGURES.CUSTOM.local,
    })
  })
})

describe('display currency precedence', () => {
  test('defaults to the site currency', async () => {
    showSite(SITES.CUSTOM)

    await useDisplayCurrencyStore.persist.rehydrate()

    expect(useDisplayCurrencyStore.getState().displayCurrency).toBe('site')
    expect(formatQuotaWithCurrency(750_000)).toBe('L 375')
  })

  test('the browser choice wins over the default', async () => {
    showSite(SITES.CUSTOM)
    writeBrowserChoice('USD')

    await useDisplayCurrencyStore.persist.rehydrate()

    expect(useDisplayCurrencyStore.getState().displayCurrency).toBe('USD')
    expect(formatQuotaWithCurrency(750_000)).toBe('$1.5')
  })

  test.each([
    { name: 'a JSON setting', setting: '{"display_currency":"site"}' },
    { name: 'an object setting', setting: { display_currency: 'site' } },
  ])(
    'the account choice in $name wins over the browser and is kept in it',
    async ({ setting }) => {
      showSite(SITES.CUSTOM)
      writeBrowserChoice('USD')
      await useDisplayCurrencyStore.persist.rehydrate()

      useAuthStore
        .getState()
        .auth.setUser({ id: 1, username: 'viewer', role: 1, setting })

      expect(useDisplayCurrencyStore.getState().displayCurrency).toBe('site')
      expect(formatQuotaWithCurrency(750_000)).toBe('L 375')
      expect(
        JSON.parse(localStorage.getItem('display-currency') ?? '{}').state
      ).toEqual({ displayCurrency: 'site' })
    }
  )

  test('an account without a choice keeps the browser choice', async () => {
    showSite(SITES.CUSTOM)
    writeBrowserChoice('USD')
    await useDisplayCurrencyStore.persist.rehydrate()

    useAuthStore.getState().auth.setUser({
      id: 1,
      username: 'viewer',
      role: 1,
      setting: '{"language":"fr"}',
    })

    expect(useDisplayCurrencyStore.getState().displayCurrency).toBe('USD')
  })
})

describe('display currency stored values', () => {
  test.each(['EUR', 'usd', '', 1, null])(
    'ignores the browser value %j',
    async (value) => {
      showSite(SITES.CUSTOM)
      writeBrowserChoice(value)

      await useDisplayCurrencyStore.persist.rehydrate()

      expect(useDisplayCurrencyStore.getState().displayCurrency).toBe('site')
      expect(formatQuotaWithCurrency(750_000)).toBe('L 375')
    }
  )

  test.each(['EUR', 'usd', '', 1])(
    'ignores the account value %j',
    async (value) => {
      showSite(SITES.CUSTOM)
      writeBrowserChoice('USD')
      await useDisplayCurrencyStore.persist.rehydrate()

      useAuthStore.getState().auth.setUser({
        id: 1,
        username: 'viewer',
        role: 1,
        setting: { display_currency: value },
      })

      expect(useDisplayCurrencyStore.getState().displayCurrency).toBe('USD')
    }
  )
})
