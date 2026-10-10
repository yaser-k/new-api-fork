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
import { CalendarDays } from 'lucide-react'
import { useId, useMemo, useState } from 'react'
import type { DateRange } from 'react-day-picker'
import { useTranslation } from 'react-i18next'

import { TimeOfDayInput } from '@/components/time-of-day-input'
import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import { Label } from '@/components/ui/label'
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

interface CompactDateTimeRangePickerProps {
  start?: Date
  end?: Date
  onChange: (range: { start?: Date; end?: Date }) => void
  className?: string
}

export function CompactDateTimeRangePicker({
  start,
  end,
  onChange,
  className,
}: CompactDateTimeRangePickerProps) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const startTimeId = useId()
  const endTimeId = useId()
  const [open, setOpen] = useState(false)
  // The draft keeps the calendar's days and the typed times apart: the
  // calendar follows the interface language (Solar Hijri in Persian) and
  // returns Gregorian Date values like every other picker.
  const [draftRange, setDraftRange] = useState<DateRange | undefined>()
  const [draftStartTime, setDraftStartTime] = useState('')
  const [draftEndTime, setDraftEndTime] = useState('')
  const [month, setMonth] = useState<Date>(() => start ?? new Date())

  const parsedStartTime = parseTimeOfDay(draftStartTime)
  const parsedEndTime = parseTimeOfDay(draftEndTime)

  const label = useMemo(() => {
    if (!start && !end) return t('Date Range')
    // The picker has minute precision, so seconds are always 00 for a start
    // and 59 for an end. Hide them in the trigger label to keep the button
    // width compact while still showing the meaningful timestamp.
    // Solar Hijri in Persian; the Date values stay Gregorian.
    const startText = start
      ? formatDisplayDate(start, 'YYYY-MM-DD HH:mm', locale)
      : '-'
    const endText = end
      ? formatDisplayDate(end, 'YYYY-MM-DD HH:mm', locale)
      : '-'
    return `${startText} ~ ${endText}`
  }, [end, start, t, locale])

  const mobileLabel = useMemo(() => {
    if (!start || !end) return label
    if (dayjs(start).isSame(end, 'day')) {
      return `${formatDisplayDate(start, 'MM/DD HH:mm', locale)}–${formatDisplayDate(end, 'HH:mm', locale)}`
    }
    return label
  }, [start, end, label, locale])

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      setDraftRange(start || end ? { from: start ?? end, to: end } : undefined)
      // The times show the interface's digits; parseTimeOfDay reads them back.
      const today = dayjs()
      setDraftStartTime(
        formatDisplayDate(start ?? today.startOf('day').toDate(), 'HH:mm', locale)
      )
      setDraftEndTime(
        formatDisplayDate(end ?? today.endOf('day').toDate(), 'HH:mm', locale)
      )
      setMonth(start ?? end ?? new Date())
    }
    setOpen(nextOpen)
  }

  const applyDraft = () => {
    if (!parsedStartTime || !parsedEndTime) return
    const fromDay = draftRange?.from
    const toDay = draftRange?.to ?? fromDay
    let nextStart = fromDay
      ? dayjs(fromDay)
          .hour(parsedStartTime.hours)
          .minute(parsedStartTime.minutes)
          .startOf('minute')
          .toDate()
      : undefined
    // An end time includes its whole minute, as the presets' end of day does.
    let nextEnd = toDay
      ? dayjs(toDay)
          .hour(parsedEndTime.hours)
          .minute(parsedEndTime.minutes)
          .endOf('minute')
          .toDate()
      : undefined
    // A side left as it was keeps its exact value.
    if (start && nextStart && dayjs(start).isSame(nextStart, 'minute')) {
      nextStart = start
    }
    if (end && nextEnd && dayjs(end).isSame(nextEnd, 'minute')) {
      nextEnd = end
    }
    onChange({ start: nextStart, end: nextEnd })
    setOpen(false)
  }

  const applyPreset = (kind: 'today' | '7d' | 'week' | '30d' | 'month') => {
    const now = dayjs()
    const presets = {
      today: {
        start: now.startOf('day').toDate(),
        end: now.endOf('day').toDate(),
      },
      '7d': {
        start: now.subtract(6, 'day').startOf('day').toDate(),
        end: now.endOf('day').toDate(),
      },
      week: {
        start: now.startOf('week').toDate(),
        end: now.endOf('week').toDate(),
      },
      '30d': {
        start: now.subtract(29, 'day').startOf('day').toDate(),
        end: now.endOf('day').toDate(),
      },
      month: {
        start: now.startOf('month').toDate(),
        end: now.endOf('month').toDate(),
      },
    }
    onChange(presets[kind])
    setOpen(false)
  }

  const draftEndDay = draftRange?.to ?? draftRange?.from

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger
        render={
          <Button
            type='button'
            variant='outline'
            aria-label={label}
            className={cn(
              'w-full justify-start gap-2 px-2.5 text-sm leading-5 font-normal tabular-nums',
              !start && !end && 'text-muted-foreground',
              className
            )}
          />
        }
      >
        <CalendarDays className='text-muted-foreground size-4 shrink-0' />
        {/* dir='auto' keeps a digits-only range in reading order inside RTL. */}
        <span dir='auto' className='hidden truncate sm:block'>
          {label}
        </span>
        <span
          dir='auto'
          className='min-w-0 [overflow-wrap:anywhere] whitespace-normal sm:hidden'
        >
          {mobileLabel}
        </span>
      </PopoverTrigger>
      <PopoverContent
        align='start'
        className='w-auto max-w-[calc(100vw-2rem)] p-3'
      >
        <div className='flex flex-col gap-3 sm:flex-row'>
          <Calendar
            mode='range'
            selected={draftRange}
            onSelect={setDraftRange}
            resetOnSelect
            month={month}
            onMonthChange={setMonth}
            className='self-center p-0 sm:self-start'
          />

          <div className='flex flex-col gap-3 sm:w-44'>
            <div className='grid gap-2'>
              <div className='space-y-1.5'>
                <Label
                  htmlFor={startTimeId}
                  className='text-muted-foreground text-xs font-normal'
                >
                  {t('Start Time')}
                </Label>
                <div className='flex items-center justify-between gap-2'>
                  <span
                    dir='auto'
                    className='min-w-0 truncate text-sm tabular-nums'
                  >
                    {draftRange?.from
                      ? formatDisplayDate(draftRange.from, 'YYYY-MM-DD', locale)
                      : '-'}
                  </span>
                  <TimeOfDayInput
                    id={startTimeId}
                    value={draftStartTime}
                    onValueChange={setDraftStartTime}
                    placeholder='00:00'
                    className='h-8 w-18 shrink-0 px-2 text-sm leading-5'
                  />
                </div>
              </div>
              <div className='space-y-1.5'>
                <Label
                  htmlFor={endTimeId}
                  className='text-muted-foreground text-xs font-normal'
                >
                  {t('End Time')}
                </Label>
                <div className='flex items-center justify-between gap-2'>
                  <span
                    dir='auto'
                    className='min-w-0 truncate text-sm tabular-nums'
                  >
                    {draftEndDay
                      ? formatDisplayDate(draftEndDay, 'YYYY-MM-DD', locale)
                      : '-'}
                  </span>
                  <TimeOfDayInput
                    id={endTimeId}
                    value={draftEndTime}
                    onValueChange={setDraftEndTime}
                    placeholder='23:59'
                    className='h-8 w-18 shrink-0 px-2 text-sm leading-5'
                  />
                </div>
              </div>
            </div>

            <div className='flex flex-wrap gap-1.5'>
              <Button
                type='button'
                variant='secondary'
                size='sm'
                className='h-7 flex-1 px-2 text-xs'
                onClick={() => applyPreset('today')}
              >
                {t('Today')}
              </Button>
              <Button
                type='button'
                variant='secondary'
                size='sm'
                className='h-7 flex-1 px-2 text-xs'
                onClick={() => applyPreset('7d')}
              >
                {t('7 Days')}
              </Button>
              <Button
                type='button'
                variant='secondary'
                size='sm'
                className='h-7 flex-1 px-2 text-xs'
                onClick={() => applyPreset('week')}
              >
                {t('This week')}
              </Button>
              <Button
                type='button'
                variant='secondary'
                size='sm'
                className='h-7 flex-1 px-2 text-xs'
                onClick={() => applyPreset('30d')}
              >
                {t('30 Days')}
              </Button>
              <Button
                type='button'
                variant='secondary'
                size='sm'
                className='h-7 flex-1 px-2 text-xs'
                onClick={() => applyPreset('month')}
              >
                {t('Current month')}
              </Button>
            </div>

            <div className='mt-auto flex justify-end'>
              <Button
                size='sm'
                className='h-8'
                disabled={!parsedStartTime || !parsedEndTime}
                onClick={applyDraft}
              >
                {t('Confirm')}
              </Button>
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}
