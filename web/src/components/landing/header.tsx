"use client";

import { StarFilledIcon, GitHubLogoIcon } from "@radix-ui/react-icons";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { NumberTicker } from "@/components/ui/number-ticker";
import { useI18n } from "@/core/i18n/hooks";
import { env } from "@/env";
import { cn } from "@/lib/utils";

export type HeaderProps = {
  className?: string;
  homeURL?: string;
};

function StarCounter() {
  const [stars, setStars] = useState(10000);
  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const headers: Record<string, string> = {
          "Content-Type": "application/json",
        };
        const token = import.meta.env.VITE_GITHUB_OAUTH_TOKEN as
          | string
          | undefined;
        if (token) headers.Authorization = `Bearer ${token}`;
        const response = await fetch(
          "https://api.github.com/repos/bytedance/deer-flow",
          { headers },
        );
        if (!cancelled && response.ok) {
          const data = (await response.json()) as { stargazers_count?: number };
          if (data.stargazers_count != null) {
            setStars(data.stargazers_count);
          }
        }
      } catch (e) {
        console.error("Error fetching GitHub stars:", e);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);
  return (
    <>
      <StarFilledIcon className="size-4 transition-colors duration-300 group-hover:text-yellow-500" />
      {stars ? (
        <NumberTicker className="font-mono tabular-nums" value={stars} />
      ) : null}
    </>
  );
}

export function Header({ className, homeURL }: HeaderProps) {
  const isExternalHome = !homeURL;
  const { locale, t } = useI18n();
  const lang = locale.substring(0, 2);
  return (
    <header
      className={cn(
        "container-md fixed top-0 right-0 left-0 z-20 mx-auto flex h-16 items-center justify-between backdrop-blur-xs",
        className,
      )}
    >
      <div className="flex items-center gap-6">
        <a
          href={homeURL ?? "https://github.com/bytedance/deer-flow"}
          target={isExternalHome ? "_blank" : "_self"}
          rel={isExternalHome ? "noopener noreferrer" : undefined}
        >
          <h1 className="font-serif text-xl">DeerFlow</h1>
        </a>
      </div>
      <nav className="mr-8 ml-auto flex items-center gap-8 text-sm font-medium">
        <Link
          to={`/${lang}/docs`}
          className="text-secondary-foreground hover:text-foreground transition-colors"
        >
          {t.home.docs}
        </Link>
        <a
          href={`/${lang}/blog`}
          target="_self"
          className="text-secondary-foreground hover:text-foreground transition-colors"
        >
          {t.home.blog}
        </a>
      </nav>
      <div className="relative">
        <div
          className="pointer-events-none absolute inset-0 z-0 h-full w-full rounded-full opacity-30 blur-2xl"
          style={{
            background: "linear-gradient(90deg, #ff80b5 0%, #9089fc 100%)",
            filter: "blur(16px)",
          }}
        />
        <Button
          variant="outline"
          size="sm"
          asChild
          className="group relative z-10"
        >
          <a
            href="https://github.com/bytedance/deer-flow"
            target="_blank"
            rel="noopener noreferrer"
          >
            <GitHubLogoIcon className="size-4" />
            Star on GitHub
            {env.NEXT_PUBLIC_STATIC_WEBSITE_ONLY === "true" && <StarCounter />}
          </a>
        </Button>
      </div>
      <hr className="from-border/0 via-border/70 to-border/0 absolute top-16 right-0 left-0 z-10 m-0 h-px w-full border-none bg-linear-to-r" />
    </header>
  );
}
