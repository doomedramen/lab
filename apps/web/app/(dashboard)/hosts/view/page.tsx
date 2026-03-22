"use client";

import { useMemo, Suspense } from "react";
import { notFound, useSearchParams } from "next/navigation";
import Link from "next/link";
import { StatusBadge, ResourceBar, TagList } from "@/components/lab-shared";
import { ResourceMetricCard } from "@/components/resource-metric-card";
import { ConfigList } from "@/components/config-list";
import { PageHeader } from "@/components/page-header";
import { PerformanceChart } from "@/components/performance-chart";
import { Shimmer } from "@/components/shimmer";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useNode, useVMs, useContainers, useMetrics } from "@/lib/api/queries";
import { useNodeMutations } from "@/lib/api/mutations";
import { ErrorDisplay } from "@/components/error-display";
import { HostShell } from "@/components/host-shell";
import type { Node } from "@/lib/gen/lab/v1/node_pb";
import { NodeStatus } from "@/lib/gen/lab/v1/node_pb";
import type { VM } from "@/lib/gen/lab/v1/vm_pb";
import type { Container } from "@/lib/gen/lab/v1/container_pb";
import {
  nodeStatusToString,
  vmStatusToString,
  containerStatusToString,
} from "@/lib/api/enum-helpers";
import { formatChartTime } from "@/lib/utils/format-time";
import {
  Server,
  Cpu,
  MemoryStick,
  HardDrive,
  Clock,
  Monitor,
  Box,
  Power,
  RotateCcw,
  Settings,
  Loader2,
  Terminal,
} from "lucide-react";
import type { MetricPoint } from "@/lib/api/types";

// Template data for shimmer
const templateNode = {
  id: "node-template",
  name: "Loading Node...",
  status: NodeStatus.ONLINE,
  ip: "Loading...",
  cpu: { used: 34, total: 100, cores: 32 },
  memory: { used: 99, total: 256 },
  disk: { used: 1.8, total: 4 },
  uptime: "Loading...",
  kernel: "Loading...",
  version: "Loading...",
  vms: 0,
  containers: 0,
  cpuModel: "Loading...",
  loadAvg: { one: 0, five: 0, fifteen: 0 },
  networkIn: 0,
  networkOut: 0,
} as unknown as Node;

const templateVms: VM[] = [];
const templateContainers: Container[] = [];

interface HostDetailContentProps {
  node: Node;
  nodeVms: VM[];
  nodeCts: Container[];
  cpuChartData: MetricPoint[];
  memoryChartData: MetricPoint[];
  mutationProps: {
    onReboot: () => void;
    onShutdown: () => void;
    loading: { reboot: boolean; shutdown: boolean };
  };
}

