"use client"

import { useTasks, useCancelTask, timestampToDate } from "@/lib/api/queries/tasks"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@workspace/ui/components/table"
import { Badge } from "@workspace/ui/components/badge"
import { Button } from "@workspace/ui/components/button"
import { Progress } from "@workspace/ui/components/progress"
import { Shimmer } from "@workspace/components/shimmer"
import { RefreshCw, XCircle, CheckCircle, AlertCircle, Clock, Activity } from "lucide-react"
import { useState } from "react"
import { TaskStatus } from "@/lib/gen/lab/v1/task_pb"
import { formatDistanceToNow } from "date-fns"
import { toast } from "sonner"

function TaskStatusBadge({ status }: { status: TaskStatus }) {
  const config: Record<TaskStatus, { label: string; variant: "default" | "secondary" | "destructive" | "outline"; icon: React.ReactNode }> = {
    [TaskStatus.UNSPECIFIED]: { label: "Unknown", variant: "secondary", icon: <AlertCircle className="h-3 w-3" /> },
    [TaskStatus.PENDING]: { label: "Pending", variant: "secondary", icon: <Clock className="h-3 w-3" /> },
    [TaskStatus.RUNNING]: { label: "Running", variant: "default", icon: <Activity className="h-3 w-3 animate-pulse" /> },
    [TaskStatus.COMPLETED]: { label: "Completed", variant: "outline", icon: <CheckCircle className="h-3 w-3 text-green-600" /> },
    [TaskStatus.FAILED]: { label: "Failed", variant: "destructive", icon: <AlertCircle className="h-3 w-3" /> },
    [TaskStatus.CANCELLED]: { label: "Cancelled", variant: "secondary", icon: <XCircle className="h-3 w-3" /> },
  }

  const { label, variant, icon } = config[status] || config[TaskStatus.UNSPECIFIED]

  return (
    <Badge variant={variant} className="gap-1">
      {icon}
      {label}
    </Badge>
  )
}

function TaskTypeBadge({ type }: { type: number }) {
  const typeLabels: Record<number, string> = {
    0: "Unknown",
    1: "Backup",
    2: "Restore",
    3: "Snapshot Create",
    4: "Snapshot Delete",
    5: "Snapshot Restore",
    6: "Clone",
    7: "Migration",
    8: "Import",
    9: "Export",
  }

  return <Badge variant="secondary">{typeLabels[type] || "Unknown"}</Badge>
}

function ResourceTypeBadge({ resourceType }: { resourceType: number }) {
  const typeLabels: Record<number, string> = {
    0: "Unknown",
    1: "VM",
    2: "Container",
    3: "Stack",
    4: "Backup",
    5: "Snapshot",
    6: "ISO",
    7: "Network",
    8: "Storage",
  }

  return <Badge variant="outline">{typeLabels[resourceType] || "Unknown"}</Badge>
}

function TasksContent() {
  const [activeOnly, setActiveOnly] = useState(true)
  const { data: tasks, isLoading, error, refetch } = useTasks({ activeOnly })
  const cancelTask = useCancelTask()

  const handleCancel = (taskId: string) => {
    if (confirm("Are you sure you want to cancel this task?")) {
      cancelTask.mutate(taskId)
    }
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="rounded-md border border-red-200 bg-red-50 p-4">
          <h3 className="text-sm font-medium text-red-800">Failed to load tasks</h3>
          <p className="mt-1 text-sm text-red-600">{error.message}</p>
          <Button variant="outline" size="sm" className="mt-2" onClick={() => refetch()}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Retry
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-foreground">Tasks</h1>
          <p className="text-sm text-muted-foreground mt-1">Monitor and manage async operations</p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant={activeOnly ? "default" : "outline"}
            size="sm"
            onClick={() => setActiveOnly(!activeOnly)}
          >
            {activeOnly ? "Active Only" : "All Tasks"}
          </Button>
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
        </div>
      </div>

      {/* Tasks Table */}
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>Resource</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="w-[200px]">Progress</TableHead>
              <TableHead>Started</TableHead>
              <TableHead>Duration</TableHead>
              <TableHead className="w-[100px]">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              // Shimmer loading rows
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell><Shimmer loading><div className="h-4 w-32" /></Shimmer></TableCell>
                  <TableCell><Shimmer loading><div className="h-4 w-24" /></Shimmer></TableCell>
                  <TableCell><Shimmer loading><div className="h-4 w-20" /></Shimmer></TableCell>
                  <TableCell><Shimmer loading><div className="h-4 w-20" /></Shimmer></TableCell>
                  <TableCell><Shimmer loading><div className="h-2 w-full" /></Shimmer></TableCell>
                  <TableCell><Shimmer loading><div className="h-4 w-28" /></Shimmer></TableCell>
                  <TableCell><Shimmer loading><div className="h-4 w-16" /></Shimmer></TableCell>
                  <TableCell><Shimmer loading><div className="h-8 w-20" /></Shimmer></TableCell>
                </TableRow>
              ))
            ) : tasks && tasks.length > 0 ? (
              tasks.map((task) => (
                <TableRow key={task.id}>
                  <TableCell className="font-mono text-xs">{task.id.slice(0, 8)}...</TableCell>
                  <TableCell>
                    <TaskTypeBadge type={task.type} />
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <ResourceTypeBadge resourceType={task.resourceType} />
                      <span className="text-sm text-muted-foreground">{task.resourceId}</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <TaskStatusBadge status={task.status} />
                  </TableCell>
                  <TableCell>
                    <div className="space-y-1">
                      <div className="flex items-center justify-between text-xs">
                        <span className="text-muted-foreground">{task.progress}%</span>
                      </div>
                      <Progress value={task.progress} className="h-2" />
                    </div>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {task.createdAt && (
                      formatDistanceToNow(timestampToDate(task.createdAt), { addSuffix: true })
                    )}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {task.createdAt && task.completedAt ? (
                      formatDistanceToNow(timestampToDate(task.createdAt), { addSuffix: false })
                    ) : task.status === TaskStatus.RUNNING ? (
                      <span className="text-green-600">Running...</span>
                    ) : (
                      "-"
                    )}
                  </TableCell>
                  <TableCell>
                    {task.status === TaskStatus.RUNNING || task.status === TaskStatus.PENDING ? (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleCancel(task.id)}
                        disabled={cancelTask.isPending}
                      >
                        <XCircle className="h-4 w-4" />
                      </Button>
                    ) : (
                      "-"
                    )}
                  </TableCell>
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={8} className="h-24 text-center">
                  <div className="flex flex-col items-center gap-2 text-muted-foreground">
                    <Activity className="h-8 w-8 opacity-50" />
                    <p>No tasks found</p>
                  </div>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      {/* Task Message/Details */}
      {tasks && tasks.some((t) => t.message) && (
        <div className="rounded-md border p-4">
          <h3 className="text-sm font-medium mb-2">Latest Task Messages</h3>
          <div className="space-y-2">
            {tasks
              .filter((t) => t.message)
              .slice(0, 5)
              .map((task) => (
                <div key={task.id} className="flex items-start gap-2 text-sm">
                  <Badge variant="outline" className="shrink-0">
                    {task.id.slice(0, 8)}
                  </Badge>
                  <span className="text-muted-foreground">{task.message}</span>
                  {task.error && (
                    <span className="text-red-600 text-xs ml-auto">{task.error}</span>
                  )}
                </div>
              ))}
          </div>
        </div>
      )}
    </div>
  )
}

export default function TasksPage() {
  return <TasksContent />
}
