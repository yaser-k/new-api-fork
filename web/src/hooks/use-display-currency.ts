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
import { useTranslation } from 'react-i18next'

import { getSitePricingCurrency } from '@/features/model-pricing/currency'
import { api } from '@/lib/api'
import { createServerError } from '@/lib/server-error-message'
import { useAuthStore } from '@/stores/auth-store'
import {
  readAccountSetting,
  useDisplayCurrencyStore,
  type DisplayCurrency,
} from '@/stores/display-currency-store'
import {
  offersDisplayCurrency,
  useSystemConfigStore,
} from '@/stores/system-config-store'

/**
 * The currencies a viewer may read figures in, or null on a site that shows
 * US dollars or tokens and so offers no choice.
 */
export function useDisplayCurrencyOptions() {
  const { t } = useTranslation()
  const siteCurrency = useSystemConfigStore(
    (state) => state.config.siteCurrency
  )
  const value = useDisplayCurrencyStore((state) => state.displayCurrency)
  const site = getSitePricingCurrency(siteCurrency)
  if (!site || !offersDisplayCurrency(siteCurrency)) return null

  return {
    value,
    symbol: value === 'USD' ? '$' : site.symbol,
    siteLabel: site.label,
    items: [
      {
        value: 'site' as const,
        label: t('Site currency ({{currency}})', { currency: site.label }),
      },
      { value: 'USD' as const, label: t('US dollar (USD)') },
    ],
  }
}

/**
 * Make `displayCurrency` the viewer's choice in this browser and, for a
 * signed-in viewer, in the account. Rejects when the account refuses it.
 */
export async function saveDisplayCurrency(
  displayCurrency: DisplayCurrency
): Promise<void> {
  useDisplayCurrencyStore.getState().setDisplayCurrency(displayCurrency)
  if (!useAuthStore.getState().auth.user) return

  const response = await api.put('/api/user/self', {
    display_currency: displayCurrency,
  })
  if (!response.data?.success) throw createServerError(response.data)

  const { user, setUser } = useAuthStore.getState().auth
  if (!user) return
  setUser({
    ...user,
    setting: JSON.stringify({
      ...readAccountSetting(user),
      display_currency: displayCurrency,
    }),
  })
}
