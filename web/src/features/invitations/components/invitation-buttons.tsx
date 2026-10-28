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
import { List, Mail } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { useStatus } from '@/hooks/use-status'

import { AccountInvitationsDialog } from './account-invitations-dialog'
import { InviteByEmailDialog } from './invite-by-email-dialog'

/** Users page actions for account invitations, shown while they are enabled. */
export function InvitationButtons() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const [open, setOpen] = useState<'invite' | 'list' | null>(null)

  if (status?.invitation_register_enabled !== true) return null

  return (
    <>
      <Button size='sm' variant='outline' onClick={() => setOpen('list')}>
        <List className='h-4 w-4' />
        {t('Account invitations')}
      </Button>
      <Button size='sm' variant='outline' onClick={() => setOpen('invite')}>
        <Mail className='h-4 w-4' />
        {t('Invite by email')}
      </Button>
      <InviteByEmailDialog
        open={open === 'invite'}
        onOpenChange={(isOpen) => setOpen(isOpen ? 'invite' : null)}
      />
      <AccountInvitationsDialog
        open={open === 'list'}
        onOpenChange={(isOpen) => setOpen(isOpen ? 'list' : null)}
      />
    </>
  )
}
