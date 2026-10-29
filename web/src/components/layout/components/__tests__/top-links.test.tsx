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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { SidebarProvider } from '@/components/ui/sidebar'
import { DirectionProvider } from '@/context/direction-provider'
import { useTopNavLinks } from '@/hooks/use-top-nav-links'
import { api } from '@/lib/api'
import { STATUS_QUERY_KEY } from '@/lib/status-query'

import { AppHeader } from '../app-header'
import { MobileDrawer } from '../mobile-drawer'
import { NavLinkList } from '../nav-link-item'
import { PublicHeader } from '../public-header'
import { PublicNavigation } from '../public-navigation'
import { TopNav } from '../top-nav'

const DOCS_LINK = 'https://docs.example.com'
const CONTACT_URL = 'https://example.com/contact'

// Whether the app's own handlers cancelled each click, that is whether the
// router took the click over instead of letting the browser follow the link.
let clicksCancelled: boolean[] = []

// Runs after the app's handlers, then cancels the click so jsdom does not try
// to load the page.
function recordClick(event: MouseEvent) {
  clicksCancelled.push(event.defaultPrevented)
  event.preventDefault()
}

beforeEach(() => {
  clicksCancelled = []
  window.addEventListener('click', recordClick)
  vi.stubGlobal('localStorage', {
    getItem: () => null,
    setItem: () => undefined,
    removeItem: () => undefined,
  })
  // jsdom lacks the Web Animations API the dropdown menu waits on.
  Object.defineProperty(HTMLElement.prototype, 'getAnimations', {
    configurable: true,
    value: () => [],
  })
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: '' },
  })
})

afterEach(() => {
  window.removeEventListener('click', recordClick)
  vi.unstubAllGlobals()
  document.documentElement.removeAttribute('dir')
  Reflect.deleteProperty(HTMLElement.prototype, 'getAnimations')
})

function MobileDrawerWithTopLinks() {
  return (
    <MobileDrawer
      isOpen
      onClose={() => undefined}
      homeUrl='/'
      displayLogo={null}
      displaySiteName='Site'
      loading={false}
      logoLoaded
      mobileLinksList={useTopNavLinks()}
      showAuthButtons={false}
      user={null}
    />
  )
}

function TopNavWithTopLinks() {
  return <TopNav links={useTopNavLinks()} />
}

function NavLinkListWithTopLinks() {
  return (
    <nav aria-label='Link list'>
      <NavLinkList links={useTopNavLinks()} />
    </nav>
  )
}

/** Renders `ui` at `/` with `/api/status` holding the given contact link. */
async function renderWithContact(ui: ReactNode, link: string) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  client.setQueryData(STATUS_QUERY_KEY, {
    docs_link: DOCS_LINK,
    HeaderNavModules: JSON.stringify({ contact: { enabled: true, link } }),
  })
  const router = createRouter({
    routeTree: createRootRoute({
      component: () => (
        <DirectionProvider>
          <SidebarProvider>{ui}</SidebarProvider>
        </DirectionProvider>
      ),
    }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  await router.load()
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  return router
}

async function expectSameTabAnchor(
  contact: HTMLElement,
  router: Awaited<ReturnType<typeof renderWithContact>>
) {
  expect(contact).toHaveAttribute('href', '/contact')
  expect(contact).not.toHaveAttribute('target')

  await userEvent.click(contact)

  expect(clicksCancelled.at(-1)).toBe(false)
  expect(router.history.location.pathname).toBe('/')
}

function expectLikeExternalDocs(contact: HTMLElement, docs: HTMLElement) {
  expect(docs).toHaveAttribute('href', DOCS_LINK)
  expect(contact).toHaveAttribute('href', CONTACT_URL)
  expect(contact).toHaveAttribute('target', '_blank')
  expect(contact).toHaveAttribute('rel', 'noopener noreferrer')
  expect(contact.getAttribute('target')).toBe(docs.getAttribute('target'))
  expect(contact.getAttribute('rel')).toBe(docs.getAttribute('rel'))
}

const publicHeader = (
  <PublicHeader
    showThemeSwitch={false}
    showLanguageSwitcher={false}
    showNotifications={false}
    showAuthButtons={false}
  />
)

const appHeader = (
  <AppHeader
    showSearch={false}
    showNotifications={false}
    showConfigDrawer={false}
    showProfileDropdown={false}
  />
)

describe('public header Contact item', () => {
  test('a same-site path is a plain same-tab anchor in the desktop bar and the mobile menu, not a router link', async () => {
    const router = await renderWithContact(publicHeader, '/contact')

    const contacts = await screen.findAllByRole('link', { name: 'Contact' })
    expect(contacts).toHaveLength(2)
    for (const contact of contacts) {
      await expectSameTabAnchor(contact, router)
    }
  })

  test('an https URL opens in the desktop bar and the mobile menu the way the external Docs link does', async () => {
    await renderWithContact(publicHeader, CONTACT_URL)

    const contacts = await screen.findAllByRole('link', { name: 'Contact' })
    const docs = screen.getAllByRole('link', { name: 'Docs' })
    expect(contacts).toHaveLength(2)
    contacts.forEach((contact, i) => expectLikeExternalDocs(contact, docs[i]))
  })
})

describe('authenticated header Contact item', () => {
  test('a same-site path is a plain same-tab anchor, not a router link', async () => {
    const router = await renderWithContact(appHeader, '/contact')

    await expectSameTabAnchor(
      await screen.findByRole('link', { name: 'Contact' }),
      router
    )
  })

  test('an https URL opens the way the external Docs link does', async () => {
    await renderWithContact(appHeader, CONTACT_URL)

    expectLikeExternalDocs(
      await screen.findByRole('link', { name: 'Contact' }),
      screen.getByRole('link', { name: 'Docs' })
    )
  })

  test('the compact menu shows a same-site path as a plain same-tab anchor', async () => {
    const router = await renderWithContact(<TopNavWithTopLinks />, '/contact')

    await userEvent.click(screen.getByRole('button'))
    const menu = await screen.findByRole('menu')

    await expectSameTabAnchor(
      within(menu).getByRole('menuitem', { name: 'Contact' }),
      router
    )
  })

  test('the compact menu opens an https URL the way the external Docs link does', async () => {
    await renderWithContact(<TopNavWithTopLinks />, CONTACT_URL)

    await userEvent.click(screen.getByRole('button'))
    const menu = await screen.findByRole('menu')

    expectLikeExternalDocs(
      within(menu).getByRole('menuitem', { name: 'Contact' }),
      within(menu).getByRole('menuitem', { name: 'Docs' })
    )
  })
})

describe('other top link renderers', () => {
  test.each([
    ['mobile drawer', <MobileDrawerWithTopLinks key='drawer' />],
    ['navigation link list', <NavLinkListWithTopLinks key='list' />],
    ['public navigation', <PublicNavigation key='public' />],
  ])(
    'the %s shows a same-site path as a plain same-tab anchor',
    async (_name, ui) => {
      const router = await renderWithContact(ui, '/contact')

      await expectSameTabAnchor(
        await screen.findByRole('link', { name: 'Contact' }),
        router
      )
    }
  )
})
