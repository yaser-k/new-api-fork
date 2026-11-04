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
import { cleanup, render, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { LanguageSwitcher } from '../language-switcher'

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
})

function renderWithLanguages(languages: string[]) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  client.setQueryData(['status'], { interface_languages: languages })
  function Wrapper(props: { children: ReactNode }) {
    return (
      <QueryClientProvider client={client}>
        {props.children}
      </QueryClientProvider>
    )
  }
  render(<LanguageSwitcher />, { wrapper: Wrapper })
  return screen.getByRole('button').querySelector('svg')
}

describe('language button picture', () => {
  test('a menu of Russian and English shows the Cyrillic mark beside the A', () => {
    const icon = renderWithLanguages(['ru', 'en'])
    expect(icon).toHaveAttribute('data-mark', 'Я')
    expect(icon).toHaveTextContent('Я')
  })

  test('the order of the two languages does not change the mark', () => {
    expect(renderWithLanguages(['en', 'ja'])).toHaveAttribute('data-mark', 'あ')
  })

  test('two Latin languages keep the generic languages icon', () => {
    const icon = renderWithLanguages(['en', 'fr'])
    expect(icon).not.toHaveAttribute('data-mark')
    expect(icon).toHaveClass('lucide-languages')
  })

  test('a menu without a Latin language keeps the generic languages icon', () => {
    expect(renderWithLanguages(['zhCN', 'ja'])).toHaveClass('lucide-languages')
  })

  test('more than two languages keep the generic languages icon', () => {
    expect(renderWithLanguages(['en', 'ru', 'ja'])).toHaveClass(
      'lucide-languages'
    )
  })

  test('without a configured menu every shipped language is offered and the generic icon stays', () => {
    expect(renderWithLanguages([])).toHaveClass('lucide-languages')
  })
})
