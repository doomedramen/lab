"use client"

import { Server, Monitor, Box, Layers, Cpu, MemoryStick, Activity } from "lucide-react"
import { MetricCard, ResourceBar, StatusBadge } from "@workspace/components/lab-shared"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@workspace/ui/components/card"
import { MetricAreaChart } from "@workspace/components/metric-area-chart"
import { Shimmer } from "@workspace/components/shimmer"
import Link from "next/link"
import { useClusterSummary, useNodes, useMetrics } from "@/lib/api/queries"
import { ErrorDisplay } from "@/components/error-display"
import type { ClusterSummary, MetricPoint } from "@/lib/api/types"
import type { Node } from "@/lib/gen/lab/v1/node_pb"
import { NodeStatus } from "@/lib/gen/lab/v1/node_pb"
import { nodeStatusToString } from "@/lib/api/enum-helpers"
import { formatChartTime } from "@/lib/utils/format-time"
import { useState, useMemo } from "react"

// Template data for shimmer
const templateSummary: ClusterSummary = {
  nodes: { total: 4, running: 3 },
  vms: { total: 10, running: 8 },
  containers: { total: 9, running: 7 },
  stacks: { total: 6, running: 4 },
  cpu: { cores: 136, avgUsage: 38 },
  memory: { used: 312, total: 1088 },
  disk: { used: 9, total: 18 },
}

const templateNodes = [
  {
    id: "node-1",
    name: "lab-prod-01",
    status: NodeStatus.ONLINE,
    ip: "10.0.1.10",
    cpu: { used: 34, total: 100, cores: 32 },
    memory: { used: 99, total: 256 },
    disk: { used: 1.8, total: 4 },
    uptime: "142d 7h 23m",
    kernel: "6.8.12-generic",
    version: "1.0.0",
    vms: 8,
    containers: 12,
    cpuModel: "AMD EPYC 9654 96-Core",
    loadAvg: { one: 4.2, five: 3.8, fifteen: 3.5 },
    networkIn: 246,
    networkOut: 182,
  },
] as unknown as Node[]

const templateMetrics: MetricPoint[] = Array.from({ length: 24 }, (_, i) => ({ time: `${i}:00`, value: 50 }))

type TimeRange = "1h" | "24h" | "7d"

