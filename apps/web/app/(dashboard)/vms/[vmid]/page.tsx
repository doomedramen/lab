"use client";

import { use, useState } from "react";
import { useRouter } from "next/navigation";
import { notFound } from "next/navigation";
import Link from "next/link";
import dynamic from "next/dynamic";
import {
  StatusBadge,
  ResourceBar,
  TagList,
} from "@/components/lab-shared";
import {
  TabsPersistent,
  TabsList,
  TabsTrigger,
  TabsContent,
} from "@/components/tabs-persistent";
import { ResourceMetricCard } from "@/components/resource-metric-card";
import { ConfigList } from "@/components/config-list";
import { ResourceConfigItem } from "@/components/resource-config-item";
import { PageHeader } from "@/components/page-header";
import { PerformanceChart } from "@/components/performance-chart";
import { EntityActionButtons } from "@/components/entity-action-buttons";
import { Shimmer } from "@/components/shimmer";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  useVM,
  useVMLogs,
  useVMDiagnostics,
  useSnapshots,
  useBackups,
  useGuestNetworkInterfaces,
  useMetrics,
} from "@/lib/api/queries";
import { useVMMutations } from "@/lib/api/mutations";
import {
  useVMDisks,
  useStoragePools,
  useStorageDisks,
} from "@/lib/api/queries/storage";
import {
  useVMDiskMutations,
  useStorageDiskMutations,
} from "@/lib/api/mutations/storage";
import { ErrorDisplay } from "@/components/error-display";
import { LogViewer } from "@/components/log-viewer";
import { SnapshotList } from "@/components/snapshot-list";
import { BackupList } from "@/components/backup-list";
import { CreateSnapshotModal } from "@/components/create-snapshot-modal";
import { CreateBackupModal } from "@/components/create-backup-modal";
import type { VM } from "@/lib/gen/lab/v1/vm_pb";
import { VmStatus } from "@/lib/gen/lab/v1/vm_pb";
import {
  OsType,
  NetworkType,
  MachineType,
  BiosType,
} from "@/lib/gen/lab/v1/common_pb";
import { DiskBus, DiskFormat } from "@/lib/gen/lab/v1/storage_pb";
import {
  vmStatusToString,
  osTypeToString,
  machineTypeToString,
  biosTypeToString,
  networkModelToString,
  diskBusToString,
  diskFormatToString,
  diskBusFromString,
  diskFormatFromString,
} from "@/lib/api/enum-helpers";
import {
  Monitor,
  Cpu,
  MemoryStick,
  HardDrive,
  Clock,
  Shield,
  Network,
  RefreshCw,
  Plus,
  Loader2,
  Trash2,
  Edit,
  AlertTriangle,
} from "lucide-react";

// Lazy-load VNCConsole (requires DOM / react-vnc)
const VNCConsole = dynamic(() => import("@/components/vnc-console"), {
  ssr: false,
});

// Lazy-load SerialConsole (requires DOM / xterm.js)
const SerialConsole = dynamic(() => import("@/components/serial-console"), {
  ssr: false,
});

// Lazy-load VMDiagnosticsPanel (requires DOM)
const VMDiagnosticsPanel = dynamic(
  () => import("@/components/vm-diagnostics-panel"),
  { ssr: false },
);

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

// Template VM for shimmer
const templateVM = {
  id: "vm-template",
  vmid: 100,
  name: "Loading VM...",
  node: "Loading...",
  status: VmStatus.RUNNING,
  cpu: { used: 45, sockets: 1, cores: 4 },
  memory: { used: 6, total: 8 },
  disk: { used: 32, total: 64 },
  uptime: "Loading...",
  os: { osType: OsType.LINUX, version: "Loading..." },
  arch: "x86_64",
  machineType: MachineType.PC,
  bios: BiosType.SEABIOS,
  cpuModel: "host-passthrough",
  network: [],
  ip: "Loading...",
  tags: ["loading"],
  ha: true,
  description: "Loading description...",
  nestedVirt: false,
  startOnBoot: true,
  agent: true,
  tpm: false,
  secureBoot: false,
} as unknown as VM;

type ConfirmAction = "stop" | "shutdown" | "reboot" | "delete" | null;

interface ConfirmDialogConfig {
  action: ConfirmAction;
  title: string;
  description: string;
  confirmLabel: string;
  destructive?: boolean;
}

const confirmConfigs: Record<
  NonNullable<ConfirmAction>,
  ConfirmDialogConfig
> = {
  stop: {
    action: "stop",
    title: "Force Stop VM",
    description:
      "This will immediately power off the VM without a graceful shutdown. Any unsaved data may be lost. Are you sure?",
    confirmLabel: "Force Stop",
    destructive: true,
  },
  shutdown: {
    action: "shutdown",
    title: "Shutdown VM",
    description:
      "This will send an ACPI shutdown signal to the VM. The VM will attempt to shut down gracefully.",
    confirmLabel: "Shutdown",
  },
  reboot: {
    action: "reboot",
    title: "Reboot VM",
    description: "This will reboot the VM. Are you sure?",
    confirmLabel: "Reboot",
  },
  delete: {
    action: "delete",
    title: "Delete VM",
    description:
      "This will permanently delete the VM and all its disk data. This action cannot be undone. The VM must be stopped before deletion.",
    confirmLabel: "Delete",
    destructive: true,
  },
};

