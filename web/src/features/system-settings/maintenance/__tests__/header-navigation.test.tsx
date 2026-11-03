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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { beforeEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { SettingsPageProvider } from '../../components/settings-page-context'
import { parseHeaderNavModules, serializeHeaderNavModules } from '../config'
import { HeaderNavigationSection } from '../header-navigation-section'

const STORED = {
  home: false,
  console: true,
  pricing: { enabled: false, requireAuth: true },
  rankings: { enabled: true, requireAuth: true },
  docs: false,
  about: true,
  playground: false,
}

function Fixture(props: { stored: string }) {
  const [container, setContainer] = useState<HTMLDivElement | null>(null)
  const config = parseHeaderNavModules(props.stored)
  return (
    <>
      <div ref={setContainer} />
      <SettingsPageProvider actionsContainer={container}>
        <HeaderNavigationSection
          config={config}
          initialSerialized={serializeHeaderNavModules(config)}
        />
      </SettingsPageProvider>
    </>
  )
}

function renderSettings(stored = JSON.stringify(STORED)) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <Fixture stored={stored} />
    </QueryClientProvider>
  )
}

async function saveWithContact(enable: boolean, link: string) {
  const user = userEvent.setup()
  renderSettings()
  if (enable) {
    await user.click(screen.getByRole('switch', { name: 'Contact' }))
  }
  const input = screen.getByRole('textbox', { name: 'Contact link' })
  await user.clear(input)
  if (link) await user.type(input, link)
  await user.click(
    await screen.findByRole('button', { name: 'Save navigation' })
  )
  return input
}

beforeEach(() => {
  vi.spyOn(api, 'put').mockResolvedValue({ data: { success: true } })
})

test.each(['javascript:alert(1)', '//example.com', 'contact', 'ftp://x.io'])(
  'a contact link %j that is neither a same-site path nor an http(s) URL is refused',
  async (link) => {
    const input = await saveWithContact(false, link)

    await waitFor(() => expect(input).toHaveAttribute('aria-invalid', 'true'))
    expect(api.put).not.toHaveBeenCalled()
  }
)

test('enabling the Contact item with an empty link is refused', async () => {
  const input = await saveWithContact(true, '')

  await waitFor(() => expect(input).toHaveAttribute('aria-invalid', 'true'))
  expect(api.put).not.toHaveBeenCalled()
})

test('saving an enabled Contact item keeps every other module the stored value holds', async () => {
  await saveWithContact(true, ' /contact ')

  await waitFor(() => expect(api.put).toHaveBeenCalledTimes(1))
  const [url, body] = vi.mocked(api.put).mock.calls[0]
  expect(url).toBe('/api/option/')
  expect(body).toMatchObject({ key: 'HeaderNavModules' })
  expect(JSON.parse((body as { value: string }).value)).toEqual({
    ...STORED,
    contact: { enabled: true, link: '/contact' },
  })
})

test('an https URL is accepted as a contact link', async () => {
  await saveWithContact(true, 'https://example.com/contact')

  await waitFor(() => expect(api.put).toHaveBeenCalledTimes(1))
  const body = vi.mocked(api.put).mock.calls[0][1] as { value: string }
  expect(JSON.parse(body.value).contact).toEqual({
    enabled: true,
    link: 'https://example.com/contact',
  })
})

test('a stored Contact item fills the switch and the link field', () => {
  renderSettings(
    JSON.stringify({ ...STORED, contact: { enabled: true, link: '/contact' } })
  )

  expect(screen.getByRole('switch', { name: 'Contact' })).toBeChecked()
  expect(screen.getByRole('textbox', { name: 'Contact link' })).toHaveValue(
    '/contact'
  )
})
