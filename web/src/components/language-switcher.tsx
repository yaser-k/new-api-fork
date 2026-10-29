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
import { Languages, Check } from 'lucide-react'
import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useStatus } from '@/hooks/use-status'
import {
  getOfferedInterfaceLanguages,
  readInterfaceLanguageSettings,
  rememberChosenInterfaceLanguage,
} from '@/i18n/interface-language-policy'
import { normalizeInterfaceLanguage } from '@/i18n/languages'
import { api } from '@/lib/api'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

/**
 * The letter that stands for each interface language's script on the language
 * button. Languages written in Latin share `A`.
 */
const LANGUAGE_SCRIPT_MARKS: Record<string, string> = {
  en: 'A',
  fr: 'A',
  vi: 'A',
  ru: 'Я',
  ja: 'あ',
  zhCN: '文',
  zhTW: '文',
  fa: 'ف',
}

/**
 * The language button's picture for a two-language menu: the other script's
 * mark at the top left and lucide's `A` at the bottom right, in the layout of
 * lucide's `Languages`.
 */
function LanguagePairIcon(props: { mark: string; className?: string }) {
  return (
    <svg
      xmlns='http://www.w3.org/2000/svg'
      viewBox='0 0 24 24'
      fill='none'
      stroke='currentColor'
      strokeWidth={2}
      strokeLinecap='round'
      strokeLinejoin='round'
      aria-hidden='true'
      data-mark={props.mark}
      className={props.className}
    >
      <text
        x='7'
        y='12'
        textAnchor='middle'
        fontSize='15'
        fontWeight='700'
        fill='currentColor'
        stroke='none'
      >
        {props.mark}
      </text>
      <path d='m22 22-5-10-5 10' />
      <path d='M14 18h6' />
    </svg>
  )
}

export function LanguageSwitcher() {
  const { i18n, t } = useTranslation()
  const user = useAuthStore((s) => s.auth.user)
  const { status } = useStatus()
  const offeredLanguages = getOfferedInterfaceLanguages(
    readInterfaceLanguageSettings(status as Record<string, unknown> | null)
  )
  const currentLanguage = normalizeInterfaceLanguage(i18n.language)
  // A menu of exactly two languages, one of them written in Latin, shows the
  // other one's script beside the A; any other menu keeps lucide's icon.
  const offeredMarks = offeredLanguages.map(
    (lang) => LANGUAGE_SCRIPT_MARKS[lang.code]
  )
  const pairMark =
    offeredMarks.length === 2 && offeredMarks.includes('A')
      ? offeredMarks.find((mark) => mark && mark !== 'A')
      : undefined
  const handleChangeLanguage = useCallback(
    async (code: string) => {
      rememberChosenInterfaceLanguage(code)
      await i18n.changeLanguage(code)
      if (user) {
        try {
          await api.put('/api/user/self', { language: code })
        } catch {
          // Best-effort persistence; don't block the UI on failure
        }
      }
    },
    [i18n, user]
  )

  return (
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger
        render={<Button variant='ghost' size='icon' className='h-9 w-9' />}
      >
        {pairMark ? (
          <LanguagePairIcon mark={pairMark} className='size-[1.2rem]' />
        ) : (
          <Languages className='size-[1.2rem]' />
        )}
        <span className='sr-only'>{t('Change language')}</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='end'>
        {offeredLanguages.map((lang) => (
          <DropdownMenuItem
            key={lang.code}
            onClick={() => handleChangeLanguage(lang.code)}
          >
            {lang.label}
            <Check
              size={14}
              className={cn(
                'ms-auto',
                currentLanguage !== lang.code && 'hidden'
              )}
            />
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
