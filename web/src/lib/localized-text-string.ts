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
import { resolveLocalizedText } from './localized-text'

/**
 * Parse admin-entered text that may carry one version per language.
 *
 * Model and group descriptions are stored as plain strings. An administrator
 * who serves more than one interface language can store a JSON object instead,
 * keyed by language tag, for example `{"en": "Fast", "fr": "Rapide"}`.
 * Returns the map when the value is such an object with at least one non-empty
 * string member, and `null` for anything else, plain text included.
 */
export function parseLocalizedTextString(
  value: string | null | undefined
): Record<string, string> | null {
  if (typeof value !== 'string') return null
  const trimmed = value.trim()
  if (!trimmed.startsWith('{') || !trimmed.endsWith('}')) return null

  let parsed: unknown
  try {
    parsed = JSON.parse(trimmed)
  } catch {
    return null
  }
  if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
    return null
  }

  const texts: Record<string, string> = {}
  for (const [key, text] of Object.entries(parsed)) {
    if (typeof text !== 'string') return null
    if (key.trim() === '' || text.trim() === '') continue
    texts[key] = text
  }
  return Object.keys(texts).length > 0 ? texts : null
}

/**
 * Resolve admin-entered text for the current interface language.
 *
 * Plain text is returned unchanged, so existing data keeps working. A JSON
 * object of texts per language is resolved with `resolveLocalizedText`
 * (exact tag, primary subtag, `en`, then the first key).
 */
export function resolveLocalizedTextString(
  value: string | null | undefined,
  language: string
): string {
  if (value == null) return ''
  const texts = parseLocalizedTextString(value)
  if (!texts) return value
  return resolveLocalizedText(texts, language)
}
