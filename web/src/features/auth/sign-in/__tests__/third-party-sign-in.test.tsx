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
import { toast } from 'sonner'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { Route as WeChatCallbackRoute } from '@/routes/(auth)/oauth'
import { Route as OAuthCallbackRoute } from '@/routes/oauth/$provider'

import { UserAuthForm } from '../components/user-auth-form'

// A first third-party sign-in that would create an account while a phone
// number is required at registration is refused with this answer.
function phoneRefusal(provider: string) {
  return {
    success: false,
    message: `Registration requires a phone number. Register with a phone number first, then link ${provider} in account settings`,
    message_key:
      'Registration requires a phone number. Register with a phone number first, then link {{provider}} in account settings',
    message_params: { provider },
  }
}

const clients: QueryClient[] = []

function mockApi(status: Record<string, unknown>, oauth?: unknown) {
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/status') return { data: { success: true, data: status } }
    if (url.startsWith('/api/oauth/')) return { data: oauth }
    return { data: { success: true, data: null } }
  })
}

async function renderAt(path: string) {
  const root = createRootRoute({ component: Outlet })
  const authGroup = createRoute({
    getParentRoute: () => root,
    id: '(auth)',
    component: Outlet,
  })
  const routes = [
    createRoute({
      getParentRoute: () => root,
      path: '/sign-in',
      component: () => (
        <>
          <div>Sign-in page</div>
          <UserAuthForm />
        </>
      ),
    }),
    createRoute({
      getParentRoute: () => root,
      path: '/oauth/$provider',
      component: OAuthCallbackRoute.options.component,
    }),
    authGroup.addChildren([
      createRoute({
        getParentRoute: () => authGroup,
        path: '/oauth',
        component: WeChatCallbackRoute.options.component,
      }),
    ]),
  ]
  const router = createRouter({
    routeTree: root.addChildren(routes),
    history: createMemoryHistory({ initialEntries: [path] }),
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

beforeEach(() => {
  vi.spyOn(window, 'scrollTo').mockImplementation(() => {})
  localStorage.clear()
  sessionStorage.clear()
})

afterEach(() => {
  clients.splice(0).forEach((client) => client.clear())
  vi.restoreAllMocks()
  localStorage.clear()
  sessionStorage.clear()
})

it.each([true, false])(
  'with the phone number required set to %s, the sign-in page keeps the third-party buttons',
  async (required) => {
    mockApi({
      register_phone_required: required,
      github_oauth: true,
      wechat_login: true,
    })
    await renderAt('/sign-in')

    expect(
      await screen.findByRole('button', { name: /Continue with GitHub/ })
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: /Continue with WeChat/ })
    ).toBeInTheDocument()
  }
)

it('a refused first OAuth sign-in shows the refusal and returns to the sign-in page', async () => {
  const error = vi.spyOn(toast, 'error')
  mockApi({ register_phone_required: true }, phoneRefusal('GitHub'))
  await renderAt('/oauth/github?code=oauth-code&state=oauth-state')

  expect(await screen.findByText('Sign-in page')).toBeInTheDocument()
  expect(error).toHaveBeenCalledTimes(1)
  expect(error).toHaveBeenCalledWith(
    'Registration requires a phone number. Register with a phone number first, then link GitHub in account settings'
  )
})

it('a refused first WeChat sign-in from the code dialog shows the refusal', async () => {
  const user = userEvent.setup()
  const error = vi.spyOn(toast, 'error')
  mockApi(
    { register_phone_required: true, wechat_login: true },
    phoneRefusal('WeChat')
  )
  await renderAt('/sign-in')

  await user.click(
    await screen.findByRole('button', { name: /Continue with WeChat/ })
  )
  await user.type(
    await screen.findByLabelText('Verification code'),
    'wechat-code'
  )
  await user.click(screen.getByRole('button', { name: 'Confirm' }))

  await waitFor(() => expect(error).toHaveBeenCalledTimes(1))
  expect(error).toHaveBeenCalledWith(
    'Registration requires a phone number. Register with a phone number first, then link WeChat in account settings'
  )
})

it('a refused first WeChat sign-in from its callback shows the refusal and returns to the sign-in page', async () => {
  const error = vi.spyOn(toast, 'error')
  mockApi({ register_phone_required: true }, phoneRefusal('WeChat'))
  await renderAt('/oauth?provider=wechat&code=wechat-code')

  expect(await screen.findByText('Sign-in page')).toBeInTheDocument()
  expect(error).toHaveBeenCalledTimes(1)
  expect(error).toHaveBeenCalledWith(
    'Registration requires a phone number. Register with a phone number first, then link WeChat in account settings'
  )
})
