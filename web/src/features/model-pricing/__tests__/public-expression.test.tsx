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
import { QueryClientProvider, type QueryClient } from '@tanstack/react-query'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { AxiosAdapter } from 'axios'
import { afterEach, expect, it } from 'vitest'

import { ModelPricingEditorPanel } from '@/features/system-settings/models/model-pricing-sheet'
import { api } from '@/lib/http-client'
import { createAppQueryClient } from '@/lib/query-client'
import { useAuthStore } from '@/stores/auth-store'
import { usePricingPreferencesStore } from '@/stores/pricing-preferences-store'

import type { ModelPricingConfig } from '../api'
import { ModelPricingPanel } from '../model-pricing-panel'
import { pricingOptions, pricingRow } from '../pricing'

const originalAdapter = api.defaults.adapter
let client: QueryClient | undefined

const values = {
  'billing_setting.billing_mode': 'tiered_expr',
  'billing_setting.billing_expr':
    'param("route") == "b" ? tier("route_b", p * 2 + c * 4) : tier("base", p * 1 + c * 2)',
  'billing_setting.public_expr': 'tier("public", p * 2 + c * 4)',
}

afterEach(() => {
  cleanup()
  client?.clear()
  api.defaults.adapter = originalAdapter
  useAuthStore.getState().auth.setUser(null)
  localStorage.clear()
})

function serve(saved: unknown[]): AxiosAdapter {
  const snapshot: ModelPricingConfig = {
    entries: [
      {
        model_name: 'example',
        version: 'v1',
        configured: values,
        effective: values,
      },
    ],
    options: pricingOptions({}),
    empty_version: 'empty',
  }
  return async (config) => {
    const respond = (data: unknown) => ({
      data,
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
    })
    if (config.method === 'get' && config.url === '/api/status') {
      return respond({ success: true, data: {} })
    }
    if (config.method === 'get' && config.url === '/api/pricing') {
      return respond({ success: true, data: [], vendors: [] })
    }
    if (config.method === 'get' && config.url === '/api/option/model_pricing') {
      return respond({ success: true, data: snapshot })
    }
    if (
      config.method === 'patch' &&
      config.url === '/api/option/model_pricing'
    ) {
      saved.push(JSON.parse(String(config.data)))
      return respond({ success: true, data: { updated_models: ['example'] } })
    }
    throw new Error(`Unexpected request: ${config.method} ${config.url}`)
  }
}

it('shows the saved public price expression and saves an edited one with the model price', async () => {
  useAuthStore
    .getState()
    .auth.setUser({ id: 1, username: 'administrator', role: 100 })
  usePricingPreferencesStore.setState({ currency: 'USD' })
  const saved: unknown[] = []
  api.defaults.adapter = serve(saved)
  client = createAppQueryClient()
  render(
    <QueryClientProvider client={client}>
      <ModelPricingPanel modelName='example' />
    </QueryClientProvider>
  )
  const user = userEvent.setup()

  const field = await screen.findByRole('textbox', {
    name: 'Public price expression',
  })
  expect(field).toHaveValue('tier("public", p * 2 + c * 4)')
  await user.clear(field)
  await user.type(field, 'tier("public", p * 3 + c * 6)')
  await user.click(screen.getByRole('button', { name: 'Save model prices' }))

  await waitFor(() => expect(saved).toHaveLength(1))
  expect(saved[0]).toMatchObject({
    changes: [
      {
        model_name: 'example',
        expected_version: 'v1',
        pricing: {
          'billing_setting.billing_mode': 'tiered_expr',
          'billing_setting.public_expr': 'tier("public", p * 3 + c * 6)',
        },
      },
    ],
  })
})

it('does not offer a public price expression where the editor commits to the whole price table', async () => {
  useAuthStore
    .getState()
    .auth.setUser({ id: 1, username: 'administrator', role: 100 })
  usePricingPreferencesStore.setState({ currency: 'USD' })
  api.defaults.adapter = serve([])
  client = createAppQueryClient()
  render(
    <QueryClientProvider client={client}>
      <ModelPricingEditorPanel editData={pricingRow('example', values)} />
    </QueryClientProvider>
  )

  expect(
    await screen.findByRole('tab', { name: 'Expression' })
  ).toBeInTheDocument()
  expect(
    screen.queryByRole('textbox', { name: 'Public price expression' })
  ).not.toBeInTheDocument()
})
