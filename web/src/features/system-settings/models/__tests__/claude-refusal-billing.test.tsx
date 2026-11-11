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
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { ClaudeSettingsCard } from '../claude-settings-card'

const mocks = vi.hoisted(() => ({
  updateSystemOption: vi.fn(),
}))

vi.mock('../../api', () => ({
  updateSystemOption: mocks.updateSystemOption,
}))

const defaultValues = {
  claude: {
    model_headers_settings: '{}',
    default_max_tokens: '{"default":8192}',
    thinking_adapter_enabled: true,
    thinking_adapter_budget_tokens_percentage: 0.8,
    refusal_billing_waiver_enabled: true,
    refusal_billed_categories: '["bio","frontier_llm","reasoning_extraction"]',
  },
}

function renderCard() {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  })
  const utils = render(
    <QueryClientProvider client={queryClient}>
      <ClaudeSettingsCard defaultValues={defaultValues} />
    </QueryClientProvider>
  )
  const categories = utils.container.querySelector<HTMLTextAreaElement>(
    'textarea[name="claude.refusal_billed_categories"]'
  )
  const form = utils.container.querySelector('form')
  expect(categories).not.toBeNull()
  expect(form).not.toBeNull()
  return {
    queryClient,
    categories: categories as HTMLTextAreaElement,
    form: form as HTMLFormElement,
  }
}

describe('Claude refusal billing settings', () => {
  beforeEach(() => {
    mocks.updateSystemOption.mockReset()
    mocks.updateSystemOption.mockResolvedValue({ success: true, message: '' })
  })

  test('stored waiver option renders the switch on', () => {
    const { queryClient } = renderCard()

    const toggle = screen.getByRole('switch', {
      name: 'Waive Refusals Before Output',
    })

    expect(toggle).toHaveAttribute('aria-checked', 'true')
    queryClient.clear()
  })

  test('switching the waiver off and editing categories saves both options', async () => {
    const { queryClient, categories, form } = renderCard()

    fireEvent.click(
      screen.getByRole('switch', { name: 'Waive Refusals Before Output' })
    )
    fireEvent.input(categories, { target: { value: '[ "bio" ]' } })
    fireEvent.submit(form)

    await waitFor(() => {
      expect(mocks.updateSystemOption).toHaveBeenCalledTimes(2)
    })
    expect(mocks.updateSystemOption.mock.calls.map((call) => call[0])).toEqual([
      { key: 'claude.refusal_billing_waiver_enabled', value: false },
      { key: 'claude.refusal_billed_categories', value: '["bio"]' },
    ])
    queryClient.clear()
  })

  test('empty category list blocks saving', async () => {
    const { queryClient, categories, form } = renderCard()

    fireEvent.input(categories, { target: { value: '' } })
    fireEvent.submit(form)

    await waitFor(() => {
      expect(categories).toHaveAttribute('aria-invalid', 'true')
    })
    expect(mocks.updateSystemOption).not.toHaveBeenCalled()
    queryClient.clear()
  })
})
