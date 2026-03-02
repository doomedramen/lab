"use client"

import { useState } from "react"
import {
  ClipboardList,
  ChevronLeft,
  ChevronRight,
  CheckCircle2,
  XCircle,
  Search,
  X,
} from "lucide-react"
import { PageHeader } from "@/components/page-header"
import { Card, CardContent, CardHeader, CardTitle } from "@workspace/ui/components/card"
import { Button } from "@workspace/ui/components/button"
import { Input } from "@workspace/ui/components/input"
import { Badge } from "@workspace/ui/components/badge"
import { useAuditLogs } from "@/lib/api/queries"

const PAGE_SIZE = 50

function formatAction(action: string): string {
  return action
    .replace(/_/g, " ")
    .toLowerCase()
    .replace(/\b\w/g, (c) => c.toUpperCase())
}

export default function AuditPage() {
  const [page, setPage] = useState(0)
  const [actionFilter, setActionFilter] = useState("")
  const [debouncedAction, setDebouncedAction] = useState("")

  // Simple debounce via state update
  const handleActionChange = (value: string) => {
    setActionFilter(value)
    setPage(0)
    // use the value directly (no debounce overhead for admin page)
    setDebouncedAction(value)
  }

  const { data, isLoading } = useAuditLogs({
    action: debouncedAction || undefined,
    limit: PAGE_SIZE,
    offset: page * PAGE_SIZE,
  })

  const logs = data?.logs ?? []
  const total = data?.total ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        backHref="/"
        backLabel="Dashboard"
        title="Audit Logs"
        subtitle="Security and activity audit trail for all user actions"
        icon={<ClipboardList className="size-5 text-foreground" />}
      />

      {/* Stats */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Total Events</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{total.toLocaleString()}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Successful Actions</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-success">
              {logs.filter((l) => l.status === "success").length}
            </div>
            <p className="text-xs text-muted-foreground">on this page</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Failed Actions</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-destructive">
              {logs.filter((l) => l.status === "failure").length}
            </div>
            <p className="text-xs text-muted-foreground">on this page</p>
          </CardContent>
        </Card>
      </div>

      {/* Filters + Table */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between gap-4">
            <CardTitle className="text-base">Event Log</CardTitle>
            <div className="flex items-center gap-2 w-64">
              <div className="relative flex-1">
                <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground" />
                <Input
                  value={actionFilter}
                  onChange={(e) => handleActionChange(e.target.value)}
                  placeholder="Filter by action..."
                  className="pl-8 pr-8 h-8 text-sm"
                />
                {actionFilter && (
                  <button
                    onClick={() => handleActionChange("")}
                    className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  >
                    <X className="size-3" />
                  </button>
                )}
              </div>
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="divide-y">
              {Array.from({ length: 8 }).map((_, i) => (
                <div key={i} className="flex items-center gap-4 px-6 py-3">
                  <div className="h-4 w-4 rounded-full bg-muted animate-pulse shrink-0" />
                  <div className="flex-1 space-y-1.5">
                    <div className="h-3.5 w-32 rounded bg-muted animate-pulse" />
                    <div className="h-3 w-48 rounded bg-muted animate-pulse" />
                  </div>
                  <div className="h-3 w-24 rounded bg-muted animate-pulse" />
                  <div className="h-3 w-20 rounded bg-muted animate-pulse" />
                </div>
              ))}
            </div>
          ) : logs.length === 0 ? (
            <div className="py-16 text-center text-sm text-muted-foreground">
              No audit log entries{actionFilter ? " matching that filter" : ""}.
            </div>
          ) : (
            <div className="divide-y">
              {logs.map((log) => (
                <div key={log.id} className="flex items-start gap-4 px-6 py-3 hover:bg-muted/30 transition-colors">
                  {/* Status icon */}
                  <div className="shrink-0 mt-0.5">
                    {log.status === "success" ? (
                      <CheckCircle2 className="size-4 text-success" />
                    ) : (
                      <XCircle className="size-4 text-destructive" />
                    )}
                  </div>

                  {/* Main content */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="text-sm font-medium">{formatAction(log.action)}</span>
                      {log.resourceType && (
                        <Badge variant="outline" className="text-xs">
                          {log.resourceType}
                          {log.resourceId ? `: ${log.resourceId.slice(0, 8)}` : ""}
                        </Badge>
                      )}
                    </div>
                    <div className="flex items-center gap-3 mt-0.5 text-xs text-muted-foreground flex-wrap">
                      {log.userId && <span>User: {log.userId.slice(0, 8)}…</span>}
                      {log.ipAddress && <span>IP: {log.ipAddress}</span>}
                      {log.details && log.details !== "{}" && (
                        <span className="truncate max-w-xs font-mono">{log.details}</span>
                      )}
                    </div>
                  </div>

                  {/* Timestamp */}
                  <div className="shrink-0 text-right">
                    <p className="text-xs text-muted-foreground">
                      {log.createdAt
                        ? new Date(log.createdAt).toLocaleString(undefined, {
                            month: "short",
                            day: "numeric",
                            hour: "2-digit",
                            minute: "2-digit",
                            second: "2-digit",
                          })
                        : "—"}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between px-6 py-3 border-t">
              <span className="text-xs text-muted-foreground">
                Page {page + 1} of {totalPages} · {total.toLocaleString()} total events
              </span>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => Math.max(0, p - 1))}
                  disabled={page === 0}
                >
                  <ChevronLeft className="size-4" />
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
                  disabled={page >= totalPages - 1}
                >
                  Next
                  <ChevronRight className="size-4" />
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
