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
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import { useAuthStore, type AuthUser } from '@/stores/auth-store'

/**
 * The currency a viewer reads figures in: the site's display currency, or US
 * dollars. Display only: quota is stored and billed in USD either way.
 */
export type DisplayCurrency = 'site' | 'USD'

export const DEFAULT_DISPLAY_CURRENCY: DisplayCurrency = 'site'

export function parseDisplayCurrency(value: unknown): DisplayCurrency | null {
  return value === 'site' || value === 'USD' ? value : null
}

/** An account's settings, which the server sends as a JSON string. */
export function readAccountSetting(
  user: AuthUser | null
): Record<string, unknown> {
  const setting = user?.setting
  if (typeof setting !== 'string') return setting ?? {}
  try {
    const parsed: unknown = JSON.parse(setting)
    return parsed && typeof parsed === 'object'
      ? (parsed as Record<string, unknown>)
      : {}
  } catch {
    return {}
  }
}

type DisplayCurrencyState = {
  displayCurrency: DisplayCurrency
  setDisplayCurrency: (displayCurrency: DisplayCurrency) => void
}

/**
 * The viewer's choice, kept in the browser for every visitor. A signed-in
 * account's choice wins over the browser's and is copied into it.
 */
export const useDisplayCurrencyStore = create<DisplayCurrencyState>()(
  persist(
    (set) => ({
      displayCurrency: DEFAULT_DISPLAY_CURRENCY,
      setDisplayCurrency: (displayCurrency) => set({ displayCurrency }),
    }),
    {
      name: 'display-currency',
      partialize: (state) => ({ displayCurrency: state.displayCurrency }),
      merge: (persisted, current) => ({
        ...current,
        displayCurrency:
          parseDisplayCurrency(
            (persisted as { displayCurrency?: unknown } | undefined)
              ?.displayCurrency
          ) ?? DEFAULT_DISPLAY_CURRENCY,
      }),
    }
  )
)

useAuthStore.subscribe((state, previous) => {
  if (state.auth.user === previous.auth.user) return
  const accountChoice = parseDisplayCurrency(
    readAccountSetting(state.auth.user).display_currency
  )
  if (
    accountChoice &&
    accountChoice !== useDisplayCurrencyStore.getState().displayCurrency
  ) {
    useDisplayCurrencyStore.getState().setDisplayCurrency(accountChoice)
  }
})
