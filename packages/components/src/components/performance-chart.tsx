"use client"

import { Card, CardContent, CardHeader, CardTitle } from "@workspace/ui/components/card"
import { MetricAreaChart, type MetricAreaChartProps } from "@workspace/components/metric-area-chart"
import { LucideIcon } from "lucide-react"
import { cn } from "@workspace/components/lib/utils"

export interface PerformanceChartProps {
  title: string
  icon?: LucideIcon
  iconClassName?: string
  data: MetricAreaChartProps["data"]
  color: MetricAreaChartProps["color"]
  height?: number
  yDomain?: MetricAreaChartProps["yDomain"]
  xAxisInterval?: number
  className?: string
  /** Label shown in tooltip for the value */
  tooltipLabel?: string
  /** Unit appended to value in tooltip (e.g., "%", " MB/s") */
  tooltipUnit?: string
}

export function PerformanceChart({
  title,
  icon: Icon,
  iconClassName,
  data,
  color,
  height = 200,
  yDomain,
  xAxisInterval,
  className,
  tooltipLabel,
  tooltipUnit,
}: PerformanceChartProps) {
  return (
    <Card className={className}>
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-2 text-sm font-medium">
          {Icon && <Icon className={cn("size-4 text-primary", iconClassName)} />}
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <MetricAreaChart
          data={data}
          color={color}
          height={height}
          yDomain={yDomain}
          xAxisInterval={xAxisInterval}
          tooltipLabel={tooltipLabel}
          tooltipUnit={tooltipUnit}
        />
      </CardContent>
    </Card>
  )
}
