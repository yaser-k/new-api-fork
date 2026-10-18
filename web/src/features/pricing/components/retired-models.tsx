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
import { Link } from '@tanstack/react-router'
import { ChevronRight, ExternalLink } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardFooter, CardHeader } from '@/components/ui/card'
import { ModelLink } from '@/features/rankings/components/entity-links'

import type { RetiredModel } from '../types'

export function RetiredModelBadge() {
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

/** The successor of a retired model, as a link to its page. */
export function RetiredModelSuccessor(props: {
  model: RetiredModel
  className?: string
}) {
  const { t } = useTranslation()
  return (
    <span className={props.className}>
      {t('Use instead')}:{' '}
      <ModelLink
        modelName={props.model.successor}
        className='text-foreground font-mono'
      />
    </span>
  )
}

/**
 * The card of a retired model in the pricing list: its retirement and
 * successor instead of prices, with no way to call or copy it.
 */
export function RetiredModelCard(props: { model: RetiredModel }) {
  const { t } = useTranslation()
  const initial = props.model.model_name.charAt(0).toUpperCase() || '?'
  return (
    <Card className='hover:ring-foreground/20 h-full min-w-0 gap-3 transition-colors'>
      <CardHeader className='flex flex-row items-start gap-3'>
        <div
          aria-hidden
          className='bg-muted/50 flex size-10 shrink-0 items-center justify-center rounded-lg'
        >
          <span className='text-muted-foreground text-sm font-bold'>
            {initial}
          </span>
        </div>
        <div className='min-w-0 flex-1'>
          <h3 className='line-clamp-2 font-mono text-[15px] leading-snug font-semibold [overflow-wrap:anywhere]'>
            <ModelLink modelName={props.model.model_name} />
          </h3>
          <div className='mt-1 flex flex-wrap items-center gap-2'>
            <RetiredModelBadge />
            <span className='text-muted-foreground text-xs'>
              {t('Retired on {{date}}', { date: props.model.retired_at })}
            </span>
          </div>
        </div>
      </CardHeader>
      <CardContent className='text-muted-foreground flex flex-1 flex-col gap-1.5 text-xs'>
        <RetiredModelSuccessor model={props.model} />
        {props.model.note && <p>{props.model.note}</p>}
        {props.model.notice_url && (
          <span>
            <RetirementNoticeLink url={props.model.notice_url} />
          </span>
        )}
      </CardContent>
      <CardFooter className='mt-auto justify-end border-0 bg-transparent pt-0'>
        <Link
          to='/pricing/$modelId'
          params={{ modelId: props.model.model_name }}
          className={buttonVariants({ variant: 'ghost', size: 'sm' })}
        >
          {t('Details')}
          <ChevronRight aria-hidden className='size-3.5' />
        </Link>
      </CardFooter>
    </Card>
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
