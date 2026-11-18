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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { SignUpForm } from '../components/sign-up-form'

const clients: QueryClient[] = []

function mockApi(status: Record<string, unknown>) {
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/status') return { data: { success: true, data: status } }
    return { data: { success: true, data: null } }
  })
  return vi.spyOn(api, 'post').mockResolvedValue({
    data: { success: true, message: '' },
  })
}

async function renderSignUpForm() {
  const root = createRootRoute({ component: Outlet })
  const signUp = createRoute({
    getParentRoute: () => root,
    path: '/sign-up',
    component: () => <SignUpForm />,
  })
  const signIn = createRoute({
    getParentRoute: () => root,
    path: '/sign-in',
    component: () => <div>Sign-in page</div>,
  })
  const router = createRouter({
    routeTree: root.addChildren([signUp, signIn]),
    history: createMemoryHistory({ initialEntries: ['/sign-up'] }),
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  clients.push(client)
  await router.load()
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}

async function fillAccount(user: ReturnType<typeof userEvent.setup>) {
  await user.type(await screen.findByLabelText('Username'), 'phone-user')
  await user.type(screen.getByLabelText('Password'), 'phone-password')
  await user.type(screen.getByLabelText('Confirm password'), 'phone-password')
}

beforeEach(() => {
  vi.spyOn(window, 'scrollTo').mockImplementation(() => {})
  localStorage.clear()
})

afterEach(() => {
  clients.splice(0).forEach((client) => client.clear())
  vi.restoreAllMocks()
  localStorage.clear()
})

it('with the phone number required, the form posts the number as typed, Persian digits included', async () => {
  const user = userEvent.setup()
  const post = mockApi({
    register_phone_required: true,
    phone_country_code: '44',
    phone_trunk_prefix: '0',
  })
  await renderSignUpForm()

  const phone = await screen.findByLabelText('Phone number')
  expect(phone).toHaveAttribute('type', 'tel')
  expect(phone).toHaveAttribute('dir', 'ltr')
  await fillAccount(user)
  await user.type(phone, '۰۷۹۱۱ ۱۲۳۴۵۶')
  await user.click(screen.getByRole('button', { name: 'Create account' }))

  await waitFor(() => expect(post).toHaveBeenCalledTimes(1))
  const [url, payload] = post.mock.calls[0]
  expect(url).toBe('/api/user/register')
  expect(payload).toMatchObject({
    username: 'phone-user',
    phone: '۰۷۹۱۱ ۱۲۳۴۵۶',
  })
})

it('with the phone number required, an empty field stops the sign-up with a field error', async () => {
  const user = userEvent.setup()
  const post = mockApi({ register_phone_required: true })
  await renderSignUpForm()

  await fillAccount(user)
  await user.click(screen.getByRole('button', { name: 'Create account' }))

  const phone = screen.getByLabelText('Phone number')
  await waitFor(() => expect(phone).toHaveAttribute('aria-invalid', 'true'))
  expect(phone).toHaveAccessibleDescription(
    expect.stringContaining('Phone number is required')
  )
  expect(post).not.toHaveBeenCalled()
})

it.each([
  [
    { phone_country_code: '44', phone_trunk_prefix: '0' },
    'Enter the number starting with 0, or with +44',
  ],
  [
    { phone_country_code: '47', phone_trunk_prefix: '' },
    'Enter the national number, or the number with +47',
  ],
  [{}, 'Enter the number with + and the country code'],
])(
  'the phone field hint follows the format options %j',
  async (format, hint) => {
    mockApi({ register_phone_required: true, ...format })
    await renderSignUpForm()

    expect(
      await screen.findByLabelText('Phone number')
    ).toHaveAccessibleDescription(hint)
  }
)

it('without the requirement, the form shows no phone field and posts no number', async () => {
  const user = userEvent.setup()
  const post = mockApi({ register_phone_required: false })
  await renderSignUpForm()

  await fillAccount(user)
  expect(screen.queryByLabelText('Phone number')).toBeNull()
  await user.click(screen.getByRole('button', { name: 'Create account' }))

  await waitFor(() => expect(post).toHaveBeenCalledTimes(1))
  expect(post.mock.calls[0][1]).not.toHaveProperty('phone', expect.anything())
})
