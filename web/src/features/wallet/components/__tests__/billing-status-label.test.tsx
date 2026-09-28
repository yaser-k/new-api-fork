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
import { render, screen } from '@testing-library/react'
import { createInstance } from 'i18next'
import { I18nextProvider } from 'react-i18next'
import { afterEach, expect, it, vi } from 'vitest'

import en from '@/i18n/locales/en.json'
import zh from '@/i18n/locales/zh.json'
import { api } from '@/lib/api'

import type { TopupRecord } from '../../types'
import { BillingHistoryDialog } from '../dialogs/billing-history-dialog'

const record = {
  user_id: 1,
  amount: 10,
  money: 10,
  payment_method: 'stripe',
  create_time: 1_750_000_000,
} as const

const records: TopupRecord[] = [
  { ...record, id: 1, trade_no: 'ORDER-1', status: 'success' },
  { ...record, id: 2, trade_no: 'ORDER-2', status: 'pending' },
  { ...record, id: 3, trade_no: 'ORDER-3', status: 'expired' },
]

afterEach(() => {
  vi.restoreAllMocks()
})

it.each([
  ['en', ['Success', 'Pending', 'Expired']],
  ['zh', ['成功', '待确认', '已过期']],
])(
  'shows the top-up status labels in the interface language (%s)',
  async (lng, labels) => {
    const i18n = createInstance()
    await i18n.init({ lng, fallbackLng: 'en', resources: { en, zh } })
    vi.spyOn(api, 'get').mockResolvedValue({
      data: { success: true, data: { items: records, total: records.length } },
    })

    render(
      <I18nextProvider i18n={i18n}>
        <BillingHistoryDialog open onOpenChange={vi.fn()} />
      </I18nextProvider>
    )

    for (const label of labels) {
      expect(await screen.findByText(label)).toBeVisible()
    }
  }
)

it('does not show the English status labels in Chinese', async () => {
  const i18n = createInstance()
  await i18n.init({ lng: 'zh', fallbackLng: 'en', resources: { en, zh } })
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: { items: records, total: records.length } },
  })

  render(
    <I18nextProvider i18n={i18n}>
      <BillingHistoryDialog open onOpenChange={vi.fn()} />
    </I18nextProvider>
  )

  expect(await screen.findByText('ORDER-1')).toBeVisible()
  for (const label of ['Success', 'Pending', 'Expired']) {
    expect(screen.queryByText(label)).not.toBeInTheDocument()
  }
})
