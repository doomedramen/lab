"use client";

import { useState } from "react";
import {
  Clock,
  MoreVertical,
  RotateCcw,
  Trash2,
  HardDrive,
  ChevronRight,
  ChevronDown,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { CreateSnapshotModal } from "./create-snapshot-modal";
import { RestoreSnapshotModal } from "./restore-snapshot-modal";
import { useSnapshotMutations } from "@/lib/api/mutations/snapshots";
import type { Snapshot, SnapshotTree } from "@/lib/gen/lab/v1/snapshot_pb";

interface SnapshotListProps {
  vmid: number;
  vmName: string;
  vmState?: string;
  snapshots?: Snapshot[];
  tree?: SnapshotTree | null;
  isLoading?: boolean;
}

export function SnapshotList({
  vmid,
  vmName,
  vmState,
  snapshots,
  tree,
  isLoading,
}: SnapshotListProps) {
  const [expandedSnapshots, setExpandedSnapshots] = useState<Set<string>>(
    new Set(),
  );
  const [restoreSnapshot, setRestoreSnapshot] = useState<Snapshot | null>(null);

  const { deleteSnapshot, isDeleting } = useSnapshotMutations();

  const toggleExpanded = (snapshotId: string) => {
    const newExpanded = new Set(expandedSnapshots);
    if (newExpanded.has(snapshotId)) {
      newExpanded.delete(snapshotId);
    } else {
      newExpanded.add(snapshotId);
    }
    setExpandedSnapshots(newExpanded);
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

  const formatDate = (dateString: string) => {
    if (!dateString) return "Unknown";
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  const getStatusBadge = (status: number) => {
    const statusConfig: Record<
      number,
      { label: string; variant: "default" | "secondary" | "destructive" }
    > = {
      0: { label: "Unknown", variant: "secondary" },
      1: { label: "Creating", variant: "secondary" },
      2: { label: "Ready", variant: "default" },
      3: { label: "Deleting", variant: "secondary" },
      4: { label: "Error", variant: "destructive" },
    };
    const config = statusConfig[status] ?? {
      label: "Unknown",
      variant: "secondary" as const,
    };
    return <Badge variant={config.variant}>{config.label}</Badge>;
  };

  const handleDelete = (snapshotId: string) => {
    if (
      confirm(
        "Are you sure you want to delete this snapshot? This action cannot be undone.",
      )
    ) {
      deleteSnapshot.mutate({ vmid, snapshotId });
    }
  };

  const renderSnapshotTree = (
    treeNode: SnapshotTree | null | undefined,
    level: number = 0,
  ) => {
    if (!treeNode) return null;

    const snapshot = treeNode.snapshot;
    if (!snapshot) return null;

    const isExpanded = expandedSnapshots.has(snapshot.id);
    const hasChildren = treeNode.children && treeNode.children.length > 0;

    return (
      <div key={snapshot.id}>
        <div
          className={cn(
            "flex items-center gap-2 p-3 rounded-lg border bg-card hover:bg-accent/50 transition-colors",
            level > 0 && "ml-6",
          )}
          style={{ marginLeft: `${level * 24}px` }}
        >
          {hasChildren ? (
            <Button
              variant="ghost"
              size="sm"
              className="h-6 w-6 p-0"
              onClick={() => toggleExpanded(snapshot.id)}
            >
              {isExpanded ? (
                <ChevronDown className="w-4 h-4" />
              ) : (
                <ChevronRight className="w-4 h-4" />
              )}
            </Button>
          ) : (
            <div className="w-6" />
          )}

          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="font-medium truncate">{snapshot.name}</span>
              {getStatusBadge(snapshot.status)}
              {snapshot.hasChildren && (
                <Badge variant="secondary" className="text-xs">
                  {treeNode.children?.length || 0} children
                </Badge>
              )}
            </div>
            <div className="flex items-center gap-4 mt-1 text-xs text-muted-foreground">
              <span className="flex items-center gap-1">
                <Clock className="w-3 h-3" />
                {formatDate(snapshot.createdAt)}
              </span>
              <span className="flex items-center gap-1">
                <HardDrive className="w-3 h-3" />
                {formatSize(Number(snapshot.sizeBytes))}
              </span>
              {snapshot.description && (
                <span className="truncate max-w-md">
                  {snapshot.description}
                </span>
              )}
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setRestoreSnapshot(snapshot)}
              disabled={isDeleting}
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
                <DropdownMenuItem onClick={() => setRestoreSnapshot(snapshot)}>
                  <RotateCcw className="w-4 h-4 mr-2" />
                  Restore to this snapshot
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => handleDelete(snapshot.id)}
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

        {isExpanded && hasChildren && (
          <div className="mt-2">
            {treeNode.children?.map((child) =>
              renderSnapshotTree(child, level + 1),
            )}
          </div>
        )}
      </div>
    );
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Clock className="w-6 h-6 animate-spin text-muted-foreground" />
        <span className="ml-2 text-muted-foreground">Loading snapshots...</span>
      </div>
    );
  }

  if (!snapshots || snapshots.length === 0) {
    return (
      <div className="text-center py-8">
        <Clock className="w-12 h-12 mx-auto text-muted-foreground/50" />
        <h3 className="mt-4 text-lg font-medium">No snapshots yet</h3>
        <p className="mt-2 text-muted-foreground">
          Create a snapshot to capture the current state of your VM
        </p>
        <div className="mt-4">
          <CreateSnapshotModal vmid={vmid} vmName={vmName} vmState={vmState} />
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {tree
        ? renderSnapshotTree(tree)
        : snapshots.map((snapshot) => (
            <div
              key={snapshot.id}
              className="flex items-center gap-4 p-3 rounded-lg border bg-card"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="font-medium">{snapshot.name}</span>
                  {getStatusBadge(snapshot.status)}
                </div>
                <div className="flex items-center gap-4 mt-1 text-xs text-muted-foreground">
                  <span className="flex items-center gap-1">
                    <Clock className="w-3 h-3" />
                    {formatDate(snapshot.createdAt)}
                  </span>
                  <span className="flex items-center gap-1">
                    <HardDrive className="w-3 h-3" />
                    {formatSize(Number(snapshot.sizeBytes))}
                  </span>
                </div>
              </div>

              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setRestoreSnapshot(snapshot)}
                >
                  <RotateCcw className="w-3 h-3 mr-1" />
                  Restore
                </Button>

                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleDelete(snapshot.id)}
                >
                  <Trash2 className="w-4 h-4" />
                </Button>
              </div>
            </div>
          ))}

      {restoreSnapshot && (
        <RestoreSnapshotModal
          vmid={vmid}
          vmName={vmName}
          snapshot={restoreSnapshot}
          onClose={() => setRestoreSnapshot(null)}
        />
      )}
    </div>
  );
}
