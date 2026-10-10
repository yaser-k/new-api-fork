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
import { describe, expect, it } from 'vitest'

import { parseTimeOfDay } from '../time-of-day'

describe('parseTimeOfDay', () => {
  it.each([
    ['09:30', { hours: 9, minutes: 30 }],
    ['9:30', { hours: 9, minutes: 30 }],
    ['0930', { hours: 9, minutes: 30 }],
    [' 23:59 ', { hours: 23, minutes: 59 }],
    ['00:00', { hours: 0, minutes: 0 }],
    ['۰۹:۳۰', { hours: 9, minutes: 30 }],
    ['٠٩:٣٠', { hours: 9, minutes: 30 }],
  ])('reads %j as a 24-hour time', (text, expected) => {
    expect(parseTimeOfDay(text)).toEqual(expected)
  })

  it.each(['', '24:00', '12:60', '1:2', '9.30', '09:30 PM', '123:45'])(
    'rejects %j',
    (text) => {
      expect(parseTimeOfDay(text)).toBeUndefined()
    }
  )
})