function VMDetailContent({
  vm,
  mutationProps,
}: {
  vm: VM;
  mutationProps: {
    onPlay: () => void;
    onStop: () => void;
    onShutdown: () => void;
    onPause: () => void;
    onResume: () => void;
    onReboot: () => void;
    onConsole: (
      type: import("@/components/entity-action-buttons").ConsoleType,
    ) => void;
    onClone: () => void;
    onDelete: () => void;
    loading: {
      start: boolean;
      stop: boolean;
      shutdown: boolean;
      pause: boolean;
      resume: boolean;
      reboot: boolean;
      clone: boolean;
      delete: boolean;
      console: boolean;
    };
  };
}) {
  const {
    onPlay,
    onStop,
    onShutdown,
    onPause,
    onResume,
    onReboot,
    onConsole,
    onClone,
    onDelete,
    loading,
  } = mutationProps;

  // Guest network interfaces from QEMU agent
  const { data: guestNetworkData } = useGuestNetworkInterfaces(vm.vmid);

  // VM performance metrics (24h)
  const now = Math.floor(Date.now() / 1000)
  const start24h = now - 86400

  const { data: cpuData } = useMetrics({
    resource_type: "cpu",
    resource_id: vm.vmid.toString(),
    start_time: start24h,
    end_time: now,
    aggregate: "avg",
  })

  const { data: memoryData } = useMetrics({
    resource_type: "memory",
    resource_id: vm.vmid.toString(),
    start_time: start24h,
    end_time: now,
    aggregate: "avg",
  })

  const { data: diskData } = useMetrics({
    resource_type: "disk",
    resource_id: vm.vmid.toString(),
    start_time: start24h,
    end_time: now,
    aggregate: "avg",
  })

  // Transform metrics to chart format
  const cpuChartData = cpuData?.metrics?.map((m) => ({
    time: new Date(m.time).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
    value: Math.round(m.value * 10) / 10,
  })) ?? []

  const memoryChartData = memoryData?.metrics?.map((m) => ({
    time: new Date(m.time).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
    value: Math.round(m.value * 10) / 10,
  })) ?? []

  const diskChartData = diskData?.metrics?.map((m) => ({
    time: new Date(m.time).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
    value: Math.round(m.value * 100) / 100,
  })) ?? []

  const osLabel =
    vm.os?.version || osTypeToString(vm.os?.osType ?? OsType.UNSPECIFIED);

  const badges = (
    <>
      <Badge
        variant="outline"
        className="font-mono text-xs"
        data-testid="vm-detail-vmid"
      >
        {vm.vmid}
      </Badge>
      <StatusBadge
        status={vmStatusToString(vm.status)}
        data-testid="vm-detail-status"
      />
      {vm.ha && (
        <Badge
          variant="outline"
          className="text-primary border-primary/30 gap-1 text-xs"
        >
          <Shield className="size-3" />
          HA
        </Badge>
      )}
    </>
  );

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        backHref="/vms"
        backLabel="Back to Virtual Machines"
        title={vm.name}
        subtitle={`${osLabel} - ${vm.node} - ${vm.ip}`}
        icon={<Monitor className="size-5 text-foreground" />}
        badges={badges}
      />

      <div className="flex justify-end">
        <EntityActionButtons
          status={vmStatusToString(vm.status)}
          variant="vm"
          onPlay={onPlay}
          onStop={onStop}
          onShutdown={onShutdown}
          onPause={onPause}
          onResume={onResume}
          onReboot={onReboot}
          onConsole={onConsole}
          onClone={onClone}
          onDelete={onDelete}
          loading={loading}
        />
      </div>

      <TabsPersistent
        defaultValue="summary"
        className="flex flex-col"
        paramKey="vm-tab"
      >
        <TabsList>
          <TabsTrigger value="summary">Summary</TabsTrigger>
          <TabsTrigger value="hardware">Hardware</TabsTrigger>
          <TabsTrigger value="disks">Disks</TabsTrigger>
          <TabsTrigger value="performance">Performance</TabsTrigger>
          <TabsTrigger value="network">Network</TabsTrigger>
          <TabsTrigger value="logs">Logs</TabsTrigger>
          <TabsTrigger value="diagnostics">Diagnostics</TabsTrigger>
          <TabsTrigger value="snapshots">Snapshots</TabsTrigger>
          <TabsTrigger value="backups">Backups</TabsTrigger>
        </TabsList>

        {/* Summary */}
        <TabsContent value="summary" className="space-y-4 mt-4">
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <ResourceMetricCard
              label="CPU Usage"
              value={`${vm.cpu?.used ?? 0}%`}
              subtitle={`${vm.cpu?.cores ?? 0} vCPUs`}
              icon={<Cpu className="size-4" />}
            />
            <ResourceMetricCard
              label="Memory"
              value={`${vm.memory?.used ?? 0} GB`}
              subtitle={`of ${vm.memory?.total ?? 0} GB`}
              icon={<MemoryStick className="size-4" />}
            />
            <ResourceMetricCard
              label="Disk"
              value={`${vm.disk?.used ?? 0} GB`}
              subtitle={`of ${vm.disk?.total ?? 0} GB`}
              icon={<HardDrive className="size-4" />}
            />
            <ResourceMetricCard
              label="Uptime"
              value={vm.uptime}
              subtitle={osLabel}
              icon={<Clock className="size-4" />}
            />
          </div>

          <div className="grid gap-4 lg:grid-cols-2">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">
                  Configuration
                </CardTitle>
              </CardHeader>
              <CardContent>
                <ConfigList
                  items={[
                    { label: "VMID", value: String(vm.vmid) },
                    { label: "Name", value: vm.name },
                    { label: "OS", value: osLabel },
                    { label: "Node", value: vm.node },
                    { label: "IP Address", value: vm.ip },
                    { label: "HA Enabled", value: vm.ha ? "Yes" : "No" },
                    {
                      label: "Start on Boot",
                      value: vm.startOnBoot ? "Yes" : "No",
                    },
                    {
                      label: "QEMU Agent",
                      value: vm.agent ? "Enabled" : "Disabled",
                    },
                    {
                      label: "Nested Virt",
                      value: vm.nestedVirt ? "Enabled" : "Disabled",
                    },
                    {
                      label: "TPM 2.0",
                      value: vm.tpm ? "Enabled" : "Disabled",
                    },
                    {
                      label: "Secure Boot",
                      value: vm.secureBoot ? "Enabled" : "Disabled",
                    },
                  ]}
                />
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Resources</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <ResourceBar
                  label="CPU"
                  used={vm.cpu?.used ?? 0}
                  total={100}
                  unit="%"
                  showPercent={false}
                />
                <ResourceBar
                  label="Memory"
                  used={vm.memory?.used ?? 0}
                  total={vm.memory?.total ?? 1}
                  unit="GB"
                />
                <ResourceBar
                  label="Disk"
                  used={vm.disk?.used ?? 0}
                  total={vm.disk?.total ?? 1}
                  unit="GB"
                />
                <div className="pt-2">
                  <div className="text-xs text-muted-foreground mb-1.5">
                    Tags
                  </div>
                  <TagList tags={vm.tags} />
                </div>
                {vm.description && (
                  <div className="pt-1">
                    <div className="text-xs text-muted-foreground mb-1">
                      Description
                    </div>
                    <p className="text-sm text-foreground">{vm.description}</p>
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        {/* Hardware */}
        <TabsContent value="hardware" className="mt-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">
                Hardware Configuration
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <ResourceConfigItem
                  icon={<Cpu className="size-4 text-muted-foreground" />}
                  label="Processors"
                  detail={`${vm.cpu?.sockets ?? 1} socket${(vm.cpu?.sockets ?? 1) !== 1 ? "s" : ""} × ${vm.cpu?.cores ?? 0} core${(vm.cpu?.cores ?? 0) !== 1 ? "s" : ""} — ${vm.cpuModel}`}
                  value=""
                  badge={machineTypeToString(vm.machineType)}
                />
                <ResourceConfigItem
                  icon={
                    <MemoryStick className="size-4 text-muted-foreground" />
                  }
                  label="Memory"
                  detail={`${vm.memory?.total ?? 0} GB`}
                  value=""
                  badge={biosTypeToString(vm.bios)}
                />
                <ResourceConfigItem
                  icon={<HardDrive className="size-4 text-muted-foreground" />}
                  label="Hard Disk"
                  detail={`${vm.disk?.total ?? 0} GB`}
                  value=""
                  badge={vm.arch}
                />
                {(vm.network ?? []).length > 0 ? (
                  (vm.network ?? []).map((nic, i) => (
                    <ResourceConfigItem
                      key={i}
                      icon={
                        <Network className="size-4 text-muted-foreground" />
                      }
                      label={`Network (net${i})`}
                      detail={
                        nic.type === NetworkType.BRIDGE
                          ? `bridge=${nic.bridge}${nic.vlan ? `, vlan=${nic.vlan}` : ""}`
                          : "user-mode (NAT)"
                      }
                      value=""
                      badge={networkModelToString(nic.model)}
                    />
                  ))
                ) : (
                  <ResourceConfigItem
                    icon={<Network className="size-4 text-muted-foreground" />}
                    label="Network"
                    value="No interfaces configured"
                  />
                )}
                <ResourceConfigItem
                  icon={<Monitor className="size-4 text-muted-foreground" />}
                  label="Display"
                  value="VNC"
                  badge="vnc"
                />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Disks */}
        <TabsContent value="disks" className="mt-4">
          <DisksTab vmid={vm.vmid} vmStatus={vmStatusToString(vm.status)} />
        </TabsContent>

        {/* Performance */}
        <TabsContent value="performance" className="space-y-4 mt-4">
          <div className="grid gap-4 lg:grid-cols-2">
            <PerformanceChart
              title="CPU Usage (24h)"
              icon={Cpu}
              data={cpuChartData}
              color="oklch(0.65 0.18 200)"
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
              tooltipUnit="%"
            />
            <PerformanceChart
              title="Disk I/O (24h)"
              icon={HardDrive}
              data={diskChartData}
              color="oklch(0.75 0.15 55)"
              iconClassName="text-chart-3"
              className="lg:col-span-2"
              tooltipLabel="I/O"
              tooltipUnit=" MB/s"
            />
          </div>
        </TabsContent>

        {/* Network */}
        <TabsContent value="network" className="mt-4">
          <Card>
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <CardTitle className="text-sm font-medium">
                  Network Interfaces
                </CardTitle>
                {vm.agent && (
                  <div className="flex items-center gap-2">
                    <Badge
                      variant={
                        guestNetworkData?.agentConnected
                          ? "default"
                          : "secondary"
                      }
                      className="text-xs"
                    >
                      {guestNetworkData?.agentConnected
                        ? "Agent Connected"
                        : "Agent Disconnected"}
                    </Badge>
                  </div>
                )}
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {/* Guest Agent Discovered Interfaces */}
                {guestNetworkData?.agentConnected &&
                guestNetworkData.interfaces.length > 0 ? (
                  guestNetworkData.interfaces.map((iface, index) => (
                    <div
                      key={index}
                      className="rounded-md border border-border bg-secondary/30 px-4 py-3"
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <Network className="size-4 text-muted-foreground" />
                          <div>
                            <div className="text-sm font-medium text-foreground">
                              {iface.name}
                            </div>
                            <div className="text-xs text-muted-foreground">
                              From QEMU Guest Agent
                            </div>
                          </div>
                        </div>
                        <div className="flex items-center gap-4">
                          <div className="text-right">
                            <div className="text-xs text-muted-foreground">
                              IP Addresses
                            </div>
                            <div className="text-sm font-medium text-foreground">
                              {iface.ipAddresses.map((ip, ipIndex) => (
                                <span key={ipIndex} className="block">
                                  {ip.address}/{ip.prefix}
                                </span>
                              ))}
                            </div>
                          </div>
                          <div className="text-right">
                            <div className="text-xs text-muted-foreground">
                              MAC
                            </div>
                            <div className="text-sm font-mono text-foreground">
                              {iface.macAddress || "N/A"}
                            </div>
                          </div>
                          <StatusBadge status="online" />
                        </div>
                      </div>
                    </div>
                  ))
                ) : vm.agent && vm.status === VmStatus.RUNNING ? (
                  // Agent enabled but not connected
                  <div className="rounded-md border border-border bg-secondary/30 px-4 py-3">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <Network className="size-4 text-muted-foreground" />
                        <div>
                          <div className="text-sm font-medium text-foreground">
                            net0 (eth0)
                          </div>
                          <div className="text-xs text-muted-foreground">
                            VirtIO, bridge=vmbr0
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-4">
                        <div className="text-right">
                          <div className="text-xs text-muted-foreground">
                            IP Address
                          </div>
                          <div className="text-sm font-medium text-foreground">
                            {vm.ip || "Discovering..."}
                          </div>
                        </div>
                        <div className="text-right">
                          <div className="text-xs text-muted-foreground">
                            MAC
                          </div>
                          <div className="text-sm font-mono text-foreground">
                            BC:24:11:AB:CD:{String(vm.vmid).slice(-2)}
                          </div>
                        </div>
                        <StatusBadge status="offline" />
                      </div>
                    </div>
                  </div>
                ) : (
                  // No agent or VM stopped
                  <div className="rounded-md border border-border bg-secondary/30 px-4 py-3">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <Network className="size-4 text-muted-foreground" />
                        <div>
                          <div className="text-sm font-medium text-foreground">
                            net0 (eth0)
                          </div>
                          <div className="text-xs text-muted-foreground">
                            VirtIO, bridge=vmbr0
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-4">
                        <div className="text-right">
                          <div className="text-xs text-muted-foreground">
                            IP Address
                          </div>
                          <div className="text-sm font-medium text-foreground">
                            {vm.ip || "Unknown"}
                          </div>
                        </div>
                        <div className="text-right">
                          <div className="text-xs text-muted-foreground">
                            MAC
                          </div>
                          <div className="text-sm font-mono text-foreground">
                            BC:24:11:AB:CD:{String(vm.vmid).slice(-2)}
                          </div>
                        </div>
                        <StatusBadge
                          status={
                            vm.status === VmStatus.RUNNING
                              ? "online"
                              : "offline"
                          }
                        />
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Logs */}
        <TabsContent value="logs" className="mt-4">
          <Card className="flex flex-col">
            <CardContent className="p-0 h-[600px]">
              <LogsTab vmid={vm?.vmid ?? 0} />
            </CardContent>
          </Card>
        </TabsContent>

        {/* Diagnostics */}
        <TabsContent value="diagnostics" className="mt-4">
          <DiagnosticsTab vmid={vm?.vmid ?? 0} />
        </TabsContent>
      </TabsPersistent>
    </div>
  );
}

export default function VMDetailPage({
  params,
}: {
  params: Promise<{ vmid: string }>;
}) {
  const { vmid } = use(params);
  const router = useRouter();
  const { data: vm, isLoading, error, refetch } = useVM(vmid);

  // Confirmation dialog state
  const [confirmAction, setConfirmAction] = useState<ConfirmAction>(null);

  // Clone dialog state
  const [cloneDialogOpen, setCloneDialogOpen] = useState(false);
  const [cloneName, setCloneName] = useState("");
  const [cloneFull, setCloneFull] = useState(true);
  const [cloneDescription, setCloneDescription] = useState("");
  const [cloneStartAfter, setCloneStartAfter] = useState(false);

  // Console dialog state
  const [consoleOpen, setConsoleOpen] = useState(false);
  const [consoleWsUrl, setConsoleWsUrl] = useState<string | null>(null);
  const [consoleType, setConsoleType] =
    useState<import("@/components/entity-action-buttons").ConsoleType>(
      "serial",
    );

  const {
    startVM,
    stopVM,
    shutdownVM,
    pauseVM,
    resumeVM,
    rebootVM,
    deleteVM,
    cloneVM,
    getConsole,
    isStarting,
    isStopping,
    isShuttingDown,
    isPausing,
    isResuming,
    isRebooting,
    isDeleting,
    isCloning,
    isGettingConsole,
  } = useVMMutations({
    onDeleteSuccess: () => router.push("/vms"),
    onCloneSuccess: () => {
      setCloneDialogOpen(false);
      setCloneName("");
      setCloneDescription("");
    },
  });

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

  if (!isLoading && !vm) return notFound();

  const handleStart = () => vm && startVM.mutate(vm.vmid);
  const handlePause = () => vm && pauseVM.mutate(vm.vmid);
  const handleResume = () => vm && resumeVM.mutate(vm.vmid);

  // Actions that need confirmation
  const handleStop = () => setConfirmAction("stop");
  const handleShutdown = () => setConfirmAction("shutdown");
  const handleReboot = () => setConfirmAction("reboot");
  const handleDelete = () => setConfirmAction("delete");

  // Clone action - opens clone dialog
  const handleClone = () => {
    if (vm) {
      setCloneName(`${vm.name}-clone`);
      setCloneDialogOpen(true);
    }
  };

  const handleCloneConfirm = () => {
    if (!vm) return;
    cloneVM.mutate({
      sourceVmid: vm.vmid,
      name: cloneName,
      full: cloneFull,
      description: cloneDescription,
      startAfterClone: cloneStartAfter,
    });
  };

  const handleConfirm = () => {
    if (!vm || !confirmAction) return;
    switch (confirmAction) {
      case "stop":
        stopVM.mutate(vm.vmid);
        break;
      case "shutdown":
        shutdownVM.mutate(vm.vmid);
        break;
      case "reboot":
        rebootVM.mutate(vm.vmid);
        break;
      case "delete":
        deleteVM.mutate(vm.vmid);
        break;
    }
    setConfirmAction(null);
  };

  const handleConsole = (
    type: import("@/components/entity-action-buttons").ConsoleType,
  ) => {
    if (!vm) return;
    // Set the console type and open dialog
    setConsoleType(type);
    setConsoleWsUrl(null);
    setConsoleOpen(true);

    getConsole.mutate(
      { vmid: vm.vmid, consoleType: type },
      {
        onSuccess: (res) => {
          // Construct WebSocket URL from API base URL and the token
          const wsBase = API_BASE_URL.replace(/^http/, "ws");
          setConsoleWsUrl(`${wsBase}${res.websocketUrl}?token=${res.token}`);
        },
        onError: (error) => {
          // Keep dialog open to show error state
          setConsoleWsUrl(null);
        },
      },
    );
  };

  const currentConfig = confirmAction ? confirmConfigs[confirmAction] : null;

  return (
    <>
      <Shimmer
        loading={isLoading}
        templateProps={{ vm: templateVM, mutationProps: {} }}
      >
        <VMDetailContent
          vm={vm || templateVM}
          mutationProps={{
            onPlay: handleStart,
            onStop: handleStop,
            onShutdown: handleShutdown,
            onPause: handlePause,
            onResume: handleResume,
            onReboot: handleReboot,
            onConsole: handleConsole,
            onClone: handleClone,
            onDelete: handleDelete,
            loading: {
              start: isStarting,
              stop: isStopping,
              shutdown: isShuttingDown,
              pause: isPausing,
              resume: isResuming,
              reboot: isRebooting,
              clone: isCloning,
              delete: isDeleting,
              console: isGettingConsole,
            },
          }}
        />
      </Shimmer>

      {/* Confirmation dialog */}
      <Dialog
        open={!!confirmAction}
        onOpenChange={(open) => !open && setConfirmAction(null)}
      >
        <DialogContent data-testid="confirm-dialog">
          <DialogHeader>
            <DialogTitle data-testid="confirm-dialog-title">
              {currentConfig?.title}
            </DialogTitle>
            <DialogDescription>{currentConfig?.description}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setConfirmAction(null)}
              data-testid="confirm-dialog-cancel"
            >
              Cancel
            </Button>
            <Button
              variant={currentConfig?.destructive ? "destructive" : "default"}
              onClick={handleConfirm}
              data-testid="confirm-dialog-confirm"
            >
              {currentConfig?.confirmLabel}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Console dialog (supports Serial, VNC, websockify) */}
      <Dialog
        open={consoleOpen}
        onOpenChange={(open) => {
          setConsoleOpen(open);
          if (!open) {
            setConsoleWsUrl(null);
          }
        }}
      >
        <DialogContent
          className="max-w-[95vw] w-[95vw] h-[90vh] flex flex-col p-0"
          data-testid="console-dialog"
        >
          <DialogHeader className="px-4 pt-4 pb-2 shrink-0">
            <DialogTitle>
              {consoleType === "serial"
                ? "Serial"
                : consoleType === "vnc"
                  ? "VNC"
                  : "Console"}{" "}
              — {vm?.name}
            </DialogTitle>
          </DialogHeader>
          <div
            className="flex-1 overflow-hidden"
            data-testid="console-container"
          >
            {consoleOpen && !consoleWsUrl && !isGettingConsole && (
              <div className="flex items-center justify-center h-full">
                <div className="text-center text-muted-foreground">
                  <p className="text-sm mb-2">Failed to load console</p>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleConsole(consoleType)}
                  >
                    <RefreshCw className="size-3 mr-2" />
                    Retry
                  </Button>
                </div>
              </div>
            )}
            {consoleOpen && !consoleWsUrl && isGettingConsole && (
              <div className="flex items-center justify-center h-full">
                <div className="text-center text-muted-foreground">
                  <RefreshCw className="size-6 animate-spin mx-auto mb-2" />
                  <p className="text-sm">Connecting to console...</p>
                </div>
              </div>
            )}
            {consoleOpen && consoleWsUrl && consoleType === "serial" && (
              <SerialConsole websocketUrl={consoleWsUrl} />
            )}
            {consoleOpen && consoleWsUrl && consoleType === "vnc" && (
              <VNCConsole websocketUrl={consoleWsUrl} />
            )}
            {consoleOpen && consoleWsUrl && consoleType === "websockify" && (
              <div className="flex items-center justify-center h-full">
                <div className="text-center text-muted-foreground">
                  <p className="text-sm">websockify is not yet implemented</p>
                </div>
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>

      {/* Clone dialog */}
      <Dialog open={cloneDialogOpen} onOpenChange={setCloneDialogOpen}>
        <DialogContent data-testid="clone-dialog">
          <DialogHeader>
            <DialogTitle>Clone VM</DialogTitle>
            <DialogDescription>
              Create a copy of "{vm?.name}". Full clones are independent copies,
              while linked clones share the base disk with the source VM.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="clone-name">Clone Name</Label>
              <Input
                id="clone-name"
                value={cloneName}
                onChange={(e) => setCloneName(e.target.value)}
                placeholder="Enter name for the cloned VM"
                data-testid="clone-name-input"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="clone-description">Description (optional)</Label>
              <Input
                id="clone-description"
                value={cloneDescription}
                onChange={(e) => setCloneDescription(e.target.value)}
                placeholder="Enter a description for the clone"
                data-testid="clone-description-input"
              />
            </div>
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="clone-full">Full Clone</Label>
                <p className="text-xs text-muted-foreground">
                  Full clones are independent but use more disk space
                </p>
              </div>
              <Switch
                id="clone-full"
                checked={cloneFull}
                onCheckedChange={setCloneFull}
                data-testid="clone-full-switch"
              />
            </div>
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="clone-start">Start after clone</Label>
                <p className="text-xs text-muted-foreground">
                  Automatically start the cloned VM when ready
                </p>
              </div>
              <Switch
                id="clone-start"
                checked={cloneStartAfter}
                onCheckedChange={setCloneStartAfter}
                data-testid="clone-start-switch"
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setCloneDialogOpen(false)}
              data-testid="clone-cancel-button"
            >
              Cancel
            </Button>
            <Button
              onClick={handleCloneConfirm}
              disabled={!cloneName.trim() || isCloning}
              data-testid="clone-confirm-button"
            >
              {isCloning ? (
                <>
                  <Loader2 className="size-4 animate-spin mr-2" />
                  Cloning...
                </>
              ) : (
                "Clone"
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

function LogsTab({ vmid }: { vmid: number }) {
  const { data, isLoading, refetch } = useVMLogs({ vmid });
  const entries = data?.entries ?? [];

  return (
    <LogViewer
      entries={entries}
      isLoading={isLoading}
      onRefresh={() => refetch()}
    />
  );
}

function DiagnosticsTab({ vmid }: { vmid: number }) {
  const { data: diagnostics, isLoading, refetch } = useVMDiagnostics(vmid);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <RefreshCw className="size-8 animate-spin text-muted-foreground" />
        <span className="ml-2 text-muted-foreground">
          Loading diagnostics...
        </span>
      </div>
    );
  }

  if (!diagnostics) {
    return (
      <div className="text-center text-muted-foreground py-8">
        Failed to load diagnostics information
      </div>
    );
  }

  return (
    <VMDiagnosticsPanel
      vmid={vmid}
      data={{
        info: diagnostics.info
          ? {
              id: diagnostics.info.id,
              name: diagnostics.info.name,
              uuid: diagnostics.info.uuid,
              osType: diagnostics.info.osType,
              state: diagnostics.info.state,
              maxMemoryKb: diagnostics.info.maxMemoryKb,
              usedMemoryKb: diagnostics.info.usedMemoryKb,
              cpuCount: diagnostics.info.cpuCount,
              autostart: diagnostics.info.autostart,
              persistent: diagnostics.info.persistent,
            }
          : undefined,
        xmlConfig: diagnostics.xmlConfig,
        networkInterfaces: diagnostics.networkInterfaces?.map((ni) => ({
          name: ni.name,
          macAddress: ni.macAddress,
          protocol: ni.protocol,
          address: ni.address,
          prefix: Number(ni.prefix),
        })),
        disks: diagnostics.disks?.map((d) => ({
          targetDev: d.targetDev,
          sourceFile: d.sourceFile,
          driverType: d.driverType,
          bus: d.bus,
        })),
        qemuMonitor: diagnostics.qemuMonitor
          ? {
              vncServer: diagnostics.qemuMonitor.vncServer,
              vncPort: diagnostics.qemuMonitor.vncPort,
              charDevices: diagnostics.qemuMonitor.charDevices?.map((cd) => ({
                name: cd.name,
                sourcePath: cd.sourcePath,
              })),
            }
          : undefined,
        host: diagnostics.host
          ? {
              hostname: diagnostics.host.hostname,
              arch: diagnostics.host.arch,
              libvirtUri: diagnostics.host.libvirtUri,
              libvirtVersion: diagnostics.host.libvirtVersion,
            }
          : undefined,
      }}
      onRefresh={refetch}
      isLoading={isLoading}
    />
  );
}

function SnapshotsTab({
  vmid,
  vmName,
  vmState,
}: {
  vmid: number;
  vmName: string;
  vmState?: string;
}) {
  const { data, isLoading } = useSnapshots(vmid);

  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium">Snapshots</CardTitle>
          <CreateSnapshotModal vmid={vmid} vmName={vmName} vmState={vmState} />
        </div>
      </CardHeader>
      <CardContent>
        <SnapshotList
          vmid={vmid}
          vmName={vmName}
          vmState={vmState}
          snapshots={data?.snapshots}
          tree={data?.tree}
          isLoading={isLoading}
        />
      </CardContent>
    </Card>
  );
}

