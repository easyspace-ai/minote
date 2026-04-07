import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { QueryClientProvider } from "@tanstack/react-query";

import { AuthProvider } from "@/auth/AuthContext";
import { ThemeProvider } from "@/components/theme-provider";
import { I18nProvider } from "@/core/i18n/context";
import { DEFAULT_LOCALE, normalizeLocale, type Locale } from "@/core/i18n/locale";
import { getLocaleFromCookie } from "@/core/i18n/cookies";
import { queryClient } from "@/lib/queryClient";

import App from "./App";

import "@/styles/globals.css";
import "katex/dist/katex.min.css";

function initialLocale(): Locale {
  return normalizeLocale(getLocaleFromCookie() ?? DEFAULT_LOCALE);
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <BrowserRouter>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <ThemeProvider attribute="class" enableSystem disableTransitionOnChange>
            <I18nProvider initialLocale={initialLocale()}>
              <App />
            </I18nProvider>
          </ThemeProvider>
        </AuthProvider>
      </QueryClientProvider>
    </BrowserRouter>
  </StrictMode>,
);
