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
import { ChevronDownIcon } from 'lucide-react'
import * as React from 'react'
import { enUS, fr, ja, ru, vi, zhCN, zhTW } from 'react-day-picker/locale'
import { useTranslation } from 'react-i18next'

import { TimeOfDayInput } from '@/components/time-of-day-input'
import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { toIntlLocale } from '@/i18n/languages'
import dayjs from '@/lib/dayjs'
import { formatDisplayDate } from '@/lib/format'
import { parseTimeOfDay } from '@/lib/time-of-day'
import { cn } from '@/lib/utils'

const calendarLocales = {
  en: enUS,
  zhCN,
  fr,
  ru,
  ja,
  vi,
  zhTW,
} as const

interface DateTimePickerProps {
  value?: Date
  onChange?: (date: Date | undefined) => void
  placeholder?: string
  className?: string
}

export function DateTimePicker({
  value,
  onChange,
  placeholder,
  className,
}: DateTimePickerProps) {
  const { t, i18n } = useTranslation()
  const placeholderText = placeholder ?? t('Select date')
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const calendarLocale =
    calendarLocales[
      (i18n.resolvedLanguage || i18n.language) as keyof typeof calendarLocales
    ] ?? enUS
  const currentYear = new Date().getFullYear()
  const [open, setOpen] = React.useState(false)
  const [date, setDate] = React.useState<Date | undefined>(value)
  const [month, setMonth] = React.useState<Date | undefined>(value)
  // The time is the text of a 24-hour field in the interface's digits.
  const [time, setTime] = React.useState<string>(() =>
    formatDisplayDate(dayjs().startOf('day').toDate(), 'HH:mm', locale)
  )

  React.useEffect(() => {
    setDate(value)
    setMonth(value)
    if (value) {
      setTime(formatDisplayDate(value, 'HH:mm', locale))
    }
  }, [value, locale])

  const handleDateSelect = (selectedDate: Date | undefined) => {
    if (selectedDate) {
      const { hours, minutes } = parseTimeOfDay(time) ?? {
        hours: 0,
        minutes: 0,
      }
      const newDate = new Date(selectedDate)
      newDate.setHours(hours, minutes, 0, 0)
      setDate(newDate)
      setMonth(newDate)
      onChange?.(newDate)
      setOpen(false)
    } else {
      setDate(undefined)
      setMonth(undefined)
      onChange?.(undefined)
    }
  }

  const handleTimeChange = (newTime: string) => {
    setTime(newTime)

    const parsed = parseTimeOfDay(newTime)
    if (!date || !parsed) return
    if (
      date.getHours() === parsed.hours &&
      date.getMinutes() === parsed.minutes
    ) {
      return
    }
    const newDate = new Date(date)
    newDate.setHours(parsed.hours, parsed.minutes, 0, 0)
    setDate(newDate)
    onChange?.(newDate)
  }

  const handleClear = () => {
    setDate(undefined)
    setMonth(undefined)
    setTime(formatDisplayDate(dayjs().startOf('day').toDate(), 'HH:mm', locale))
    onChange?.(undefined)
  }

  return (
    <div className={cn('flex gap-2', className)}>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger
          render={
            <Button
              variant='outline'
              className={cn(
                'flex-1 justify-between font-normal',
                !date && 'text-muted-foreground'
              )}
            />
          }
        >
          {date
            ? formatDisplayDate(date, 'YYYY-MM-DD', locale)
            : placeholderText}
          <ChevronDownIcon className='h-4 w-4 opacity-50' />
        </PopoverTrigger>
        <PopoverContent className='w-auto overflow-hidden p-0' align='start'>
          <Calendar
            mode='single'
            selected={date}
            month={month}
            onMonthChange={setMonth}
            captionLayout='dropdown'
            onSelect={handleDateSelect}
            locale={calendarLocale}
            startMonth={new Date(currentYear - 100, 0)}
            endMonth={new Date(currentYear + 100, 11)}
          />
        </PopoverContent>
      </Popover>
      <TimeOfDayInput
        value={time}
        onValueChange={handleTimeChange}
        aria-label={t('Time')}
        className='w-20'
        disabled={!date}
      />
      {date && (
        <Button
          type='button'
          variant='outline'
          size='icon'
          onClick={handleClear}
          className='shrink-0'
          aria-label='Clear'
        >
          <span aria-hidden='true'>✕</span>
        </Button>
      )}
    </div>
  )
}
