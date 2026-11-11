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
import { type Direction, useDirection } from '@/context/direction-provider'
import { useStatus } from '@/hooks/use-status'

/**
 * `dir` attributes for a top bar. Both are undefined while the bar follows the
 * page, so the markup is unchanged.
 */
export type HeaderDirection = {
  /** For the bar's own element: where its ends sit. */
  barDir?: Direction
  /** For the navigation links inside the bar: the page's reading order. */
  linksDir?: Direction
}

/**
 * The direction of the top bars.
 *
 * By default a bar mirrors with the page. With the operator's
 * `interface_setting.keep_header_ltr` (`keep_header_ltr` in `/api/status`) and
 * a right-to-left page, a bar is laid out left to right instead, so the logo
 * stays at the left and the actions at the right in every language. The
 * navigation links inside it keep the page's direction, and menus, sheets, the
 * sidebar and the page are not affected.
 */
export function useHeaderDirection(): HeaderDirection {
  const { dir } = useDirection()
  const { status } = useStatus()
  if (status?.keep_header_ltr !== true || dir !== 'rtl') return {}
  return { barDir: 'ltr', linksDir: 'rtl' }
}
