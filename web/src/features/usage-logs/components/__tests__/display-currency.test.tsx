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
import { render, screen } from '@testing-library/react'
import { afterEach, beforeEach, expect, test } from 'vitest'

import { useDisplayCurrencyStore } from '@/stores/display-currency-store'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
} from '@/stores/system-config-store'

import { LogCostDisplay } from '../log-cost-display'

function resetStores(): void {
  localStorage.clear()
  useDisplayCurrencyStore.setState(
    useDisplayCurrencyStore.getInitialState(),
    true
  )
  useSystemConfigStore.setState(useSystemConfigStore.getInitialState(), true)
}

beforeEach(() => {
  resetStores()
  useSystemConfigStore.getState().setConfig({
    currency: {
      ...DEFAULT_CURRENCY_CONFIG,
      quotaDisplayType: 'CUSTOM',
      customCurrencySymbol: 'L',
      customCurrencyExchangeRate: 250,
    },
  })
})
afterEach(resetStores)

test('a usage-log cost reads in the site currency by default', () => {
  render(<LogCostDisplay quota={5000} other={null} />)

  expect(screen.getByText('L 2.5')).toBeVisible()
})

test('a usage-log cost reads in $ at rate 1 when the viewer chose US dollars', () => {
  useDisplayCurrencyStore.getState().setDisplayCurrency('USD')

  render(<LogCostDisplay quota={5000} other={null} />)

  expect(screen.getByText('$0.01')).toBeVisible()
  expect(screen.queryByText('L 2.5')).not.toBeInTheDocument()
})
