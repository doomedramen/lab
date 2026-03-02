import { useQuery } from "@tanstack/react-query"
import { storageClient } from "../client"

export function useStoragePools(type?: number, status?: number, enabledOnly?: boolean) {
  return useQuery({
    queryKey: ["storage-pools", { type, status, enabledOnly }],
    queryFn: () => storageClient.listStoragePools({ 
      type: type ?? 0, 
      status: status ?? 0,
      enabledOnly: enabledOnly ?? false 
    }),
    select: (res) => ({
      pools: res.pools,
      total: res.total,
    }),
  })
}

export function useStoragePool(id: string | undefined) {
  return useQuery({
    queryKey: ["storage-pool", id],
    queryFn: () => storageClient.getStoragePool({ id: id! }),
    select: (res) => res.pool,
    enabled: !!id,
  })
}

export function useStorageDisks(poolId?: string, vmid?: number, unassignedOnly?: boolean) {
  return useQuery({
    queryKey: ["storage-disks", { poolId, vmid, unassignedOnly }],
    queryFn: () => storageClient.listStorageDisks({
      poolId: poolId ?? "",
      vmid: vmid ?? 0,
      unassignedOnly: unassignedOnly ?? false
    }),
    select: (res) => ({
      disks: res.disks,
      total: res.total,
    }),
  })
}

export function useVMDisks(vmid: number | undefined) {
  return useQuery({
    queryKey: ["vm-disks", vmid],
    queryFn: () => storageClient.listVMDisks({ vmid: vmid! }),
    select: (res) => res.disks,
    enabled: vmid !== undefined && vmid > 0,
    refetchInterval: 5000,
  })
}
