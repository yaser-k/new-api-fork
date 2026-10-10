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
import userEvent from '@testing-library/user-event'
import i18next from 'i18next'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { CompactDateTimeRangePicker } from '../compact-date-time-range-picker'

// 2026-09-23 is 1 Mehr 1405, 2026-09-25 is 3 Mehr and 2026-09-27 is 5 Mehr.
afterEach(async () => {
  vi.useRealTimers()
  await i18next.changeLanguage('en')
})

describe('CompactDateTimeRangePicker in Persian', () => {
  it('shows the range in Solar Hijri and keeps the values when confirmed unchanged', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    const start = new Date(2026, 8, 25, 0, 0)
    const end = new Date(2026, 8, 25, 11, 35)
    await i18next.changeLanguage('fa')
    render(
      <CompactDateTimeRangePicker start={start} end={end} onChange={onChange} />
    )

    expect(
      screen.getByText('۱۴۰۵/۰۷/۰۳ ۰۰:۰۰ ~ ۱۴۰۵/۰۷/۰۳ ۱۱:۳۵')
    ).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /۱۴۰۵\/۰۷\/۰۳/ }))
    expect(
      await screen.findByRole('grid', { name: 'مهر ۱۴۰۵' })
    ).toBeInTheDocument()
    expect(screen.getByLabelText('Start Time')).toHaveValue('۰۰:۰۰')
    expect(screen.getByLabelText('End Time')).toHaveValue('۱۱:۳۵')

    await user.click(screen.getByRole('button', { name: 'Confirm' }))
    expect(onChange).toHaveBeenCalledWith({ start, end })
  })

  it('picks a range on the Solar Hijri calendar with 24-hour times typed in Persian digits', async () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date(2026, 8, 25, 12))
    const user = userEvent.setup()
    const onChange = vi.fn()
    await i18next.changeLanguage('fa')
    render(<CompactDateTimeRangePicker onChange={onChange} />)

    await user.click(screen.getAllByRole('button', { name: 'Date Range' })[0])
    await screen.findByRole('grid', { name: 'مهر ۱۴۰۵' })
    await user.click(
      screen.getByRole('button', { name: /(^|\s)۱-ام مهر ۱۴۰۵/ })
    )
    await user.click(
      screen.getByRole('button', { name: /(^|\s)۵-ام مهر ۱۴۰۵/ })
    )
    const startTime = screen.getByLabelText('Start Time')
    await user.clear(startTime)
    await user.type(startTime, '۰۹:۳۰')
    const endTime = screen.getByLabelText('End Time')
    await user.clear(endTime)
    await user.type(endTime, '18:05')
    await user.click(screen.getByRole('button', { name: 'Confirm' }))

    expect(onChange).toHaveBeenCalledWith({
      start: new Date(2026, 8, 23, 9, 30, 0, 0),
      end: new Date(2026, 8, 27, 18, 5, 59, 999),
    })
  })
})

describe('CompactDateTimeRangePicker time fields', () => {
  it('in English, shows the Gregorian month and ends a picked day at its last minute', async () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date(2026, 8, 25, 12))
    const user = userEvent.setup()
    const onChange = vi.fn()
    render(<CompactDateTimeRangePicker onChange={onChange} />)

    await user.click(screen.getAllByRole('button', { name: 'Date Range' })[0])
    expect(
      screen.getByRole('grid', { name: 'September 2026' })
    ).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /September 25th, 2026/ }))
    await user.click(screen.getByRole('button', { name: 'Confirm' }))

    expect(onChange).toHaveBeenCalledWith({
      start: new Date(2026, 8, 25, 0, 0, 0, 0),
      end: new Date(2026, 8, 25, 23, 59, 59, 999),
    })
  })

  it('marks a time outside 00:00-23:59 invalid and disables Confirm', async () => {
    const user = userEvent.setup()
    render(
      <CompactDateTimeRangePicker
        start={new Date(2026, 8, 25, 0, 0)}
        end={new Date(2026, 8, 25, 11, 35)}
        onChange={() => {}}
      />
    )

    await user.click(screen.getAllByRole('button', { name: /^2026/ })[0])
    const endTime = screen.getByLabelText('End Time')
    await user.clear(endTime)
    await user.type(endTime, '24:10')

    expect(endTime).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByRole('button', { name: 'Confirm' })).toBeDisabled()
  })
})
