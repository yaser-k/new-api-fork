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
import { useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'
import { safeNumberFieldProps } from '../utils/numeric-field'

const invitationHoursMessage =
  'Invitation validity must be between 1 and 720 hours'

const basicAuthSchema = z.object({
  PasswordLoginEnabled: z.boolean(),
  PasswordRegisterEnabled: z.boolean(),
  EmailVerificationEnabled: z.boolean(),
  RegisterEnabled: z.boolean(),
  InvitationRegisterEnabled: z.boolean(),
  InvitationValidHours: z
    .number()
    .int(invitationHoursMessage)
    .min(1, invitationHoursMessage)
    .max(720, invitationHoursMessage),
  EmailDomainRestrictionEnabled: z.boolean(),
  EmailAliasRestrictionEnabled: z.boolean(),
  EmailDomainWhitelist: z.string(),
  // The server checks these too; it alone can check the pattern.
  RegisterPhoneRequiredEnabled: z.boolean(),
  PhoneCountryCode: z
    .string()
    .trim()
    .regex(
      /^([1-9]\d{0,2})?$/,
      'The country calling code must be one to three digits that do not start with 0'
    ),
  PhoneTrunkPrefix: z
    .string()
    .trim()
    .regex(/^(\d{1,3})?$/, 'The trunk prefix must be one to three digits'),
  PhoneNationalNumberPattern: z.string().trim(),
  RegisterPhoneHint: z
    .string()
    .trim()
    .max(200, 'The phone number hint must be at most 200 characters'),
})

type BasicAuthFormValues = z.infer<typeof basicAuthSchema>

type BasicAuthSectionProps = {
  defaultValues: BasicAuthFormValues
}

export function BasicAuthSection({ defaultValues }: BasicAuthSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const formDefaults = useMemo<BasicAuthFormValues>(
    () => ({
      ...defaultValues,
      EmailDomainWhitelist: defaultValues.EmailDomainWhitelist.split(',')
        .map((domain) => domain.trim())
        .filter(Boolean)
        .join('\n'),
    }),
    [defaultValues]
  )

  const form = useForm<BasicAuthFormValues>({
    resolver: zodResolver(basicAuthSchema),
    defaultValues: formDefaults,
  })

  useResetForm(form, formDefaults)

  const onSubmit = async (data: BasicAuthFormValues) => {
    const updates: Array<{ key: string; value: string | boolean | number }> = []

    Object.entries(data).forEach(([key, value]) => {
      if (key === 'EmailDomainWhitelist') {
        if (typeof value !== 'string') return
        const domains = value
          .split('\n')
          .map((domain) => domain.trim())
          .filter(Boolean)
          .join(',')
        if (domains !== defaultValues.EmailDomainWhitelist) {
          updates.push({ key, value: domains })
        }
      } else if (value !== defaultValues[key as keyof typeof defaultValues]) {
        updates.push({ key, value })
      }
    })

    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }
  }

  return (
    <SettingsSection title={t('Basic Authentication')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />
          <FormField
            control={form.control}
            name='PasswordLoginEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Password Login')}</FormLabel>
                  <FormDescription>
                    {t('Allow users to log in with password')}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='RegisterEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Registration Enabled')}</FormLabel>
                  <FormDescription>
                    {t('Allow new users to register')}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='PasswordRegisterEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Password Registration')}</FormLabel>
                  <FormDescription>
                    {t('Allow registration with password')}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='InvitationRegisterEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Registration by email invitation')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Administrators can invite an email address from the users page. The mailed link opens the sign-up form for that address, even when registration is closed.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='InvitationValidHours'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Invitation validity (hours)')}</FormLabel>
                <FormControl>
                  <Input
                    type='number'
                    min={1}
                    max={720}
                    step={1}
                    {...safeNumberFieldProps(field)}
                  />
                </FormControl>
                <FormDescription>
                  {t('How long an invitation link works, from 1 to 720 hours.')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='EmailVerificationEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Email Verification')}</FormLabel>
                  <FormDescription>
                    {t('Require email verification for new accounts')}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='RegisterPhoneRequiredEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>
                    {t('Require a phone number at registration')}
                  </FormLabel>
                  <FormDescription>
                    {t(
                      'Sign-up asks for a phone number that no other account holds. While this is on, third-party sign-ins cannot create an account; accounts already linked still sign in with them. Existing accounts without a number keep working. No message is sent to the number.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='PhoneCountryCode'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Country calling code')}</FormLabel>
                <FormControl>
                  <Input dir='ltr' inputMode='numeric' {...field} />
                </FormControl>
                <FormDescription>
                  {t(
                    'Digits without the plus sign. Leave empty to accept any number written with + and its country code.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='PhoneTrunkPrefix'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Trunk prefix')}</FormLabel>
                <FormControl>
                  <Input dir='ltr' inputMode='numeric' {...field} />
                </FormControl>
                <FormDescription>
                  {t(
                    'Written before a national number when it is dialed without the country code. Used only with a country code.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='PhoneNationalNumberPattern'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('National number pattern')}</FormLabel>
                <FormControl>
                  <Input dir='ltr' className='font-mono' {...field} />
                </FormControl>
                <FormDescription>
                  {t(
                    'A regular expression the whole national number must match, without the country code or the trunk prefix. Used only with a country code.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='RegisterPhoneHint'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Phone number hint')}</FormLabel>
                <FormControl>
                  <Input maxLength={200} {...field} />
                </FormControl>
                <FormDescription>
                  {t(
                    'Shown under the phone field on the sign-up page, as plain text. Leave empty to show none.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='EmailDomainRestrictionEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Email Domain Restriction')}</FormLabel>
                  <FormDescription>
                    {t('Only allow specific email domains')}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='EmailAliasRestrictionEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Email Alias Restriction')}</FormLabel>
                  <FormDescription>
                    {t('Block email aliases (e.g., user+alias@domain.com)')}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='EmailDomainWhitelist'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Email Domain Whitelist')}</FormLabel>
                <FormControl>
                  <Textarea
                    dir='ltr'
                    placeholder={t('example.com&#10;company.com')}
                    rows={4}
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'One domain per line (only used when domain restriction is enabled)'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
