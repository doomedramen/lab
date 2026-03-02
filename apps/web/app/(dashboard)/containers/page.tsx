"use client"

import { StatusBadge, ResourceUsageBar, TagList } from "@workspace/components/lab-shared"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@workspace/ui/components/table"
import { Badge } from "@workspace/ui/components/badge"
import { Box, Lock, Unlock } from "lucide-react"
import Link from "next/link"
import { useContainers } from "@/lib/api/queries"
import { Shimmer } from "@workspace/components/shimmer"
import { ErrorDisplay } from "@/components/error-display"
import type { Container } from "@/lib/gen/lab/v1/container_pb"
import { ContainerStatus } from "@/lib/gen/lab/v1/container_pb"
import { containerStatusToString } from "@/lib/api/enum-helpers"

// Template data for shimmer
const templateContainers = [
  {
    id: "ct-1000",
    ctid: 1000,
    name: "nginx-proxy",
    node: "lab-prod-01",
    status: ContainerStatus.RUNNING,
    cpu: { used: 8, sockets: 1, cores: 2 },
    memory: { used: 0.4, total: 1 },
    disk: { used: 2, total: 8 },
    uptime: "142d 7h",
    os: "Alpine 3.20",
    ip: "10.0.10.50",
    tags: ["production", "proxy"],
    unprivileged: true,
    swap: { used: 0, total: 0.5 },
    description: "Reverse proxy",
    startOnBoot: true,
  },
  {
    id: "ct-1001",
    ctid: 1001,
    name: "dns-primary",
    node: "lab-prod-01",
    status: ContainerStatus.RUNNING,
    cpu: { used: 3, sockets: 1, cores: 1 },
    memory: { used: 0.2, total: 0.5 },
    disk: { used: 1, total: 4 },
    uptime: "142d 7h",
    os: "Debian 12",
    ip: "10.0.10.51",
    tags: ["production", "dns"],
    unprivileged: true,
    swap: { used: 0, total: 0.25 },
    description: "Primary DNS",
    startOnBoot: true,
  },
  {
    id: "ct-1002",
    ctid: 1002,
    name: "log-collector",
    node: "lab-prod-01",
    status: ContainerStatus.RUNNING,
    cpu: { used: 22, sockets: 1, cores: 2 },
    memory: { used: 1.8, total: 4 },
    disk: { used: 45, total: 100 },
    uptime: "142d 7h",
    os: "Ubuntu 24.04",
    ip: "10.0.10.52",
    tags: ["production", "logging"],
    unprivileged: true,
    swap: { used: 0.1, total: 1 },
    description: "Log collection",
    startOnBoot: true,
  },
] as unknown as Container[]

function ContainersContent({ containers }: { containers: Container[] }) {
  const running = containers.filter((c) => c.status === ContainerStatus.RUNNING).length
  const stopped = containers.filter((c) => c.status === ContainerStatus.STOPPED).length

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-foreground text-balance">Containers</h1>
        <p className="text-sm text-muted-foreground mt-1">
          {containers.length} total - {running} running, {stopped} stopped
        </p>
      </div>

      <div className="rounded-lg border border-border bg-card overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow className="bg-secondary/30 hover:bg-secondary/30">
              <TableHead>CTID</TableHead>
              <TableHead>Name</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Node</TableHead>
              <TableHead>CPU</TableHead>
              <TableHead>Memory</TableHead>
              <TableHead>Disk</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>Tags</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {containers.map((ct) => (
              <TableRow key={ct.id}>
                <TableCell className="font-mono text-sm">{ct.ctid}</TableCell>
                <TableCell>
                  <Link href={`/containers/${ct.ctid}`} className="flex items-center gap-2 hover:text-primary transition-colors">
                    <Box className="size-4 text-muted-foreground shrink-0" />
                    <div>
                      <div className="font-medium text-foreground">{ct.name}</div>
                      <div className="text-xs text-muted-foreground">{ct.os}</div>
                    </div>
                  </Link>
                </TableCell>
                <TableCell>
                  <StatusBadge status={containerStatusToString(ct.status)} />
                </TableCell>
                <TableCell>
                  <span className="text-sm text-muted-foreground">{ct.node}</span>
                </TableCell>
                <TableCell>
                  <div className="space-y-1">
                    <ResourceUsageBar value={ct.cpu?.used ?? 0} />
                    <div className="text-[11px] text-muted-foreground">{ct.cpu?.cores ?? 0} cores</div>
                  </div>
                </TableCell>
                <TableCell>
                  <div className="space-y-1">
                    <ResourceUsageBar value={(ct.memory?.total ?? 0) > 0 ? Math.round(((ct.memory?.used ?? 0) / (ct.memory?.total || 1)) * 100) : 0} />
                    <div className="text-[11px] text-muted-foreground">
                      {ct.memory?.used ?? 0}/{ct.memory?.total ?? 0} GB
                    </div>
                  </div>
                </TableCell>
                <TableCell>
                  <div className="text-sm text-foreground">
                    {ct.disk?.used ?? 0}/{ct.disk?.total ?? 0} GB
                  </div>
                </TableCell>
                <TableCell>
                  {ct.unprivileged ? (
                    <Badge variant="secondary" className="text-[11px] gap-1">
                      <Unlock className="size-3" />
                      Unpriv
                    </Badge>
                  ) : (
                    <Badge variant="outline" className="text-[11px] gap-1 text-warning border-warning/30">
                      <Lock className="size-3" />
                      Priv
                    </Badge>
                  )}
                </TableCell>
                <TableCell>
                  <TagList tags={ct.tags} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}

export default function ContainersPage() {
  const { data: containers, isLoading, error, refetch } = useContainers()

  if (error) {
    return (
      <div className="p-6">
        <ErrorDisplay message={error.message} onRetry={() => refetch()} className="h-[50vh]" />
      </div>
    )
  }

  return (
    <Shimmer loading={isLoading} templateProps={{ containers: templateContainers }}>
      <ContainersContent containers={containers || templateContainers} />
    </Shimmer>
  )
}
