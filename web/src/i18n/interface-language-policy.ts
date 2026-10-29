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
import i18n from 'i18next'

import { useSystemConfigStore } from '@/stores/system-config-store'

import {
  INTERFACE_LANGUAGE_OPTIONS,
  normalizeInterfaceLanguage,
} from './languages'

/**
 * The operator's interface language setting, from `/api/status`
 * (`interface_setting` on the backend). Every field is optional: unset, the
 * interface offers every shipped language, starts in the browser's language and
 * shows the general custom currency symbol, exactly as without the setting.
 */
export type InterfaceLanguageSettings = {
  /** Offered codes, in menu order; empty offers every shipped language. */
  languages: string[]
  /** The language a visitor sees until they choose one; '' keeps the browser's. */
  defaultLanguage: string
  /** Custom currency symbol per interface language. */
  currencySymbols: Record<string, string>
}

type InterfaceLanguageOption = (typeof INTERFACE_LANGUAGE_OPTIONS)[number]

/**
 * localStorage key holding the language a person chose in the language menu or
 * in their profile. i18next caches every language it detects under its own key,
 * so its cache cannot tell a choice from a browser default; this key can.
 */
export const CHOSEN_LANGUAGE_STORAGE_KEY = 'interface-language-chosen'

function isShippedLanguage(code: string): boolean {
  return INTERFACE_LANGUAGE_OPTIONS.some((lang) => lang.code === code)
}

/** Normalize a code, keeping it only when the interface ships that language. */
function toShippedLanguage(value: unknown): string | null {
  if (typeof value !== 'string' || !value.trim()) return null
  const code = normalizeInterfaceLanguage(value)
  // normalizeInterfaceLanguage maps every unknown value to 'en'; an 'en' that
  // did not come from an English tag is an unknown language, not English.
  if (code === 'en' && !value.trim().toLowerCase().startsWith('en')) return null
  return isShippedLanguage(code) ? code : null
}

export function readInterfaceLanguageSettings(
  status: Record<string, unknown> | null | undefined
): InterfaceLanguageSettings {
  const rawLanguages = Array.isArray(status?.interface_languages)
    ? (status?.interface_languages as unknown[])
    : []
  const languages: string[] = []
  for (const value of rawLanguages) {
    const code = toShippedLanguage(value)
    if (code && !languages.includes(code)) languages.push(code)
  }
  const defaultLanguage =
    toShippedLanguage(status?.default_interface_language) ?? ''
  const currencySymbols: Record<string, string> = {}
  const rawSymbols = status?.custom_currency_symbols
  if (
    rawSymbols &&
    typeof rawSymbols === 'object' &&
    !Array.isArray(rawSymbols)
  ) {
    for (const [lang, symbol] of Object.entries(rawSymbols)) {
      const code = toShippedLanguage(lang)
      if (code && typeof symbol === 'string' && symbol.trim()) {
        currencySymbols[code] = symbol.trim()
      }
    }
  }
  return { languages, defaultLanguage, currencySymbols }
}

/** The languages the menu offers, in the configured order. */
export function getOfferedInterfaceLanguages(
  settings: InterfaceLanguageSettings
): InterfaceLanguageOption[] {
  if (settings.languages.length === 0) return [...INTERFACE_LANGUAGE_OPTIONS]
  return settings.languages
    .map((code) =>
      INTERFACE_LANGUAGE_OPTIONS.find((lang) => lang.code === code)
    )
    .filter((lang): lang is InterfaceLanguageOption => Boolean(lang))
}

export function isInterfaceLanguageOffered(
  settings: InterfaceLanguageSettings,
  value: string | null | undefined
): boolean {
  const code = toShippedLanguage(value)
  if (!code) return false
  return settings.languages.length === 0 || settings.languages.includes(code)
}

/**
 * The language the interface should switch to, or null to keep the current one.
 * Without a configured list or default it is always null.
 *
 * - A language the person chose, if it is still offered, is kept.
 * - Otherwise, with a default configured, the default (when offered) applies;
 *   a language that is not offered gives way to the default, or to the first
 *   offered language when there is no usable default.
 */
export function resolveInterfaceLanguage(
  current: string | null | undefined,
  settings: InterfaceLanguageSettings,
  chosen: string | null | undefined
): string | null {
  // Unset, the interface keeps i18next's own choice, exactly as without it.
  if (settings.languages.length === 0 && !settings.defaultLanguage) return null
  const currentCode = normalizeInterfaceLanguage(current)
  if (chosen && isInterfaceLanguageOffered(settings, chosen)) {
    const chosenCode = normalizeInterfaceLanguage(chosen)
    return chosenCode === currentCode ? null : chosenCode
  }
  const offeredDefault = isInterfaceLanguageOffered(
    settings,
    settings.defaultLanguage
  )
    ? settings.defaultLanguage
    : (settings.languages[0] ?? '')
  if (settings.defaultLanguage && offeredDefault) {
    return offeredDefault === currentCode ? null : offeredDefault
  }
  if (!isInterfaceLanguageOffered(settings, currentCode) && offeredDefault) {
    return offeredDefault
  }
  return null
}

function readChosenLanguage(): string | null {
  try {
    return window.localStorage.getItem(CHOSEN_LANGUAGE_STORAGE_KEY)
  } catch {
    return null
  }
}

/** Remember that the person chose this language (menu, profile, account). */
export function rememberChosenInterfaceLanguage(code: string): void {
  try {
    window.localStorage.setItem(
      CHOSEN_LANGUAGE_STORAGE_KEY,
      normalizeInterfaceLanguage(code)
    )
  } catch {
    /* Storage can be unavailable in private mode. */
  }
}

let currentSettings: InterfaceLanguageSettings = {
  languages: [],
  defaultLanguage: '',
  currencySymbols: {},
}
let baseCurrencySymbol: string | null = null
let listening = false

function applyCurrencySymbol(language: string | undefined): void {
  const store = useSystemConfigStore.getState()
  if (baseCurrencySymbol === null) {
    baseCurrencySymbol = store.config.currency.customCurrencySymbol
  }
  const symbol =
    currentSettings.currencySymbols[normalizeInterfaceLanguage(language)] ??
    baseCurrencySymbol
  if (store.config.currency.customCurrencySymbol !== symbol) {
    store.setConfig({
      currency: { ...store.config.currency, customCurrencySymbol: symbol },
    })
  }
}

/**
 * Apply the operator's interface language setting from a status payload: switch
 * to the language the setting calls for, and show the currency symbol of the
 * current language. Called with the cached status at start-up and with every
 * fresh `/api/status`. Without the setting it changes nothing.
 */
export function applyInterfaceLanguageSettings(
  status: Record<string, unknown> | null | undefined
): void {
  if (!status) return
  currentSettings = readInterfaceLanguageSettings(status)
  const base = status.custom_currency_symbol
  if (typeof base === 'string' && base.trim()) baseCurrencySymbol = base.trim()

  const hasSymbols = Object.keys(currentSettings.currencySymbols).length > 0
  if (hasSymbols && !listening) {
    listening = true
    i18n.on('languageChanged', (lng) => applyCurrencySymbol(lng))
  }

  const next = resolveInterfaceLanguage(
    i18n.language,
    currentSettings,
    readChosenLanguage()
  )
  if (next) {
    void i18n.changeLanguage(next)
  } else if (hasSymbols) {
    applyCurrencySymbol(i18n.language)
  }
}
