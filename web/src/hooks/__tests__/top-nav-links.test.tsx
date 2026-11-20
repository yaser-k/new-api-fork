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
import { renderHook } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { STATUS_QUERY_KEY } from '@/lib/status-query'

import { useTopNavLinks } from '../use-top-nav-links'

beforeEach(() => {
  vi.stubGlobal('localStorage', {
    getItem: () => null,
    setItem: () => undefined,
    removeItem: () => undefined,
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

const TAG_LINKS = [
  { title: 'Home', href: '/' },
  { title: 'Console', href: '/dashboard' },
  { title: 'Model Square', href: '/pricing', requiresAuth: false },
  { title: 'Rankings', href: '/rankings', requiresAuth: false },
  { title: 'Docs', href: '/docs' },
  { title: 'About', href: '/about' },
]

function linksFor(status: Record<string, unknown>) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  client.setQueryData(STATUS_QUERY_KEY, status)
  function Wrapper(props: { children: ReactNode }) {
    return (
      <QueryClientProvider client={client}>
        {props.children}
      </QueryClientProvider>
    )
  }
  return renderHook(() => useTopNavLinks(), { wrapper: Wrapper }).result.current
}

function linksWithContact(contact: unknown) {
  return linksFor({ HeaderNavModules: JSON.stringify({ contact }) })
}

describe('useTopNavLinks contact item', () => {
  test('without a contact module the links are the ones the header always had', () => {
    expect(linksFor({ HeaderNavModules: '{"home":true}' })).toEqual(TAG_LINKS)
  })

  test('an enabled same-site path adds a last Contact item loaded by the browser in the same tab', () => {
    expect(linksWithContact({ enabled: true, link: ' /contact ' })).toEqual([
      ...TAG_LINKS,
      { title: 'Contact', href: '/contact', reloadDocument: true },
    ])
  })

  test('an enabled https URL adds a last Contact item that opens like the external Docs link', () => {
    const links = linksFor({
      docs_link: 'https://docs.example.com',
      HeaderNavModules: JSON.stringify({
        contact: { enabled: true, link: 'https://example.com/contact' },
      }),
    })

    expect(links.find((link) => link.title === 'Docs')).toEqual({
      title: 'Docs',
      href: 'https://docs.example.com',
      external: true,
    })
    expect(links.at(-1)).toEqual({
      title: 'Contact',
      href: 'https://example.com/contact',
      external: true,
    })
  })

  test('a disabled contact module with a link adds no item', () => {
    expect(linksWithContact({ enabled: false, link: '/contact' })).toEqual(
      TAG_LINKS
    )
  })

  test.each([
    '',
    '   ',
    'javascript:alert(1)',
    '//example.com',
    '/\\example.com',
    'mailto:team@example.com',
    'contact',
  ])('an enabled contact module with the link %j adds no item', (link) => {
    expect(linksWithContact({ enabled: true, link })).toEqual(TAG_LINKS)
  })
})
