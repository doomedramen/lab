import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { storageClient } from "../client";

interface UseStoragePoolMutationsOptions {
  onCreateSuccess?: () => void;
  onDeleteSuccess?: () => void;
  onUpdateSuccess?: () => void;
}

export function useStoragePoolMutations({
  onCreateSuccess,
  onDeleteSuccess,
  onUpdateSuccess,
}: UseStoragePoolMutationsOptions = {}) {
  const queryClient = useQueryClient();

  const createPool = useMutation({
    mutationFn: (data: Parameters<typeof storageClient.createStoragePool>[0]) =>
      storageClient.createStoragePool(data),
    onSuccess: () => {
      toast.success("Storage pool created successfully");
      queryClient.invalidateQueries({ queryKey: ["storage-pools"] });
      onCreateSuccess?.();
    },
    onError: (error: Error) => {
      toast.error(`Failed to create storage pool: ${error.message}`);
    },
  });

  const updatePool = useMutation({
    mutationFn: (data: Parameters<typeof storageClient.updateStoragePool>[0]) =>
      storageClient.updateStoragePool(data),
    onSuccess: () => {
      toast.success("Storage pool updated successfully");
      queryClient.invalidateQueries({ queryKey: ["storage-pools"] });
      onUpdateSuccess?.();
    },
    onError: (error: Error) => {
      toast.error(`Failed to update storage pool: ${error.message}`);
    },
  });

  const deletePool = useMutation({
    mutationFn: ({ id, force }: { id: string; force?: boolean }) =>
      storageClient.deleteStoragePool({ id, force }),
    onSuccess: () => {
      toast.success("Storage pool deleted successfully");
      queryClient.invalidateQueries({ queryKey: ["storage-pools"] });
      onDeleteSuccess?.();
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete storage pool: ${error.message}`);
    },
  });

  const refreshPool = useMutation({
    mutationFn: (id: string) => storageClient.refreshStoragePool({ id }),
    onSuccess: () => {
      toast.success("Storage pool refreshed");
      queryClient.invalidateQueries({ queryKey: ["storage-pools"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to refresh storage pool: ${error.message}`);
    },
  });

  return {
    createPool,
    updatePool,
    deletePool,
    refreshPool,
    isCreating: createPool.isPending,
    isUpdating: updatePool.isPending,
    isDeleting: deletePool.isPending,
    isRefreshing: refreshPool.isPending,
  };
}

interface UseStorageDiskMutationsOptions {
  onCreateSuccess?: () => void;
  onDeleteSuccess?: () => void;
}

export function useStorageDiskMutations({
  onCreateSuccess,
  onDeleteSuccess,
}: UseStorageDiskMutationsOptions = {}) {
  const queryClient = useQueryClient();

  const createDisk = useMutation({
    mutationFn: (data: Parameters<typeof storageClient.createStorageDisk>[0]) =>
      storageClient.createStorageDisk(data),
    onSuccess: () => {
      toast.success("Disk created successfully");
      queryClient.invalidateQueries({ queryKey: ["storage-disks"] });
      onCreateSuccess?.();
    },
    onError: (error: Error) => {
      toast.error(`Failed to create disk: ${error.message}`);
    },
  });

  const resizeDisk = useMutation({
    mutationFn: (data: Parameters<typeof storageClient.resizeStorageDisk>[0]) =>
      storageClient.resizeStorageDisk(data),
    onSuccess: () => {
      toast.success("Disk resized successfully");
      queryClient.invalidateQueries({ queryKey: ["storage-disks"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to resize disk: ${error.message}`);
    },
  });

  const deleteDisk = useMutation({
    mutationFn: ({ diskId, purge }: { diskId: string; purge?: boolean }) =>
      storageClient.deleteStorageDisk({ diskId, purge }),
    onSuccess: () => {
      toast.success("Disk deleted successfully");
      queryClient.invalidateQueries({ queryKey: ["storage-disks"] });
      onDeleteSuccess?.();
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete disk: ${error.message}`);
    },
  });

  const moveDisk = useMutation({
    mutationFn: (data: Parameters<typeof storageClient.moveStorageDisk>[0]) =>
      storageClient.moveStorageDisk(data),
    onSuccess: () => {
      toast.success("Disk move initiated");
      queryClient.invalidateQueries({ queryKey: ["storage-disks"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to move disk: ${error.message}`);
    },
  });

  return {
    createDisk,
    resizeDisk,
    deleteDisk,
    moveDisk,
    isCreating: createDisk.isPending,
    isResizing: resizeDisk.isPending,
    isDeleting: deleteDisk.isPending,
    isMoving: moveDisk.isPending,
  };
}

// VM Disk mutations
export function useVMDiskMutations(vmid: number) {
  const queryClient = useQueryClient();

  const attachDisk = useMutation({
    mutationFn: (data: Parameters<typeof storageClient.attachVMdisk>[0]) =>
      storageClient.attachVMdisk(data),
    onSuccess: () => {
      toast.success("Disk attached to VM successfully");
      queryClient.invalidateQueries({ queryKey: ["vm-disks", vmid] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to attach disk: ${error.message}`);
    },
  });

  const detachDisk = useMutation({
    mutationFn: (data: Parameters<typeof storageClient.detachVMdisk>[0]) =>
      storageClient.detachVMdisk(data),
    onSuccess: () => {
      toast.success("Disk detached from VM successfully");
      queryClient.invalidateQueries({ queryKey: ["vm-disks", vmid] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to detach disk: ${error.message}`);
    },
  });

  const resizeDisk = useMutation({
    mutationFn: (data: Parameters<typeof storageClient.resizeVMdisk>[0]) =>
      storageClient.resizeVMdisk(data),
    onSuccess: () => {
      toast.success("VM disk resized successfully");
      queryClient.invalidateQueries({ queryKey: ["vm-disks", vmid] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to resize disk: ${error.message}`);
    },
  });

  return {
    attachDisk,
    detachDisk,
    resizeDisk,
    isAttaching: attachDisk.isPending,
    isDetaching: detachDisk.isPending,
    isResizing: resizeDisk.isPending,
  };
}
