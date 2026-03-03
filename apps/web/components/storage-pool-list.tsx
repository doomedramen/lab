"use client"

import { useState } from "react"
import { MoreVertical, Trash2, RefreshCw, HardDrive, Folder, Database, Server } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { CreateStoragePoolModal } from "./create-storage-pool-modal"
import { useStoragePoolMutations } from "@/lib/api/mutations/storage"
import type { StoragePool } from "@/lib/gen/lab/v1/storage_pb"

interface StoragePoolListProps {
  pools?: StoragePool[]
  isLoading?: boolean
}

export function StoragePoolList({ pools, isLoading }: StoragePoolListProps) {
  const [selectedPool, setSelectedPool] = useState<StoragePool | null>(null)
  const { deletePool, refreshPool, isDeleting, isRefreshing } = useStoragePoolMutations()

  const formatSize = (bytes: number) => {
    if (bytes === 0) return "Unknown"
    const tb = bytes / (1024 * 1024 * 1024 * 1024)
    if (tb >= 1) return `${tb.toFixed(2)} TB`
    const gb = bytes / (1024 * 1024 * 1024)
    if (gb >= 1) return `${gb.toFixed(2)} GB`
    const mb = bytes / (1024 * 1024)
    return `${mb.toFixed(2)} MB`
  }

  const getTypeBadge = (type: number) => {
    const typeMap: Record<number, { label: string; icon: React.ReactNode }> = {
      0: { label: "Unknown", icon: <HardDrive className="w-3 h-3" /> },
      1: { label: "Directory", icon: <Folder className="w-3 h-3" /> },
      2: { label: "LVM", icon: <Database className="w-3 h-3" /> },
      3: { label: "ZFS", icon: <Server className="w-3 h-3" /> },
      4: { label: "NFS", icon: <Server className="w-3 h-3" /> },
      5: { label: "iSCSI", icon: <Database className="w-3 h-3" /> },
      6: { label: "Ceph", icon: <Database className="w-3 h-3" /> },
      7: { label: "GlusterFS", icon: <Server className="w-3 h-3" /> },
    }
    const config = typeMap[type] ?? typeMap[0]!
    return (
      <Badge variant="outline" className="gap-1">
        {config!.icon}
        {config!.label}
      </Badge>
    )
  }

  const getStatusBadge = (status: number) => {
    const statusConfig: Record<number, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
      0: { label: "Unknown", variant: "secondary" },
      1: { label: "Active", variant: "default" },
      2: { label: "Inactive", variant: "secondary" },
      3: { label: "Maintenance", variant: "outline" },
      4: { label: "Error", variant: "destructive" },
    }
    const config = statusConfig[status] ?? { label: "Unknown", variant: "secondary" as const }
    return <Badge variant={config.variant}>{config.label}</Badge>
  }

  const handleDelete = (poolId: string, diskCount: number) => {
    const message = diskCount > 0
      ? `This pool contains ${diskCount} disk(s). Deleting will remove all disks. Are you sure?`
      : "Are you sure you want to delete this storage pool? This action cannot be undone."
    
    if (confirm(message)) {
      deletePool.mutate({ id: poolId, force: true })
    }
  }

  const handleRefresh = (poolId: string) => {
    refreshPool.mutate(poolId)
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <RefreshCw className="w-6 h-6 animate-spin text-muted-foreground" />
        <span className="ml-2 text-muted-foreground">Loading storage pools...</span>
      </div>
    )
  }

  if (!pools || pools.length === 0) {
    return (
      <div className="text-center py-8">
        <HardDrive className="w-12 h-12 mx-auto text-muted-foreground/50" />
        <h3 className="mt-4 text-lg font-medium">No storage pools yet</h3>
        <p className="mt-2 text-muted-foreground">
          Create a storage pool to store VM disks, ISOs, and backups
        </p>
        <div className="mt-4">
          <CreateStoragePoolModal />
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <CreateStoragePoolModal />
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {pools.map((pool) => (
          <Card key={pool.id} className="relative">
            <CardContent className="pt-4">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-2">
                  {getTypeBadge(pool.type)}
                  {getStatusBadge(pool.status)}
                </div>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                      <MoreVertical className="w-4 h-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem onClick={() => handleRefresh(pool.id)}>
                      <RefreshCw className="w-4 h-4 mr-2" />
                      Refresh
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      onClick={() => handleDelete(pool.id, pool.diskCount)}
                      className="text-destructive"
                      disabled={isDeleting}
                    >
                      <Trash2 className="w-4 h-4 mr-2" />
                      Delete
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>

              <h3 className="font-semibold text-lg mb-1">{pool.name}</h3>
              {pool.description && (
                <p className="text-sm text-muted-foreground mb-3">{pool.description}</p>
              )}

              <div className="space-y-2 mb-4">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Usage</span>
                  <span className="font-medium">{pool.usagePercent.toFixed(1)}%</span>
                </div>
                <Progress value={pool.usagePercent} className="h-2" />
              </div>

              <div className="grid grid-cols-2 gap-2 text-sm">
                <div>
                  <p className="text-muted-foreground">Capacity</p>
                  <p className="font-medium">{formatSize(Number(pool.capacityBytes))}</p>
                </div>
                <div>
                  <p className="text-muted-foreground">Used</p>
                  <p className="font-medium">{formatSize(Number(pool.usedBytes))}</p>
                </div>
                <div>
                  <p className="text-muted-foreground">Available</p>
                  <p className="font-medium">{formatSize(Number(pool.availableBytes))}</p>
                </div>
                <div>
                  <p className="text-muted-foreground">Disks</p>
                  <p className="font-medium">{pool.diskCount}</p>
                </div>
              </div>

              <div className="mt-3 pt-3 border-t text-xs text-muted-foreground">
                <p>Path: {pool.path}</p>
                {!pool.enabled && <p className="text-amber-600 mt-1">Disabled</p>}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
