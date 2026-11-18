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
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { beforeEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { SettingsPageProvider } from '../../components/settings-page-context'
import type { UpdateOptionRequest } from '../../types'
import { BasicAuthSection } from '../basic-auth-section'

const defaults = {
  PasswordLoginEnabled: true,
  PasswordRegisterEnabled: true,
  EmailVerificationEnabled: false,
  RegisterEnabled: true,
  InvitationRegisterEnabled: false,
  InvitationValidHours: 72,
  EmailDomainRestrictionEnabled: false,
  EmailAliasRestrictionEnabled: false,
  EmailDomainWhitelist: '',
  RegisterPhoneRequiredEnabled: false,
  PhoneCountryCode: '',
  PhoneTrunkPrefix: '',
  PhoneNationalNumberPattern: '',
  RegisterPhoneHint: '',
}

function Fixture() {
  const [container, setContainer] = useState<HTMLDivElement | null>(null)
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { mutations: { retry: false } } })
  )

  return (
    <QueryClientProvider client={client}>
      <div ref={setContainer} />
      <SettingsPageProvider actionsContainer={container}>
        <BasicAuthSection defaultValues={defaults} />
      </SettingsPageProvider>
    </QueryClientProvider>
  )
}

async function renderSettings() {
  const route = createRootRoute({ component: Fixture })
  const router = createRouter({
    routeTree: route,
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  await router.load()
  render(<RouterProvider router={router} />)
}

function mockSave() {
  return vi.spyOn(api, 'put').mockResolvedValue({
    data: { success: true, message: '' },
  })
}

beforeEach(() => {
  vi.spyOn(api, 'get').mockResolvedValue({ data: { success: true, data: {} } })
})

it('turning the phone requirement on with a format saves each option, trimmed', async () => {
  const put = mockSave()
  const user = userEvent.setup()
  await renderSettings()

  await user.click(
    screen.getByRole('switch', {
      name: 'Require a phone number at registration',
    })
  )
  await user.type(screen.getByLabelText('Country calling code'), ' 44 ')
  await user.type(screen.getByLabelText('Trunk prefix'), '0')
  await user.type(screen.getByLabelText('National number pattern'), '7\\d{{9}')
  await user.click(screen.getByRole('button', { name: 'Save Changes' }))

  await waitFor(() => expect(put).toHaveBeenCalledTimes(4))
  const saved = Object.fromEntries(
    put.mock.calls.map(([, body]) => {
      const request = body as UpdateOptionRequest
      return [request.key, request.value]
    })
  )
  expect(saved).toEqual({
    RegisterPhoneRequiredEnabled: true,
    PhoneCountryCode: '44',
    PhoneTrunkPrefix: '0',
    PhoneNationalNumberPattern: '7\\d{9}',
  })
})

it('a country calling code that starts with 0 is refused on the field before saving', async () => {
  const put = mockSave()
  const user = userEvent.setup()
  await renderSettings()

  const code = screen.getByLabelText('Country calling code')
  await user.type(code, '044')
  await user.click(screen.getByRole('button', { name: 'Save Changes' }))

  await waitFor(() => expect(code).toHaveAttribute('aria-invalid', 'true'))
  expect(code).toHaveAccessibleDescription(
    expect.stringContaining(
      'The country calling code must be one to three digits that do not start with 0'
    )
  )
  expect(put).not.toHaveBeenCalled()
})

it('the phone format fields are written left to right', async () => {
  await renderSettings()

  for (const label of [
    'Country calling code',
    'Trunk prefix',
    'National number pattern',
  ]) {
    expect(screen.getByLabelText(label)).toHaveAttribute('dir', 'ltr')
  }
})

it('the administrator hint for the sign-up phone field is saved trimmed', async () => {
  const put = mockSave()
  const user = userEvent.setup()
  await renderSettings()

  await user.type(
    screen.getByLabelText('Phone number hint'),
    '  Mobile numbers only.  '
  )
  await user.click(screen.getByRole('button', { name: 'Save Changes' }))

  await waitFor(() => expect(put).toHaveBeenCalledTimes(1))
  expect(put.mock.calls[0][1]).toEqual({
    key: 'RegisterPhoneHint',
    value: 'Mobile numbers only.',
  })
})

it('the administrator hint field takes at most 200 characters', async () => {
  const user = userEvent.setup()
  await renderSettings()

  const hint = screen.getByLabelText('Phone number hint')
  await user.click(hint)
  await user.paste('x'.repeat(201))

  expect(hint).toHaveAttribute('maxLength', '200')
  expect(hint).toHaveValue('x'.repeat(200))
})