function DashboardContent({
  summary,
  nodes,
  timeRange,
  onTimeRangeChange,
  cpuMetrics,
  memoryMetrics,
  networkMetrics,
}: {
  summary: ClusterSummary
  nodes: Node[]
  timeRange: TimeRange
  onTimeRangeChange: (range: TimeRange) => void
  cpuMetrics: MetricPoint[]
  memoryMetrics: MetricPoint[]
  networkMetrics: MetricPoint[]
}) {
  const onlineNodes = nodes.filter((n) => n.status === NodeStatus.ONLINE)

  // Use the most recent chart data point for the live throughput stat.
  // networkMetrics values are MiB/s rates (combined in+out).
  const latestThroughput = networkMetrics.at(-1)?.value ?? 0

  const formatThroughput = (value: number) => {
    if (value >= 1000) {
      return `${(value / 1000).toFixed(1)} GiB/s`
    }
    if (value < 0.1) {
      return `${(value * 1024).toFixed(0)} KiB/s`
    }
    return `${value.toFixed(1)} MiB/s`
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-foreground text-balance">Datacenter Overview</h1>
          <p className="text-sm text-muted-foreground mt-1">Cluster resource summary and status</p>
        </div>
        <div className="flex items-center gap-2">
          {(["1h", "24h", "7d"] as TimeRange[]).map((range) => (
            <button
              key={range}
              onClick={() => onTimeRangeChange(range)}
              className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                timeRange === range
                  ? "bg-primary text-primary-foreground"
                  : "bg-secondary text-secondary-foreground hover:bg-secondary/80"
              }`}
            >
              {range}
            </button>
          ))}
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <MetricCard
          label="Hosts"
          value={summary.nodes.running}
          subtitle={`${summary.nodes.total} total, ${summary.nodes.running} online`}
          icon={<Server className="size-4" />}
        />
        <MetricCard
          label="Virtual Machines"
          value={summary.vms.running}
          subtitle={`${summary.vms.total} total, ${summary.vms.running} running`}
          icon={<Monitor className="size-4" />}
        />
        <MetricCard
          label="Containers"
          value={summary.containers.running}
          subtitle={`${summary.containers.total} total, ${summary.containers.running} running`}
          icon={<Box className="size-4" />}
        />
        <MetricCard
          label="Stacks"
          value={summary.stacks.running}
          subtitle={`${summary.stacks.total} total, ${summary.stacks.running} running`}
          icon={<Layers className="size-4" />}
        />
      </div>

      {/* Resource utilization */}
      <div className="grid gap-4 lg:grid-cols-3">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-sm font-medium">
              <Cpu className="size-4 text-primary" />
              CPU Usage
            </CardTitle>
          </CardHeader>
          <CardContent>
            <ResourceBar label="Cluster Average" used={summary.cpu.avgUsage} total={100} unit="%" showPercent={false} />
            <div className="mt-3">
              <MetricAreaChart
                data={cpuMetrics}
                color="oklch(0.65 0.18 200)"
                yDomain={[0, 100]}
                tooltipLabel="CPU"
                tooltipUnit="%"
              />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-sm font-medium">
              <MemoryStick className="size-4 text-chart-2" />
              Memory Usage
            </CardTitle>
          </CardHeader>
          <CardContent>
            <ResourceBar label="Cluster Memory" used={Math.round(summary.memory.used)} total={summary.memory.total} unit="GB" />
            <div className="mt-3">
              <MetricAreaChart
                data={memoryMetrics}
                color="oklch(0.70 0.15 145)"
                yDomain={[0, 100]}
                tooltipLabel="Memory"
                tooltipUnit="%"
              />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-sm font-medium">
              <Activity className="size-4 text-chart-3" />
              Network I/O
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between text-xs mb-2">
              <span className="text-muted-foreground">Cluster Throughput</span>
              <span className="text-foreground font-medium">{formatThroughput(latestThroughput)}</span>
            </div>
            <div className="mt-1">
              <MetricAreaChart
                data={networkMetrics}
                color="oklch(0.75 0.15 55)"
                height={148}
                tooltipLabel="Throughput"
                valueFormatter={(v) => {
                  if (v >= 1024) return `${(v / 1024).toFixed(1)} GiB/s`
                  if (v < 0.1) return `${(v * 1024).toFixed(0)} KiB/s`
                  return `${v.toFixed(1)} MiB/s`
                }}
              />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Node overview table */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-medium">Host Nodes</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            {nodes.map((node) => (
              <Link
                key={node.id}
                href={`/hosts/${node.id}`}
                className="flex items-center gap-4 rounded-md border border-border bg-secondary/30 px-4 py-3 hover:bg-secondary/60 transition-colors"
              >
                <Server className="size-5 text-muted-foreground shrink-0" />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-foreground">{node.name}</span>
                    <StatusBadge status={nodeStatusToString(node.status)} />
                  </div>
                  <span className="text-xs text-muted-foreground">{node.ip}</span>
                </div>
                <div className="hidden md:flex items-center gap-6">
                  <div className="text-center">
                    <div className="text-xs text-muted-foreground">CPU</div>
                    <div className="text-sm font-medium text-foreground">{node.cpu?.used ?? 0}%</div>
                  </div>
                  <div className="text-center">
                    <div className="text-xs text-muted-foreground">Memory</div>
                    <div className="text-sm font-medium text-foreground">
                      {Math.round(node.memory?.used ?? 0)}/{node.memory?.total ?? 0} GB
                    </div>
                  </div>
                  <div className="text-center">
                    <div className="text-xs text-muted-foreground">VMs</div>
                    <div className="text-sm font-medium text-foreground">{node.vms}</div>
                  </div>
                  <div className="text-center">
                    <div className="text-xs text-muted-foreground">CTs</div>
                    <div className="text-sm font-medium text-foreground">{node.containers}</div>
                  </div>
                </div>
              </Link>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

export default function DashboardPage() {
  const [timeRange, setTimeRange] = useState<TimeRange>("24h")

  const {
    data: summary,
    isLoading: summaryLoading,
    error: summaryError,
    refetch: refetchSummary,
  } = useClusterSummary()
  const {
    data: nodes,
    isLoading: nodesLoading,
    error: nodesError,
    refetch: refetchNodes,
  } = useNodes()

  // Calculate time range in seconds
  const timeRangeSeconds = useMemo(() => {
    const now = Math.floor(Date.now() / 1000)
    switch (timeRange) {
      case "1h":
        return { start: now - 3600, end: now }
      case "24h":
        return { start: now - 86400, end: now }
      case "7d":
        return { start: now - 604800, end: now }
    }
  }, [timeRange])

  // Fetch metrics from SQLite — host_only filters out VM-level metrics so we get
  // true host CPU/memory averages across the cluster.
  const { data: cpuData } = useMetrics({
    resource_type: "cpu",
    host_only: true,
    start_time: timeRangeSeconds.start,
    end_time: timeRangeSeconds.end,
    aggregate: "avg",
  })

  const { data: memoryData } = useMetrics({
    resource_type: "memory",
    host_only: true,
    start_time: timeRangeSeconds.start,
    end_time: timeRangeSeconds.end,
    aggregate: "avg",
  })

  const { data: networkInData } = useMetrics({
    resource_type: "network_in",
    start_time: timeRangeSeconds.start,
    end_time: timeRangeSeconds.end,
    aggregate: "avg",
  })

  const { data: networkOutData } = useMetrics({
    resource_type: "network_out",
    start_time: timeRangeSeconds.start,
    end_time: timeRangeSeconds.end,
    aggregate: "avg",
  })

  const cpuMetrics = useMemo(() => {
    if (!cpuData?.metrics?.length) return templateMetrics
    return cpuData.metrics.map((m) => ({
      time: formatChartTime(m.time),
      value: Math.round(m.value * 10) / 10,
    }))
  }, [cpuData])

  const memoryMetrics = useMemo(() => {
    if (!memoryData?.metrics?.length) return templateMetrics
    return memoryData.metrics.map((m) => ({
      time: formatChartTime(m.time),
      value: Math.round(m.value * 10) / 10,
    }))
  }, [memoryData])

  const networkMetrics = useMemo(() => {
    if (!networkInData?.metrics?.length && !networkOutData?.metrics?.length) return templateMetrics

    // Combine network in and out by matching on the ISO time bucket string.
    const networkMap = new Map<string, { in: number; out: number }>()

    networkInData?.metrics.forEach((m) => {
      if (!networkMap.has(m.time)) networkMap.set(m.time, { in: 0, out: 0 })
      networkMap.get(m.time)!.in = m.value
    })

    networkOutData?.metrics.forEach((m) => {
      if (!networkMap.has(m.time)) networkMap.set(m.time, { in: 0, out: 0 })
      networkMap.get(m.time)!.out = m.value
    })

    return Array.from(networkMap.entries())
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([time, data]) => ({
        time: formatChartTime(time),
        value: Math.round((data.in + data.out) * 10) / 10,
      }))
  }, [networkInData, networkOutData])

  const isLoading = summaryLoading || nodesLoading
  const hasError = summaryError || nodesError

  if (hasError) {
    return (
      <ErrorDisplay
        message={summaryError?.message || nodesError?.message || "Failed to load dashboard data"}
        onRetry={() => {
          refetchSummary()
          refetchNodes()
        }}
        className="h-[50vh]"
      />
    )
  }

  return (
    <Shimmer
      loading={isLoading}
      templateProps={{
        summary: templateSummary,
        nodes: templateNodes,
        metrics: templateMetrics,
      }}
    >
      <DashboardContent
        summary={summary || templateSummary}
        nodes={nodes || templateNodes}
        timeRange={timeRange}
        onTimeRangeChange={setTimeRange}
        cpuMetrics={cpuMetrics}
        memoryMetrics={memoryMetrics}
        networkMetrics={networkMetrics}
      />
    </Shimmer>
  )
}
