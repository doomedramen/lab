"use client";

import { useState } from "react";
import { Plus, Loader2, Clock, HardDrive, Server } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useSnapshotMutations } from "@/lib/api/mutations/snapshots";
import type { Snapshot } from "@/lib/gen/lab/v1/snapshot_pb";

interface CreateSnapshotModalProps {
  vmid: number;
  vmName: string;
  vmState?: string;
  trigger?: React.ReactNode;
  onSuccess?: () => void;
}

export function CreateSnapshotModal({
  vmid,
  vmName,
  vmState,
  trigger,
  onSuccess,
}: CreateSnapshotModalProps) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [live, setLive] = useState(true);
  const [includeMemory, setIncludeMemory] = useState(false);

  const { createSnapshot, isCreating } = useSnapshotMutations({
    onCreateSuccess: () => {
      setOpen(false);
      setName("");
      setDescription("");
      setLive(true);
      setIncludeMemory(false);
      onSuccess?.();
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!name.trim()) {
      return;
    }

    createSnapshot.mutate({
      vmid,
      name: name.trim(),
      description: description.trim(),
      live: live && vmState === "running",
      includeMemory: includeMemory && live && vmState === "running",
    });
  };

  const isVMRunning = vmState === "running";

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger || (
          <Button size="sm">
            <Plus className="w-4 h-4 mr-2" />
            Create Snapshot
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="sm:max-w-[500px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Create Snapshot</DialogTitle>
            <DialogDescription>
              Create a point-in-time snapshot of {vmName}. Snapshots capture the
              disk state and optionally memory.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="snapshot-name">Name</Label>
              <Input
                id="snapshot-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g., before-update"
                required
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="snapshot-description">
                Description (optional)
              </Label>
              <Textarea
                id="snapshot-description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Describe what this snapshot is for..."
                rows={3}
              />
            </div>

            <Card>
              <CardContent className="pt-4">
                <div className="grid gap-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Server className="w-4 h-4 text-muted-foreground" />
                      <div>
                        <Label
                          htmlFor="snapshot-live"
                          className="cursor-pointer"
                        >
                          Live Snapshot
                        </Label>
                        <p className="text-xs text-muted-foreground">
                          Create snapshot while VM is running
                        </p>
                      </div>
                    </div>
                    <Switch
                      id="snapshot-live"
                      checked={live && isVMRunning}
                      onCheckedChange={setLive}
                      disabled={!isVMRunning}
                    />
                  </div>

                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Clock className="w-4 h-4 text-muted-foreground" />
                      <div>
                        <Label
                          htmlFor="snapshot-memory"
                          className="cursor-pointer"
                        >
                          Include Memory
                        </Label>
                        <p className="text-xs text-muted-foreground">
                          Capture VM memory state (requires live snapshot)
                        </p>
                      </div>
                    </div>
                    <Switch
                      id="snapshot-memory"
                      checked={includeMemory}
                      onCheckedChange={setIncludeMemory}
                      disabled={!live || !isVMRunning}
                    />
                  </div>
                </div>
              </CardContent>
            </Card>

            {!isVMRunning && (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Clock className="w-4 h-4" />
                <span>
                  VM is stopped. Snapshot will capture disk state only.
                </span>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isCreating || !name.trim()}>
              {isCreating && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
              {isCreating ? "Creating..." : "Create Snapshot"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