function BackupsTab({ vmid }: { vmid: number }) {
  const { data, isLoading } = useBackups(vmid);

  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium">Backups</CardTitle>
          <CreateBackupModal vmid={vmid} vmName="this VM" />
        </div>
      </CardHeader>
      <CardContent>
        <BackupList vmid={vmid} backups={data?.backups} isLoading={isLoading} />
      </CardContent>
    </Card>
  );
}

// Helper function to format bytes to human-readable size
function formatBytes(bytes: number | bigint): string {
  const numBytes = typeof bytes === "bigint" ? Number(bytes) : bytes;
  if (numBytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(numBytes) / Math.log(k));
  return `${parseFloat((numBytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
}

// Helper function to convert GB to bytes
function gbToBytes(gb: number): bigint {
  return BigInt(Math.floor(gb * 1024 * 1024 * 1024));
}

// Disk table row actions type
interface DiskRowActionsProps {
  disk: { target: string; sizeBytes: bigint; path: string };
  isRootDisk: boolean;
  vmStatus: string;
  onResize: () => void;
  onDetach: () => void;
}

function DiskRowActions({
  disk,
  isRootDisk,
  vmStatus,
  onResize,
  onDetach,
}: DiskRowActionsProps) {
  const isRunning = vmStatus === "running";

  return (
    <div className="flex items-center gap-1">
      <Button
        variant="ghost"
        size="sm"
        onClick={onResize}
        className="h-7 px-2"
        title="Resize disk"
      >
        <Edit className="size-3.5" />
      </Button>
      <Button
        variant="ghost"
        size="sm"
        onClick={onDetach}
        className="h-7 px-2 text-destructive hover:text-destructive"
        disabled={isRootDisk}
        title={isRootDisk ? "Cannot detach root disk" : "Detach disk"}
      >
        <Trash2 className="size-3.5" />
      </Button>
    </div>
  );
}

function DisksTab({ vmid, vmStatus }: { vmid: number; vmStatus: string }) {
  // Fetch VM disks
  const {
    data: disks,
    isLoading: disksLoading,
    refetch: refetchDisks,
  } = useVMDisks(vmid);

  // Fetch storage pools for disk creation
  const { data: poolsData } = useStoragePools(undefined, undefined, true);
  const pools = poolsData?.pools ?? [];

  // Fetch unassigned disks for attach dialog
  const { data: unassignedDisksData, refetch: refetchUnassigned } =
    useStorageDisks(undefined, undefined, true);
  const unassignedDisks = unassignedDisksData?.disks ?? [];

  // Mutations
  const {
    attachDisk,
    detachDisk,
    resizeDisk,
    isAttaching,
    isDetaching,
    isResizing,
  } = useVMDiskMutations(vmid);
  const { createDisk, isCreating: isCreatingDisk } = useStorageDiskMutations({
    onCreateSuccess: () => {
      refetchUnassigned();
    },
  });

  // Dialog states
  const [attachDialogOpen, setAttachDialogOpen] = useState(false);
  const [detachDialogOpen, setDetachDialogOpen] = useState(false);
  const [resizeDialogOpen, setResizeDialogOpen] = useState(false);

  // Selected disk for actions
  const [selectedDisk, setSelectedDisk] = useState<{
    target: string;
    sizeBytes: bigint;
    path: string;
  } | null>(null);

  // Attach mode: 'existing' | 'create'
  const [attachMode, setAttachMode] = useState<"existing" | "create">(
    "existing",
  );

  // Attach form state
  const [selectedPoolId, setSelectedPoolId] = useState("");
  const [selectedDiskPath, setSelectedDiskPath] = useState("");
  const [newDiskName, setNewDiskName] = useState("");
  const [newDiskSizeGb, setNewDiskSizeGb] = useState(10);
  const [newDiskFormat, setNewDiskFormat] = useState<"qcow2" | "raw">("qcow2");
  const [newDiskBus, setNewDiskBus] = useState<"virtio" | "sata" | "scsi">(
    "virtio",
  );
  const [attachReadonly, setAttachReadonly] = useState(false);

  // Resize form state
  const [resizeSizeGb, setResizeSizeGb] = useState(0);

  // Detach form state
  const [deleteAfterDetach, setDeleteAfterDetach] = useState(false);

  // Filter unassigned disks by selected pool (when in existing mode)
  const filteredUnassignedDisks = selectedPoolId
    ? unassignedDisks.filter((d) => d.poolId === selectedPoolId)
    : unassignedDisks;

  // Reset attach form when dialog opens
  const handleAttachDialogOpen = (open: boolean) => {
    setAttachDialogOpen(open);
    if (open) {
      setAttachMode("existing");
      setSelectedPoolId(pools[0]?.id ?? "");
      setSelectedDiskPath("");
      setNewDiskName("");
      setNewDiskSizeGb(10);
      setNewDiskFormat("qcow2");
      setNewDiskBus("virtio");
      setAttachReadonly(false);
      refetchUnassigned();
    }
  };

  // Handle resize dialog open
  const handleResizeDialogOpen = (open: boolean) => {
    setResizeDialogOpen(open);
    if (!open) {
      setSelectedDisk(null);
      setResizeSizeGb(0);
    }
  };

  // Handle detach dialog open
  const handleDetachDialogOpen = (open: boolean) => {
    setDetachDialogOpen(open);
    if (!open) {
      setSelectedDisk(null);
      setDeleteAfterDetach(false);
    }
  };

  // Handle attach disk submit
  const handleAttachSubmit = async () => {
    if (attachMode === "existing") {
      if (!selectedDiskPath) return;
      attachDisk.mutate(
        {
          vmid,
          diskPath: selectedDiskPath,
          bus: diskBusFromString(newDiskBus),
          readonly: attachReadonly,
        },
        {
          onSuccess: () => {
            setAttachDialogOpen(false);
            refetchDisks();
          },
        },
      );
    } else {
      // Create new disk first
      if (!selectedPoolId || !newDiskName.trim()) return;

      createDisk.mutate(
        {
          poolId: selectedPoolId,
          name: newDiskName.trim(),
          sizeBytes: gbToBytes(newDiskSizeGb),
          format: diskFormatFromString(newDiskFormat),
          bus: diskBusFromString(newDiskBus),
          sparse: true,
        },
        {
          onSuccess: (res) => {
            // Then attach the newly created disk
            if (res.disk?.path) {
              attachDisk.mutate(
                {
                  vmid,
                  diskPath: res.disk.path,
                  bus: diskBusFromString(newDiskBus),
                  readonly: attachReadonly,
                },
                {
                  onSuccess: () => {
                    setAttachDialogOpen(false);
                    refetchDisks();
                  },
                },
              );
            }
          },
        },
      );
    }
  };

  // Handle resize submit
  const handleResizeSubmit = () => {
    if (!selectedDisk || resizeSizeGb <= 0) return;

    const currentSizeGb = Number(selectedDisk.sizeBytes) / (1024 * 1024 * 1024);
    if (resizeSizeGb < currentSizeGb) return;

    resizeDisk.mutate(
      {
        vmid,
        target: selectedDisk.target,
        newSizeBytes: gbToBytes(resizeSizeGb),
      },
      {
        onSuccess: () => {
          handleResizeDialogOpen(false);
          refetchDisks();
        },
      },
    );
  };

  // Handle detach submit
  const handleDetachSubmit = () => {
    if (!selectedDisk) return;

    detachDisk.mutate(
      {
        vmid,
        target: selectedDisk.target,
        deleteDisk: deleteAfterDetach,
      },
      {
        onSuccess: () => {
          handleDetachDialogOpen(false);
          refetchDisks();
        },
      },
    );
  };

  const isRunning = vmStatus === "running";

  return (
    <>
      <Card>
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-sm font-medium">
                Virtual Disks
              </CardTitle>
              <CardDescription>
                Manage disks attached to this VM
              </CardDescription>
            </div>
            <Button size="sm" onClick={() => handleAttachDialogOpen(true)}>
              <Plus className="size-4 mr-2" />
              Attach Disk
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {disksLoading ? (
            <div className="flex items-center justify-center py-8">
              <RefreshCw className="size-6 animate-spin text-muted-foreground" />
            </div>
          ) : !disks || disks.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <HardDrive className="size-8 mx-auto mb-2 opacity-50" />
              <p>No disks attached to this VM</p>
            </div>
          ) : (
            <div className="rounded-md border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/50">
                    <th className="px-3 py-2 text-left font-medium">Target</th>
                    <th className="px-3 py-2 text-left font-medium">Path</th>
                    <th className="px-3 py-2 text-left font-medium">Size</th>
                    <th className="px-3 py-2 text-left font-medium">Bus</th>
                    <th className="px-3 py-2 text-left font-medium">Format</th>
                    <th className="px-3 py-2 text-left font-medium">Boot</th>
                    <th className="px-3 py-2 text-right font-medium">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {disks.map((disk, index) => {
                    const isRootDisk = index === 0;
                    return (
                      <tr
                        key={disk.target}
                        className="border-b last:border-b-0"
                      >
                        <td className="px-3 py-2">
                          <div className="flex items-center gap-2">
                            <span className="font-mono">{disk.target}</span>
                            {disk.readonly && (
                              <Badge variant="outline" className="text-xs">
                                RO
                              </Badge>
                            )}
                            {isRootDisk && (
                              <Badge variant="secondary" className="text-xs">
                                Root
                              </Badge>
                            )}
                          </div>
                        </td>
                        <td
                          className="px-3 py-2 font-mono text-xs text-muted-foreground max-w-[200px] truncate"
                          title={disk.path}
                        >
                          {disk.path}
                        </td>
                        <td className="px-3 py-2">
                          {formatBytes(disk.sizeBytes)}
                        </td>
                        <td className="px-3 py-2">
                          {diskBusToString(disk.bus)}
                        </td>
                        <td className="px-3 py-2">
                          {diskFormatToString(disk.format)}
                        </td>
                        <td className="px-3 py-2">
                          {disk.bootOrder > 0 ? `#${disk.bootOrder}` : "-"}
                        </td>
                        <td className="px-3 py-2 text-right">
                          <DiskRowActions
                            disk={disk}
                            isRootDisk={isRootDisk}
                            vmStatus={vmStatus}
                            onResize={() => {
                              setSelectedDisk(disk);
                              setResizeSizeGb(
                                Math.ceil(
                                  Number(disk.sizeBytes) / (1024 * 1024 * 1024),
                                ),
                              );
                              setResizeDialogOpen(true);
                            }}
                            onDetach={() => {
                              setSelectedDisk(disk);
                              setDeleteAfterDetach(false);
                              setDetachDialogOpen(true);
                            }}
                          />
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Attach Disk Dialog */}
      <Dialog open={attachDialogOpen} onOpenChange={handleAttachDialogOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>Attach Disk</DialogTitle>
            <DialogDescription>
              Attach an existing disk or create a new one for this VM.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            {/* Attach mode toggle */}
            <div className="flex gap-2">
              <Button
                type="button"
                variant={attachMode === "existing" ? "default" : "outline"}
                size="sm"
                onClick={() => setAttachMode("existing")}
                className="flex-1"
              >
                Existing Disk
              </Button>
              <Button
                type="button"
                variant={attachMode === "create" ? "default" : "outline"}
                size="sm"
                onClick={() => setAttachMode("create")}
                className="flex-1"
              >
                Create New
              </Button>
            </div>

            {/* Storage pool selection */}
            <div className="grid gap-2">
              <Label>Storage Pool</Label>
              <Select value={selectedPoolId} onValueChange={setSelectedPoolId}>
                <SelectTrigger>
                  <SelectValue placeholder="Select a pool" />
                </SelectTrigger>
                <SelectContent>
                  {pools.map((pool) => (
                    <SelectItem key={pool.id} value={pool.id}>
                      {pool.name} ({formatBytes(pool.availableBytes)})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {attachMode === "existing" ? (
              /* Existing disk selection */
              <div className="grid gap-2">
                <Label>Select Disk</Label>
                <Select
                  value={selectedDiskPath}
                  onValueChange={setSelectedDiskPath}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select an unassigned disk" />
                  </SelectTrigger>
                  <SelectContent>
                    {filteredUnassignedDisks.length > 0 ? (
                      filteredUnassignedDisks.map((disk) => (
                        <SelectItem key={disk.id} value={disk.path}>
                          {disk.name} ({formatBytes(disk.sizeBytes)})
                        </SelectItem>
                      ))
                    ) : (
                      <SelectItem value="none" disabled>
                        No unassigned disks available
                      </SelectItem>
                    )}
                  </SelectContent>
                </Select>
                {filteredUnassignedDisks.length === 0 && (
                  <p className="text-xs text-muted-foreground">
                    No unassigned disks found. Create a new disk or check your
                    storage pools.
                  </p>
                )}
              </div>
            ) : (
              /* New disk creation fields */
              <>
                <div className="grid gap-2">
                  <Label htmlFor="disk-name">Disk Name</Label>
                  <Input
                    id="disk-name"
                    value={newDiskName}
                    onChange={(e) => setNewDiskName(e.target.value)}
                    placeholder="my-vm-disk"
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="disk-size">Size (GB)</Label>
                  <Input
                    id="disk-size"
                    type="number"
                    min={1}
                    value={newDiskSizeGb}
                    onChange={(e) =>
                      setNewDiskSizeGb(parseInt(e.target.value) || 1)
                    }
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="grid gap-2">
                    <Label>Format</Label>
                    <Select
                      value={newDiskFormat}
                      onValueChange={(v) =>
                        setNewDiskFormat(v as "qcow2" | "raw")
                      }
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="qcow2">QCOW2</SelectItem>
                        <SelectItem value="raw">Raw</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              </>
            )}

            {/* Common fields for both modes */}
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label>Bus Type</Label>
                <Select
                  value={newDiskBus}
                  onValueChange={(v) =>
                    setNewDiskBus(v as "virtio" | "sata" | "scsi")
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="virtio">VirtIO</SelectItem>
                    <SelectItem value="sata">SATA</SelectItem>
                    <SelectItem value="scsi">SCSI</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label>&nbsp;</Label>
                <div className="flex items-center gap-2 h-9">
                  <Switch
                    id="readonly"
                    checked={attachReadonly}
                    onCheckedChange={setAttachReadonly}
                  />
                  <Label htmlFor="readonly" className="cursor-pointer">
                    Read-only
                  </Label>
                </div>
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setAttachDialogOpen(false)}
            >
              Cancel
            </Button>
            <Button
              onClick={handleAttachSubmit}
              disabled={
                isAttaching ||
                isCreatingDisk ||
                (attachMode === "existing" && !selectedDiskPath) ||
                (attachMode === "create" &&
                  (!selectedPoolId || !newDiskName.trim()))
              }
            >
              {(isAttaching || isCreatingDisk) && (
                <Loader2 className="size-4 mr-2 animate-spin" />
              )}
              {isAttaching || isCreatingDisk ? "Processing..." : "Attach Disk"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Resize Disk Dialog */}
      <Dialog open={resizeDialogOpen} onOpenChange={handleResizeDialogOpen}>
        <DialogContent className="sm:max-w-[400px]">
          <DialogHeader>
            <DialogTitle>Resize Disk</DialogTitle>
            <DialogDescription>
              Increase the size of this disk. You may need to resize the
              filesystem inside the VM afterward.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            {selectedDisk && (
              <div className="grid gap-2">
                <Label>Current Size</Label>
                <div className="text-sm font-medium">
                  {formatBytes(selectedDisk.sizeBytes)}
                </div>
              </div>
            )}
            <div className="grid gap-2">
              <Label htmlFor="resize-size">New Size (GB)</Label>
              <Input
                id="resize-size"
                type="number"
                min={
                  selectedDisk
                    ? Math.ceil(
                        Number(selectedDisk.sizeBytes) / (1024 * 1024 * 1024),
                      )
                    : 1
                }
                value={resizeSizeGb}
                onChange={(e) => setResizeSizeGb(parseInt(e.target.value) || 0)}
              />
            </div>
            <div className="flex items-start gap-2 text-sm text-amber-600 bg-amber-500/10 p-3 rounded-lg">
              <AlertTriangle className="size-4 shrink-0 mt-0.5" />
              <span>
                You may need to resize the filesystem inside the VM after
                resizing the disk.
              </span>
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => handleResizeDialogOpen(false)}
            >
              Cancel
            </Button>
            <Button
              onClick={handleResizeSubmit}
              disabled={
                isResizing ||
                !selectedDisk ||
                resizeSizeGb <=
                  Math.ceil(
                    Number(selectedDisk?.sizeBytes ?? 0) / (1024 * 1024 * 1024),
                  )
              }
            >
              {isResizing && <Loader2 className="size-4 mr-2 animate-spin" />}
              {isResizing ? "Resizing..." : "Resize Disk"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Detach Disk Dialog */}
      <Dialog open={detachDialogOpen} onOpenChange={handleDetachDialogOpen}>
        <DialogContent className="sm:max-w-[400px]">
          <DialogHeader>
            <DialogTitle>Detach Disk</DialogTitle>
            <DialogDescription>
              Detach this disk from the VM. The disk file can optionally be
              deleted.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            {selectedDisk && (
              <div className="grid gap-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Target:</span>
                  <span className="font-mono">{selectedDisk.target}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Path:</span>
                  <span
                    className="font-mono text-xs truncate max-w-[200px]"
                    title={selectedDisk.path}
                  >
                    {selectedDisk.path}
                  </span>
                </div>
              </div>
            )}
            {isRunning && (
              <div className="flex items-start gap-2 text-sm text-amber-600 bg-amber-500/10 p-3 rounded-lg">
                <AlertTriangle className="size-4 shrink-0 mt-0.5" />
                <span>
                  VM is running. Hot-detaching may cause data loss or
                  instability.
                </span>
              </div>
            )}
            <div className="flex items-center gap-2">
              <Switch
                id="delete-disk"
                checked={deleteAfterDetach}
                onCheckedChange={setDeleteAfterDetach}
              />
              <Label htmlFor="delete-disk" className="cursor-pointer">
                Delete disk file after detach
              </Label>
            </div>
            {deleteAfterDetach && (
              <p className="text-xs text-destructive">
                Warning: This will permanently delete the disk file and cannot
                be undone.
              </p>
            )}
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => handleDetachDialogOpen(false)}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDetachSubmit}
              disabled={isDetaching}
            >
              {isDetaching && <Loader2 className="size-4 mr-2 animate-spin" />}
              {isDetaching ? "Detaching..." : "Detach Disk"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
