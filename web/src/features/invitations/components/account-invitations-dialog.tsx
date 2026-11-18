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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { TFunction } from 'i18next'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { StaticDataTable } from '@/components/data-table'
import { Dialog } from '@/components/dialog'
import { LoadingState } from '@/components/loading-state'
import { StatusBadge, type StatusVariant } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { formatTimestampToDate } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import {
  createServerError,
  requireServerSuccess,
} from '@/lib/server-error-message'

import {
  getInvitations,
  INVITATIONS_QUERY_KEY,
  resendInvitation,
  revokeInvitation,
} from '../api'
import type { Invitation, InvitationStatus } from '../types'
import { InvitationLink } from './invitation-link'

const PAGE_SIZE = 10

const STATUS_VARIANTS: Record<InvitationStatus, StatusVariant> = {
  pending: 'info',
  used: 'success',
  expired: 'warning',
  revoked: 'neutral',
}

function invitationStatusLabel(status: InvitationStatus, t: TFunction) {
  switch (status) {
    case 'pending':
      return t('Pending', { context: 'invitation' })
    case 'used':
      return t('Used', { context: 'invitation' })
    case 'expired':
      return t('Expired', { context: 'invitation' })
    case 'revoked':
      return t('Revoked', { context: 'invitation' })
  }
}

export function AccountInvitationsDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [sentLink, setSentLink] = useState<string | null>(null)
  const [revokeTarget, setRevokeTarget] = useState<Invitation | null>(null)

  const invitations = useQuery({
    queryKey: [...INVITATIONS_QUERY_KEY, page],
    queryFn: async () =>
      requireServerSuccess(
        await getInvitations({ p: page, page_size: PAGE_SIZE })
      ).data,
    enabled: props.open,
  })
  const items = invitations.data?.items ?? []
  const totalPages = Math.max(
    1,
    Math.ceil((invitations.data?.total ?? 0) / PAGE_SIZE)
  )

  const refresh = () =>
    queryClient.invalidateQueries({ queryKey: INVITATIONS_QUERY_KEY })

  const resend = useMutation({
    mutationFn: async (invitation: Invitation) => {
      const result = await resendInvitation(invitation.id)
      if (!result.success || !result.data) throw createServerError(result)
      return result.data
    },
    onSuccess: (sent) => {
      setSentLink(sent.link)
      toast.success(t('Invitation sent'))
    },
    onError: (error) => {
      handleServerError(error, t('Failed to send the invitation'))
    },
    onSettled: refresh,
  })

  const revoke = useMutation({
    mutationFn: async (invitation: Invitation) => {
      const result = await revokeInvitation(invitation.id)
      if (!result.success) throw createServerError(result)
    },
    onSuccess: () => {
      setRevokeTarget(null)
      toast.success(t('Invitation revoked'))
    },
    onError: (error) => {
      handleServerError(error, t('Failed to revoke the invitation'))
    },
    onSettled: refresh,
  })

  const handleOpenChange = (open: boolean) => {
    if (!open) setSentLink(null)
    props.onOpenChange(open)
  }

  // Used and revoked invitations are final; the others can be sent again.
  const canResend = (invitation: Invitation) =>
    invitation.status === 'pending' || invitation.status === 'expired'

  return (
    <>
      <Dialog
        open={props.open}
        onOpenChange={handleOpenChange}
        title={t('Account invitations')}
        contentClassName='sm:max-w-4xl'
        contentHeight='auto'
        bodyClassName='space-y-4'
        footer={
          <Button onClick={() => handleOpenChange(false)}>{t('Close')}</Button>
        }
      >
        {sentLink && <InvitationLink link={sentLink} />}
        {invitations.isLoading && <LoadingState />}
        {!invitations.isLoading && items.length === 0 && (
          <div className='text-muted-foreground py-12 text-center'>
            {t('No invitations yet')}
          </div>
        )}
        {!invitations.isLoading && items.length > 0 && (
          <StaticDataTable
            tableClassName='min-w-[720px]'
            data={items}
            getRowKey={(invitation) => invitation.id}
            columns={[
              {
                id: 'email',
                header: t('Email'),
                cellClassName: 'max-w-56 truncate',
                cell: (invitation) => <span dir='ltr'>{invitation.email}</span>,
              },
              {
                id: 'status',
                header: t('Status'),
                className: 'w-28',
                cell: (invitation) => (
                  <StatusBadge
                    label={invitationStatusLabel(invitation.status, t)}
                    variant={STATUS_VARIANTS[invitation.status]}
                    copyable={false}
                  />
                ),
              },
              {
                id: 'created',
                header: t('Created'),
                className: 'w-44',
                cellClassName: 'text-muted-foreground text-sm',
                cell: (invitation) =>
                  formatTimestampToDate(invitation.created_time),
              },
              {
                id: 'expires',
                header: t('Expires'),
                className: 'w-44',
                cellClassName: 'text-muted-foreground text-sm',
                cell: (invitation) =>
                  formatTimestampToDate(invitation.expired_time),
              },
              {
                id: 'used-by',
                header: t('Used by'),
                className: 'w-24',
                cellClassName: 'font-mono text-sm',
                cell: (invitation) =>
                  invitation.used_user_id ? `#${invitation.used_user_id}` : '-',
              },
              {
                id: 'actions',
                header: t('Actions'),
                className: 'text-end',
                cellClassName: 'text-end',
                cell: (invitation) =>
                  canResend(invitation) ? (
                    <div className='flex justify-end gap-1'>
                      <Button
                        variant='outline'
                        size='sm'
                        disabled={resend.isPending}
                        onClick={() => resend.mutate(invitation)}
                      >
                        {t('Resend')}
                      </Button>
                      <Button
                        variant='ghost'
                        size='sm'
                        className='text-destructive'
                        onClick={() => setRevokeTarget(invitation)}
                      >
                        {t('Revoke')}
                      </Button>
                    </div>
                  ) : null,
              },
            ]}
          />
        )}
        {totalPages > 1 && (
          <div className='flex items-center justify-between'>
            <div className='text-muted-foreground text-sm'>
              {t('Page {{current}} of {{total}}', {
                current: page,
                total: totalPages,
              })}
            </div>
            <div className='flex gap-2'>
              <Button
                variant='outline'
                size='sm'
                onClick={() => setPage(page - 1)}
                disabled={page === 1 || invitations.isFetching}
              >
                {t('Previous')}
              </Button>
              <Button
                variant='outline'
                size='sm'
                onClick={() => setPage(page + 1)}
                disabled={page >= totalPages || invitations.isFetching}
              >
                {t('Next')}
              </Button>
            </div>
          </div>
        )}
      </Dialog>

      <ConfirmDialog
        open={revokeTarget !== null}
        onOpenChange={(isOpen) => !isOpen && setRevokeTarget(null)}
        title={t('Revoke this invitation?')}
        desc={t(
          'The link sent to {{email}} stops working. The invitation stays in the list.',
          { email: revokeTarget?.email ?? '' }
        )}
        confirmText={t('Revoke')}
        destructive
        isLoading={revoke.isPending}
        handleConfirm={() => revokeTarget && revoke.mutate(revokeTarget)}
      />
    </>
  )
}
