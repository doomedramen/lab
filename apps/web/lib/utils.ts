import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/**
 * Format bytes to human-readable string
 */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B"
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB", "TB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i]
}

/**
 * Calculate download speed from bytes downloaded and start time
 */
export function calculateSpeed(downloaded: number, startTime: string): string {
  const start = new Date(startTime).getTime()
  const now = Date.now()
  const elapsedSec = (now - start) / 1000
  if (elapsedSec <= 0) return ""
  const bytesPerSec = downloaded / elapsedSec
  return formatBytes(bytesPerSec) + "/s"
}
