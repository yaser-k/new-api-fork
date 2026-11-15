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
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  RouterProvider,
} from '@tanstack/react-router'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { RetiredModelNotice } from '../components/retired-models'
import { SORT_OPTIONS } from '../constants'
import { Pricing } from '../index'
import { sortModels, toRetiredEntries } from '../lib/filters'
import type { PricingModel, RetiredModel } from '../types'

const pricing = vi.hoisted(() => ({
  models: [] as PricingModel[],
  retiredModels: [] as RetiredModel[],
}))

vi.mock('../hooks/use-pricing-data', () => ({
  usePricingData: () => ({
    models: pricing.models,
    vendors: [],
    groupRatio: { default: 1 },
    usableGroup: { default: { desc: 'Default', ratio: 1 } },
    endpointMap: {},
    autoGroups: [],
    retiredModels: pricing.retiredModels,
    isLoading: false,
    priceRate: 1,
    usdExchangeRate: 1,
  }),
}))

vi.mock('@/features/performance-metrics/api', () => ({
  getPerfMetricsSummary: async () => ({
    success: true,
    data: { models: [], recent_success_series: {} },
  }),
}))

// The details drawer loads the chart library, which jsdom cannot run
vi.mock('../components/model-details', () => ({
  ModelDetails: () => null,
  ModelDetailsContent: () => null,
  ModelDetailsDrawer: () => null,
}))

vi.mock('@/components/layout', () => ({
  PublicLayout: (props: { children: React.ReactNode }) => (
    <div>{props.children}</div>
  ),
}))

const liveModels: PricingModel[] = [
  {
    id: 1,
    model_name: 'example-model-v2',
    quota_type: 0,
    model_ratio: 1,
    completion_ratio: 2,
    enable_groups: ['default'],
  },
  {
    id: 2,
    model_name: 'other-model',
    quota_type: 1,
    model_ratio: 0,
    completion_ratio: 0,
    model_price: 0.01,
    enable_groups: ['default'],
  },
]

const retiredModels: RetiredModel[] = [
  {
    model_name: 'example-model-v1',
    retired_at: '2026-09-30',
    successor: 'example-model-v2',
    notice_url: 'https://example.com/retirements/example-model-v1',
    note: 'Replaced by v2',
  },
  {
    model_name: 'example-model-v0',
    retired_at: '2026-03-01',
    successor: 'example-model-v2',
  },
]

let queryClient: QueryClient
beforeEach(() => {
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  pricing.models = liveModels
  pricing.retiredModels = retiredModels
})
afterEach(() => {
  queryClient.clear()
})

