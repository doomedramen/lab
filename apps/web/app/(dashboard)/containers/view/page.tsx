"use client"

import { Suspense } from "react"
import { notFound, useSearchParams } from "next/navigation"
import { StatusBadge, ResourceBar, TagList } from "@/components/lab-shared"
import { ResourceMetricCard } from "@/components/resource-metric-card"
import { ConfigList } from "@/components/config-list"
import { ResourceConfigItem } from "@/components/resource-config-item"
import { PageHeader } from "@/components/page-header"
import { PerformanceChart } from "@/components/performance-chart"
import { EntityActionButtons } from "@/components/entity-action-buttons"
import { Shimmer } from "@/components/shimmer"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import { useContainer } from "@/lib/api/queries"
import { useContainerMutations } from "@/lib/api/mutations"
import { ErrorDisplay } from "@/components/error-display"
import type { Container } from "@/lib/gen/lab/v1/container_pb"
import { ContainerStatus } from "@/lib/gen/lab/v1/container_pb"
import { containerStatusToString } from "@/lib/api/enum-helpers"
import {
  Box,
  Cpu,
  MemoryStick,
  HardDrive,
  Clock,
  Lock,
  Unlock,
  Network,
} from "lucide-react"

// Template container for shimmer
const templateContainer = {
  id: "ct-template",
  ctid: 1000,
  name: "Loading Container...",
  node: "Loading...",
  status: ContainerStatus.RUNNING,
  cpu: { used: 12, sockets: 1, cores: 2 },
  memory: { used: 0.8, total: 2 },
  disk: { used: 4, total: 16 },
  uptime: "Loading...",
  os: "Loading...",
  ip: "Loading...",
  tags: ["loading"],
  unprivileged: true,
  swap: { used: 0, total: 0.5 },
  description: "Loading description...",
  startOnBoot: true,
} as unknown as Container

