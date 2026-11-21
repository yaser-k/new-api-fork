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

import { DEFAULT_SYSTEM_NAME, DEFAULT_LOGO } from '@/lib/constants'
import {
  shownDisplayCurrency,
  useDisplayCurrencyStore,
  type DisplayCurrency,
} from '@/stores/display-currency-store'

export type CurrencyDisplayType = 'USD' | 'CNY' | 'TOKENS' | 'CUSTOM'

export interface CurrencyConfig {
  /** Whether to render quota values as currency instead of raw units */
  displayInCurrency: boolean
  /** Currency presentation strategy configured by the admin */
  quotaDisplayType: CurrencyDisplayType
  /** Number of quota units that equal one USD */
  quotaPerUnit: number
  /** Exchange rate from USD to the configured local currency */
  usdExchangeRate: number
  /** Custom currency symbol configured by the admin (used when type === CUSTOM) */
  customCurrencySymbol: string
  /** Exchange rate from USD to the custom currency (used when type === CUSTOM) */
  customCurrencyExchangeRate: number
}

export interface SystemConfig {
  systemName: string
  logo: string
  footerHtml?: string
  demoSiteEnabled?: boolean
  displayTokenStatEnabled?: boolean
  /** Currency settings as this viewer reads them (see applyDisplayCurrency) */
  currency: CurrencyConfig
  /**
   * The admin's currency settings, whatever the viewer chose: for amounts the
   * site charges in its own currency and for editors of the site's values.
   */
  siteCurrency: CurrencyConfig
}

export const DEFAULT_CURRENCY_CONFIG: CurrencyConfig = {
  displayInCurrency: true,
  quotaDisplayType: 'USD',
  quotaPerUnit: 500000,
  usdExchangeRate: 1,
  customCurrencySymbol: '¤',
  customCurrencyExchangeRate: 1,
}

/** Whether a viewer may read this site's figures in US dollars instead. */
export function offersDisplayCurrency(site: CurrencyConfig): boolean {
  return site.quotaDisplayType === 'CUSTOM' || site.quotaDisplayType === 'CNY'
}

/**
 * The currency settings a viewer reads figures in: the site's own, or US
 * dollars (rate 1, symbol $) when the viewer chose them on a site that shows
 * a currency of its own. Every formatter follows from the display type.
 */
export function applyDisplayCurrency(
  site: CurrencyConfig,
  displayCurrency: DisplayCurrency
): CurrencyConfig {
  if (displayCurrency !== 'USD' || !offersDisplayCurrency(site)) return site
  return { ...site, quotaDisplayType: 'USD' }
}

interface SystemConfigState {
  config: SystemConfig
  loading: boolean
  loadedLogoUrl: string
  setConfig: (config: Partial<SystemConfig>) => void
  setLoadedLogoUrl: (url: string) => void
  setLoading: (loading: boolean) => void
}

/**
 * System configuration store with automatic persistence
 * Manages system name, logo, footer HTML and loading states
 */
export const useSystemConfigStore = create<SystemConfigState>()(
  persist(
    (set) => ({
      config: {
        systemName: DEFAULT_SYSTEM_NAME,
        logo: DEFAULT_LOGO,
        currency: { ...DEFAULT_CURRENCY_CONFIG },
        siteCurrency: { ...DEFAULT_CURRENCY_CONFIG },
      },
      loading: true,
      loadedLogoUrl: DEFAULT_LOGO,
      setConfig: (newConfig) =>
        set((state) => {
          // `currency` edits the settings as displayed. The viewer's choice
          // shows only in the display type, so a type that just repeats it is
          // not a change of the site's; `siteCurrency` sets the site's values.
          const edit = { ...newConfig.currency }
          const shown = state.config.currency.quotaDisplayType
          if (
            edit.quotaDisplayType === shown &&
            shown !== state.config.siteCurrency.quotaDisplayType
          ) {
            delete edit.quotaDisplayType
          }
          const siteCurrency = {
            ...state.config.siteCurrency,
            ...edit,
            ...newConfig.siteCurrency,
          }
          return {
            config: {
              ...state.config,
              ...newConfig,
              siteCurrency,
              currency: applyDisplayCurrency(
                siteCurrency,
                shownDisplayCurrency(useDisplayCurrencyStore.getState())
              ),
            },
          }
        }),
      setLoadedLogoUrl: (url) => set({ loadedLogoUrl: url }),
      setLoading: (loading) => set({ loading }),
    }),
    {
      name: 'system-config-storage',
      partialize: (state) => ({
        config: state.config,
        loadedLogoUrl: state.loadedLogoUrl,
      }),
      merge: (persisted, current) => {
        const saved = (persisted ?? {}) as Partial<SystemConfigState>
        // A config saved before siteCurrency existed holds the site's values
        // in currency.
        const siteCurrency = {
          ...DEFAULT_CURRENCY_CONFIG,
          ...(saved.config?.siteCurrency ?? saved.config?.currency),
        }
        return {
          ...current,
          ...saved,
          config: {
            ...current.config,
            ...saved.config,
            siteCurrency,
            currency: applyDisplayCurrency(
              siteCurrency,
              shownDisplayCurrency(useDisplayCurrencyStore.getState())
            ),
          },
        }
      },
    }
  )
)

useDisplayCurrencyStore.subscribe((state, previous) => {
  const shown = shownDisplayCurrency(state)
  if (shown === shownDisplayCurrency(previous)) return
  useSystemConfigStore.setState((current) => ({
    config: {
      ...current.config,
      currency: applyDisplayCurrency(current.config.siteCurrency, shown),
    },
  }))
})

// Selector helpers for convenience
export const getSystemName = () =>
  useSystemConfigStore.getState().config.systemName

export const getLogo = () => useSystemConfigStore.getState().config.logo

export const getFooterHtml = () =>
  useSystemConfigStore.getState().config.footerHtml
