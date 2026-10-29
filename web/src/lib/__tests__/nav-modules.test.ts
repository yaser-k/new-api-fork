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
import { describe, expect, test } from 'vitest'

import { parseHeaderNavModules } from '../nav-modules'

describe('parseHeaderNavModules contact module', () => {
  test('an absent contact module is off with an empty link and the other modules keep their defaults', () => {
    expect(parseHeaderNavModules('{"docs":false}')).toEqual({
      home: true,
      console: true,
      pricing: { enabled: true, requireAuth: false },
      rankings: { enabled: true, requireAuth: false },
      docs: false,
      about: true,
      contact: { enabled: false, link: '' },
    })
  })

  test('a contact object is read as its enabled flag and its link', () => {
    const modules = parseHeaderNavModules(
      JSON.stringify({ contact: { enabled: true, link: '/contact' } })
    )

    expect(modules.contact).toEqual({ enabled: true, link: '/contact' })
  })

  test.each([true, 'true', 1])(
    'a contact value %j that is not an object leaves the module off',
    (value) => {
      const modules = parseHeaderNavModules(JSON.stringify({ contact: value }))

      expect(modules.contact).toEqual({ enabled: false, link: '' })
    }
  )

  test('an object value for a key the parser does not know is still ignored', () => {
    const modules = parseHeaderNavModules(
      JSON.stringify({ chat: { enabled: true, link: '/chat' } })
    )

    expect(modules).not.toHaveProperty('chat')
  })
})
