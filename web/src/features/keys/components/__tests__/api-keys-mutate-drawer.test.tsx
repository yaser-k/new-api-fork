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
import {
  fireEvent,
  render,
  screen,
  waitFor,
  type RenderResult,
} from '@testing-library/react'
import { afterEach, describe, expect, test } from 'vitest'

import type { ApiKey } from '../../types'

const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')
const { api } = await import('@/lib/api')
const { ApiKeysProvider } = await import('../api-keys-provider')
const { ApiKeysMutateDrawer } = await import('../api-keys-mutate-drawer')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

type ApiMethod = (url: string, data?: unknown) => Promise<{ data: unknown }>
type MockableApi = {
  get: ApiMethod
  post: ApiMethod
  put: ApiMethod
}
type RenderedDrawer = {
  queryClient: InstanceType<typeof QueryClient>
  result?: RenderResult
}

const apiClient = api as unknown as MockableApi
const originalGet = apiClient.get
const originalPost = apiClient.post
const originalPut = apiClient.put
let renderedDrawer: RenderedDrawer | null = null

function installApiFixtures(createdPayloads: Array<Record<string, unknown>>) {
  apiClient.get = async (url) => {
    switch (url) {
      case '/api/status':
        return { data: { data: { default_use_auto_group: true } } }
      case '/api/user/models':
        return { data: { success: true, data: [] } }
      case '/api/user/self/groups':
        return {
          data: {
            success: true,
            data: {
              auto: { desc: 'Automatic routing', ratio: 'auto' },
              default: { desc: 'Standard access', ratio: 1 },
              vip: { desc: 'Priority access', ratio: 2 },
            },
          },
        }
      case '/api/token/auto-groups':
        return {
          data: {
            success: true,
            data: { groups: ['vip', 'default'], max_count: 3 },
          },
        }
      default:
        throw new Error(`Unexpected GET ${url}`)
    }
  }
  apiClient.post = async (url, data) => {
    expect(url).toBe('/api/token/')
    expect(data && typeof data === 'object').toBeTruthy()
    createdPayloads.push(data as Record<string, unknown>)
    return { data: { success: true, data: {} } }
  }
}

function createSeededQueryClient(): InstanceType<typeof QueryClient> {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const freshAt = Date.now() + 60_000
  queryClient.setQueryData(
    ['status'],
    { default_use_auto_group: true },
    { updatedAt: freshAt }
  )
  queryClient.setQueryData(
    ['user-models'],
    { success: true, data: [] },
    { updatedAt: freshAt }
  )
  queryClient.setQueryData(
    ['user-groups'],
    {
      success: true,
      data: {
        auto: { desc: 'Automatic routing', ratio: 'auto' },
        default: { desc: 'Standard access', ratio: 1 },
        vip: { desc: 'Priority access', ratio: 2 },
      },
    },
    { updatedAt: freshAt }
  )
  queryClient.setQueryData(
    ['token-auto-groups'],
    {
      success: true,
      data: { groups: ['vip', 'default'], max_count: 3 },
    },
    { updatedAt: freshAt }
  )
  return queryClient
}

async function renderCreateDrawer(): Promise<void> {
  const queryClient = createSeededQueryClient()
  renderedDrawer = { queryClient }

  render(
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <ApiKeysProvider>
          <ApiKeysMutateDrawer open onOpenChange={() => undefined} />
        </ApiKeysProvider>
      </I18nextProvider>
    </QueryClientProvider>
  )
  await waitFor(
    () => {
      const saveButton = findButton('Save changes', false)
      expect(saveButton).toBeEnabled()
    },
    { timeout: 1500 }
  )
}

function findButton(text: string, required: true): HTMLButtonElement
function findButton(text: string, required: false): HTMLButtonElement | null
function findButton(text: string, required = true): HTMLButtonElement | null {
  const button = screen
    .queryAllByRole<HTMLButtonElement>('button')
    .find((candidate) => candidate.textContent?.includes(text))
  if (required && !button) {
    throw new Error(`Expected button containing "${text}"`)
  }
  return button ?? null
}

