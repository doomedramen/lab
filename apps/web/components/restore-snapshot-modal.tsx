"use client"

import { useState } from "react"
import { Loader2, RotateCcw, AlertTriangle } from "lucide-react"
import { Button } from "@workspace/ui/components/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@workspace/ui/components/dialog"
import { Label } from "@workspace/ui/components/label"
import { Switch } from "@workspace/ui/components/switch"
import { Alert, AlertDescription } from "@workspace/ui/components/alert"
import { useSnapshotMutations } from "@/lib/api/mutations/snapshots"
import type { Snapshot } from "@/lib/gen/lab/v1/snapshot_pb"

interface RestoreSnapshotModalProps {
  vmid: number
  vmName: string
  snapshot: Snapshot
  onClose: () => void
}

export function RestoreSnapshotModal({ vmid, vmName, snapshot, onClose }: RestoreSnapshotModalProps) {
  const [startAfter, setStartAfter] = useState(false)
  const { restoreSnapshot, isRestoring } = useSnapshotMutations({
    onRestoreSuccess: () => {
      onClose()
    },
  })

  const handleSubmit = () => {
    restoreSnapshot.mutate({
      vmid,
      snapshotId: snapshot.id,
      startAfter,
    })
  }

  const formatDate = (dateString: string) => {
    if (!dateString) return "Unknown"
    const date = new Date(dateString)
    return date.toLocaleString()
  }

  return (
    <Dialog open={true} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <RotateCcw className="w-5 h-5" />
            Restore Snapshot
          </DialogTitle>
          <DialogDescription>
            Restore {vmName} to a previous state
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          <Alert variant="destructive" className="mb-4">
            <AlertTriangle className="w-4 h-4" />
            <AlertDescription>
              Restoring a snapshot will revert the VM to its state when the snapshot was created.
              Any changes made since then will be lost.
            </AlertDescription>
          </Alert>

          <div className="bg-muted p-4 rounded-lg space-y-2">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label className="text-xs text-muted-foreground">Snapshot</Label>
                <p className="font-medium">{snapshot.name}</p>
              </div>
              <div>
                <Label className="text-xs text-muted-foreground">Created</Label>
                <p className="font-medium">{formatDate(snapshot.createdAt)}</p>
              </div>
            </div>
            {snapshot.description && (
              <div>
                <Label className="text-xs text-muted-foreground">Description</Label>
                <p className="text-sm">{snapshot.description}</p>
              </div>
            )}
          </div>

          <div className="mt-4 flex items-center justify-between p-3 border rounded-lg">
            <div>
              <Label htmlFor="start-after" className="cursor-pointer">
                Start VM after restore
              </Label>
              <p className="text-xs text-muted-foreground">
                Automatically start the VM after restoring
              </p>
            </div>
            <Switch
              id="start-after"
              checked={startAfter}
              onCheckedChange={setStartAfter}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isRestoring}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={isRestoring}>
            {isRestoring && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
            {isRestoring ? "Restoring..." : "Restore Snapshot"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
