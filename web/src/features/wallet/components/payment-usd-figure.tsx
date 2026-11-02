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
import { formatCurrencyFromUSD } from '@/lib/currency'
import { useDisplayCurrencyStore } from '@/stores/display-currency-store'
import {
  offersDisplayCurrency,
  useSystemConfigStore,
} from '@/stores/system-config-store'

/**
 * The US dollar figure of a payment in the site's currency, shown beside it to
 * a viewer who chose to read figures in US dollars. `priceRatio` is what the
 * site charges for one US dollar of balance.
 */
export function PaymentUsdFigure(props: {
  amount: number
  priceRatio: number
}) {
  const displayCurrency = useDisplayCurrencyStore(
    (state) => state.displayCurrency
  )
  const siteCurrency = useSystemConfigStore(
    (state) => state.config.siteCurrency
  )
  if (displayCurrency !== 'USD' || !offersDisplayCurrency(siteCurrency)) {
    return null
  }
  if (!(props.priceRatio > 0) || !Number.isFinite(props.amount)) return null

  return (
    <span className='text-muted-foreground ms-1.5 text-xs font-normal'>
      ≈ {formatCurrencyFromUSD(props.amount / props.priceRatio)}
    </span>
  )
}
