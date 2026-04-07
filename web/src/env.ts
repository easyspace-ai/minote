import { z } from "zod";

const clientSchema = z.object({
  NEXT_PUBLIC_BACKEND_BASE_URL: z.string().optional(),
  NEXT_PUBLIC_LANGGRAPH_BASE_URL: z.string().optional(),
  NEXT_PUBLIC_STATIC_WEBSITE_ONLY: z.string().optional(),
});

const parsed = clientSchema.safeParse({
  NEXT_PUBLIC_BACKEND_BASE_URL: import.meta.env.VITE_BACKEND_BASE_URL,
  NEXT_PUBLIC_LANGGRAPH_BASE_URL: import.meta.env.VITE_LANGGRAPH_BASE_URL,
  NEXT_PUBLIC_STATIC_WEBSITE_ONLY: import.meta.env.VITE_STATIC_WEBSITE_ONLY,
});

export const env = parsed.success
  ? parsed.data
  : {
      NEXT_PUBLIC_BACKEND_BASE_URL: undefined as string | undefined,
      NEXT_PUBLIC_LANGGRAPH_BASE_URL: undefined as string | undefined,
      NEXT_PUBLIC_STATIC_WEBSITE_ONLY: undefined as string | undefined,
    };