function HostDetailContent({
  node,
  nodeVms,
  nodeCts,
  cpuChartData,
  memoryChartData,
  mutationProps,
}: HostDetailContentProps) {
  const { onReboot, onShutdown, loading } = mutationProps;

  // Format network values with appropriate units
  const formatNetwork = (value: number) => {
    if (value >= 1000) {
      return `${(value / 1000).toFixed(1)} GB/s`;
    }
    return `${Math.round(value)} MB/s`;
  };

  const loadAvgStr = [
    node.loadAvg?.one ?? 0,
    node.loadAvg?.five ?? 0,
    node.loadAvg?.fifteen ?? 0,
  ].join(", ");

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        backHref="/hosts"
        backLabel="Back to Hosts"
        title={node.name}
        subtitle={node.ip}
        icon={<Server className="size-5 text-foreground" />}
        badges={<StatusBadge status={nodeStatusToString(node.status)} />}
      />

      <div className="flex justify-end gap-2">
        <Button
          variant="outline"
          size="sm"
          className="gap-1.5"
          onClick={onReboot}
          disabled={loading.reboot}
        >
          {loading.reboot ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <RotateCcw className="size-3.5" />
          )}
          Reboot
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="gap-1.5"
          onClick={onShutdown}
          disabled={loading.shutdown}
        >
          {loading.shutdown ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <Power className="size-3.5" />
          )}
          Shutdown
        </Button>
        <Button variant="outline" size="sm" className="gap-1.5">
          <Settings className="size-3.5" />
          Configure
        </Button>
      </div>

      <Tabs defaultValue="summary" className="flex flex-col">
        <TabsList>
          <TabsTrigger value="summary">Summary</TabsTrigger>
          <TabsTrigger value="vms">VMs ({nodeVms.length})</TabsTrigger>
          <TabsTrigger value="containers">
            Containers ({nodeCts.length})
          </TabsTrigger>
          <TabsTrigger value="performance">Performance</TabsTrigger>
          <TabsTrigger value="shell" className="gap-1.5">
            <Terminal className="size-3.5" />
            Shell
          </TabsTrigger>
        </TabsList>

        {/* Summary */}
        <TabsContent value="summary" className="space-y-4 mt-4">
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <ResourceMetricCard
              label="CPU Cores"
              value={node.cpu?.cores ?? 0}
              subtitle={`${node.cpu?.used ?? 0}% used`}
              icon={<Cpu className="size-4" />}
            />
            <ResourceMetricCard
              label="Memory"
              value={`${Math.round(node.memory?.used ?? 0)} GB`}
              subtitle={`of ${node.memory?.total ?? 0} GB`}
              icon={<MemoryStick className="size-4" />}
            />
            <ResourceMetricCard
              label="Disk"
              value={`${node.disk?.used ?? 0} TB`}
              subtitle={`of ${node.disk?.total ?? 0} TB`}
              icon={<HardDrive className="size-4" />}
            />
            <ResourceMetricCard
              label="Uptime"
              value={node.uptime}
              subtitle={`Kernel ${node.kernel}`}
              icon={<Clock className="size-4" />}
            />
          </div>

          <div className="grid gap-4 lg:grid-cols-2">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">
                  System Information
                </CardTitle>
              </CardHeader>
              <CardContent>
                <ConfigList
                  items={[
                    { label: "CPU Model", value: node.cpuModel },
                    { label: "Kernel", value: node.kernel },
                    { label: "Version", value: node.version },
                    { label: "IP Address", value: node.ip },
                    { label: "Load Average", value: loadAvgStr },
                    {
                      label: "Network In",
                      value: formatNetwork(node.networkIn),
                    },
                    {
                      label: "Network Out",
                      value: formatNetwork(node.networkOut),
                    },
                  ]}
                />
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">
                  Resource Utilization
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <ResourceBar
                  label="CPU"
                  used={node.cpu?.used ?? 0}
                  total={100}
                  unit="%"
                  showPercent={false}
                />
                <ResourceBar
                  label="Memory"
                  used={Math.round(node.memory?.used ?? 0)}
                  total={node.memory?.total ?? 1}
                  unit="GB"
                />
                <ResourceBar
                  label="Disk"
                  used={node.disk?.used ?? 0}
                  total={node.disk?.total ?? 1}
                  unit="TB"
                />
                <div className="pt-2 flex items-center gap-4">
                  <div className="flex items-center gap-1.5">
                    <Monitor className="size-4 text-muted-foreground" />
                    <span className="text-sm text-foreground">
                      {nodeVms.length} VMs
                    </span>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <Box className="size-4 text-muted-foreground" />
                    <span className="text-sm text-foreground">
                      {nodeCts.length} Containers
                    </span>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        {/* VMs Tab */}
        <TabsContent value="vms" className="mt-4">
          <div className="rounded-lg border border-border bg-card overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow className="bg-secondary/30 hover:bg-secondary/30">
                  <TableHead>VMID</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>CPU</TableHead>
                  <TableHead>Memory</TableHead>
                  <TableHead>Tags</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {nodeVms.map((vm) => (
                  <TableRow key={vm.id}>
                    <TableCell className="font-mono text-sm">
                      {vm.vmid}
                    </TableCell>
                    <TableCell>
                      <Link
                        href={`/vms/view?id=${vm.vmid}`}
                        className="text-foreground hover:text-primary transition-colors font-medium"
                      >
                        {vm.name}
                      </Link>
                    </TableCell>
                    <TableCell>
                      <StatusBadge status={vmStatusToString(vm.status)} />
                    </TableCell>
                    <TableCell className="text-sm text-foreground">
                      {vm.cpu?.used ?? 0}% ({vm.cpu?.cores ?? 0}c)
                    </TableCell>
                    <TableCell className="text-sm text-foreground">
                      {vm.memory?.used ?? 0}/{vm.memory?.total ?? 0} GB
                    </TableCell>
                    <TableCell>
                      <TagList tags={vm.tags} />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </TabsContent>

        {/* Containers Tab */}
        <TabsContent value="containers" className="mt-4">
          <div className="rounded-lg border border-border bg-card overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow className="bg-secondary/30 hover:bg-secondary/30">
                  <TableHead>CTID</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>CPU</TableHead>
                  <TableHead>Memory</TableHead>
                  <TableHead>Tags</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {nodeCts.map((ct) => (
                  <TableRow key={ct.id}>
                    <TableCell className="font-mono text-sm">
                      {ct.ctid}
                    </TableCell>
                    <TableCell>
                      <Link
                        href={`/containers/view?id=${ct.ctid}`}
                        className="text-foreground hover:text-primary transition-colors font-medium"
                      >
                        {ct.name}
                      </Link>
                    </TableCell>
                    <TableCell>
                      <StatusBadge
                        status={containerStatusToString(ct.status)}
                      />
                    </TableCell>
                    <TableCell className="text-sm text-foreground">
                      {ct.cpu?.used ?? 0}% ({ct.cpu?.cores ?? 0}c)
                    </TableCell>
                    <TableCell className="text-sm text-foreground">
                      {ct.memory?.used ?? 0}/{ct.memory?.total ?? 0} GB
                    </TableCell>
                    <TableCell>
                      <TagList tags={ct.tags} />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </TabsContent>

        {/* Performance Tab */}
        <TabsContent value="performance" className="space-y-4 mt-4">
          <div className="grid gap-4 lg:grid-cols-2">
            <PerformanceChart
              title="CPU Usage (24h)"
              icon={Cpu}
              data={cpuChartData}
              color="oklch(0.65 0.18 200)"
              yDomain={[0, 100]}
              tooltipLabel="CPU"
              tooltipUnit="%"
            />
            <PerformanceChart
              title="Memory Usage (24h)"
              icon={MemoryStick}
              data={memoryChartData}
              color="oklch(0.70 0.15 145)"
              iconClassName="text-chart-2"
              tooltipLabel="Memory"
              tooltipUnit=" GB"
            />
          </div>
        </TabsContent>

        {/* Shell Tab */}
        <TabsContent value="shell" className="mt-4">
          <HostShell nodeId={node.id} nodeName={node.name} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

// Main content component that uses search params
function HostDetailView() {
  const searchParams = useSearchParams();
  const id = searchParams.get("id") || "";
  const { data: node, isLoading, error, refetch } = useNode(id);
  const { data: allVms } = useVMs();
  const { data: allContainers } = useContainers();
  const { rebootNode, shutdownNode, isRebooting, isShuttingDown } =
    useNodeMutations();

  const now = Math.floor(Date.now() / 1000);
  const start24h = now - 86400;

  const { data: cpuData } = useMetrics({
    node_id: node?.id,
    resource_type: "cpu",
    host_only: true,
    start_time: start24h,
    end_time: now,
    aggregate: "avg",
  });

  const { data: memoryData } = useMetrics({
    node_id: node?.id,
    resource_type: "memory",
    host_only: true,
    start_time: start24h,
    end_time: now,
    aggregate: "avg",
  });

  const cpuChartData = useMemo(() => {
    if (!cpuData?.metrics?.length) return [];
    return cpuData.metrics.map((m) => ({
      time: formatChartTime(m.time),
      value: Math.round(m.value * 10) / 10,
    }));
  }, [cpuData]);

  const memoryChartData = useMemo(() => {
    if (!memoryData?.metrics?.length) return [];
    return memoryData.metrics.map((m) => ({
      time: formatChartTime(m.time),
      value: Math.round(m.value * 100) / 100,
    }));
  }, [memoryData]);

  if (error) {
    return (
      <div className="p-6">
        <ErrorDisplay
          message={error.message}
          onRetry={() => refetch()}
          className="h-[50vh]"
        />
      </div>
    );
  }

  // If not loading and no node found, show 404
  if (!isLoading && !node) return notFound();

  const nodeVms =
    node && allVms ? allVms.filter((v) => v.node === node.name) : templateVms;
  const nodeCts =
    node && allContainers
      ? allContainers.filter((c) => c.node === node.name)
      : templateContainers;

  const handleReboot = () => node && rebootNode.mutate(node.id);
  const handleShutdown = () => node && shutdownNode.mutate(node.id);

  return (
    <Shimmer
      loading={isLoading}
      templateProps={{
        node: templateNode,
        nodeVms: templateVms,
        nodeCts: templateContainers,
        cpuChartData: [],
        memoryChartData: [],
        mutationProps: {
          onReboot: () => {},
          onShutdown: () => {},
          loading: { reboot: false, shutdown: false },
        },
      }}
    >
      <HostDetailContent
        node={node || templateNode}
        nodeVms={nodeVms}
        nodeCts={nodeCts}
        cpuChartData={cpuChartData}
        memoryChartData={memoryChartData}
        mutationProps={{
          onReboot: handleReboot,
          onShutdown: handleShutdown,
          loading: { reboot: isRebooting, shutdown: isShuttingDown },
        }}
      />
    </Shimmer>
  );
}

// Default export with Suspense boundary for static export compatibility
export default function HostViewPage() {
  return (
    <Suspense fallback={<div className="p-8 text-center">Loading...</div>}>
      <HostDetailView />
    </Suspense>
  );
}
