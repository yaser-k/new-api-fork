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
import { toast } from 'sonner'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { SettingsPageProvider } from '../../components/settings-page-context'
import { SSRFSection } from '../ssrf-section'

const portsError =
  'Enter ports from 1 to 65535 or ranges such as 8000-9000, separated by commas'

function Fixture(props: { allowedPorts: Array<number | string> }) {
  const [container, setContainer] = useState<HTMLDivElement | null>(null)
  const [client] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: { retry: false },
          mutations: { retry: false },
        },
      })
  )
  return (
    <QueryClientProvider client={client}>
      <div ref={setContainer} />
      <SettingsPageProvider actionsContainer={container}>
        <SSRFSection
          defaultValues={{
            'fetch_setting.enable_ssrf_protection': true,
            'fetch_setting.allow_private_ip': false,
            'fetch_setting.domain_filter_mode': false,
            'fetch_setting.ip_filter_mode': false,
            'fetch_setting.domain_list': [],
            'fetch_setting.ip_list': [],
            'fetch_setting.allowed_ports': props.allowedPorts,
            'fetch_setting.apply_ip_filter_for_domain': true,
          }}
        />
      </SettingsPageProvider>
    </QueryClientProvider>
  )
}

function portsInput() {
  return screen.getByRole('textbox', { name: 'Allowed Ports' })
}

function saveButton() {
  return screen.getByRole('button', { name: 'Save SSRF settings' })
}

beforeEach(() => {
  vi.spyOn(api, 'put').mockResolvedValue({ data: { success: true } })
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('SSRF allowed ports', () => {
  it('saves typed ports and a port range as the same entries in text', async () => {
    const user = userEvent.setup()
    render(<Fixture allowedPorts={['80', '443']} />)
    await user.clear(portsInput())
    await user.type(portsInput(), '80, 443, 8000-9000')
    await user.click(saveButton())

    await waitFor(() =>
      expect(api.put).toHaveBeenCalledExactlyOnceWith('/api/option/', {
        key: 'fetch_setting.allowed_ports',
        value: '["80","443","8000-9000"]',
      })
    )
  })

  it.each([
    [['80', '443', '8000-9000'], '80,443,8000-9000'],
    [[80, 443], '80,443'],
  ])(
    'shows saved ports %j as "%s" and saves nothing when they are unchanged',
    async (saved, displayed) => {
      const info = vi.spyOn(toast, 'info')
      const user = userEvent.setup()
      render(<Fixture allowedPorts={saved} />)
      expect(portsInput()).toHaveValue(displayed)
      await user.click(saveButton())

      await waitFor(() =>
        expect(info).toHaveBeenCalledExactlyOnceWith('No changes to save')
      )
      expect(api.put).not.toHaveBeenCalled()
    }
  )

  it.each(['0', '70000', '9000-8000', 'abc'])(
    'refuses "%s" with a field error and saves nothing',
    async (entry) => {
      const user = userEvent.setup()
      render(<Fixture allowedPorts={['80']} />)
      await user.clear(portsInput())
      await user.type(portsInput(), `443, ${entry}`)
      await user.click(saveButton())

      expect(await screen.findByText(portsError)).toBeVisible()
      expect(portsInput()).toHaveAttribute('aria-invalid', 'true')
      expect(api.put).not.toHaveBeenCalled()
    }
  )
})