function ContainerDetailContent({ ct, mutationProps }: { ct: Container; mutationProps: {
  onPlay: () => void
  onStop: () => void
  onReboot: () => void
  loading: { start: boolean; stop: boolean; reboot: boolean }
} }) {
  const { onPlay, onStop, onReboot, loading } = mutationProps

  const badges = (
    <>
      <Badge variant="outline" className="font-mono text-xs">{ct.ctid}</Badge>
      <StatusBadge status={containerStatusToString(ct.status)} />
      {ct.unprivileged ? (
        <Badge variant="secondary" className="text-[11px] gap-1">
          <Unlock className="size-3" /> Unprivileged
        </Badge>
      ) : (
        <Badge variant="outline" className="text-[11px] gap-1 text-warning border-warning/30">
          <Lock className="size-3" /> Privileged
        </Badge>
      )}
    </>
  )

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        backHref="/containers"
        backLabel="Back to Containers"
        title={ct.name}
        subtitle={`${ct.os} - ${ct.node} - ${ct.ip}`}
        icon={<Box className="size-5 text-foreground" />}
        badges={badges}
      />

      <div className="flex justify-end">
        <EntityActionButtons
          status={containerStatusToString(ct.status)}
          variant="container"
          onPlay={onPlay}
          onStop={onStop}
          onReboot={onReboot}
          loading={loading}
        />
      </div>

      <Tabs defaultValue="summary" className="flex flex-col">
        <TabsList>
          <TabsTrigger value="summary">Summary</TabsTrigger>
          <TabsTrigger value="resources">Resources</TabsTrigger>
          <TabsTrigger value="performance">Performance</TabsTrigger>
          <TabsTrigger value="network">Network</TabsTrigger>
        </TabsList>

        {/* Summary */}
        <TabsContent value="summary" className="space-y-4 mt-4">
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <ResourceMetricCard label="CPU Usage" value={`${ct.cpu?.used ?? 0}%`} subtitle={`${ct.cpu?.cores ?? 0} vCPUs`} icon={<Cpu className="size-4" />} />
            <ResourceMetricCard label="Memory" value={`${ct.memory?.used ?? 0} GB`} subtitle={`of ${ct.memory?.total ?? 0} GB`} icon={<MemoryStick className="size-4" />} />
            <ResourceMetricCard label="Disk" value={`${ct.disk?.used ?? 0} GB`} subtitle={`of ${ct.disk?.total ?? 0} GB`} icon={<HardDrive className="size-4" />} />
            <ResourceMetricCard label="Uptime" value={ct.uptime} subtitle={ct.os} icon={<Clock className="size-4" />} />
          </div>

          <div className="grid gap-4 lg:grid-cols-2">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Configuration</CardTitle>
              </CardHeader>
              <CardContent>
                <ConfigList items={[
                  { label: "CTID", value: String(ct.ctid) },
                  { label: "Name", value: ct.name },
                  { label: "OS Template", value: ct.os },
                  { label: "Node", value: ct.node },
                  { label: "IP Address", value: ct.ip },
                  { label: "Unprivileged", value: ct.unprivileged ? "Yes" : "No" },
                  { label: "Start on Boot", value: ct.startOnBoot ? "Yes" : "No" },
                  { label: "Swap", value: `${ct.swap?.used ?? 0} / ${ct.swap?.total ?? 0} GB` },
                ]} />
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Resources</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <ResourceBar label="CPU" used={ct.cpu?.used ?? 0} total={100} unit="%" showPercent={false} />
                <ResourceBar label="Memory" used={ct.memory?.used ?? 0} total={ct.memory?.total ?? 1} unit="GB" />
                <ResourceBar label="Disk" used={ct.disk?.used ?? 0} total={ct.disk?.total ?? 1} unit="GB" />
                <ResourceBar label="Swap" used={ct.swap?.used ?? 0} total={ct.swap?.total ?? 1} unit="GB" />
                <div className="pt-2">
                  <div className="text-xs text-muted-foreground mb-1.5">Tags</div>
                  <TagList tags={ct.tags} />
                </div>
                {ct.description && (
                  <div className="pt-1">
                    <div className="text-xs text-muted-foreground mb-1">Description</div>
                    <p className="text-sm text-foreground">{ct.description}</p>
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        {/* Resources */}
        <TabsContent value="resources" className="mt-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">Resource Configuration</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <ResourceConfigItem icon={<Cpu className="size-4 text-muted-foreground" />} label="CPU" value={`${ct.cpu?.cores ?? 0} cores`} detail="Unlimited CPU limit" />
                <ResourceConfigItem icon={<MemoryStick className="size-4 text-muted-foreground" />} label="Memory" value={`${ct.memory?.total ?? 0} GB`} detail={`Swap: ${ct.swap?.total ?? 0} GB`} />
                <ResourceConfigItem icon={<HardDrive className="size-4 text-muted-foreground" />} label="Root Disk" value={`${ct.disk?.total ?? 0} GB`} detail="Storage: local-lvm" />
                <ResourceConfigItem icon={<Network className="size-4 text-muted-foreground" />} label="Network" value={ct.ip} detail="Bridge: vmbr0, DHCP: No" />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Performance */}
        <TabsContent value="performance" className="space-y-4 mt-4">
          <div className="grid gap-4 lg:grid-cols-2">
            <PerformanceChart
              title="CPU Usage (24h)"
              icon={Cpu}
              data={[]}
              color="oklch(0.65 0.18 200)"
              tooltipLabel="CPU"
              tooltipUnit="%"
            />
            <PerformanceChart
              title="Memory Usage (24h)"
              icon={MemoryStick}
              data={[]}
              color="oklch(0.70 0.15 145)"
              iconClassName="text-chart-2"
              tooltipLabel="Memory"
              tooltipUnit="%"
            />
          </div>
        </TabsContent>

        {/* Network */}
        <TabsContent value="network" className="mt-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">Network Interfaces</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                <div className="rounded-md border border-border bg-secondary/30 px-4 py-3">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <Network className="size-4 text-muted-foreground" />
                      <div>
                        <div className="text-sm font-medium text-foreground">eth0</div>
                        <div className="text-xs text-muted-foreground">Bridge: vmbr0, firewall=1</div>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <div className="text-right">
                        <div className="text-xs text-muted-foreground">IP Address</div>
                        <div className="text-sm font-medium text-foreground">{ct.ip}/24</div>
                      </div>
                      <div className="text-right">
                        <div className="text-xs text-muted-foreground">Gateway</div>
                        <div className="text-sm font-mono text-foreground">
                          {ct.ip.split(".").slice(0, 3).join(".")}.1
                        </div>
                      </div>
                      <StatusBadge status={ct.status === ContainerStatus.RUNNING ? "online" : "offline"} />
                    </div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}

// Main content component that uses search params
function ContainerDetailView() {
  const searchParams = useSearchParams();
  const ctid = searchParams.get("id") || "";
  const { data: ct, isLoading, error, refetch } = useContainer(ctid)
  const { startContainer, stopContainer, rebootContainer, isStarting, isStopping, isRebooting } = useContainerMutations()

  if (error) {
    return (
      <div className="p-6">
        <ErrorDisplay
          message={error.message}
          onRetry={() => refetch()}
          className="h-[50vh]"
        />
      </div>
    )
  }

  // If not loading and no container found, show 404
  if (!isLoading && !ct) return notFound()

  const handleStart = () => ct && startContainer.mutate(ct.ctid)
  const handleStop = () => ct && stopContainer.mutate(ct.ctid)
  const handleReboot = () => ct && rebootContainer.mutate(ct.ctid)

  return (
    <Shimmer loading={isLoading} templateProps={{ ct: templateContainer, mutationProps: {} }}>
      <ContainerDetailContent
        ct={ct || templateContainer}
        mutationProps={{
          onPlay: handleStart,
          onStop: handleStop,
          onReboot: handleReboot,
          loading: { start: isStarting, stop: isStopping, reboot: isRebooting },
        }}
      />
    </Shimmer>
  )
}

// Default export with Suspense boundary for static export compatibility
export default function ContainerViewPage() {
  return (
    <Suspense fallback={<div className="p-8 text-center">Loading...</div>}>
      <ContainerDetailView />
    </Suspense>
  );
}
