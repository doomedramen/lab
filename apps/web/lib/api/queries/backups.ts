import { useQuery } from "@tanstack/react-query";
import { backupClient } from "../client";

export function useBackups(vmid?: number, storagePool?: string) {
  return useQuery({
    queryKey: ["backups", { vmid, storagePool }],
    queryFn: () =>
      backupClient.listBackups({
        vmid: vmid ?? 0,
        storagePool: storagePool ?? "",
      }),
    select: (res) => ({
      backups: res.backups,
      total: res.total,
    }),
  });
}

export function useBackup(backupId: string | undefined) {
  return useQuery({
    queryKey: ["backup", backupId],
    queryFn: () => backupClient.getBackup({ backupId: backupId! }),
    select: (res) => res.backup,
    enabled: !!backupId,
  });
}

export function useBackupSchedules(entityType?: string, entityId?: number) {
  return useQuery({
    queryKey: ["backup-schedules", { entityType, entityId }],
    queryFn: () =>
      backupClient.listBackupSchedules({
        entityType: entityType ?? "",
        entityId: entityId ?? 0,
      }),
    select: (res) => res.schedules,
  });
}
