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
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'

import { TooltipProvider } from '@/components/ui/tooltip'
import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import type { User } from '../../types'
import { useUsersColumns } from '../users-columns'
import { UsersMutateDrawer } from '../users-mutate-drawer'
import { UsersProvider } from '../users-provider'

const target: User = {
  id: 2,
  username: 'managed-user',
  display_name: 'Managed user',
  role: 1,
  status: 1,
  quota: 0,
  used_quota: 0,
  request_count: 0,
  group: 'default',
  phone: '07911123456',
}

function PhoneColumn(props: { users: User[] }) {
  const columns = useUsersColumns().filter(
    (column) => 'accessorKey' in column && column.accessorKey === 'phone'
  )
  const table = useReactTable({
    columns,
    data: props.users,
    getCoreRowModel: getCoreRowModel(),
  })
  return (
    <table>
      <thead>
        {table.getHeaderGroups().map((group) => (
          <tr key={group.id}>
            {group.headers.map((header) => (
              <th key={header.id}>
                {flexRender(
                  header.column.columnDef.header,
                  header.getContext()
                )}
              </th>
            ))}
          </tr>
        ))}
      </thead>
      <tbody>
        {table.getRowModel().rows.map((row) => (
          <tr key={row.id}>
            {row.getVisibleCells().map((cell) => (
              <td key={cell.id}>
                {flexRender(cell.column.columnDef.cell, cell.getContext())}
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  )
}

function renderDrawer(currentRow?: User) {
  useAuthStore.getState().auth.setUser({ id: 1, username: 'root', role: 100 })
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/authz/catalog') {
      return { data: { success: true, data: { resources: [], roles: [] } } }
    }
    if (url === '/api/group/') {
      return { data: { success: true, data: ['default'] } }
    }
    return { data: { success: true, data: target } }
  })
  const put = vi.spyOn(api, 'put').mockResolvedValue({
    data: { success: true, message: '' },
  })
  const post = vi.spyOn(api, 'post').mockResolvedValue({
    data: { success: true, message: '' },
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <UsersProvider>
        <UsersMutateDrawer
          open
          onOpenChange={() => undefined}
          currentRow={currentRow}
        />
      </UsersProvider>
    </QueryClientProvider>
  )
  return { put, post }
}

afterEach(() => {
  vi.restoreAllMocks()
  useAuthStore.getState().auth.reset()
})

it('the users table shows the stored phone number left to right, and a dash for none', () => {
  render(
    <TooltipProvider>
      <PhoneColumn users={[target, { ...target, id: 3, phone: null }]} />
    </TooltipProvider>
  )

  expect(
    screen.getByRole('columnheader', { name: 'Phone number' })
  ).toBeInTheDocument()
  expect(screen.getByText('07911123456')).toHaveAttribute('dir', 'ltr')
  expect(screen.getByText('-')).toBeInTheDocument()
})

it('editing a user shows the stored phone number and saves a changed one', async () => {
  const user = userEvent.setup()
  const { put } = renderDrawer(target)
  const phone = await screen.findByLabelText('Phone number')
  await waitFor(() => expect(phone).toHaveValue('07911123456'))
  expect(phone).toHaveAttribute('dir', 'ltr')

  await user.clear(phone)
  await user.type(phone, '+44 7911 654321')
  await user.click(screen.getByRole('button', { name: 'Save changes' }))

  await waitFor(() => expect(put).toHaveBeenCalledTimes(1))
  expect(put.mock.calls[0][1]).toMatchObject({
    id: target.id,
    phone: '+44 7911 654321',
  })
})

it('editing a user with the phone field emptied sends an empty number, which clears it', async () => {
  const user = userEvent.setup()
  const { put } = renderDrawer(target)
  const phone = await screen.findByLabelText('Phone number')
  await waitFor(() => expect(phone).toHaveValue('07911123456'))

  await user.clear(phone)
  await user.click(screen.getByRole('button', { name: 'Save changes' }))

  await waitFor(() => expect(put).toHaveBeenCalledTimes(1))
  expect(put.mock.calls[0][1]).toMatchObject({ id: target.id, phone: '' })
})

it('creating a user without a phone number sends none', async () => {
  const user = userEvent.setup()
  const { post } = renderDrawer()
  await user.type(await screen.findByLabelText('Username'), 'new-user')
  await user.type(screen.getByLabelText('Password'), 'new-user-password')
  await user.click(screen.getByRole('button', { name: 'Save changes' }))

  await waitFor(() => expect(post).toHaveBeenCalledTimes(1))
  expect(post.mock.calls[0][1]).not.toHaveProperty('phone', expect.anything())
})