function getControlByLabel(labelText: 'Name' | 'Quantity'): HTMLInputElement
function getControlByLabel(labelText: 'Group'): HTMLButtonElement
function getControlByLabel(labelText: 'Auto group order'): HTMLElement
function getControlByLabel(labelText: string): HTMLElement {
  const label = [...document.querySelectorAll<HTMLLabelElement>('label')].find(
    (candidate) => candidate.textContent?.trim() === labelText
  )
  if (!label) {
    throw new Error(`Expected label "${labelText}"`)
  }

  const control =
    label.control ??
    label
      .closest('[data-slot="form-item"]')
      ?.querySelector<HTMLElement>(
        '[data-slot="form-control"], input, textarea, button[role="combobox"], [role="group"]'
      )
  if (!control) {
    throw new Error(`Expected control for label "${labelText}"`)
  }
  return control
}

function changeInput(input: HTMLInputElement, value: string): void {
  fireEvent.input(input, { target: { value } })
}

function selectComboboxOption(
  trigger: HTMLButtonElement,
  optionDescription: string
): void {
  fireEvent.click(trigger)
  const option = [
    ...document.querySelectorAll<HTMLElement>('[data-slot="command-item"]'),
  ].find((candidate) => candidate.textContent?.includes(optionDescription))
  if (!option) {
    throw new Error(`Expected option containing "${optionDescription}"`)
  }
  fireEvent.click(option)
}

afterEach(() => {
  apiClient.get = originalGet
  apiClient.post = originalPost
  apiClient.put = originalPut
  localStorage.clear()
  if (renderedDrawer) {
    renderedDrawer.queryClient.clear()
    renderedDrawer = null
  }
})

describe('API keys mutate drawer Auto group integration', () => {
  test('inherits the root Auto order and sends an empty override for every batch-created key', async () => {
    const createdPayloads: Array<Record<string, unknown>> = []
    installApiFixtures(createdPayloads)
    await renderCreateDrawer()

    const groupTrigger = getControlByLabel('Group')
    expect(groupTrigger.textContent?.includes('auto')).toBe(true)
    expect(
      document.body.textContent?.includes(
        'Using the complete global Auto order (2 groups)'
      )
    ).toBe(true)
    expect(
      [
        ...document.querySelectorAll('[data-slot="global-auto-order-name"]'),
      ].map((item) => item.textContent)
    ).toEqual(['vip', 'default'])
    expect(findButton('Restore global Auto', true).disabled).toBe(true)

    changeInput(getControlByLabel('Name'), 'batch')
    changeInput(getControlByLabel('Quantity'), '2')
    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(createdPayloads).toHaveLength(2))

    expect(createdPayloads.length).toBe(2)
    expect(createdPayloads[0]?.name).toBe('batch')
    for (const payload of createdPayloads) {
      expect(payload.group).toBe('auto')
      expect(payload.auto_groups).toEqual([])
      expect(payload.cross_group_retry).toBe(true)
    }
  })

  test('preserves an unsaved custom order and mode after Auto to ordinary to Auto changes', async () => {
    const createdPayloads: Array<Record<string, unknown>> = []
    installApiFixtures(createdPayloads)
    await renderCreateDrawer()

    const autoOrderControl = getControlByLabel('Auto group order')
    const addGroupTrigger = autoOrderControl.querySelector<HTMLButtonElement>(
      'button[role="combobox"]'
    )
    if (!addGroupTrigger) {
      throw new Error('Expected Auto group order combobox')
    }
    selectComboboxOption(addGroupTrigger, 'Priority access')

    expect(
      document.querySelector('button[aria-label="Remove vip"]')
    ).toBeTruthy()
    expect(document.body.textContent?.includes('1 / 3 groups selected')).toBe(
      true
    )
    expect(findButton('Restore global Auto', true).disabled).toBe(false)

    const groupTrigger = getControlByLabel('Group')
    selectComboboxOption(groupTrigger, 'Standard access')
    expect(document.querySelector('button[aria-label="Remove vip"]')).toBe(null)
    selectComboboxOption(groupTrigger, 'Automatic routing')

    expect(
      document.querySelector('button[aria-label="Remove vip"]')
    ).toBeTruthy()
    expect(document.body.textContent?.includes('1 / 3 groups selected')).toBe(
      true
    )
    expect(findButton('Restore global Auto', true).disabled).toBe(false)

    changeInput(getControlByLabel('Name'), 'custom')
    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(createdPayloads).toHaveLength(1))
    expect(createdPayloads[0]?.auto_groups).toEqual(['vip'])
  })
})

