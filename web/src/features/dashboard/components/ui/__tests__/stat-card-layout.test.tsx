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
import { WalletCards } from 'lucide-react'
import { expect, it } from 'vitest'

import { StatCard } from '../stat-card'

it('wraps a stat amount only between the number and the unit word', () => {
  render(
    <StatCard
      title='Current Balance'
      value='203,700 Credits'
      description='Remaining quota'
      icon={WalletCards}
    />
  )

  const value = screen.getByText('203,700 Credits')
  expect(value).toHaveClass('break-normal')
  expect(value).not.toHaveClass('break-all')
})
