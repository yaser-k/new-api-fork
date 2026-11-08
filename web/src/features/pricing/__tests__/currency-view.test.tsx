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
  createRootRouteWithContext,
  createRoute,
  createRouter,
  Outlet,
  RouterProvider,
} from '@tanstack/react-router'
import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { formatQuotaWithCurrency } from '@/lib/currency'
import { Route as ModelDetailsRoute } from '@/routes/pricing/$modelId/index'
import { Route as PricingRoute } from '@/routes/pricing/index'
import { useAuthStore } from '@/stores/auth-store'
import {
  useDisplayCurrencyStore,
  type DisplayCurrency,
} from '@/stores/display-currency-store'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
  type CurrencyDisplayType,
} from '@/stores/system-config-store'

import type { PricingData, PricingModel } from '../types'

vi.mock('@/components/layout', () => ({
  PublicLayout: (props: { children: ReactNode }) => <div>{props.children}</div>,
}))

// The performance charts need a canvas, which jsdom does not have.
vi.mock('../components/model-details-charts', () => ({
  LatencyTrendChart: () => null,
  UptimeTrendChart: () => null,
}))

vi.mock('@/features/performance-metrics/api', () => ({
  getPerfMetricsSummary: async () => ({
    success: true,
    data: { models: [], recent_success_series: {} },
  }),
  getPerfMetrics: () => new Promise(() => undefined),
}))

const model: PricingModel = {
  id: 1,
  model_name: 'example-model',
  quota_type: 0,
  model_ratio: 1,
  completion_ratio: 4,
  enable_groups: ['default'],
}

const pricingData: PricingData = {
  success: true,
  data: [model],
  vendors: [],
  group_ratio: { default: 1 },
  usable_group: { default: { desc: 'Default', ratio: 1 } },
  supported_endpoint: {},
  auto_groups: [],
}

/** A page outside pricing that reads figures without subscribing. */
function Elsewhere() {
  return <p>Balance {formatQuotaWithCurrency(750_000)}</p>
}

const rootRoute = createRootRouteWithContext<{ queryClient: QueryClient }>()({
  component: Outlet,
})
const routeTree = rootRoute.addChildren([
  PricingRoute.update({
    id: '/pricing/',
    path: '/pricing/',
    getParentRoute: () => rootRoute,
  } as never),
  ModelDetailsRoute.update({
    id: '/pricing/$modelId/',
    path: '/pricing/$modelId/',
    getParentRoute: () => rootRoute,
  } as never),
  createRoute({
    getParentRoute: () => rootRoute,
    path: '/elsewhere',
    component: Elsewhere,
  }),
])

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

function preferCurrency(displayCurrency: DisplayCurrency): void {
  useDisplayCurrencyStore.getState().setDisplayCurrency(displayCurrency)
}

function storedPreference(): unknown {
  return JSON.parse(localStorage.getItem('display-currency') ?? '{}').state
}

async function renderAt(url: string) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  queryClient.setQueryData(['status'], {})
  queryClient.setQueryData(['pricing'], pricingData)
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: [url] }),
    context: { queryClient },
  })
  await router.load()
  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  return router
}

function currencySwitch(): HTMLElement {
  return screen.getByRole('group', { name: 'Currency' })
}

function resetStores(): void {
  localStorage.clear()
  useAuthStore.getState().auth.setUser(null)
  useDisplayCurrencyStore.setState(
    useDisplayCurrencyStore.getInitialState(),
    true
  )
  useSystemConfigStore.setState(useSystemConfigStore.getInitialState(), true)
}

beforeEach(() => {
  resetStores()
  showSite('CUSTOM')
})

afterEach(() => {
  vi.restoreAllMocks()
  resetStores()
})

