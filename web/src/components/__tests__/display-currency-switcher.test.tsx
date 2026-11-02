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
import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { AppHeader } from '@/components/layout/components/app-header'
import { PublicHeader } from '@/components/layout/components/public-header'
import { SidebarProvider } from '@/components/ui/sidebar'
import { SearchProvider } from '@/context/search-provider'
import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'
import { useDisplayCurrencyStore } from '@/stores/display-currency-store'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
  type CurrencyDisplayType,
} from '@/stores/system-config-store'

import { DisplayCurrencySwitcher } from '../display-currency-switcher'

function resetStores(): void {
  localStorage.clear()
  useAuthStore.getState().auth.setUser(null)
  useDisplayCurrencyStore.setState(
    useDisplayCurrencyStore.getInitialState(),
    true
  )
  useSystemConfigStore.setState(useSystemConfigStore.getInitialState(), true)
}

function showSite(type: CurrencyDisplayType): void {
  useSystemConfigStore.getState().setConfig({
    currency: {
      ...DEFAULT_CURRENCY_CONFIG,
      quotaDisplayType: type,
      customCurrencySymbol: 'L',
      customCurrencyExchangeRate: 250,
    },
  })
}

async function renderInRouter(node: ReactNode): Promise<void> {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  client.setQueryData(['status'], {})
  const router = createRouter({
    routeTree: createRootRoute({ component: () => node }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  await router.load()
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}

async function chooseCurrency(name: string): Promise<void> {
  await userEvent.click(screen.getByRole('button', { name: 'Select currency' }))
  await userEvent.click(await screen.findByRole('menuitemradio', { name }))
}

beforeEach(() => {
  resetStores()
  vi.spyOn(api, 'get').mockResolvedValue({ data: { success: true, data: '' } })
})

afterEach(() => {
  vi.restoreAllMocks()
  resetStores()
})

describe('display currency switch', () => {
  test.each(['USD', 'TOKENS'] as const)('is not shown on a %s site', (type) => {
    showSite(type)
    useDisplayCurrencyStore.getState().setDisplayCurrency('USD')

    render(<DisplayCurrencySwitcher />)

    expect(
      screen.queryByRole('button', { name: 'Select currency' })
    ).not.toBeInTheDocument()
  })

  test.each([
    { type: 'CUSTOM' as const, site: 'Site currency (L)', symbol: 'L' },
    { type: 'CNY' as const, site: 'Site currency (CNY)', symbol: '¥' },
  ])(
    'offers the site currency and US dollars on a $type site',
    async ({ type, site, symbol }) => {
      showSite(type)

      render(<DisplayCurrencySwitcher />)
      const trigger = screen.getByRole('button', { name: 'Select currency' })
      expect(trigger).toHaveTextContent(symbol)
      await userEvent.click(trigger)

      expect(
        await screen.findByRole('menuitemradio', { name: site })
      ).toHaveAttribute('aria-checked', 'true')
      expect(
        screen.getByRole('menuitemradio', { name: 'US dollar (USD)' })
      ).toHaveAttribute('aria-checked', 'false')
    }
  )

  test('keeps a visitor choice in the browser only', async () => {
    showSite('CUSTOM')
    const put = vi.spyOn(api, 'put')
    render(<DisplayCurrencySwitcher />)

    await chooseCurrency('US dollar (USD)')

    expect(useDisplayCurrencyStore.getState().displayCurrency).toBe('USD')
    expect(
      JSON.parse(localStorage.getItem('display-currency') ?? '{}').state
    ).toEqual({ displayCurrency: 'USD' })
    expect(
      screen.getByRole('button', { name: 'Select currency' })
    ).toHaveTextContent('$')
    expect(put).not.toHaveBeenCalled()
  })

  test('saves a signed-in choice to the account', async () => {
    showSite('CUSTOM')
    useAuthStore.getState().auth.setUser({
      id: 1,
      username: 'viewer',
      role: 1,
      setting: '{"language":"fr"}',
    })
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true } })
    render(<DisplayCurrencySwitcher />)

    await chooseCurrency('US dollar (USD)')

    expect(put).toHaveBeenCalledWith('/api/user/self', {
      display_currency: 'USD',
    })
    await waitFor(() =>
      expect(
        JSON.parse(String(useAuthStore.getState().auth.user?.setting))
      ).toEqual({ language: 'fr', display_currency: 'USD' })
    )
    expect(useDisplayCurrencyStore.getState().displayCurrency).toBe('USD')
  })

  test('sits beside the language switch in the public header', async () => {
    showSite('CUSTOM')

    await renderInRouter(<PublicHeader showNotifications={false} />)

    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Change language' })
          .nextElementSibling
      ).toHaveAccessibleName('Select currency')
    )
    // The compact bar of narrow screens carries it too.
    expect(
      screen.getAllByRole('button', { name: 'Select currency' })
    ).toHaveLength(2)
  })

  test('sits beside the language switch in the console header', async () => {
    showSite('CNY')

    await renderInRouter(
      <SearchProvider>
        <SidebarProvider>
          <AppHeader showConfigDrawer={false} showProfileDropdown={false} />
        </SidebarProvider>
      </SearchProvider>
    )

    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Change language' })
          .nextElementSibling
      ).toHaveAccessibleName('Select currency')
    )
  })
})
