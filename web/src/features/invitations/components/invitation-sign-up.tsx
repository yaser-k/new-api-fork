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
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { SignUpForm } from '@/features/auth/sign-up/components/sign-up-form'

import { checkInvitation } from '../api'

/**
 * The sign-up opened by an invitation link: the form for the invited address,
 * or one message for every token that opens no pending invitation.
 */
export function InvitationSignUp(props: { token: string }) {
  const { t } = useTranslation()
  const check = useQuery({
    queryKey: ['invitation-check', props.token],
    queryFn: () => checkInvitation(props.token),
    retry: false,
    staleTime: Infinity,
    meta: { errorToast: false },
  })

  if (check.isLoading) return <LoadingState />

  const email = check.data?.success ? check.data.data?.email : undefined
  if (!email) {
    return (
      <div className='space-y-4' role='alert'>
        <p className='text-sm'>
          {t('This invitation is invalid or has expired')}
        </p>
        <Button className='w-full' render={<Link to='/sign-in' />}>
          {t('Sign in')}
        </Button>
      </div>
    )
  }

  return <SignUpForm invitation={{ token: props.token, email }} />
}
