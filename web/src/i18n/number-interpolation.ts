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
import { createInstance, type FormatterModule } from 'i18next'

import { toIntlLocale } from './languages'

// i18next's own formatter, kept for named formats such as {{value, number}}.
const builtInInstance = createInstance()
void builtInInstance.init({ initAsync: false, resources: {} })
const builtInFormatter = builtInInstance.services.formatter

// Intl number format per locale, or null for a locale that writes Latin
// digits (its numbers are interpolated unchanged).
const numberFormats = new Map<string, Intl.NumberFormat | null>()

/**
 * i18next formatter that writes a number interpolated into a translated
 * string in the numbering system of the interface language, as Intl writes
 * it elsewhere (Persian digits for Persian). Grouping stays off and every
 * fraction digit is kept, so only the digits and the decimal mark change;
 * a language that writes Latin digits gets the number unchanged. Strings
 * and named formats are passed on as before.
 *
 * i18next only calls a formatter for a value without a named format when
 * `interpolation.alwaysFormat` is on, so the module turns it on.
 */
export const localeNumberFormatter: FormatterModule = {
  type: 'formatter',
  init(_services, options) {
    options.interpolation = { ...options.interpolation, alwaysFormat: true }
  },
  add(name, fc) {
    builtInFormatter?.add(name, fc)
  },
  addCached(name, fc) {
    builtInFormatter?.addCached(name, fc)
  },
  format(value, format, lng, options) {
    if (format) {
      return builtInFormatter
        ? builtInFormatter.format(value, format, lng, options)
        : value
    }
    if (typeof value !== 'number' || !Number.isFinite(value)) return value

    const locale = toIntlLocale(lng)
    const key = locale ?? ''
    let numberFormat = numberFormats.get(key)
    if (numberFormat === undefined) {
      const candidate = new Intl.NumberFormat(locale, {
        useGrouping: false,
        maximumFractionDigits: 20,
      })
      numberFormat =
        candidate.resolvedOptions().numberingSystem === 'latn'
          ? null
          : candidate
      numberFormats.set(key, numberFormat)
    }
    return numberFormat ? numberFormat.format(value) : value
  },
}
