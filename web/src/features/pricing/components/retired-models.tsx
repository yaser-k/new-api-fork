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
import { ExternalLink } from 'lucide-react'
import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { ModelLink } from '@/features/rankings/components/entity-links'

import type { RetiredModel } from '../types'

function RetiredModelBadge() {
  const { t } = useTranslation()
  return <Badge variant='secondary'>{t('Retired')}</Badge>
}

function RetirementNoticeLink(props: { url: string }) {
  const { t } = useTranslation()
  return (
    <a
      href={props.url}
      target='_blank'
      rel='noopener noreferrer'
      className='text-muted-foreground hover:text-foreground inline-flex items-center gap-1 underline decoration-current/40 underline-offset-2'
    >
      {t('Retirement notice')}
      <ExternalLink className='size-3' aria-hidden='true' />
    </a>
  )
}

function RetiredModelItem(props: { model: RetiredModel }) {
  const { t } = useTranslation()
  return (
    <li className='flex flex-col gap-1 py-3 first:pt-0 last:pb-0'>
      <div className='flex flex-wrap items-center gap-2'>
        <ModelLink
          modelName={props.model.model_name}
          className='font-mono text-sm font-semibold'
        />
        <RetiredModelBadge />
        <span className='text-muted-foreground text-xs'>
          {t('Retired on {{date}}', { date: props.model.retired_at })}
        </span>
      </div>
      <div className='text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-xs'>
        <span>
          {t('Use instead')}:{' '}
          <ModelLink
            modelName={props.model.successor}
            className='text-foreground font-mono'
          />
        </span>
        {props.model.notice_url && (
          <RetirementNoticeLink url={props.model.notice_url} />
        )}
      </div>
      {props.model.note && (
        <p className='text-muted-foreground text-xs'>{props.model.note}</p>
      )}
    </li>
  )
}

/**
 * The retired models of the pricing page, hidden until the visitor opens
 * them. Retired models cannot be called, so they are listed apart from the
 * live models, each with its successor.
 */
export function RetiredModelsSection(props: { models: RetiredModel[] }) {
  const { t } = useTranslation()
  const headingId = useId()
  const [open, setOpen] = useState(false)

  if (props.models.length === 0) return null

  return (
    <section
      aria-labelledby={headingId}
      className='bg-card/60 rounded-xl border p-4'
    >
      <Collapsible open={open} onOpenChange={setOpen}>
        <div className='flex flex-wrap items-center justify-between gap-2'>
          <div>
            <h2 id={headingId} className='text-sm font-semibold'>
              {t('Retired models')}
            </h2>
            <p className='text-muted-foreground mt-0.5 text-xs'>
              {t(
                'These models can no longer be called. Use their successors instead.'
              )}
            </p>
          </div>
          <CollapsibleTrigger render={<Button variant='outline' size='sm' />}>
            {open
              ? t('Hide retired models')
              : t('Show retired models ({{count}})', {
                  count: props.models.length,
                })}
          </CollapsibleTrigger>
        </div>
        <CollapsibleContent>
          <ul className='mt-4 divide-y'>
            {props.models.map((model) => (
              <RetiredModelItem key={model.model_name} model={model} />
            ))}
          </ul>
        </CollapsibleContent>
      </Collapsible>
    </section>
  )
}

/** The page of a retired model: its retirement instead of its prices. */
export function RetiredModelNotice(props: { model: RetiredModel }) {
  const { t } = useTranslation()
  return (
    <div className='space-y-4'>
      <header>
        <div className='flex flex-wrap items-center gap-2.5'>
          <h1 className='font-mono text-xl font-bold tracking-tight sm:text-2xl'>
            {props.model.model_name}
          </h1>
          <RetiredModelBadge />
        </div>
      </header>
      <section className='bg-muted/20 space-y-3 rounded-xl border p-4'>
        <p className='text-sm'>
          {t(
            'This model was retired on {{date}} and can no longer be called.',
            { date: props.model.retired_at }
          )}
        </p>
        <p className='text-sm'>
          {t('Use instead')}:{' '}
          <ModelLink
            modelName={props.model.successor}
            className='font-mono font-semibold'
          />
        </p>
        {props.model.note && (
          <p className='text-muted-foreground text-sm'>{props.model.note}</p>
        )}
        {props.model.notice_url && (
          <p className='text-sm'>
            <RetirementNoticeLink url={props.model.notice_url} />
          </p>
        )}
      </section>
    </div>
  )
}
