"use client"

import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import type { HTMLAttributes } from "react"

export function StatusBadge({ status, ...props }: { status: string } & HTMLAttributes<HTMLDivElement>) {
  const config: Record<string, { label: string; className: string }> = {
    online: { label: "Online", className: "bg-success/15 text-success border-success/30" },
    running: { label: "Running", className: "bg-success/15 text-success border-success/30" },
    offline: { label: "Offline", className: "bg-muted text-muted-foreground border-muted" },
    stopped: { label: "Stopped", className: "bg-muted text-muted-foreground border-muted" },
    maintenance: { label: "Maintenance", className: "bg-warning/15 text-warning border-warning/30" },
    paused: { label: "Paused", className: "bg-warning/15 text-warning border-warning/30" },
    suspended: { label: "Suspended", className: "bg-warning/15 text-warning border-warning/30" },
    frozen: { label: "Frozen", className: "bg-warning/15 text-warning border-warning/30" },
    partially_running: { label: "Partial", className: "bg-warning/15 text-warning border-warning/30" },
  }
  const c = config[status] ?? { label: status, className: "" }
  return <Badge variant="outline" className={cn("text-[11px] font-medium", c.className)} {...props}>{c.label}</Badge>
}

export function ResourceBar({ label, used, total, unit, showPercent = true }: {
  label: string
  used: number
  total: number
  unit: string
  showPercent?: boolean
}) {
  const pct = total > 0 ? Math.round((used / total) * 100) : 0
  const color =
    pct >= 90 ? "bg-destructive" : pct >= 70 ? "bg-warning" : "bg-primary"

  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center justify-between text-xs">
        <span className="text-muted-foreground">{label}</span>
        <span className="text-foreground font-medium">
          {used} / {total} {unit}
          {showPercent && <span className="ml-1 text-muted-foreground">({pct}%)</span>}
        </span>
      </div>
      <div className="relative h-1.5 w-full overflow-hidden rounded-full bg-secondary">
        <div
          className={cn("h-full rounded-full transition-all", color)}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  )
}

export function MetricCard({
  label,
  value,
  subtitle,
  icon,
}: {
  label: string
  value: string | number
  subtitle?: string
  icon?: React.ReactNode
}) {
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="flex items-center gap-2 text-muted-foreground">
        {icon}
        <span className="text-xs font-medium">{label}</span>
      </div>
      <div className="mt-1.5 text-2xl font-semibold text-foreground">{value}</div>
      {subtitle && <div className="mt-0.5 text-xs text-muted-foreground">{subtitle}</div>}
    </div>
  )
}

export function ResourceUsageBar({ value }: { value: number }) {
  const color =
    value >= 90 ? "bg-destructive" : value >= 70 ? "bg-warning" : "bg-primary"
  return (
    <div className="flex items-center gap-2 min-w-[100px]">
      <div className="relative h-1.5 w-full overflow-hidden rounded-full bg-secondary">
        <div className={cn("h-full rounded-full transition-all", color)} style={{ width: `${value}%` }} />
      </div>
      <span className="text-xs text-muted-foreground w-8 text-right">{value}%</span>
    </div>
  )
}

export function TagList({ tags }: { tags: string[] }) {
  return (
    <div className="flex flex-wrap gap-1">
      {tags.map((tag) => (
        <Badge key={tag} variant="secondary" className="text-[10px] px-1.5 py-0">
          {tag}
        </Badge>
      ))}
    </div>
  )
}
