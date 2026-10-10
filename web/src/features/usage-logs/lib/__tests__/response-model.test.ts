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

import {
  isResponseModelMismatch,
  type ResponseModelObservation,
} from '../response-model'

function observation(
  returned: string,
  overrides: Partial<ResponseModelObservation> = {}
): ResponseModelObservation {
  return {
    requested_model: 'requested',
    upstream_model: 'mapped',
    returned_model: returned,
    ...overrides,
  }
}

describe('isResponseModelMismatch', () => {
  test.each([
    ['requested', 'exact requested model'],
    ['mapped', 'exact upstream model'],
    ['Requested', 'case-only difference'],
    ['requested-2026-09-01', 'dated variant of the requested model'],
    ['mapped-2026-09-01', 'dated variant of the upstream model'],
    ['deepseek/requested', 'provider path in front of the requested model'],
    ['vendor/MAPPED', 'provider path with a case-only difference'],
    ['accounts/vendor/models/requested', 'nested provider path'],
  ])('treats %s as compatible (%s)', (returned) => {
    expect(isResponseModelMismatch(observation(returned))).toBe(false)
  })

  test.each([
    ['other', 'different model'],
    ['request', 'shorter name that the expected model extends'],
    ['vendor/other', 'different model behind a provider path'],
    ['vendor/request', 'provider path with a shorter name'],
    ['vendor/', 'provider path with no model segment'],
    ['vendor', 'provider segment alone'],
  ])('flags %s as a mismatch (%s)', (returned) => {
    expect(isResponseModelMismatch(observation(returned))).toBe(true)
  })

  test('compares against provider-qualified requested and upstream names as written', () => {
    expect(
      isResponseModelMismatch(
        observation('vendor/requested-2026-09-01', {
          requested_model: 'vendor/requested',
        })
      )
    ).toBe(false)
    expect(
      isResponseModelMismatch(
        observation('vendor/other', { requested_model: 'vendor/requested' })
      )
    ).toBe(true)
  })

  test.each([
    ['global.vendor.model-v1:0', 'vendor.model-v1:0', 'an inference profile prefix'],
    ['eu.vendor.model-v1:0', 'vendor.model-v1:0', 'a regional profile prefix'],
    ['GLOBAL.Vendor.Model-v1:0', 'vendor.model-v1:0', 'a profile prefix with a case-only difference'],
    ['us.vendor.model', 'vendor.model-2026-09-01', 'a dated name behind a profile prefix'],
  ])(
    'treats the upstream model %s answered as %s as compatible (%s)',
    (upstream, returned) => {
      expect(
        isResponseModelMismatch(
          observation(returned, { upstream_model: upstream })
        )
      ).toBe(false)
    }
  )

  test.each([
    ['global.vendor.model-v1:0', 'vendor.other-v1:0', 'another model behind the prefix'],
    ['global.vendor.model-v1:0', 'model-v1:0', 'two stripped segments'],
    ['global.vendor.model-v1:0', 'vendor.model', 'a shorter name behind the prefix'],
    ['vendor/global.model', 'model', 'a prefix holding a provider path'],
    ['global.', 'other', 'an empty name after the prefix'],
  ])(
    'flags the upstream model %s answered as %s as a mismatch (%s)',
    (upstream, returned) => {
      expect(
        isResponseModelMismatch(
          observation(returned, { upstream_model: upstream })
        )
      ).toBe(true)
    }
  )

  test('empty expected names never match', () => {
    expect(
      isResponseModelMismatch(
        observation('other', { requested_model: '', upstream_model: '' })
      )
    ).toBe(true)
  })

  test('returns false without an observation or a returned model', () => {
    expect(isResponseModelMismatch(undefined)).toBe(false)
    expect(isResponseModelMismatch(observation(''))).toBe(false)
  })
})
