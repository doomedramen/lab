import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { snapshotClient } from "../client";

interface UseSnapshotMutationsOptions {
  onCreateSuccess?: () => void;
  onDeleteSuccess?: () => void;
  onRestoreSuccess?: () => void;
}

export function useSnapshotMutations({
  onCreateSuccess,
  onDeleteSuccess,
  onRestoreSuccess,
}: UseSnapshotMutationsOptions = {}) {
  const queryClient = useQueryClient();

  const createSnapshot = useMutation({
    mutationFn: (data: Parameters<typeof snapshotClient.createSnapshot>[0]) =>
      snapshotClient.createSnapshot(data),
    onSuccess: (res) => {
      toast.success(`Snapshot "${res.snapshot?.name}" created successfully`);
      queryClient.invalidateQueries({ queryKey: ["snapshots"] });
      queryClient.invalidateQueries({ queryKey: ["vm-diagnostics"] });
      onCreateSuccess?.();
    },
    onError: (error: Error) => {
      toast.error(`Failed to create snapshot: ${error.message}`);
    },
  });

  const deleteSnapshot = useMutation({
    mutationFn: ({
      vmid,
      snapshotId,
      includeChildren,
    }: {
      vmid: number;
      snapshotId: string;
      includeChildren?: boolean;
    }) => snapshotClient.deleteSnapshot({ vmid, snapshotId, includeChildren }),
    onSuccess: () => {
      toast.success("Snapshot deleted successfully");
      queryClient.invalidateQueries({ queryKey: ["snapshots"] });
      queryClient.invalidateQueries({ queryKey: ["vm-diagnostics"] });
      onDeleteSuccess?.();
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete snapshot: ${error.message}`);
    },
  });

  const restoreSnapshot = useMutation({
    mutationFn: ({
      vmid,
      snapshotId,
      startAfter,
    }: {
      vmid: number;
      snapshotId: string;
      startAfter?: boolean;
    }) => snapshotClient.restoreSnapshot({ vmid, snapshotId, startAfter }),
    onSuccess: () => {
      toast.success("Snapshot restored successfully");
      queryClient.invalidateQueries({ queryKey: ["snapshots"] });
      queryClient.invalidateQueries({ queryKey: ["vms"] });
      queryClient.invalidateQueries({ queryKey: ["vm-diagnostics"] });
      onRestoreSuccess?.();
    },
    onError: (error: Error) => {
      toast.error(`Failed to restore snapshot: ${error.message}`);
    },
  });

  return {
    createSnapshot,
    deleteSnapshot,
    restoreSnapshot,
    isCreating: createSnapshot.isPending,
    isDeleting: deleteSnapshot.isPending,
    isRestoring: restoreSnapshot.isPending,
  };
}
