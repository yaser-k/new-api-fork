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
import { I18nextProvider, initReactI18next } from 'react-i18next'
import { beforeEach, expect, it, vi } from 'vitest'

import en from '@/i18n/locales/en.json'

import type { UserWalletData } from '../../types'
import { AffiliateRewardsCard } from '../affiliate-rewards-card'

const PROMISE =
  'Earn rewards when users join through your referral link. Transfer accumulated rewards to your balance anytime.'
const HISTORY_ONLY =
  'Inviting users earns no reward at the moment. Your earlier referral rewards stay here.'
const LINK = 'https://example.com/register?aff=abcd'

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

function walletUser(overrides: Partial<UserWalletData> = {}): UserWalletData {
  return {
    id: 1,
    username: 'wallet-user',
    quota: 0,
    used_quota: 0,
    request_count: 0,
    aff_quota: 0,
    aff_history_quota: 0,
    aff_count: 0,
    group: 'default',
    ...overrides,
  }
}

function renderCard(user: UserWalletData, rewardEnabled: boolean) {
  return render(
    <I18nextProvider i18n={i18n}>
      <AffiliateRewardsCard
        user={user}
        affiliateLink={LINK}
        onTransfer={vi.fn()}
        rewardEnabled={rewardEnabled}
      />
    </I18nextProvider>
  )
}

it('offers the referral link and its reward while an inviter reward is set', () => {
  renderCard(walletUser(), true)

  expect(screen.getByText('Referral Program')).toBeVisible()
  expect(screen.getByText(PROMISE)).toBeVisible()
  expect(screen.getByDisplayValue(LINK)).toBeVisible()
  expect(
    screen.getByRole('button', { name: 'Copy referral link' })
  ).toBeVisible()
})

it('hides the card when no inviter reward is set and the user has no referral history', () => {
  const { container } = renderCard(walletUser(), false)

  expect(container).toBeEmptyDOMElement()
})

it('keeps an earlier referral history readable without the reward promise or the link', () => {
  renderCard(
    walletUser({ aff_count: 3, aff_history_quota: 1500, aff_quota: 500 }),
    false
  )

  expect(screen.getByText('Referral Program')).toBeVisible()
  expect(screen.getByText(HISTORY_ONLY)).toBeVisible()
  expect(screen.queryByText(PROMISE)).not.toBeInTheDocument()
  expect(screen.queryByDisplayValue(LINK)).not.toBeInTheDocument()
  expect(
    screen.queryByRole('button', { name: 'Copy referral link' })
  ).not.toBeInTheDocument()
  expect(screen.getByText('3')).toBeVisible()
  expect(
    screen.getByRole('button', { name: 'Transfer to Balance' })
  ).toBeEnabled()
})

it('shows a history of invites alone, without a transfer, once the pending rewards are spent', () => {
  renderCard(walletUser({ aff_count: 2, aff_history_quota: 1000 }), false)

  expect(screen.getByText(HISTORY_ONLY)).toBeVisible()
  expect(screen.getByText('2')).toBeVisible()
  expect(
    screen.queryByRole('button', { name: 'Transfer to Balance' })
  ).not.toBeInTheDocument()
})
