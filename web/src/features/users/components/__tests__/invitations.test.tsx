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
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { toast } from 'sonner'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { TooltipProvider } from '@/components/ui/tooltip'
import type { Invitation } from '@/features/invitations/types'
import { api } from '@/lib/api'

import { UsersPrimaryButtons } from '../users-primary-buttons'
import { UsersProvider } from '../users-provider'

const link =
  'https://console.example.com/sign-up?invite=q3Jv0Xb8T1mN5yZ2kL7pC4sW9dF6gH0aE3rU8iO1xVw'
const pending: Invitation = {
  id: 7,
  email: 'invitee@example.com',
  language: 'fr',
  created_by: 1,
  created_time: 1761696000,
  expired_time: 1761955200,
  used_time: 0,
  used_user_id: 0,
  revoked_time: 0,
  status: 'pending',
}
const clients: QueryClient[] = []

function mockApi(options: { enabled: boolean; invitations?: Invitation[] }) {
  const get = vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/status') {
      return {
        data: {
          success: true,
          data: { invitation_register_enabled: options.enabled },
        },
      }
    }
    if (url === '/api/user/invitation') {
      const items = options.invitations ?? []
      return {
        data: {
          success: true,
          data: { page: 1, page_size: 10, total: items.length, items },
        },
      }
    }
    return { data: { success: true, data: null } }
  })
  return { get }
}

function renderButtons() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  clients.push(client)
  render(
    <QueryClientProvider client={client}>
      <TooltipProvider>
        <UsersProvider>
          <UsersPrimaryButtons />
        </UsersProvider>
      </TooltipProvider>
    </QueryClientProvider>
  )
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

it('hides the invitation actions while invitation registration is off', async () => {
  const { get } = mockApi({ enabled: false })
  renderButtons()

  await waitFor(() => expect(get).toHaveBeenCalledWith('/api/status'))
  expect(screen.getByRole('button', { name: 'Add User' })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Invite by email' })).toBeNull()
  expect(
    screen.queryByRole('button', { name: 'Account invitations' })
  ).toBeNull()
})

it('the invite dialog posts the email and the chosen language and shows the link once', async () => {
  const user = userEvent.setup()
  mockApi({ enabled: true })
  const post = vi.spyOn(api, 'post').mockResolvedValue({
    data: { success: true, data: { invitation: pending, link } },
  })
  renderButtons()

  await user.click(
    await screen.findByRole('button', { name: 'Invite by email' })
  )
  const dialog = await screen.findByRole('dialog')
  await user.type(within(dialog).getByLabelText('Email'), 'invitee@example.com')
  await user.click(within(dialog).getByRole('combobox'))
  await user.click(await screen.findByRole('option', { name: 'Français' }))
  await user.click(within(dialog).getByRole('button', { name: 'Send' }))

  await waitFor(() =>
    expect(post).toHaveBeenCalledWith('/api/user/invitation', {
      email: 'invitee@example.com',
      language: 'fr',
    })
  )
  expect(await within(dialog).findByLabelText('Invitation link')).toHaveValue(
    link
  )
  expect(
    within(dialog).getByRole('button', { name: 'Copy invitation link' })
  ).toBeInTheDocument()

  await user.keyboard('{Escape}')
  await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
  await user.click(screen.getByRole('button', { name: 'Invite by email' }))
  const reopened = await screen.findByRole('dialog')
  expect(within(reopened).queryByLabelText('Invitation link')).toBeNull()
  expect(within(reopened).getByLabelText('Email')).toHaveValue('')
})

it('a failed mail shows its error and leaves the invitation pending in the list, where it can be resent', async () => {
  const user = userEvent.setup()
  const { get } = mockApi({ enabled: true, invitations: [pending] })
  const error = vi.spyOn(toast, 'error')
  const post = vi.spyOn(api, 'post').mockImplementation(async (url) => {
    if (url === '/api/user/invitation') {
      return {
        data: {
          success: false,
          message: 'Resend email delivery failed (HTTP 500)',
        },
      }
    }
    return { data: { success: true, data: { invitation: pending, link } } }
  })
  renderButtons()

  await user.click(
    await screen.findByRole('button', { name: 'Invite by email' })
  )
  const dialog = await screen.findByRole('dialog')
  await user.type(within(dialog).getByLabelText('Email'), 'invitee@example.com')
  await user.click(within(dialog).getByRole('button', { name: 'Send' }))
  await waitFor(() =>
    expect(error).toHaveBeenCalledWith(
      'Resend email delivery failed (HTTP 500)'
    )
  )
  expect(within(dialog).queryByLabelText('Invitation link')).toBeNull()
  await user.click(within(dialog).getByRole('button', { name: 'Cancel' }))
  await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())

  await user.click(screen.getByRole('button', { name: 'Account invitations' }))
  const list = await screen.findByRole('dialog')
  expect(
    await within(list).findByText('invitee@example.com')
  ).toBeInTheDocument()
  expect(within(list).getByText('Pending')).toBeInTheDocument()
  expect(get).toHaveBeenCalledWith('/api/user/invitation', {
    params: { p: 1, page_size: 10 },
  })

  await user.click(within(list).getByRole('button', { name: 'Resend' }))
  await waitFor(() =>
    expect(post).toHaveBeenCalledWith('/api/user/invitation/7/resend')
  )
  expect(await within(list).findByLabelText('Invitation link')).toHaveValue(
    link
  )
})

it('revoking an invitation asks for confirmation before it is sent', async () => {
  const user = userEvent.setup()
  mockApi({ enabled: true, invitations: [pending] })
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { success: true, data: pending } })
  renderButtons()

  await user.click(
    await screen.findByRole('button', { name: 'Account invitations' })
  )
  const list = await screen.findByRole('dialog')
  await user.click(await within(list).findByRole('button', { name: 'Revoke' }))

  const confirm = await screen.findByRole('alertdialog')
  expect(confirm).toHaveTextContent('invitee@example.com')
  expect(post).not.toHaveBeenCalled()
  await user.click(within(confirm).getByRole('button', { name: 'Revoke' }))
  await waitFor(() =>
    expect(post).toHaveBeenCalledWith('/api/user/invitation/7/revoke')
  )
})
