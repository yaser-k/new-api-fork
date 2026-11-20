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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'
import { useDisplayCurrencyStore } from '@/stores/display-currency-store'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
  type CurrencyDisplayType,
} from '@/stores/system-config-store'

import { LanguagePreferencesCard } from '../components/language-preferences-card'

function resetStores(): void {
  localStorage.clear()
  useAuthStore.getState().auth.setUser(null)
  useDisplayCurrencyStore.setState(
    useDisplayCurrencyStore.getInitialState(),
    true
  )
  useSystemConfigStore.setState(useSystemConfigStore.getInitialState(), true)
}

function renderCard(type: CurrencyDisplayType, onProfileUpdate = vi.fn()) {
  useSystemConfigStore.getState().setConfig({
    currency: {
      ...DEFAULT_CURRENCY_CONFIG,
      quotaDisplayType: type,
      customCurrencySymbol: 'L',
      customCurrencyExchangeRate: 250,
    },
  })
  useAuthStore
    .getState()
    .auth.setUser({ id: 1, username: 'viewer', role: 1, setting: '{}' })
  const client = new QueryClient()
  client.setQueryData(['status'], {})
  render(
    <QueryClientProvider client={client}>
      <LanguagePreferencesCard
        profile={null}
        onProfileUpdate={onProfileUpdate}
      />
    </QueryClientProvider>
  )
}

beforeEach(resetStores)
afterEach(() => {
  vi.restoreAllMocks()
  resetStores()
})

describe('currency preference', () => {
  test.each(['USD', 'TOKENS'] as const)(
    'is not offered on a %s site',
    (type) => {
      renderCard(type)

      expect(screen.getByText('Interface Language')).toBeVisible()
      expect(
        screen.queryByRole('combobox', { name: 'Currency' })
      ).not.toBeInTheDocument()
    }
  )

  test('sits beside the interface language and saves the choice to the account', async () => {
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true } })
    const onProfileUpdate = vi.fn()
    renderCard('CUSTOM', onProfileUpdate)
    const select = screen.getByRole('combobox', { name: 'Currency' })
    expect(select).toHaveTextContent('Site currency (L)')

    await userEvent.click(select)
    await userEvent.click(
      await screen.findByRole('option', { name: 'US dollar (USD)' })
    )

    await waitFor(() => expect(onProfileUpdate).toHaveBeenCalled())
    expect(put).toHaveBeenCalledWith('/api/user/self', {
      display_currency: 'USD',
    })
    expect(useDisplayCurrencyStore.getState().displayCurrency).toBe('USD')
    expect(select).toHaveTextContent('US dollar (USD)')
  })

  test('goes back to the previous choice when the account refuses it', async () => {
    const put = vi.spyOn(api, 'put').mockResolvedValue({
      data: { success: false, message: 'Invalid parameters' },
    })
    renderCard('CNY')
    const select = screen.getByRole('combobox', { name: 'Currency' })

    await userEvent.click(select)
    await userEvent.click(
      await screen.findByRole('option', { name: 'US dollar (USD)' })
    )

    await waitFor(() => expect(select).toHaveTextContent('Site currency (CNY)'))
    expect(useDisplayCurrencyStore.getState().displayCurrency).toBe('site')
    expect(put).toHaveBeenCalledTimes(1)
  })
})
