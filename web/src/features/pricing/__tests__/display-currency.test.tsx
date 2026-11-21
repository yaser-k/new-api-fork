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
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { useDisplayCurrencyStore } from '@/stores/display-currency-store'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
} from '@/stores/system-config-store'

import { DynamicPricingBreakdown } from '../components/dynamic-pricing-breakdown'
import { ModelCard } from '../components/model-card'
import type { PricingModel } from '../types'

const EXPRESSION = 'tier("standard", p * 2 + c * 8)'

const model: PricingModel = {
  id: 1,
  model_name: 'example-model',
  quota_type: 0,
  model_ratio: 1,
  completion_ratio: 4,
  enable_groups: ['default'],
  group_ratio: { default: 1 },
  billing_mode: 'tiered_expr',
  billing_expr: EXPRESSION,
}

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

describe('prices on a custom-currency site', () => {
  test('a model price per 1M tokens reads in the site currency by default', () => {
    render(<ModelCard model={model} onClick={vi.fn()} tokenUnit='M' />)

    expect(screen.getByText('L 500')).toBeVisible()
    expect(screen.queryByText('$2')).not.toBeInTheDocument()
  })

  test('a model price per 1M tokens reads in $ at rate 1 when the viewer chose US dollars', () => {
    useDisplayCurrencyStore.getState().setDisplayCurrency('USD')

    render(<ModelCard model={model} onClick={vi.fn()} tokenUnit='M' />)

    expect(screen.getByText('$2')).toBeVisible()
    expect(screen.queryByText('L 500')).not.toBeInTheDocument()
  })

  test('a dynamic-pricing breakdown reads in the site currency by default', () => {
    render(<DynamicPricingBreakdown billingExpr={EXPRESSION} />)

    expect(screen.getAllByText('L500.0000').length).toBeGreaterThan(0)
    expect(screen.queryByText('$2.0000')).not.toBeInTheDocument()
  })

  test('a dynamic-pricing breakdown reads in $ at rate 1 when the viewer chose US dollars', () => {
    useDisplayCurrencyStore.getState().setDisplayCurrency('USD')

    render(<DynamicPricingBreakdown billingExpr={EXPRESSION} />)

    expect(screen.getAllByText('$2.0000').length).toBeGreaterThan(0)
    expect(screen.getAllByText('$8.0000').length).toBeGreaterThan(0)
    expect(screen.queryByText('L500.0000')).not.toBeInTheDocument()
  })
})
