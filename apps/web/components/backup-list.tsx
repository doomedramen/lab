"use client"

import { useState } from "react"
import { Clock, MoreVertical, RotateCcw, Trash2, HardDrive, FileArchive } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu"
import { Badge } from "@/components/ui/badge"
import { CreateBackupModal } from "./create-backup-modal"
import { RestoreBackupModal } from "./restore-backup-modal"
import { useBackupMutations } from "@/lib/api/mutations/backups"
import type { Backup } from "@/lib/gen/lab/v1/backup_pb"

interface BackupListProps {
  vmid?: number
  backups?: Backup[]
  isLoading?: boolean
}

export function BackupList({ vmid, backups, isLoading }: BackupListProps) {
  const [restoreBackup, setRestoreBackup] = useState<Backup | null>(null)
  const { deleteBackup, isDeleting } = useBackupMutations()

  const formatSize = (bytes: number) => {
    if (bytes === 0) return "Unknown"
    const gb = bytes / (1024 * 1024 * 1024)
    if (gb < 1) {
      const mb = bytes / (1024 * 1024)
      return `${mb.toFixed(1)} MB`
    }
    return `${gb.toFixed(2)} GB`
  }

  const formatDate = (dateString: string) => {
    if (!dateString) return "Unknown"
    const date = new Date(dateString)
    return date.toLocaleString()
  }

  const getStatusBadge = (status: number) => {
    const statusMap: Record<number, { label: string; variant: "default" | "secondary" | "destructive" }> = {
      0: { label: "Unknown", variant: "secondary" },
      1: { label: "Pending", variant: "secondary" },
      2: { label: "Running", variant: "secondary" },
      3: { label: "Completed", variant: "default" },
      4: { label: "Failed", variant: "destructive" },
      5: { label: "Deleting", variant: "secondary" },
    }
    const config = statusMap[status] ?? { label: "Unknown", variant: "secondary" as const }
    return <Badge variant={config.variant}>{config.label}</Badge>
  }

  const getTypeBadge = (type: number) => {
    const typeMap: Record<number, string> = {
      0: "Unknown",
      1: "Full",
      2: "Incremental",
      3: "Snapshot",
    }
    return <Badge variant="outline">{typeMap[type] ?? "Unknown"}</Badge>
  }

  const handleDelete = (backupId: string) => {
    if (confirm("Are you sure you want to delete this backup? This action cannot be undone.")) {
      deleteBackup.mutate(backupId)
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Clock className="w-6 h-6 animate-spin text-muted-foreground" />
        <span className="ml-2 text-muted-foreground">Loading backups...</span>
      </div>
    )
  }

  if (!backups || backups.length === 0) {
    return (
      <div className="text-center py-8">
        <FileArchive className="w-12 h-12 mx-auto text-muted-foreground/50" />
        <h3 className="mt-4 text-lg font-medium">No backups yet</h3>
        <p className="mt-2 text-muted-foreground">
          Create a backup to protect your VM data
        </p>
        {vmid && (
          <div className="mt-4">
            <CreateBackupModal vmid={vmid} vmName="this VM" />
          </div>
        )}
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {backups.map((backup) => (
        <div
          key={backup.id}
          className="flex items-center gap-4 p-3 rounded-lg border bg-card"
        >
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="font-medium">{backup.name || backup.id}</span>
              {getStatusBadge(backup.status)}
              {getTypeBadge(backup.type)}
            </div>
            <div className="flex items-center gap-4 mt-1 text-xs text-muted-foreground">
              <span className="flex items-center gap-1">
                <Clock className="w-3 h-3" />
                {formatDate(backup.createdAt)}
              </span>
              <span className="flex items-center gap-1">
                <HardDrive className="w-3 h-3" />
                {formatSize(Number(backup.sizeBytes))}
              </span>
              {backup.retentionDays > 0 && (
                <span>Expires: {formatDate(backup.expiresAt)}</span>
              )}
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setRestoreBackup(backup)}
            >
              <RotateCcw className="w-3 h-3 mr-1" />
              Restore
            </Button>

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                  <MoreVertical className="w-4 h-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => setRestoreBackup(backup)}>
                  <RotateCcw className="w-4 h-4 mr-2" />
                  Restore backup
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => handleDelete(backup.id)}
                  className="text-destructive"
                  disabled={isDeleting}
                >
                  <Trash2 className="w-4 h-4 mr-2" />
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      ))}

      {restoreBackup && (
        <RestoreBackupModal
          backup={restoreBackup}
          onClose={() => setRestoreBackup(null)}
        />
      )}
    </div>
  )
}