describe('API keys mutate drawer user quota query switch', () => {
  test('sends false for a new key after the switch, which starts on, is turned off', async () => {
    const createdPayloads: Array<Record<string, unknown>> = []
    installApiFixtures(createdPayloads)
    await renderCreateDrawer()

    fireEvent.click(findButton('Advanced Settings', true))
    const quotaQuerySwitch = await screen.findByRole('switch', {
      name: 'Allow querying user quota',
    })
    expect(quotaQuerySwitch).toBeChecked()
    fireEvent.click(quotaQuerySwitch)
    expect(quotaQuerySwitch).not.toBeChecked()

    changeInput(getControlByLabel('Name'), 'no-user-quota')
    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(createdPayloads).toHaveLength(1))
    expect(createdPayloads[0]?.allow_user_quota_query).toBe(false)
  })
})

// 2023-11-14 12:00 UTC: the same calendar day in every common time zone.
const PAST_EXPIRY = 1699963200

function apiKey(id: number, expiredTime: number): ApiKey {
  return {
    id,
    name: `key-${id}`,
    key: `sk-${id}`,
    status: 1,
    remain_quota: 0,
    used_quota: 0,
    unlimited_quota: true,
    expired_time: expiredTime,
    created_time: 1,
    accessed_time: 1,
    group: 'default',
    auto_groups: null,
    cross_group_retry: false,
    model_limits_enabled: false,
    model_limits: '',
    allow_ips: '',
    allow_user_quota_query: true,
  }
}

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((promiseResolve) => {
    resolve = promiseResolve
  })
  return { promise, resolve }
}

function installEditFixtures(
  keysById: Map<number, Promise<ApiKey>>,
  updates: Array<Record<string, unknown>>,
  creates: Array<Record<string, unknown>> = []
) {
  installApiFixtures(creates)
  const fixtureGet = apiClient.get
  apiClient.get = async (url, data) => {
    const id = Number(url.match(/^\/api\/token\/(\d+)$/)?.[1])
    const key = keysById.get(id)
    if (key) {
      return { data: { success: true, data: await key } }
    }
    return fixtureGet(url, data)
  }
  apiClient.put = async (url, data) => {
    expect(url).toBe('/api/token/')
    expect(data && typeof data === 'object').toBeTruthy()
    updates.push(data as Record<string, unknown>)
    return { data: { success: true, data: {} } }
  }
}

function keyDrawerTree(open: boolean, currentRow?: ApiKey) {
  if (!renderedDrawer) {
    throw new Error('Expected a seeded query client')
  }
  return (
    <QueryClientProvider client={renderedDrawer.queryClient}>
      <I18nextProvider i18n={i18n}>
        <ApiKeysProvider>
          <ApiKeysMutateDrawer
            open={open}
            onOpenChange={() => undefined}
            currentRow={currentRow}
          />
        </ApiKeysProvider>
      </I18nextProvider>
    </QueryClientProvider>
  )
}

function openKeyDrawer(currentRow?: ApiKey): void {
  if (renderedDrawer?.result) {
    renderedDrawer.result.rerender(keyDrawerTree(true, currentRow))
    return
  }
  renderedDrawer = { queryClient: createSeededQueryClient() }
  renderedDrawer.result = render(keyDrawerTree(true, currentRow))
}

// Closes the drawer the way a person does: the Close button, then the parent
// clears the open state and the row.
function closeKeyDrawer(): void {
  fireEvent.click(findButton('Close', true))
  renderedDrawer?.result?.rerender(keyDrawerTree(false))
}

