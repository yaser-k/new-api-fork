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
import { cleanup, render, renderHook, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { SidebarProvider } from '@/components/ui/sidebar'
import { DirectionProvider } from '@/context/direction-provider'
import { useHeaderDirection } from '@/hooks/use-header-direction'

import { Header } from '../header'

beforeEach(() => {
  vi.stubGlobal('localStorage', {
    getItem: () => null,
    setItem: () => undefined,
    removeItem: () => undefined,
  })
})

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  document.cookie = 'dir=; max-age=0; path=/'
  document.documentElement.removeAttribute('dir')
})

function wrapperFor(status: Record<string, unknown>) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  client.setQueryData(['status'], status)
  return function Wrapper(props: { children: ReactNode }) {
    return (
      <QueryClientProvider client={client}>
        <DirectionProvider>
          <SidebarProvider>{props.children}</SidebarProvider>
        </DirectionProvider>
      </QueryClientProvider>
    )
  }
}

// The page direction a visitor chose is kept in the `dir` cookie, which the
// direction provider reads when it mounts.
function setPageDirection(dir: 'ltr' | 'rtl') {
  document.cookie = `dir=${dir}; path=/`
}

function headerDirectionIn(pageDir: 'ltr' | 'rtl', keepHeaderLtr: boolean) {
  setPageDirection(pageDir)
  const { result } = renderHook(() => useHeaderDirection(), {
    wrapper: wrapperFor({ keep_header_ltr: keepHeaderLtr }),
  })
  return result.current
}

describe('top bar direction', () => {
  test('without keep_header_ltr a right-to-left bar mirrors with the page', () => {
    expect(headerDirectionIn('rtl', false)).toEqual({})
  })

  test('with keep_header_ltr a left-to-right page is left as it is', () => {
    expect(headerDirectionIn('ltr', true)).toEqual({})
  })

  test('with keep_header_ltr a right-to-left bar is laid out left to right and its links keep the page order', () => {
    expect(headerDirectionIn('rtl', true)).toEqual({
      barDir: 'ltr',
      linksDir: 'rtl',
    })
  })

  test('the console bar carries dir="ltr" only when the setting pins it', () => {
    setPageDirection('rtl')
    const pinned = render(<Header aria-label='Console bar' />, {
      wrapper: wrapperFor({ keep_header_ltr: true }),
    })
    expect(screen.getByRole('banner', { name: 'Console bar' })).toHaveAttribute(
      'dir',
      'ltr'
    )
    pinned.unmount()

    render(<Header aria-label='Console bar' />, {
      wrapper: wrapperFor({ keep_header_ltr: false }),
    })
    expect(
      screen.getByRole('banner', { name: 'Console bar' })
    ).not.toHaveAttribute('dir')
  })
})
