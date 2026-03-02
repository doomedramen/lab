"use client"

import { HardDrive } from "lucide-react"
import { PageHeader } from "@/components/page-header"
import { StoragePoolList } from "@/components/storage-pool-list"
import { useStoragePools } from "@/lib/api/queries"
import { Card, CardContent, CardHeader, CardTitle } from "@workspace/ui/components/card"

export default function StoragePage() {
  const { data, isLoading } = useStoragePools()

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        backHref="/"
        backLabel="Dashboard"
        title="Storage"
        subtitle="Manage storage pools and disk images"
        icon={<HardDrive className="size-5 text-foreground" />}
      />

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Total Pools</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data?.total ?? 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Active Pools</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {data?.pools?.filter(p => p.status === 1).length ?? 0}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Total Capacity</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {formatTotalSize(data?.pools)}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Total Disks</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {data?.pools?.reduce((sum, p) => sum + (p.diskCount ?? 0), 0) ?? 0}
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Storage Pools</CardTitle>
        </CardHeader>
        <CardContent>
          <StoragePoolList pools={data?.pools} isLoading={isLoading} />
        </CardContent>
      </Card>
    </div>
  )
}

function formatTotalSize(pools?: any[]) {
  if (!pools || pools.length === 0) return "0 GB"
  const totalBytes = pools.reduce((sum, p) => sum + Number(p.capacityBytes ?? 0), 0)
  const tb = totalBytes / (1024 * 1024 * 1024 * 1024)
  if (tb >= 1) return `${tb.toFixed(2)} TB`
  const gb = totalBytes / (1024 * 1024 * 1024)
  return `${gb.toFixed(2)} GB`
}
