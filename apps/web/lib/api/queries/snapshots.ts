import { useQuery } from "@tanstack/react-query"
import { snapshotClient } from "../client"

export function useSnapshots(vmid: number | undefined) {
  return useQuery({
    queryKey: ["snapshots", vmid],
    queryFn: () => snapshotClient.listSnapshots({ vmid: vmid! }),
    select: (res) => ({
      snapshots: res.snapshots,
      tree: res.tree,
    }),
    enabled: !!vmid,
    refetchInterval: 10000, // Auto-refresh every 10 seconds
  })
}

export function useSnapshotInfo(vmid: number | undefined, snapshotId: string | undefined) {
  return useQuery({
    queryKey: ["snapshot-info", vmid, snapshotId],
    queryFn: () => snapshotClient.getSnapshotInfo({ vmid: vmid!, snapshotId: snapshotId! }),
    select: (res) => ({
      snapshot: res.snapshot,
      tree: res.tree,
    }),
    enabled: !!vmid && !!snapshotId,
  })
}
