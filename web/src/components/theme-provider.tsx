"use client";

import { useLocation } from "react-router-dom";
import { ThemeProvider as NextThemesProvider } from "next-themes";

export function ThemeProvider({
  children,
  ...props
}: React.ComponentProps<typeof NextThemesProvider>) {
  const { pathname } = useLocation();
  return (
    <NextThemesProvider
      {...props}
      forcedTheme={pathname === "/landing" ? "dark" : undefined}
    >
      {children}
    </NextThemesProvider>
  );
}
