import { getLocaleFromCookie, setLocaleInCookie } from "./cookies";
import { DEFAULT_LOCALE, normalizeLocale, type Locale } from "./locale";
import { translations } from "./translations";

export async function detectLocaleServer(): Promise<Locale> {
  return normalizeLocale(getLocaleFromCookie() ?? undefined);
}

export async function setLocale(locale: string | Locale): Promise<Locale> {
  const normalizedLocale = normalizeLocale(locale);
  setLocaleInCookie(normalizedLocale);
  return normalizedLocale;
}

export async function getI18n(localeOverride?: string | Locale) {
  const locale = localeOverride
    ? normalizeLocale(localeOverride)
    : normalizeLocale(getLocaleFromCookie() ?? undefined);
  const t = translations[locale] ?? translations[DEFAULT_LOCALE];
  return {
    locale,
    t,
  };
}
