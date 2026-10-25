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

import { getTieredBillingSummary } from '@/features/usage-logs/lib/format'

import type { PricingModel } from '../../types'
import { parseTiersFromExpr } from '../billing-expr'
import { compileBillingExpression } from '../billing-expression/parser'
import {
  getDynamicPriceUnitLabelKey,
  getDynamicPricingSummary,
} from '../dynamic-price'
import {
  buildEstimatorTokens,
  evalExprLocally,
  type ExtraTokenValues,
} from '../tier-expr'

// USD 7.50 per 1M output tokens and USD 0.00049 per second of input video.
const videoExpression = 'tier("base", c * 7.5 + vid_sec * 490)'

const noExtras: ExtraTokenValues = {
  cacheReadTokens: 0,
  cacheCreateTokens: 0,
  cacheCreate1hTokens: 0,
  imageTokens: 0,
  imageOutputTokens: 0,
  imageCacheTokens: 0,
  audioInputTokens: 0,
  audioOutputTokens: 0,
  videoSeconds: 0,
}

describe('vid_sec in billing expressions', () => {
  test('the parser accepts vid_sec and still refuses an unknown name', () => {
    const compiled = compileBillingExpression(videoExpression)
    expect(compiled.status).toBe('ready')
    expect(
      compiled.status === 'ready' && compiled.variables.has('vid_sec')
    ).toBe(true)
    expect(
      compileBillingExpression('tier("base", vid_secs * 490)')
    ).toMatchObject({
      status: 'unsupported',
      diagnostic: { code: 'unsupported' },
    })
  })

  test('the estimator prices 2 seconds of video beside 18 output tokens', () => {
    const extras = { ...noExtras, videoSeconds: 2 }
    expect(evalExprLocally(videoExpression, 108, 18, extras)).toMatchObject({
      cost: 1115,
      error: null,
    })
    expect(
      evalExprLocally('tier("base", c * 7.5)', 108, 18, extras)
    ).toMatchObject({ cost: 135, error: null })
    expect(buildEstimatorTokens(108, 18, extras)).toMatchObject({
      len: 108,
      vid_sec: 2,
    })
  })

  test('a displayed video price is per second, not per 1M tokens', () => {
    const [tier] = parseTiersFromExpr(videoExpression)
    expect(tier.outputPrice).toBe(7.5)
    expect(tier.videoInputPrice).toBe(0.00049)

    const model: PricingModel = {
      id: 1,
      model_name: 'twelvelabs.pegasus-1-5-v1:0',
      quota_type: 0,
      model_ratio: 1,
      completion_ratio: 1,
      enable_groups: ['default'],
      billing_mode: 'tiered_expr',
      billing_expr: videoExpression,
    }
    const summary = getDynamicPricingSummary(model, { tokenUnit: 'K' })
    const video = summary?.entries.find((entry) => entry.key === 'vid_sec')
    expect(video).toMatchObject({ unit: 'second', value: 0.00049 })
    expect(video?.formatted).toContain('0.00049')
    expect(video && getDynamicPriceUnitLabelKey(video)).toBe('s')
  })
  test('a consume log lists the video price per second', () => {
    const summary = getTieredBillingSummary({
      billing_mode: 'tiered_expr',
      expr_b64: btoa(videoExpression),
      matched_tier: 'base',
    })
    expect(
      summary?.priceEntries.find((entry) => entry.field === 'videoInputPrice')
    ).toEqual({
      field: 'videoInputPrice',
      shortLabel: 'Video seconds',
      price: 0.00049,
      unit: 'second',
    })
  })
})
