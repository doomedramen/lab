"use client"

import { useState } from "react"
import { Loader2, Plus, HardDrive, Database } from "lucide-react"
import { Button } from "@workspace/ui/components/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@workspace/ui/components/dialog"
import { Input } from "@workspace/ui/components/input"
import { Label } from "@workspace/ui/components/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@workspace/ui/components/select"
import { Textarea } from "@workspace/ui/components/textarea"
import { Switch } from "@workspace/ui/components/switch"
import { Card, CardContent } from "@workspace/ui/components/card"
import { useBackupMutations } from "@/lib/api/mutations/backups"
import type { BackupType } from "@/lib/gen/lab/v1/backup_pb"

interface CreateBackupModalProps {
  vmid: number
  vmName: string
  trigger?: React.ReactNode
  onSuccess?: () => void
}

export function CreateBackupModal({ vmid, vmName, trigger, onSuccess }: CreateBackupModalProps) {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState("")
  const [backupType, setBackupType] = useState<BackupType>(1) // FULL
  const [storagePool, setStoragePool] = useState("default")
  const [compress, setCompress] = useState(true)
  const [retentionDays, setRetentionDays] = useState(30)

  const { createBackup, isCreating } = useBackupMutations({
    onCreateSuccess: () => {
      setOpen(false)
      setName("")
      onSuccess?.()
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    createBackup.mutate({
      vmid,
      name: name.trim() || `Backup-${vmid}-${new Date().toISOString().split('T')[0]}`,
      type: backupType,
      storagePool,
      compress,
      retentionDays,
    })
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger || (
          <Button size="sm">
            <Plus className="w-4 h-4 mr-2" />
            Create Backup
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="sm:max-w-[500px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Create Backup</DialogTitle>
            <DialogDescription>
              Create a backup of {vmName}. Backups can be restored to the same or different VM.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="backup-name">Name (optional)</Label>
              <Input
                id="backup-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Leave empty for auto-generated name"
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="backup-type">Backup Type</Label>
              <Select value={String(backupType)} onValueChange={(v) => setBackupType(Number(v) as BackupType)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="1">Full Backup</SelectItem>
                  <SelectItem value="2">Incremental Backup</SelectItem>
                  <SelectItem value="3">Snapshot Backup</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <Card>
              <CardContent className="pt-4">
                <div className="grid gap-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Database className="w-4 h-4 text-muted-foreground" />
                      <div>
                        <Label htmlFor="backup-storage" className="cursor-pointer">
                          Storage Pool
                        </Label>
                        <p className="text-xs text-muted-foreground">
                          Where to store the backup
                        </p>
                      </div>
                    </div>
                    <Select value={storagePool} onValueChange={setStoragePool}>
                      <SelectTrigger className="w-[150px]">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="default">Default</SelectItem>
                        <SelectItem value="local">Local</SelectItem>
                        <SelectItem value="nfs">NFS</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <HardDrive className="w-4 h-4 text-muted-foreground" />
                      <div>
                        <Label htmlFor="backup-compress" className="cursor-pointer">
                          Compression
                        </Label>
                        <p className="text-xs text-muted-foreground">
                          Compress backup to save space
                        </p>
                      </div>
                    </div>
                    <Switch
                      id="backup-compress"
                      checked={compress}
                      onCheckedChange={setCompress}
                    />
                  </div>

                  <div className="grid gap-2">
                    <Label htmlFor="backup-retention">Retention (days)</Label>
                    <Input
                      id="backup-retention"
                      type="number"
                      min="0"
                      value={retentionDays}
                      onChange={(e) => setRetentionDays(Number(e.target.value))}
                    />
                    <p className="text-xs text-muted-foreground">
                      0 = keep forever
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isCreating}>
              {isCreating && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
              {isCreating ? "Creating..." : "Create Backup"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
