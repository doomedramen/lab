"use client"

import { useState, useMemo, useRef, useEffect } from "react"
import { Badge } from "@workspace/ui/components/badge"
import { Button } from "@workspace/ui/components/button"
import { Input } from "@workspace/ui/components/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@workspace/ui/components/select"
import { Checkbox } from "@workspace/ui/components/checkbox"
import type { VMLogEntry } from "@/lib/gen/lab/v1/vm_pb"
import { VMLogLevel } from "@/lib/gen/lab/v1/vm_pb"
import { Clock, Filter, ChevronDown, ChevronRight, ArrowDown, RefreshCw } from "lucide-react"
import { cn } from "@workspace/ui/lib/utils"

interface LogViewerProps {
  entries: VMLogEntry[]
  isLoading?: boolean
  onRefresh?: () => void
}

const LOG_LEVEL_COLORS: Record<VMLogLevel, string> = {
  [VMLogLevel.VM_LOG_LEVEL_UNSPECIFIED]: "bg-gray-500",
  [VMLogLevel.VM_LOG_LEVEL_DEBUG]: "bg-gray-500",
  [VMLogLevel.VM_LOG_LEVEL_INFO]: "bg-blue-500",
  [VMLogLevel.VM_LOG_LEVEL_WARNING]: "bg-yellow-500",
  [VMLogLevel.VM_LOG_LEVEL_ERROR]: "bg-orange-500",
  [VMLogLevel.VM_LOG_LEVEL_CRITICAL]: "bg-red-500",
}

const LOG_LEVEL_LABELS: Record<VMLogLevel, string> = {
  [VMLogLevel.VM_LOG_LEVEL_UNSPECIFIED]: "UNSPECIFIED",
  [VMLogLevel.VM_LOG_LEVEL_DEBUG]: "DEBUG",
  [VMLogLevel.VM_LOG_LEVEL_INFO]: "INFO",
  [VMLogLevel.VM_LOG_LEVEL_WARNING]: "WARNING",
  [VMLogLevel.VM_LOG_LEVEL_ERROR]: "ERROR",
  [VMLogLevel.VM_LOG_LEVEL_CRITICAL]: "CRITICAL",
}

const SOURCE_COLORS: Record<string, string> = {
  qemu: "bg-purple-500",
  "cloud-init": "bg-green-500",
  kernel: "bg-blue-500",
  systemd: "bg-orange-500",
  default: "bg-gray-500",
}

function getLogLevelColor(level: VMLogLevel): string {
  return LOG_LEVEL_COLORS[level] ?? LOG_LEVEL_COLORS[VMLogLevel.VM_LOG_LEVEL_UNSPECIFIED]
}

function getLogLevelLabel(level: VMLogLevel): string {
  return LOG_LEVEL_LABELS[level] ?? "UNKNOWN"
}

function getSourceColor(source: string): string {
  const color = SOURCE_COLORS[source as keyof typeof SOURCE_COLORS]
  return (color ?? SOURCE_COLORS.default) as string
}

function formatTimestamp(timestamp: string): string {
  try {
    const date = new Date(timestamp)
    return date.toLocaleTimeString()
  } catch {
    return timestamp
  }
}

interface LogEntryRowProps {
  entry: VMLogEntry
  isExpanded: boolean
  onToggle: () => void
}

function LogEntryRow({ entry, isExpanded, onToggle }: LogEntryRowProps) {
  const levelColor = getLogLevelColor(entry.level)
  const sourceColor = getSourceColor(entry.source)

  return (
    <div className="border-b border-border/50">
      <div
        className="flex items-start gap-2 p-2 hover:bg-secondary/50 cursor-pointer text-xs font-mono"
        onClick={onToggle}
      >
        <div className="shrink-0 mt-0.5">
          {isExpanded ? (
            <ChevronDown className="size-3 text-muted-foreground" />
          ) : (
            <ChevronRight className="size-3 text-muted-foreground" />
          )}
        </div>
        
        <div className="shrink-0">
          <Badge variant="outline" className={`${levelColor} text-white border-0 text-[10px] px-1.5 py-0 h-4`}>
            {getLogLevelLabel(entry.level)}
          </Badge>
        </div>
        
        <div className="shrink-0 text-muted-foreground flex items-center gap-1">
          <Clock className="size-3" />
          {formatTimestamp(entry.timestamp)}
        </div>
        
        <div className="shrink-0">
          <Badge variant="outline" className={`${sourceColor} text-white border-0 text-[10px] px-1.5 py-0 h-4`}>
            {entry.source}
          </Badge>
        </div>
        
        <div className="flex-1 truncate text-foreground">
          {entry.message}
        </div>
      </div>
      
      {isExpanded && entry.metadata && Object.keys(entry.metadata).length > 0 && (
        <div className="px-4 py-2 bg-secondary/30 text-xs font-mono">
          <div className="text-muted-foreground mb-1">Metadata:</div>
          <pre className="whitespace-pre-wrap text-foreground">
            {JSON.stringify(entry.metadata, null, 2)}
          </pre>
        </div>
      )}
    </div>
  )
}

