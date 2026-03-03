"use client"

import { Shimmer as ShimmerLib } from "shimmer-from-structure"

export interface ShimmerProps {
  loading: boolean
  children: React.ReactNode
  templateProps?: Record<string, unknown>
}

export function Shimmer({ loading, children, templateProps }: ShimmerProps) {
  return (
    <ShimmerLib
      loading={loading}
      templateProps={templateProps}
      shimmerColor="oklch(0.5 0 0 / 0.15)"
      backgroundColor="oklch(0.5 0 0 / 0.08)"
      duration={1.5}
      fallbackBorderRadius={6}
    >
      {children}
    </ShimmerLib>
  )
}
