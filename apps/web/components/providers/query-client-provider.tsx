"use client"

import { QueryClient, QueryClientProvider as TanStackQueryClientProvider } from "@tanstack/react-query"
import { useState, type ReactNode } from "react"
import { AuthProvider } from "@/lib/auth"

interface QueryProviderProps {
  children: ReactNode
}

export function QueryClientProvider({ children }: QueryProviderProps) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 30 * 1000, // 30 seconds
            refetchOnWindowFocus: false,
            retry: 1,
          },
        },
      })
  )

  return (
    <TanStackQueryClientProvider client={queryClient}>
      <AuthProvider>{children}</AuthProvider>
    </TanStackQueryClientProvider>
  )
}
