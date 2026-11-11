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
import { useNavigate, useSearch } from '@tanstack/react-router'

import { USD_PRICING_CURRENCY } from '@/features/model-pricing/currency'
import { useDisplayCurrencyOptions } from '@/hooks/use-display-currency'
import {
  parseDisplayCurrency,
  useDisplayCurrencyStore,
  type DisplayCurrency,
} from '@/stores/display-currency-store'

/** The pricing toolbar's currency switch. */
export type CurrencyView = {
  value: DisplayCurrency
  items: { value: DisplayCurrency; label: string }[]
  select: (value: unknown) => void
}

function showCurrencyView(match: {
  search: { currency?: DisplayCurrency }
}): void {
  useDisplayCurrencyStore
    .getState()
    .setViewDisplayCurrency(match.search.currency ?? null)
}

/**
 * Route lifecycle of the pricing pages: while one is open, its `currency`
 * search parameter is the currency figures are shown in; leaving it gives
 * every other page the viewer's choice again. The router calls these before
 * it renders the next page, so no reader there sees this page's view.
 */
export const currencyViewLifecycle = {
  onEnter: showCurrencyView,
  onStay: showCurrencyView,
  onLeave: () =>
    useDisplayCurrencyStore.getState().setViewDisplayCurrency(null),
}

/**
 * The pricing list's currency switch, or null on a site that offers no
 * choice. Like the token unit it is a view of the page, not the viewer's
 * choice: a click rewrites the link (without `currency` when it matches the
 * choice) and never the choice kept in the browser or the account.
 */
export function useCurrencyView(): CurrencyView | null {
  const options = useDisplayCurrencyOptions()
  const currency = useSearch({
    from: '/pricing/',
    select: (search) => search.currency,
  })
  const navigate = useNavigate({ from: '/pricing/' })
  if (!options) return null

  return {
    value: currency ?? options.value,
    items: [
      { value: 'site', label: options.siteLabel },
      { value: 'USD', label: USD_PRICING_CURRENCY.label },
    ],
    select: (value) => {
      const next = parseDisplayCurrency(value)
      if (!next) return
      void navigate({
        search: (prev) => ({
          ...prev,
          currency: next === options.value ? undefined : next,
        }),
        replace: true,
      })
    },
  }
}
