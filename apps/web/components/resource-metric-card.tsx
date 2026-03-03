import { Card, CardContent } from "@/components/ui/card"
import { cn } from "@/lib/utils"

export interface ResourceMetricCardProps {
  label: string
  value: string | number
  subtitle?: string
  icon?: React.ReactNode
  className?: string
}

export function ResourceMetricCard({
  label,
  value,
  subtitle,
  icon,
  className,
}: ResourceMetricCardProps) {
  return (
    <Card className={cn("", className)}>
      <CardContent className="p-4">
        <div className="flex items-center gap-2 text-muted-foreground">
          {icon}
          <span className="text-xs font-medium">{label}</span>
        </div>
        <div className="mt-1.5 text-2xl font-semibold text-foreground">{value}</div>
        {subtitle && <div className="mt-0.5 text-xs text-muted-foreground">{subtitle}</div>}
      </CardContent>
    </Card>
  )
}
