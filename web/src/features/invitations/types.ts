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
export type InvitationStatus = 'pending' | 'used' | 'expired' | 'revoked'

/** An account invitation as the admin list returns it; never its token. */
export interface Invitation {
  id: number
  email: string
  language: string
  created_by: number
  created_time: number
  expired_time: number
  used_time: number
  used_user_id: number
  revoked_time: number
  status: InvitationStatus
}

/** Create and resend answer with the mailed link, shown this once. */
export interface SentInvitation {
  invitation: Invitation
  link: string
}

export interface InvitationCheck {
  email: string
  expired_time: number
}

export interface InvitationPage {
  page: number
  page_size: number
  total: number
  items: Invitation[] | null
}

export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}
