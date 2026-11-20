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
import { render, screen, within } from '@testing-library/react'
import { createInstance } from 'i18next'
import { I18nextProvider, initReactI18next } from 'react-i18next'
import { beforeEach, expect, it, vi } from 'vitest'

import en from '@/i18n/locales/en.json'

import type { TopupInfo } from '../../types'
import { RechargeFormCard } from '../recharge-form-card'

const DEFAULT_NOTICE =
  'Online topup is not enabled. Please use redemption code or contact administrator.'

const i18n = createInstance()

beforeEach(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en',
    fallbackLng: false,
    nsSeparator: false,
    resources: { en },
    interpolation: { escapeValue: false },
  })
})

function topupInfo(overrides: Partial<TopupInfo> = {}): TopupInfo {
  return {
    enable_online_topup: false,
    enable_stripe_topup: false,
    enable_redemption: true,
    pay_methods: [],
    min_topup: 1,
    stripe_min_topup: 1,
    amount_options: [],
    discount: {},
    ...overrides,
  }
}

function renderCard(info: TopupInfo) {
  render(
    <I18nextProvider i18n={i18n}>
      <RechargeFormCard
        topupInfo={info}
        presetAmounts={[]}
        selectedPreset={null}
        topupAmount={1}
        paymentAmount={1}
        calculating={false}
        paymentLoading={null}
        redemptionCode=''
        redeeming={false}
        onSelectPreset={vi.fn()}
        onTopupAmountChange={vi.fn()}
        onPaymentMethodSelect={vi.fn()}
        onRedemptionCodeChange={vi.fn()}
        onRedeem={vi.fn()}
      />
    </I18nextProvider>
  )
}

it('describes the card as a redemption code top-up while online top-up is off', () => {
  renderCard(topupInfo())

  expect(screen.getByText('Top up with a redemption code')).toBeVisible()
  expect(
    screen.queryByText('Choose an amount and payment method')
  ).not.toBeInTheDocument()
  expect(screen.getByRole('alert')).toHaveTextContent(DEFAULT_NOTICE)
})

it('keeps the amount and payment description while online top-up is on', () => {
  renderCard(topupInfo({ enable_online_topup: true }))

  expect(screen.getByText('Choose an amount and payment method')).toBeVisible()
  expect(
    screen.queryByText('Top up with a redemption code')
  ).not.toBeInTheDocument()
})

it('does not offer redemption codes in the description when they are disabled too', () => {
  renderCard(topupInfo({ enable_redemption: false }))

  expect(
    screen.queryByText('Top up with a redemption code')
  ).not.toBeInTheDocument()
})

it('shows the administrator notice as plain text in place of the default one', () => {
  const message = '<b>Ask</b> at the front desk.\nOpen weekdays.'
  renderCard(topupInfo({ topup_disabled_message: message }))

  const alert = screen.getByRole('alert')
  expect(alert).toHaveTextContent(
    '<b>Ask</b> at the front desk. Open weekdays.'
  )
  expect(alert).not.toHaveTextContent(DEFAULT_NOTICE)
  expect(alert.querySelector('b')).toBeNull()
  expect(within(alert).getByText(/front desk/)).toHaveClass(
    'whitespace-pre-line'
  )
})

it('falls back to the default notice when the administrator notice is blank', () => {
  renderCard(topupInfo({ topup_disabled_message: '   ' }))

  expect(screen.getByRole('alert')).toHaveTextContent(DEFAULT_NOTICE)
})
