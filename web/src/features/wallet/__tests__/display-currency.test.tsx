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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { STATUS_QUERY_KEY } from '@/lib/status-query'
import {
  useDisplayCurrencyStore,
  type DisplayCurrency,
} from '@/stores/display-currency-store'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
} from '@/stores/system-config-store'

import { Wallet } from '../index'

// The site shows CNY at 7 per USD and charges 5 CNY for 1 USD of balance.
const PRICE = 5

function resetStores(): void {
  localStorage.clear()
  useDisplayCurrencyStore.setState(
    useDisplayCurrencyStore.getInitialState(),
    true
  )
  useSystemConfigStore.setState(useSystemConfigStore.getInitialState(), true)
}

function renderWallet(): void {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  queryClient.setQueryData(STATUS_QUERY_KEY, { price: PRICE })
  render(
    <QueryClientProvider client={queryClient}>
      <Wallet />
    </QueryClientProvider>
  )
}

beforeEach(() => {
  resetStores()
  useSystemConfigStore.getState().setConfig({
    currency: {
      ...DEFAULT_CURRENCY_CONFIG,
      quotaDisplayType: 'CNY',
      usdExchangeRate: 7,
    },
  })
  vi.spyOn(api, 'get').mockImplementation(async (url: string) => {
    if (url === '/api/user/topup/info') {
      return {
        data: {
          success: true,
          data: {
            enable_online_topup: true,
            pay_methods: [{ name: 'Example Pay', type: 'example' }],
            min_topup: 10,
            amount_options: [10],
            discount: {},
          },
        },
      }
    }
    return { data: { success: true, data: null } }
  })
  vi.spyOn(api, 'post').mockImplementation(async (url: string, body) => {
    if (url !== '/api/user/amount') throw new Error(`Unexpected POST ${url}`)
    const amount = (body as { amount: number }).amount
    return { data: { success: true, data: String(amount * PRICE) } }
  })
})

afterEach(() => {
  vi.restoreAllMocks()
  resetStores()
})

describe('wallet top-up with either display currency', () => {
  test.each<DisplayCurrency>(['site', 'USD'])(
    'charges and shows the top-up in the site currency when the viewer reads %s',
    async (choice) => {
      useDisplayCurrencyStore.getState().setDisplayCurrency(choice)
      renderWallet()

      const amountToPay = (await screen.findByText('Amount to pay:'))
        .parentElement as HTMLElement
      expect(await within(amountToPay).findByText(/^50/)).toBeVisible()
      expect(api.post).toHaveBeenCalledWith(
        '/api/user/amount',
        { amount: 10 },
        expect.anything()
      )
      // The preset still shows 10 USD of balance at the site's rate.
      expect(screen.getByRole('button', { name: /^70/ })).toBeVisible()
    }
  )

  test('shows no US dollar figure to a viewer who reads the site currency', async () => {
    renderWallet()

    const amountToPay = (await screen.findByText('Amount to pay:'))
      .parentElement as HTMLElement
    expect(await within(amountToPay).findByText(/^50/)).toHaveTextContent(
      /^50$/
    )
  })

  test('shows the payment in US dollars beside it, the amount divided by the price', async () => {
    useDisplayCurrencyStore.getState().setDisplayCurrency('USD')
    renderWallet()

    const amountToPay = (await screen.findByText('Amount to pay:'))
      .parentElement as HTMLElement
    expect(await within(amountToPay).findByText(/^50/)).toHaveTextContent(
      '50≈ $10'
    )
  })

  test('the payment confirmation keeps the site currency and shows the US dollar figure', async () => {
    useDisplayCurrencyStore.getState().setDisplayCurrency('USD')
    renderWallet()
    const amountToPay = (await screen.findByText('Amount to pay:'))
      .parentElement as HTMLElement
    await within(amountToPay).findByText(/^50/)

    await userEvent.click(screen.getByRole('button', { name: /Example Pay/ }))

    const dialog = (await screen.findByText('Review your payment details'))
      .parentElement?.parentElement as HTMLElement
    expect(
      within(dialog).getByText('Topup Amount').nextSibling
    ).toHaveTextContent('¥70')
    expect(within(dialog).getByText('You Pay').nextSibling).toHaveTextContent(
      '50≈ $10'
    )
  })
})
