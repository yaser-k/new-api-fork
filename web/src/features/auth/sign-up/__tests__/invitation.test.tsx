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
import { Route as SignUpRoute } from '@/routes/(auth)/sign-up'

import { SignUp } from '..'

const token = 'q3Jv0Xb8T1mN5yZ2kL7pC4sW9dF6gH0aE3rU8iO1xVw'
const clients: QueryClient[] = []

// Email verification is on, so the ordinary form asks for a code.
const status = {
  email_verification: true,
  register_enabled: false,
  invitation_register_enabled: true,
}

function mockApi(checkSucceeds = true) {
  const get = vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/status') return { data: { success: true, data: status } }
    if (url === '/api/invitation/check') {
      return {
        data: checkSucceeds
          ? {
              success: true,
              data: { email: 'invitee@example.com', expired_time: 1 },
            }
          : {
              success: false,
              message: 'This invitation is invalid or has expired',
            },
      }
    }
    return { data: { success: true, data: null } }
  })
  const post = vi.spyOn(api, 'post').mockResolvedValue({
    data: { success: true, message: '' },
  })
  return { get, post }
}

async function renderSignUp(initialEntry: string) {
  const root = createRootRoute({ component: Outlet })
  const auth = createRoute({ getParentRoute: () => root, id: '(auth)' })
  const signUp = createRoute({
    getParentRoute: () => auth,
    path: 'sign-up',
    validateSearch: SignUpRoute.options.validateSearch,
    component: SignUp,
  })
  const signIn = createRoute({
    getParentRoute: () => auth,
    path: 'sign-in',
    component: () => <div>Sign-in page</div>,
  })
  const router = createRouter({
    routeTree: root.addChildren([auth.addChildren([signUp, signIn])]),
    history: createMemoryHistory({ initialEntries: [initialEntry] }),
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
  return router
}

async function fillAccount(user: ReturnType<typeof userEvent.setup>) {
  await user.type(await screen.findByLabelText('Username'), 'invited-user')
  await user.type(screen.getByLabelText('Password'), 'invited-password')
  await user.type(screen.getByLabelText('Confirm password'), 'invited-password')
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

it('with an invitation token, the form shows the invited email locked, asks no code and posts the token', async () => {
  const user = userEvent.setup()
  const { get, post } = mockApi()
  await renderSignUp(`/sign-up?invite=${token}`)

  const email = await screen.findByLabelText('Email')
  expect(email).toHaveValue('invitee@example.com')
  expect(email).toHaveAttribute('readonly')
  expect(get).toHaveBeenCalledWith('/api/invitation/check', {
    params: { token },
  })
  expect(screen.queryByPlaceholderText('Verification code')).toBeNull()
  expect(screen.queryByRole('button', { name: 'Send code' })).toBeNull()

  await fillAccount(user)
  await user.click(screen.getByRole('button', { name: 'Create account' }))

  await waitFor(() => expect(post).toHaveBeenCalledTimes(1))
  const [url, payload] = post.mock.calls[0]
  expect(url).toBe('/api/user/register')
  expect(payload).toMatchObject({
    username: 'invited-user',
    password: 'invited-password',
    invite_token: token,
  })
  expect(payload).not.toHaveProperty('verification_code', expect.anything())
})

it('without an invitation token, the sign-up form asks for the email code as before', async () => {
  const user = userEvent.setup()
  const { get, post } = mockApi()
  await renderSignUp('/sign-up')

  expect(
    await screen.findByLabelText('Email (required for verification)')
  ).not.toHaveAttribute('readonly')
  expect(screen.getByPlaceholderText('Verification code')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Send code' })).toBeInTheDocument()

  await fillAccount(user)
  await user.type(
    screen.getByLabelText('Email (required for verification)'),
    'person@example.com'
  )
  await user.type(screen.getByPlaceholderText('Verification code'), '123456')
  await user.click(screen.getByRole('button', { name: 'Create account' }))

  await waitFor(() => expect(post).toHaveBeenCalledTimes(1))
  expect(post.mock.calls[0][1]).toMatchObject({
    email: 'person@example.com',
    verification_code: '123456',
    invite_token: undefined,
  })
  expect(get).not.toHaveBeenCalledWith(
    '/api/invitation/check',
    expect.anything()
  )
})

it('an invalid invitation token shows one message and a way to sign in, without the form', async () => {
  const user = userEvent.setup()
  mockApi(false)
  const router = await renderSignUp(`/sign-up?invite=${token}`)

  expect(
    await screen.findByText('This invitation is invalid or has expired')
  ).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Create account' })).toBeNull()
  expect(screen.queryByLabelText('Username')).toBeNull()

  await user.click(screen.getByRole('button', { name: 'Sign in' }))
  await waitFor(() => expect(router.state.location.pathname).toBe('/sign-in'))
})
