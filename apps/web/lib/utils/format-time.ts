/**
 * Shared formatting utilities for dates and times
 */

const isToday = (dateStr: string): boolean => {
  const date = new Date(dateStr)
  const today = new Date()
  return date.toDateString() === today.toDateString()
}

/**
 * Format time for display in charts.
 * Shows time only if today, otherwise includes date.
 */
export function formatChartTime(isoStr: string): string {
  const date = new Date(isoStr)

  if (isToday(isoStr)) {
    // Same day - just show time
    return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
  }

  // Different day - show date and time
  return date.toLocaleDateString([], {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  })
}
