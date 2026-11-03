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
import { useId } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

/** The mailed sign-up link, shown once after the invitation is sent. */
export function InvitationLink(props: { link: string }) {
  const { t } = useTranslation()
  const inputId = useId()
  return (
    <div className='space-y-2' role='status'>
      <p className='text-sm font-medium'>{t('Invitation sent')}</p>
      <p className='text-muted-foreground text-sm'>
        {t('Copy this link now. It will not be shown again.')}
      </p>
      <Label htmlFor={inputId}>{t('Invitation link')}</Label>
      <div className='flex gap-2'>
        <Input
          id={inputId}
          value={props.link}
          readOnly
          autoComplete='off'
          dir='ltr'
          className='font-mono text-xs'
        />
        <CopyButton
          value={props.link}
          variant='outline'
          tooltip={t('Copy invitation link')}
          aria-label={t('Copy invitation link')}
        />
      </div>
    </div>
  )
}