async function waitForKeyForm(): Promise<void> {
  await waitFor(() => expect(findButton('Save changes', true)).toBeEnabled(), {
    timeout: 1500,
  })
}

// The picker copies its value into local state in an effect, so wait for it.
function findExpiryTrigger(name: string): Promise<HTMLButtonElement> {
  return screen.findByRole<HTMLButtonElement>('button', { name })
}

describe('API keys mutate drawer expiry between records', () => {
  test('a never-expiring key opened after a key with an expiry shows and saves never', async () => {
    const withExpiry = apiKey(1, PAST_EXPIRY)
    const neverExpires = apiKey(2, -1)
    const neverExpiresRequest = deferred<ApiKey>()
    const updates: Array<Record<string, unknown>> = []
    installEditFixtures(
      new Map([
        [1, Promise.resolve(withExpiry)],
        [2, neverExpiresRequest.promise],
      ]),
      updates
    )

    openKeyDrawer(withExpiry)
    await waitForKeyForm()
    expect(await findExpiryTrigger('2023-11-14')).toBeInTheDocument()
    closeKeyDrawer()

    openKeyDrawer(neverExpires)
    neverExpiresRequest.resolve(neverExpires)
    await waitForKeyForm()
    expect(getControlByLabel('Name').value).toBe('key-2')
    expect(await findExpiryTrigger('Never expires')).toBeInTheDocument()

    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(updates).toHaveLength(1))
    expect(updates[0]?.id).toBe(2)
    expect(updates[0]?.expired_time).toBe(-1)
  })

  test('a key with an expiry opened after a never-expiring key shows and saves its expiry', async () => {
    const neverExpires = apiKey(1, -1)
    const withExpiry = apiKey(2, PAST_EXPIRY)
    const withExpiryRequest = deferred<ApiKey>()
    const updates: Array<Record<string, unknown>> = []
    installEditFixtures(
      new Map([
        [1, Promise.resolve(neverExpires)],
        [2, withExpiryRequest.promise],
      ]),
      updates
    )

    openKeyDrawer(neverExpires)
    await waitForKeyForm()
    closeKeyDrawer()

    openKeyDrawer(withExpiry)
    withExpiryRequest.resolve(withExpiry)
    await waitForKeyForm()
    expect(await findExpiryTrigger('2023-11-14')).toBeInTheDocument()

    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(updates).toHaveLength(1))
    expect(updates[0]?.expired_time).toBe(PAST_EXPIRY)
  })

  test('create after editing a key with an expiry starts blank and saves never', async () => {
    const withExpiry = apiKey(1, PAST_EXPIRY)
    const updates: Array<Record<string, unknown>> = []
    const creates: Array<Record<string, unknown>> = []
    installEditFixtures(
      new Map([[1, Promise.resolve(withExpiry)]]),
      updates,
      creates
    )

    openKeyDrawer(withExpiry)
    await waitForKeyForm()
    closeKeyDrawer()

    openKeyDrawer()
    await waitForKeyForm()
    expect(getControlByLabel('Name').value).toBe('')
    expect(await findExpiryTrigger('Never expires')).toBeInTheDocument()

    changeInput(getControlByLabel('Name'), 'fresh')
    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(creates).toHaveLength(1))
    expect(creates[0]?.expired_time).toBe(-1)
    expect(updates).toEqual([])
  })

  test('the Never button clears a loaded expiry and saves never', async () => {
    const withExpiry = apiKey(1, PAST_EXPIRY)
    const updates: Array<Record<string, unknown>> = []
    installEditFixtures(new Map([[1, Promise.resolve(withExpiry)]]), updates)

    openKeyDrawer(withExpiry)
    await waitForKeyForm()
    expect(await findExpiryTrigger('2023-11-14')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Never' }))
    expect(await findExpiryTrigger('Never expires')).toBeInTheDocument()

    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(updates).toHaveLength(1))
    expect(updates[0]?.expired_time).toBe(-1)
  })
})
