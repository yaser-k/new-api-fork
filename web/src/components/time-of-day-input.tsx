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
import type * as React from 'react'
import { useTranslation } from 'react-i18next'

import { Input } from '@/components/ui/input'
import { toIntlLocale } from '@/i18n/languages'
import dayjs from '@/lib/dayjs'
import { formatDisplayDate } from '@/lib/format'
import { parseTimeOfDay } from '@/lib/time-of-day'
import { cn } from '@/lib/utils'

type TimeOfDayInputProps = Omit<
  React.ComponentProps<typeof Input>,
  'value' | 'onChange' | 'type'
> & {
  value: string
  onValueChange: (value: string) => void
}

/**
 * A 24-hour time field. The browser's own time input follows the browser's
 * locale (AM/PM in an English browser), not the interface language; this one
 * shows the interface's digits, reads Latin, Persian or Arabic-Indic ones and
 * marks a time outside 00:00-23:59 invalid. The caller keeps the text and
 * reads it with parseTimeOfDay.
 */
export function TimeOfDayInput(props: TimeOfDayInputProps) {
  const { value, onValueChange, className, onBlur, ...inputProps } = props
  const { i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const time = parseTimeOfDay(value)

  return (
    <Input
      placeholder='00:00'
      {...inputProps}
      value={value}
      onChange={(event) => onValueChange(event.target.value)}
      onBlur={(event) => {
        if (time) {
          onValueChange(
            formatDisplayDate(
              dayjs().hour(time.hours).minute(time.minutes).toDate(),
              'HH:mm',
              locale
            )
          )
        }
        onBlur?.(event)
      }}
      aria-invalid={!time}
      inputMode='numeric'
      autoComplete='off'
      dir='ltr'
      maxLength={5}
      className={cn('text-center tabular-nums', className)}
    />
  )
}
