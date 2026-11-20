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
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  saveDisplayCurrency,
  useDisplayCurrencyOptions,
} from '@/hooks/use-display-currency'
import { parseDisplayCurrency } from '@/stores/display-currency-store'

export function DisplayCurrencySwitcher() {
  const { t } = useTranslation()
  const options = useDisplayCurrencyOptions()
  if (!options) return null

  return (
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger
        render={
          <Button
            variant='ghost'
            size='icon'
            className='h-9 w-auto min-w-9 px-2'
          />
        }
      >
        <span aria-hidden='true' className='text-sm font-semibold'>
          {options.symbol}
        </span>
        <span className='sr-only'>{t('Select currency')}</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='end'>
        <DropdownMenuRadioGroup
          value={options.value}
          onValueChange={(next) => {
            const displayCurrency = parseDisplayCurrency(next)
            if (!displayCurrency) return
            // Best effort, as for the language: the browser keeps the choice
            // even when the account cannot be reached.
            saveDisplayCurrency(displayCurrency).catch(() => undefined)
          }}
        >
          {options.items.map((item) => (
            <DropdownMenuRadioItem
              closeOnClick
              key={item.value}
              value={item.value}
            >
              {item.label}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
