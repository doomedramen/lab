"use client";

import { StatusBadge, ResourceUsageBar } from "@/components/lab-shared";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Server } from "lucide-react";
import Link from "next/link";
import { useNodes } from "@/lib/api/queries";
import { Shimmer } from "@/components/shimmer";
import { ErrorDisplay } from "@/components/error-display";
import type { Node } from "@/lib/gen/lab/v1/node_pb";
import { NodeStatus } from "@/lib/gen/lab/v1/node_pb";
import { nodeStatusToString } from "@/lib/api/enum-helpers";

// Template data for shimmer
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
  {
    id: "node-2",
    name: "lab-prod-02",
    status: NodeStatus.ONLINE,
    ip: "10.0.1.11",
    cpu: { used: 67, total: 100, cores: 64 },
    memory: { used: 189, total: 512 },
    disk: { used: 6.2, total: 8 },
    uptime: "89d 14h 52m",
    kernel: "6.8.12-generic",
    version: "1.0.0",
    vms: 14,
    containers: 6,
    cpuModel: "Intel Xeon w9-3495X 56-Core",
    loadAvg: { one: 12.1, five: 11.4, fifteen: 10.8 },
    networkIn: 513,
    networkOut: 398,
  },
  {
    id: "node-3",
    name: "lab-dev-01",
    status: NodeStatus.ONLINE,
    ip: "10.0.1.20",
    cpu: { used: 12, total: 100, cores: 16 },
    memory: { used: 24, total: 128 },
    disk: { used: 0.8, total: 2 },
    uptime: "312d 2h 11m",
    kernel: "6.8.8-generic",
    version: "0.9.5",
    vms: 3,
    containers: 8,
    cpuModel: "AMD EPYC 7543 32-Core",
    loadAvg: { one: 1.2, five: 1.0, fifteen: 0.9 },
    networkIn: 45,
    networkOut: 33,
  },
] as unknown as Node[];

function HostsContent({ nodes }: { nodes: Node[] }) {
  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-foreground text-balance">
          Host Nodes
        </h1>
        <p className="text-sm text-muted-foreground mt-1">
          {nodes.length} nodes in cluster
        </p>
      </div>

      <div className="rounded-lg border border-border bg-card overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow className="bg-secondary/30 hover:bg-secondary/30">
              <TableHead>Node</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>CPU</TableHead>
              <TableHead>Memory</TableHead>
              <TableHead>Disk</TableHead>
              <TableHead>Uptime</TableHead>
              <TableHead>VMs / CTs</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {nodes.map((node) => (
              <TableRow key={node.id}>
                <TableCell>
                  <Link
                    href={`/hosts/view?id=${node.id}`}
                    className="flex items-center gap-2 hover:text-primary transition-colors"
                  >
                    <Server className="size-4 text-muted-foreground shrink-0" />
                    <div>
                      <div className="font-medium text-foreground">
                        {node.name}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {node.ip}
                      </div>
                    </div>
                  </Link>
                </TableCell>
                <TableCell>
                  <StatusBadge status={nodeStatusToString(node.status)} />
                </TableCell>
                <TableCell>
                  <div className="space-y-1">
                    <ResourceUsageBar value={node.cpu?.used ?? 0} />
                    <div className="text-[11px] text-muted-foreground">
                      {node.cpu?.cores ?? 0} cores
                    </div>
                  </div>
                </TableCell>
                <TableCell>
                  <div className="space-y-1">
                    <ResourceUsageBar
                      value={Math.round(
                        ((node.memory?.used ?? 0) / (node.memory?.total || 1)) *
                          100,
                      )}
                    />
                    <div className="text-[11px] text-muted-foreground">
                      {Math.round(node.memory?.used ?? 0)} /{" "}
                      {node.memory?.total ?? 0} GB
                    </div>
                  </div>
                </TableCell>
                <TableCell>
                  <div className="space-y-1">
                    <ResourceUsageBar
                      value={Math.round(
                        ((node.disk?.used ?? 0) / (node.disk?.total || 1)) *
                          100,
                      )}
                    />
                    <div className="text-[11px] text-muted-foreground">
                      {node.disk?.used ?? 0} / {node.disk?.total ?? 0} TB
                    </div>
                  </div>
                </TableCell>
                <TableCell>
                  <span className="text-sm text-foreground">{node.uptime}</span>
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary" className="text-[11px]">
                      {node.vms} VMs
                    </Badge>
                    <Badge variant="secondary" className="text-[11px]">
                      {node.containers} CTs
                    </Badge>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}

export default function HostsPage() {
  const { data: nodes, isLoading, error, refetch } = useNodes();

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

  return (
    <Shimmer loading={isLoading} templateProps={{ nodes: templateNodes }}>
      <HostsContent nodes={nodes || templateNodes} />
    </Shimmer>
  );
}