describe('pricing page currency switch', () => {
  test.each([
    { preference: 'site' as const, pressed: 'L', other: 'USD' },
    { preference: 'USD' as const, pressed: 'USD', other: 'L' },
  ])(
    'on a custom-currency site sits next to the token unit toggle with the $preference preference pressed',
    async ({ preference, pressed, other }) => {
      preferCurrency(preference)

      await renderAt('/pricing')

      const group = await screen.findByRole('group', { name: 'Currency' })
      expect(group.previousElementSibling).toHaveAccessibleName(
        'Price display mode'
      )
      expect(group.nextElementSibling).toHaveAccessibleName('Token unit')
      expect(screen.getByRole('button', { name: pressed, pressed: true })).toBe(
        group.querySelector('[aria-pressed="true"]')
      )
      expect(
        screen.getByRole('button', { name: other, pressed: false })
      ).toBeVisible()
    }
  )

  test('a click on USD shows prices in $ at rate 1 and leaves the preference as it was', async () => {
    useAuthStore.getState().auth.setUser({
      id: 1,
      username: 'viewer',
      role: 1,
      setting: '{"display_currency":"site"}',
    })
    preferCurrency('site')
    const put = vi.spyOn(api, 'put')
    await renderAt('/pricing')
    expect(await screen.findByText('L 500')).toBeVisible()

    await userEvent.click(screen.getByRole('button', { name: 'USD' }))

    expect(await screen.findByText('$2')).toBeVisible()
    expect(screen.queryByText('L 500')).not.toBeInTheDocument()
    expect(useDisplayCurrencyStore.getState().displayCurrency).toBe('site')
    expect(storedPreference()).toEqual({ displayCurrency: 'site' })
    expect(put).not.toHaveBeenCalled()
  })

  test.each([
    { link: 'USD', preference: 'site' as const, price: '$2', pressed: 'USD' },
    { link: 'site', preference: 'USD' as const, price: 'L 500', pressed: 'L' },
    { link: 'EUR', preference: 'site' as const, price: 'L 500', pressed: 'L' },
  ])(
    'a link with currency=$link opens with $price for the $preference preference',
    async ({ link, preference, price, pressed }) => {
      preferCurrency(preference)

      await renderAt(`/pricing?currency=${link}`)

      expect(await screen.findByText(price)).toBeVisible()
      expect(screen.getByRole('button', { name: pressed, pressed: true })).toBe(
        currencySwitch().querySelector('[aria-pressed="true"]')
      )
      expect(useDisplayCurrencyStore.getState().displayCurrency).toBe(
        preference
      )
    }
  )

  test('a click writes currency into the address without a new entry and removes it on the preference', async () => {
    preferCurrency('site')
    const router = await renderAt('/pricing?tokenUnit=K')
    const entries = router.history.length

    await userEvent.click(await screen.findByRole('button', { name: 'USD' }))

    expect(router.state.location.search).toEqual({
      tokenUnit: 'K',
      currency: 'USD',
    })
    expect(router.history.length).toBe(entries)

    await userEvent.click(screen.getByRole('button', { name: 'L' }))

    expect(router.state.location.search).toEqual({ tokenUnit: 'K' })
    expect(router.history.length).toBe(entries)
  })

  test('leaving the pricing page gives every other reader the preference again', async () => {
    preferCurrency('site')
    const router = await renderAt('/pricing?currency=USD')
    expect(await screen.findByText('$2')).toBeVisible()

    act(() => router.history.push('/elsewhere'))

    expect(await screen.findByText('Balance L 375')).toBeVisible()
    expect(formatQuotaWithCurrency(750_000)).toBe('L 375')
  })

  test('the model details page shows US dollars from its link and keeps the link on the way back', async () => {
    preferCurrency('site')
    const router = await renderAt('/pricing/example-model?currency=USD')

    expect((await screen.findAllByText('$2')).length).toBeGreaterThan(0)
    expect(screen.queryByText('L 500')).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Back' }))

    expect(router.state.location.pathname).toBe('/pricing')
    expect(router.state.location.search).toEqual({ currency: 'USD' })
    expect(await screen.findByText('$2')).toBeVisible()
  })

  test.each([
    { type: 'USD' as const, balance: '$1.5' },
    { type: 'TOKENS' as const, balance: '750k' },
  ])(
    'on a $type site shows no switch and ignores the currency link',
    async ({ type, balance }) => {
      showSite(type)

      await renderAt('/pricing?currency=USD')

      expect(await screen.findByText('$2')).toBeVisible()
      expect(
        screen.queryByRole('group', { name: 'Currency' })
      ).not.toBeInTheDocument()
      expect(formatQuotaWithCurrency(750_000)).toBe(balance)
    }
  )
})
