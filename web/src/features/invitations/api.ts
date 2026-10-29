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
import { api } from '@/lib/api'

import type {
  ApiResponse,
  Invitation,
  InvitationCheck,
  InvitationPage,
  SentInvitation,
} from './types'

export const INVITATIONS_QUERY_KEY = ['account-invitations'] as const

/** Invite an address; the mail goes out in `language`. */
export async function createInvitation(payload: {
  email: string
  language: string
}): Promise<ApiResponse<SentInvitation>> {
  const res = await api.post('/api/user/invitation', payload)
  return res.data
}

/** Newest invitations first. */
export async function getInvitations(params: {
  p: number
  page_size: number
}): Promise<ApiResponse<InvitationPage>> {
  const res = await api.get('/api/user/invitation', { params })
  return res.data
}

/** Mail a new link; the previous one stops working. */
export async function resendInvitation(
  id: number
): Promise<ApiResponse<SentInvitation>> {
  const res = await api.post(`/api/user/invitation/${id}/resend`)
  return res.data
}

export async function revokeInvitation(
  id: number
): Promise<ApiResponse<Invitation>> {
  const res = await api.post(`/api/user/invitation/${id}/revoke`)
  return res.data
}

/** Look up the pending invitation a sign-up link opens. */
export async function checkInvitation(
  token: string
): Promise<ApiResponse<InvitationCheck>> {
  const res = await api.get('/api/invitation/check', { params: { token } })
  return res.data
}
