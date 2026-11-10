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
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
  type CurrencyConfig,
} from '@/stores/system-config-store'

import { CachedPriceCell } from '../components/cached-price-cell'
import { ModelCard } from '../components/model-card'
import { ModelPriceCell } from '../components/model-price-cell'
import type { PricingModel } from '../types'

const TOMAN = '⁨تومان⁩'

// $3 input, $12 output and $0.06 cached per 1M tokens.
function pricingModel(overrides: Partial<PricingModel> = {}): PricingModel {
  return {
    id: 1,
    model_name: 'example-model',
    quota_type: 0,
    model_ratio: 1.5,
    completion_ratio: 4,
    cache_ratio: 0.02,
    enable_groups: ['default'],
    group_ratio: { default: 1 },
    ...overrides,
  }
}

function useCurrency(currency: Partial<CurrencyConfig>) {
  useSystemConfigStore.getState().setConfig({
    currency: { ...DEFAULT_CURRENCY_CONFIG, ...currency },
  })
}

const useToman = () =>
  useCurrency({
    quotaDisplayType: 'CUSTOM',
    customCurrencySymbol: 'تومان',
    customCurrencyExchangeRate: 672050,
  })

beforeEach(() => {
  // The pricing formatters use the runtime locale; pin it so digit grouping
  // does not depend on the machine running the tests.
  const NativeNumberFormat = Intl.NumberFormat
  vi.spyOn(Intl, 'NumberFormat').mockImplementation(function (
    locales?: Intl.LocalesArgument,
    options?: Intl.NumberFormatOptions
  ) {
    return new NativeNumberFormat(locales ?? 'en-US', options)
  } as typeof Intl.NumberFormat)
})

afterEach(() => {
  useSystemConfigStore
    .getState()
    .setConfig({ currency: { ...DEFAULT_CURRENCY_CONFIG } })
})

describe('model card prices', () => {
  it('keeps a long right-to-left price whole and puts the unit after it', () => {
    useToman()
    render(<ModelCard model={pricingModel()} onClick={vi.fn()} />)

    const amount = screen.getByText(`${TOMAN} 2,016,150`)
    expect(amount).toHaveClass('whitespace-nowrap')
    expect(amount.parentElement).toHaveTextContent(`${TOMAN} 2,016,150 / 1M`)
    expect(screen.getByText(`${TOMAN} 8,064,600`)).toBeVisible()
    expect(screen.getByText(`${TOMAN} 40,323`)).toBeVisible()
  })

  it('puts each long price on its own line with the label beside it', () => {
    useToman()
    render(<ModelCard model={pricingModel()} onClick={vi.fn()} />)

    const input = screen.getByText('Input').parentElement
    expect(input).not.toHaveClass('flex-col')
    expect(input?.parentElement).toHaveClass('grid-cols-1')
  })

  it('keeps short prices side by side with the label above each', () => {
    useCurrency({ quotaDisplayType: 'USD' })
    render(<ModelCard model={pricingModel()} onClick={vi.fn()} />)

    const input = screen.getByText('Input').parentElement
    expect(input).toHaveClass('flex-col')
    expect(input?.parentElement).not.toHaveClass('grid-cols-1')
    expect(input).toHaveTextContent('$3 / 1M')
  })
})

describe('pricing table prices', () => {
  it('does not break a long price and names the currency before the unit', () => {
    useToman()
    render(<ModelPriceCell model={pricingModel()} />)

    // The number has no spaces; overflow-wrap: normal keeps it whole while
    // the text around it may still wrap in narrow layouts.
    const input = screen.getByText('2,016,150')
    expect(input).toHaveClass('wrap-normal')
    expect(input).not.toHaveClass('break-words')
    expect(screen.getByText(`${TOMAN} / 1M tokens`)).toBeVisible()
  })

  it('groups the digits of a cached price like the card does', () => {
    useCurrency({ quotaDisplayType: 'USD' })
    const model = pricingModel({ model_ratio: 20161.5, cache_ratio: 1 })
    render(<CachedPriceCell model={model} options={{ tokenUnit: 'M' }} />)

    expect(screen.getByText('$40,323')).toBeVisible()
  })
})
