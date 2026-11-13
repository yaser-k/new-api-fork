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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { useId, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  INTERFACE_LANGUAGE_OPTIONS,
  normalizeInterfaceLanguage,
} from '@/i18n/languages'
import { handleServerError } from '@/lib/handle-server-error'
import { createServerError } from '@/lib/server-error-message'

import { createInvitation, INVITATIONS_QUERY_KEY } from '../api'
import { inviteFormSchema, type InviteFormValues } from '../lib/invite-form'
import { InvitationLink } from './invitation-link'

const LANGUAGE_ITEMS = INTERFACE_LANGUAGE_OPTIONS.map((language) => ({
  value: language.code,
  label: language.label,
}))

export function InviteByEmailDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t, i18n } = useTranslation()
  const queryClient = useQueryClient()
  const formId = useId()
  const [link, setLink] = useState<string | null>(null)
  const defaultValues = {
    email: '',
    language: normalizeInterfaceLanguage(
      i18n.resolvedLanguage || i18n.language
    ),
  }
  const form = useForm<InviteFormValues>({
    resolver: zodResolver(inviteFormSchema),
    defaultValues,
  })

  const invite = useMutation({
    mutationFn: async (values: InviteFormValues) => {
      const result = await createInvitation(values)
      if (!result.success || !result.data) throw createServerError(result)
      return result.data
    },
    onSuccess: (sent) => {
      setLink(sent.link)
      toast.success(t('Invitation sent'))
    },
    onError: (error) => {
      handleServerError(error, t('Failed to send the invitation'))
    },
    // A failed mail still leaves a pending invitation that can be resent.
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: INVITATIONS_QUERY_KEY }),
  })

  const handleOpenChange = (open: boolean) => {
    if (!open) {
      setLink(null)
      form.reset(defaultValues)
      invite.reset()
    }
    props.onOpenChange(open)
  }

  const footer = link ? (
    <Button onClick={() => handleOpenChange(false)}>{t('Close')}</Button>
  ) : (
    <>
      <Button
        variant='outline'
        onClick={() => handleOpenChange(false)}
        disabled={invite.isPending}
      >
        {t('Cancel')}
      </Button>
      <Button type='submit' form={formId} disabled={invite.isPending}>
        {invite.isPending ? <Loader2 className='h-4 w-4 animate-spin' /> : null}
        {t('Send')}
      </Button>
    </>
  )

  return (
    <Dialog
      open={props.open}
      onOpenChange={handleOpenChange}
      title={t('Invite by email')}
      description={t(
        'The person gets a single-use link that opens the sign-up form with this email address.'
      )}
      contentClassName='sm:max-w-md'
      contentHeight='auto'
      footer={footer}
    >
      {link ? (
        <InvitationLink link={link} />
      ) : (
        <Form {...form}>
          <form
            id={formId}
            onSubmit={form.handleSubmit((values) => invite.mutate(values))}
            className='grid gap-4'
          >
            <FormField
              control={form.control}
              name='email'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Email')}</FormLabel>
                  <FormControl>
                    <Input
                      type='email'
                      dir='ltr'
                      autoComplete='off'
                      placeholder={t('name@example.com')}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='language'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Email language')}</FormLabel>
                  <Select
                    items={LANGUAGE_ITEMS}
                    value={field.value}
                    onValueChange={(value) =>
                      value !== null && field.onChange(value)
                    }
                  >
                    <FormControl>
                      <SelectTrigger className='w-full'>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent alignItemWithTrigger={false}>
                      <SelectGroup>
                        {LANGUAGE_ITEMS.map((language) => (
                          <SelectItem
                            key={language.value}
                            value={language.value}
                          >
                            {language.label}
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
          </form>
        </Form>
      )}
    </Dialog>
  )
}
