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
import { Loader2 } from 'lucide-react'
import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  saveDisplayCurrency,
  useDisplayCurrencyOptions,
} from '@/hooks/use-display-currency'
import { handleServerError } from '@/lib/handle-server-error'
import {
  parseDisplayCurrency,
  useDisplayCurrencyStore,
} from '@/stores/display-currency-store'

/** The currency preference, shown beside the interface language. */
export function DisplayCurrencyPreference(props: {
  onProfileUpdate: () => void
}) {
  const { t } = useTranslation()
  const id = useId()
  const options = useDisplayCurrencyOptions()
  const [saving, setSaving] = useState(false)
  if (!options) return null

  const handleChange = async (next: unknown) => {
    const displayCurrency = parseDisplayCurrency(next)
    if (!displayCurrency || displayCurrency === options.value) return
    const previous = options.value
    setSaving(true)
    try {
      await saveDisplayCurrency(displayCurrency)
      props.onProfileUpdate()
      toast.success(t('Currency preference saved'))
    } catch (error) {
      // The account still holds the previous choice.
      useDisplayCurrencyStore.getState().setDisplayCurrency(previous)
      handleServerError(error, t('Failed to update settings'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className='mt-4 flex flex-col gap-3 border-t pt-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4'>
      <div className='space-y-1'>
        <label htmlFor={id} className='text-sm font-medium'>
          {t('Currency')}
        </label>
        <p className='text-muted-foreground line-clamp-2 text-xs sm:text-sm'>
          {t(
            'Show balances and prices in the site currency or in US dollars. Payments are still charged in the site currency.'
          )}
        </p>
      </div>
      <div className='flex items-center gap-2 sm:min-w-48'>
        <Select
          items={options.items}
          value={options.value}
          onValueChange={handleChange}
          disabled={saving}
        >
          <SelectTrigger id={id} className='w-full sm:w-48'>
            <SelectValue />
          </SelectTrigger>
          <SelectContent alignItemWithTrigger={false}>
            {options.items.map((item) => (
              <SelectItem key={item.value} value={item.value}>
                {item.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {saving && (
          <Loader2 className='text-muted-foreground size-4 animate-spin' />
        )}
      </div>
    </div>
  )
}