function renderAt(url: string, content?: React.ReactNode) {
  const root = createRootRoute({ component: Outlet })
  const pricingRoute = createRoute({
    getParentRoute: () => root,
    path: 'pricing',
    component: Outlet,
  })
  const list = createRoute({
    getParentRoute: () => pricingRoute,
    path: '/',
    component: () => content ?? <Pricing />,
  })
  const details = createRoute({
    getParentRoute: () => pricingRoute,
    path: '$modelId',
    component: () => <p>Model page</p>,
  })
  const router = createRouter({
    routeTree: root.addChildren([pricingRoute.addChildren([list, details])]),
    history: createMemoryHistory({ initialEntries: [url] }),
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}

function tagChip(name: RegExp) {
  return screen.getByRole('button', { name })
}

function retiredCard(modelName: string) {
  const link = screen.getByRole('link', { name: modelName })
  const card = link.closest('[data-slot="card"]')
  if (!(card instanceof HTMLElement)) throw new Error('no card')
  return card
}

describe('pricing page model tags', () => {
  it('opens on the active models when some are retired, without the retired models section', async () => {
    renderAt('/pricing')

    expect(await screen.findByText('example-model-v2')).toBeVisible()
    expect(tagChip(/^Active\s*2$/)).toHaveAttribute('aria-pressed', 'true')
    expect(tagChip(/^Retired\s*2$/)).toHaveAttribute('aria-pressed', 'false')
    expect(tagChip(/^All Tags\s*4$/)).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByText('other-model')).toBeVisible()
    expect(
      screen.queryByRole('link', { name: 'example-model-v1' })
    ).not.toBeInTheDocument()
    expect(screen.queryByText('Retired models')).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: /retired models/i })
    ).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Reset' })).toBeDisabled()
  })

  it('shows only the retired models under Retired, each with its date and successor', async () => {
    const user = userEvent.setup()
    renderAt('/pricing')
    await user.click(await screen.findByRole('button', { name: /^Retired/ }))

    expect(tagChip(/^Retired\s*2$/)).toHaveAttribute('aria-pressed', 'true')
    expect(screen.queryByText('other-model')).not.toBeInTheDocument()
    const newest = retiredCard('example-model-v1')
    expect(
      within(newest).getByRole('link', { name: 'example-model-v1' })
    ).toHaveAttribute('href', '/pricing/example-model-v1')
    expect(within(newest).getByText('Retired')).toBeVisible()
    expect(within(newest).getByText('Retired on 2026-09-30')).toBeVisible()
    expect(
      within(newest).getByRole('link', { name: 'example-model-v2' })
    ).toHaveAttribute('href', '/pricing/example-model-v2')
    expect(
      within(newest).getByRole('link', { name: 'Retirement notice' })
    ).toHaveAttribute(
      'href',
      'https://example.com/retirements/example-model-v1'
    )
    expect(within(newest).getByText('Replaced by v2')).toBeVisible()
    expect(
      within(newest).getByRole('link', { name: 'Details' })
    ).toHaveAttribute('href', '/pricing/example-model-v1')
    expect(
      within(newest).queryByRole('button', { name: 'Copy model name' })
    ).not.toBeInTheDocument()
    expect(within(newest).queryByText('Input')).not.toBeInTheDocument()

    const oldest = retiredCard('example-model-v0')
    expect(within(oldest).getByText('Retired on 2026-03-01')).toBeVisible()
    expect(
      within(oldest).queryByRole('link', { name: 'Retirement notice' })
    ).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Reset' })).toBeEnabled()
  })

  it('shows the active and the retired models under All Tags', async () => {
    const user = userEvent.setup()
    renderAt('/pricing')
    await user.click(await screen.findByRole('button', { name: /^All Tags/ }))

    expect(tagChip(/^All Tags\s*4$/)).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByText('other-model')).toBeVisible()
    expect(screen.getByRole('link', { name: 'example-model-v1' })).toBeVisible()
    expect(screen.getByRole('link', { name: 'example-model-v0' })).toBeVisible()
  })

  it('counts only the callable models on the other filters when some are retired', async () => {
    renderAt('/pricing')

    expect(await screen.findByText('example-model-v2')).toBeVisible()
    expect(tagChip(/^All Vendors\s*2$/)).toBeVisible()
    expect(tagChip(/^All Models\s*2$/)).toBeVisible()
    expect(tagChip(/^All Types\s*2$/)).toBeVisible()
    expect(tagChip(/^All Tags\s*4$/)).toBeVisible()
  })

  it('opens on the tag in the address', async () => {
    renderAt('/pricing?tag=status:retired')

    expect(
      await screen.findByRole('link', { name: 'example-model-v1' })
    ).toBeVisible()
    expect(tagChip(/^Retired\s*2$/)).toHaveAttribute('aria-pressed', 'true')
    expect(screen.queryByText('other-model')).not.toBeInTheDocument()
  })

  it('lists a retired model in the table with its badge and successor and no price', async () => {
    renderAt('/pricing?view=table&tag=status:retired')

    const name = await screen.findByRole('link', { name: 'example-model-v1' })
    const row = name.closest('tr')
    if (!row) throw new Error('no row')
    expect(name).toHaveAttribute('href', '/pricing/example-model-v1')
    expect(within(row).getByText('Retired')).toBeVisible()
    expect(within(row).getByText('Retired on 2026-09-30')).toBeVisible()
    expect(
      within(row).getByRole('link', { name: 'example-model-v2' })
    ).toHaveAttribute('href', '/pricing/example-model-v2')
    expect(row).not.toHaveTextContent('$')
  })

  it('keeps upstream behaviour when no model is retired', async () => {
    pricing.retiredModels = []
    renderAt('/pricing')

    expect(await screen.findByText('example-model-v2')).toBeVisible()
    expect(screen.getByText('other-model')).toBeVisible()
    expect(tagChip(/^All Tags\s*2$/)).toHaveAttribute('aria-pressed', 'true')
    expect(
      screen.queryByRole('button', { name: /^Active/ })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: /^Retired/ })
    ).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Reset' })).toBeDisabled()
  })
})

describe('retired model sorting', () => {
  it('puts retired models last when sorting by price in either direction', () => {
    const models = [...toRetiredEntries(retiredModels), ...liveModels]
    for (const sort of [
      SORT_OPTIONS.INPUT_PRICE_LOW,
      SORT_OPTIONS.INPUT_PRICE_HIGH,
      SORT_OPTIONS.REQUEST_PRICE_LOW,
      SORT_OPTIONS.REQUEST_PRICE_HIGH,
    ]) {
      const sorted = sortModels(models, sort)
      expect(
        sorted.slice(-2).map((model) => model.model_name),
        sort
      ).toEqual(['example-model-v1', 'example-model-v0'])
    }
  })
})

describe('retired model page', () => {
  it('shows the retirement, the successor link and the notice instead of prices', async () => {
    renderAt('/pricing', <RetiredModelNotice model={retiredModels[0]} />)

    expect(
      await screen.findByRole('heading', { name: 'example-model-v1' })
    ).toBeVisible()
    expect(screen.getByText('Retired')).toBeVisible()
    expect(
      screen.getByText(
        'This model was retired on 2026-09-30 and can no longer be called.'
      )
    ).toBeVisible()
    expect(
      screen.getByRole('link', { name: 'example-model-v2' })
    ).toHaveAttribute('href', '/pricing/example-model-v2')
    expect(
      screen.getByRole('link', { name: 'Retirement notice' })
    ).toHaveAttribute(
      'href',
      'https://example.com/retirements/example-model-v1'
    )
    expect(screen.queryByText('Base Price')).not.toBeInTheDocument()
  })
})
