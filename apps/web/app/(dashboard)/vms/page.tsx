"use client";

import {
  StatusBadge,
  ResourceUsageBar,
  TagList,
} from "@/components/lab-shared";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Monitor, Shield, Plus } from "lucide-react";
import Link from "next/link";
import { useVMs, useNodes } from "@/lib/api/queries";
import { Shimmer } from "@/components/shimmer";
import { ErrorDisplay } from "@/components/error-display";
import { CreateVMModal } from "@/components/create-vm-modal";
import type { VM } from "@/lib/gen/lab/v1/vm_pb";
import type { Node } from "@/lib/gen/lab/v1/node_pb";
import { VmStatus } from "@/lib/gen/lab/v1/vm_pb";
import {
  OsType,
  NetworkType,
  NetworkModel,
  MachineType,
  BiosType,
} from "@/lib/gen/lab/v1/common_pb";
import { vmStatusToString, osTypeToString } from "@/lib/api/enum-helpers";

// Template data for shimmer
const templateVMs = [
  {
    id: "vm-100",
    vmid: 100,
    name: "web-frontend-prod",
    node: "lab-prod-01",
    status: VmStatus.RUNNING,
    cpu: { used: 45, sockets: 1, cores: 4 },
    memory: { used: 6, total: 8 },
    disk: { used: 32, total: 64 },
    uptime: "45d 12h",
    os: { osType: OsType.LINUX, version: "ubuntu-24.04" },
    arch: "x86_64",
    machineType: MachineType.PC,
    bios: BiosType.SEABIOS,
    cpuModel: "host-passthrough",
    network: [
      {
        type: NetworkType.USER,
        model: NetworkModel.VIRTIO,
        bridge: "",
        vlan: 0,
      },
    ],
    ip: "10.0.10.100",
    tags: ["production", "web"],
    ha: true,
    description: "Primary web frontend server",
    nestedVirt: false,
    startOnBoot: true,
    agent: true,
  },
  {
    id: "vm-101",
    vmid: 101,
    name: "api-gateway",
    node: "lab-prod-01",
    status: VmStatus.RUNNING,
    cpu: { used: 62, sockets: 1, cores: 8 },
    memory: { used: 14, total: 16 },
    disk: { used: 48, total: 128 },
    uptime: "45d 12h",
    os: { osType: OsType.LINUX, version: "debian-12" },
    arch: "x86_64",
    machineType: MachineType.PC,
    bios: BiosType.SEABIOS,
    cpuModel: "host-passthrough",
    network: [
      {
        type: NetworkType.USER,
        model: NetworkModel.VIRTIO,
        bridge: "",
        vlan: 0,
      },
    ],
    ip: "10.0.10.101",
    tags: ["production", "api"],
    ha: true,
    description: "API gateway",
    nestedVirt: false,
    startOnBoot: true,
    agent: true,
  },
  {
    id: "vm-102",
    vmid: 102,
    name: "database-primary",
    node: "lab-prod-02",
    status: VmStatus.RUNNING,
    cpu: { used: 78, sockets: 2, cores: 8 },
    memory: { used: 57, total: 64 },
    disk: { used: 820, total: 1024 },
    uptime: "89d 14h",
    os: { osType: OsType.LINUX, version: "rocky-9" },
    arch: "x86_64",
    machineType: MachineType.PC,
    bios: BiosType.SEABIOS,
    cpuModel: "host-passthrough",
    network: [
      {
        type: NetworkType.USER,
        model: NetworkModel.VIRTIO,
        bridge: "",
        vlan: 0,
      },
    ],
    ip: "10.0.10.102",
    tags: ["production", "database"],
    ha: true,
    description: "PostgreSQL primary",
    nestedVirt: false,
    startOnBoot: true,
    agent: true,
  },
] as unknown as VM[];

