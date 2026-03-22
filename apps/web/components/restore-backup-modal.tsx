"use client";

import { useState } from "react";
import { Loader2, RotateCcw, AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Input } from "@/components/ui/input";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useBackupMutations } from "@/lib/api/mutations/backups";
import type { Backup } from "@/lib/gen/lab/v1/backup_pb";

interface RestoreBackupModalProps {
  backup: Backup;
  onClose: () => void;
}

export function RestoreBackupModal({
  backup,
  onClose,
}: RestoreBackupModalProps) {
  const [restoreToNewVm, setRestoreToNewVm] = useState(false);
  const [targetVmid, setTargetVmid] = useState("");
  const [startAfter, setStartAfter] = useState(false);

  const { restoreBackup, isRestoring } = useBackupMutations({
    onRestoreSuccess: () => {
      onClose();
    },
  });

  const handleSubmit = () => {
    restoreBackup.mutate({
      backupId: backup.id,
      targetVmid: restoreToNewVm ? Number(targetVmid) : 0,
      startAfter,
    });
  };

  const formatDate = (dateString: string) => {
    if (!dateString) return "Unknown";
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  const formatSize = (bytes: number) => {
    if (bytes === 0) return "Unknown";
    const gb = bytes / (1024 * 1024 * 1024);
    if (gb < 1) {
      const mb = bytes / (1024 * 1024);
      return `${mb.toFixed(1)} MB`;
    }
    return `${gb.toFixed(2)} GB`;
  };

  return (
    <Dialog open={true} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <RotateCcw className="w-5 h-5" />
            Restore Backup
          </DialogTitle>
          <DialogDescription>Restore a VM from backup</DialogDescription>
        </DialogHeader>

        <div className="py-4">
          <Alert variant="destructive" className="mb-4">
            <AlertTriangle className="w-4 h-4" />
            <AlertDescription>
              Restoring to the original VM will overwrite its current state. Any
              data not in the backup will be lost.
            </AlertDescription>
          </Alert>

          <div className="bg-muted p-4 rounded-lg space-y-2 mb-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label className="text-xs text-muted-foreground">Backup</Label>
                <p className="font-medium">{backup.name || backup.id}</p>
              </div>
              <div>
                <Label className="text-xs text-muted-foreground">Created</Label>
                <p className="font-medium">{formatDate(backup.createdAt)}</p>
              </div>
              <div>
                <Label className="text-xs text-muted-foreground">Size</Label>
                <p className="font-medium">
                  {formatSize(Number(backup.sizeBytes))}
                </p>
              </div>
              <div>
                <Label className="text-xs text-muted-foreground">Type</Label>
                <p className="font-medium capitalize">
                  {backup.type === 1
                    ? "Full"
                    : backup.type === 2
                      ? "Incremental"
                      : "Snapshot"}
                </p>
              </div>
            </div>
          </div>

          <div className="space-y-4">
            <div className="flex items-center justify-between p-3 border rounded-lg">
              <div>
                <Label htmlFor="restore-new" className="cursor-pointer">
                  Restore to different VM
                </Label>
                <p className="text-xs text-muted-foreground">
                  Create a new VM or restore to existing
                </p>
              </div>
              <Switch
                id="restore-new"
                checked={restoreToNewVm}
                onCheckedChange={setRestoreToNewVm}
              />
            </div>

            {restoreToNewVm && (
              <div className="grid gap-2">
                <Label htmlFor="target-vmid">Target VM ID</Label>
                <Input
                  id="target-vmid"
                  type="number"
                  value={targetVmid}
                  onChange={(e) => setTargetVmid(e.target.value)}
                  placeholder="Leave empty to create new VM"
                />
              </div>
            )}

            <div className="flex items-center justify-between p-3 border rounded-lg">
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
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isRestoring}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={isRestoring}>
            {isRestoring && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
            {isRestoring ? "Restoring..." : "Restore Backup"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