export function LogViewer({ entries, isLoading, onRefresh }: LogViewerProps) {
  const [searchQuery, setSearchQuery] = useState("")
  const [selectedLevels, setSelectedLevels] = useState<VMLogLevel[]>([])
  const [selectedSources, setSelectedSources] = useState<string[]>([])
  const [expandedLogs, setExpandedLogs] = useState<Set<string>>(new Set())
  const [autoScroll, setAutoScroll] = useState(true)
  const scrollViewportRef = useRef<HTMLDivElement>(null)

  // Extract unique sources from logs
  const uniqueSources = useMemo(() => {
    const sources = new Set(entries.map((e) => e.source))
    return Array.from(sources).sort()
  }, [entries])

  // Sort logs by timestamp (newest first) and filter
  const filteredEntries = useMemo(() => {
    // First sort by timestamp (newest first)
    const sorted = [...entries].sort((a, b) => {
      const timeA = new Date(a.timestamp).getTime()
      const timeB = new Date(b.timestamp).getTime()
      return timeB - timeA // Descending order (newest first)
    })

    return sorted.filter((entry) => {
      // Search filter
      if (searchQuery && !entry.message.toLowerCase().includes(searchQuery.toLowerCase())) {
        return false
      }

      // Level filter
      if (selectedLevels.length > 0 && !selectedLevels.includes(entry.level)) {
        return false
      }

      // Source filter
      if (selectedSources.length > 0 && !selectedSources.includes(entry.source)) {
        return false
      }

      return true
    })
  }, [entries, searchQuery, selectedLevels, selectedSources])

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (autoScroll && scrollViewportRef.current) {
      scrollViewportRef.current.scrollTop = scrollViewportRef.current.scrollHeight
    }
  }, [filteredEntries, autoScroll])

  const toggleLogLevel = (level: VMLogLevel) => {
    setSelectedLevels((prev) =>
      prev.includes(level) ? prev.filter((l) => l !== level) : [...prev, level]
    )
  }

  const toggleSource = (source: string) => {
    setSelectedSources((prev) =>
      prev.includes(source) ? prev.filter((s) => s !== source) : [...prev, source]
    )
  }

  const toggleExpandLog = (logId: string) => {
    setExpandedLogs((prev) => {
      const next = new Set(prev)
      if (next.has(logId)) {
        next.delete(logId)
      } else {
        next.add(logId)
      }
      return next
    })
  }

  const clearFilters = () => {
    setSearchQuery("")
    setSelectedLevels([])
    setSelectedSources([])
  }

  const hasActiveFilters = searchQuery || selectedLevels.length > 0 || selectedSources.length > 0

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center gap-2 p-2 border-b border-border shrink-0">
        <div className="relative flex-1">
          <Input
            placeholder="Search logs..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="h-8 pl-8 pr-8 text-xs"
          />
          <Filter className="absolute left-2 top-1/2 -translate-y-1/2 size-3 text-muted-foreground" />
          {searchQuery && (
            <Button
              variant="ghost"
              size="sm"
              className="absolute right-1 top-1/2 -translate-y-1/2 h-6 w-6 p-0"
              onClick={() => setSearchQuery("")}
            >
              ×
            </Button>
          )}
        </div>

        <Select
          value={selectedLevels.length.toString()}
          onValueChange={(val) => {
            if (val === "all") setSelectedLevels([])
            else if (val === "errors") setSelectedLevels([VMLogLevel.VM_LOG_LEVEL_ERROR, VMLogLevel.VM_LOG_LEVEL_CRITICAL])
            else if (val === "warnings") setSelectedLevels([VMLogLevel.VM_LOG_LEVEL_WARNING, VMLogLevel.VM_LOG_LEVEL_ERROR, VMLogLevel.VM_LOG_LEVEL_CRITICAL])
          }}
        >
          <SelectTrigger className="h-8 w-32 text-xs">
            <SelectValue placeholder="Log Levels" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Levels</SelectItem>
            <SelectItem value="errors">Errors Only</SelectItem>
            <SelectItem value="warnings">Warnings+</SelectItem>
          </SelectContent>
        </Select>

        <Button
          variant="outline"
          size="sm"
          className="h-8 text-xs gap-1"
          onClick={() => setAutoScroll(!autoScroll)}
        >
          <ArrowDown className={cn("size-3", autoScroll && "text-primary")} />
          Auto-scroll
        </Button>

        <Button
          variant="outline"
          size="sm"
          className="h-8 w-8 p-0"
          onClick={onRefresh}
          disabled={isLoading}
        >
          <RefreshCw className={cn("size-3", isLoading && "animate-spin")} />
        </Button>

        {hasActiveFilters && (
          <Button variant="ghost" size="sm" className="h-8 text-xs" onClick={clearFilters}>
            Clear filters
          </Button>
        )}
      </div>

      {/* Filter chips */}
      {(selectedLevels.length > 0 || selectedSources.length > 0) && (
        <div className="flex items-center gap-2 p-2 border-b border-border shrink-0 flex-wrap">
          {selectedLevels.map((level) => (
            <Badge
              key={level}
              variant="secondary"
              className="gap-1 cursor-pointer"
              onClick={() => toggleLogLevel(level)}
            >
              {getLogLevelLabel(level)} ×
            </Badge>
          ))}
          {selectedSources.map((source) => (
            <Badge
              key={source}
              variant="secondary"
              className="gap-1 cursor-pointer"
              onClick={() => toggleSource(source)}
            >
              {source} ×
            </Badge>
          ))}
        </div>
      )}

      {/* Source filter dropdown */}
      {uniqueSources.length > 0 && (
        <div className="flex items-center gap-2 p-2 border-b border-border shrink-0 flex-wrap">
          <span className="text-xs text-muted-foreground">Sources:</span>
          {uniqueSources.map((source) => (
            <div key={source} className="flex items-center gap-1">
              <Checkbox
                id={`source-${source}`}
                checked={!selectedSources.includes(source)}
                onCheckedChange={() => toggleSource(source)}
                className="size-3"
              />
              <label
                htmlFor={`source-${source}`}
                className="text-xs cursor-pointer flex items-center gap-1"
              >
                <div className={`size-2 rounded-full ${getSourceColor(source)}`} />
                {source}
              </label>
            </div>
          ))}
        </div>
      )}

      {/* Log entries */}
      <div className="flex-1 overflow-auto" ref={scrollViewportRef}>
        <div className="min-h-full py-4">
          {isLoading && entries.length === 0 ? (
            <div className="flex items-center justify-center h-full">
              <div className="text-center text-muted-foreground">
                <RefreshCw className="size-6 animate-spin mx-auto mb-2" />
                <p className="text-sm">Loading logs...</p>
              </div>
            </div>
          ) : filteredEntries.length === 0 ? (
            <div className="flex items-center justify-center h-full">
              <div className="text-center text-muted-foreground">
                <p className="text-sm">No logs found</p>
                {hasActiveFilters && (
                  <Button variant="link" onClick={clearFilters} className="text-xs">
                    Clear filters
                  </Button>
                )}
              </div>
            </div>
          ) : (
            filteredEntries.map((entry) => (
              <LogEntryRow
                key={entry.id}
                entry={entry}
                isExpanded={expandedLogs.has(entry.id)}
                onToggle={() => toggleExpandLog(entry.id)}
              />
            ))
          )}
        </div>
      </div>

      {/* Footer with count */}
      <div className="flex items-center justify-between p-2 border-t border-border shrink-0 text-xs text-muted-foreground">
        <span>
          {filteredEntries.length} of {entries.length} logs
        </span>
        <span>Auto-refresh: 5s</span>
      </div>
    </div>
  )
}