function VMsContent({ vms, nodes }: { vms: VM[]; nodes: Node[] | undefined }) {
  const running = vms.filter((v) => v.status === VmStatus.RUNNING).length;
  const stopped = vms.filter((v) => v.status === VmStatus.STOPPED).length;

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1
            className="text-2xl font-semibold text-foreground text-balance"
            data-testid="vms-page-title"
          >
            Virtual Machines
          </h1>
          <p
            className="text-sm text-muted-foreground mt-1"
            data-testid="vms-page-summary"
          >
            {vms.length} total - {running} running, {stopped} stopped
          </p>
        </div>
        <CreateVMModal nodes={nodes} />
      </div>

      <div
        className="rounded-lg border border-border bg-card overflow-hidden"
        data-testid="vms-table"
      >
        <Table>
          <TableHeader>
            <TableRow className="bg-secondary/30 hover:bg-secondary/30">
              <TableHead>VMID</TableHead>
              <TableHead>Name</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Node</TableHead>
              <TableHead>CPU</TableHead>
              <TableHead>Memory</TableHead>
              <TableHead>Disk</TableHead>
              <TableHead>HA</TableHead>
              <TableHead>Tags</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {vms.map((vm) => {
              const node = nodes?.find((n) => n.name === vm.node);
              return (
                <TableRow key={vm.id} data-testid={`vm-table-row-${vm.vmid}`}>
                  <TableCell
                    className="font-mono text-sm"
                    data-testid={`vm-table-vmid-${vm.vmid}`}
                  >
                    {vm.vmid}
                  </TableCell>
                  <TableCell>
                    <Link
                      href={`/vms/view?id=${vm.vmid}`}
                      className="flex items-center gap-2 hover:text-primary transition-colors"
                      data-testid={`vm-table-link-${vm.vmid}`}
                    >
                      <Monitor className="size-4 text-muted-foreground shrink-0" />
                      <div>
                        <div
                          className="font-medium text-foreground"
                          data-testid={`vm-table-name-${vm.vmid}`}
                        >
                          {vm.name}
                        </div>
                        <div
                          className="text-xs text-muted-foreground"
                          data-testid={`vm-table-os-${vm.vmid}`}
                        >
                          {vm.os?.version ||
                            osTypeToString(vm.os?.osType ?? OsType.UNSPECIFIED)}
                        </div>
                      </div>
                    </Link>
                  </TableCell>
                  <TableCell>
                    <StatusBadge
                      status={vmStatusToString(vm.status)}
                      data-testid={`vm-table-status-${vm.vmid}`}
                    />
                  </TableCell>
                  <TableCell>
                    {node ? (
                      <Link
                        href={`/hosts/view?id=${node.id}`}
                        className="text-sm text-muted-foreground hover:text-foreground transition-colors"
                      >
                        {vm.node}
                      </Link>
                    ) : (
                      <span className="text-sm text-muted-foreground">
                        {vm.node}
                      </span>
                    )}
                  </TableCell>
                  <TableCell>
                    <div className="space-y-1">
                      <ResourceUsageBar value={vm.cpu?.used ?? 0} />
                      <div className="text-[11px] text-muted-foreground">
                        {vm.cpu?.cores ?? 0} cores
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="space-y-1">
                      <ResourceUsageBar
                        value={Math.round(
                          ((vm.memory?.used ?? 0) / (vm.memory?.total || 1)) *
                            100,
                        )}
                      />
                      <div className="text-[11px] text-muted-foreground">
                        {vm.memory?.used ?? 0}/{vm.memory?.total ?? 0} GB
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="text-sm text-foreground">
                      {vm.disk?.used ?? 0}/{vm.disk?.total ?? 0} GB
                    </div>
                  </TableCell>
                  <TableCell>
                    {vm.ha && <Shield className="size-4 text-primary" />}
                  </TableCell>
                  <TableCell>
                    <TagList tags={vm.tags} />
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}

export default function VMsPage() {
  const {
    data: vms,
    isLoading: vmsLoading,
    error: vmsError,
    refetch: refetchVMs,
  } = useVMs();
  const { data: nodes } = useNodes();

  if (vmsError) {
    return (
      <div className="p-6">
        <ErrorDisplay
          message={vmsError.message}
          onRetry={() => refetchVMs()}
          className="h-[50vh]"
        />
      </div>
    );
  }

  return (
    <Shimmer
      loading={vmsLoading}
      templateProps={{ vms: templateVMs, nodes: nodes || [] }}
    >
      <VMsContent vms={vms || templateVMs} nodes={nodes} />
    </Shimmer>
  );
}
