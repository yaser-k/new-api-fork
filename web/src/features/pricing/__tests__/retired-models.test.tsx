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
import { describe, expect, it } from 'vitest'

import {
  RetiredModelNotice,
  RetiredModelsSection,
} from '../components/retired-models'
import type { RetiredModel } from '../types'

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

function renderAtPricing(content: React.ReactNode) {
  const root = createRootRoute({ component: Outlet })
  const pricing = createRoute({
    getParentRoute: () => root,
    path: '/pricing',
    component: () => <div>{content}</div>,
  })
  const details = createRoute({
    getParentRoute: () => root,
    path: '/pricing/$modelId',
    component: () => <p>Model page</p>,
  })
  const router = createRouter({
    routeTree: root.addChildren([pricing, details]),
    history: createMemoryHistory({ initialEntries: ['/pricing'] }),
  })
  return render(<RouterProvider router={router} />)
}

describe('retired models section', () => {
  it('renders nothing when no model is retired', async () => {
    renderAtPricing(
      <>
        <p>Live models</p>
        <RetiredModelsSection models={[]} />
      </>
    )
    await screen.findByText('Live models')
    expect(
      screen.queryByRole('region', { name: 'Retired models' })
    ).not.toBeInTheDocument()
  })

  it('hides retired models until the visitor opens them, then lists each as retired with its successor', async () => {
    const user = userEvent.setup()
    renderAtPricing(<RetiredModelsSection models={retiredModels} />)

    const section = await screen.findByRole('region', {
      name: 'Retired models',
    })
    const trigger = within(section).getByRole('button', {
      name: 'Show retired models (2)',
    })
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
    expect(
      within(section).queryByRole('link', { name: 'example-model-v1' })
    ).not.toBeInTheDocument()

    await user.click(trigger)

    expect(
      within(section).getByRole('button', { name: 'Hide retired models' })
    ).toHaveAttribute('aria-expanded', 'true')
    const items = within(section).getAllByRole('listitem')
    expect(items).toHaveLength(2)
    const [newest] = items
    expect(
      within(newest).getByRole('link', { name: 'example-model-v1' })
    ).toHaveAttribute('href', '/pricing/example-model-v1')
    expect(within(newest).getByText('Retired')).toBeVisible()
    expect(within(newest).getByText('Retired on 2026-09-30')).toBeVisible()
    expect(
      within(newest).getByRole('link', { name: 'example-model-v2' })
    ).toHaveAttribute('href', '/pricing/example-model-v2')
    const notice = within(newest).getByRole('link', {
      name: 'Retirement notice',
    })
    expect(notice).toHaveAttribute(
      'href',
      'https://example.com/retirements/example-model-v1'
    )
    expect(notice).toHaveAttribute('target', '_blank')
    expect(within(newest).getByText('Replaced by v2')).toBeVisible()
    expect(
      within(items[1]).queryByRole('link', { name: 'Retirement notice' })
    ).not.toBeInTheDocument()
  })
})

describe('retired model page', () => {
  it('shows the retirement, the successor link and the notice instead of prices', async () => {
    renderAtPricing(<RetiredModelNotice model={retiredModels[0]} />)

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
