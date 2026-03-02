"use client"

import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip as RTooltip,
  ResponsiveContainer,
} from "recharts"
import { useId, useEffect, useState } from "react"

export interface MetricAreaChartProps {
  /** Chart data with time and value keys */
  data: Array<{ time: string; value: number }>
  /** Color for the stroke and gradient (oklch or any CSS color) */
  color: string
  /** Chart height in pixels */
  height?: number
  /** Y-axis domain [min, max] */
  yDomain?: [number, number]
  /** X-axis tick interval */
  xAxisInterval?: number
  /** Width reserved for Y-axis labels */
  yAxisWidth?: number
  /** Chart margin */
  margin?: { top?: number; right?: number; bottom?: number; left?: number }
  /** Show Y-axis */
  showYAxis?: boolean
  /** Show X-axis */
  showXAxis?: boolean
  /** Show Cartesian grid */
  showGrid?: boolean
  /** Line stroke width */
  strokeWidth?: number
  /** Unique ID for gradient (auto-generated if not provided) */
  gradientId?: string
  /** Label shown in tooltip for the value (default: "Value") */
  tooltipLabel?: string
  /** Unit appended to value in tooltip (e.g., "%", " MB/s") */
  tooltipUnit?: string
  /** Custom value formatter for tooltip — overrides tooltipUnit when provided */
  valueFormatter?: (value: number) => string
}

const defaultMargin = { left: 0, right: 5, top: 5, bottom: 5 }

function getCSSVariable(name: string): string {
  if (typeof document === "undefined") return ""
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

export function MetricAreaChart({
  data,
  color,
  height = 140,
  yDomain,
  xAxisInterval = 5,
  yAxisWidth = 35,
  margin = defaultMargin,
  showYAxis = true,
  showXAxis = true,
  showGrid = true,
  strokeWidth = 1.5,
  gradientId,
  tooltipLabel = "Value",
  tooltipUnit = "",
  valueFormatter,
}: MetricAreaChartProps) {
  const autoGradientId = useId()
  const finalGradientId = gradientId || `grad-${autoGradientId.replace(/:/g, "-")}`

  const [themeColors, setThemeColors] = useState({
    grid: "oklch(0.85 0.005 260)",
    tick: "oklch(0.45 0.01 260)",
    popover: "oklch(0.99 0.003 260)",
    popoverForeground: "oklch(0.13 0.005 260)",
    border: "oklch(0.88 0.005 260)",
  })

  useEffect(() => {
    setThemeColors({
      grid: getCSSVariable("--chart-grid") || "oklch(0.85 0.005 260)",
      tick: getCSSVariable("--muted-foreground") || "oklch(0.45 0.01 260)",
      popover: getCSSVariable("--popover") || "oklch(0.99 0.003 260)",
      popoverForeground: getCSSVariable("--popover-foreground") || "oklch(0.13 0.005 260)",
      border: getCSSVariable("--border") || "oklch(0.88 0.005 260)",
    })
  }, [])

  const tooltipFormatter = (value: number | undefined) => [
    value === undefined
      ? ""
      : valueFormatter
        ? valueFormatter(value)
        : `${value}${tooltipUnit}`,
    tooltipLabel,
  ]

  const tickStyle = { fontSize: 10, fill: themeColors.tick }
  const tooltipStyle = {
    background: themeColors.popover,
    border: `1px solid ${themeColors.border}`,
    borderRadius: 6,
    fontSize: 12,
    color: themeColors.popoverForeground,
  }

  return (
    <div style={{ height }}>
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data} margin={margin}>
          <defs>
            <linearGradient id={finalGradientId} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={color} stopOpacity={0.3} />
              <stop offset="95%" stopColor={color} stopOpacity={0} />
            </linearGradient>
          </defs>
          {showGrid && <CartesianGrid strokeDasharray="3 3" stroke={themeColors.grid} />}
          {showXAxis && (
            <XAxis
              dataKey="time"
              tick={tickStyle}
              interval={xAxisInterval}
              axisLine={false}
              tickLine={false}
            />
          )}
          {showYAxis && (
            <YAxis
              width={yAxisWidth}
              tick={tickStyle}
              domain={yDomain}
              axisLine={false}
              tickLine={false}
            />
          )}
          <RTooltip contentStyle={tooltipStyle} formatter={tooltipFormatter} />
          <Area
            type="monotone"
            dataKey="value"
            stroke={color}
            fill={`url(#${finalGradientId})`}
            strokeWidth={strokeWidth}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}
