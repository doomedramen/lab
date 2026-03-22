import { useQuery } from "@tanstack/react-query";
import { clusterClient } from "../client";
import type { ClusterSummary } from "../types";

export function useClusterSummary() {
  return useQuery({
    queryKey: ["cluster", "summary"],
    queryFn: () => clusterClient.getClusterSummary({}),
    select: (res): ClusterSummary => ({
      nodes: { total: res.nodes?.total ?? 0, running: res.nodes?.running ?? 0 },
      vms: { total: res.vms?.total ?? 0, running: res.vms?.running ?? 0 },
      containers: {
        total: res.containers?.total ?? 0,
        running: res.containers?.running ?? 0,
      },
      stacks: {
        total: res.stacks?.total ?? 0,
        running: res.stacks?.running ?? 0,
      },
      cpu: { cores: res.cpu?.cores ?? 0, avgUsage: res.cpu?.avgUsage ?? 0 },
      memory: { used: res.memory?.used ?? 0, total: res.memory?.total ?? 0 },
      disk: { used: res.disk?.used ?? 0, total: res.disk?.total ?? 0 },
    }),
  });
}

export function useClusterMetrics(points?: number) {
  return useQuery({
    queryKey: ["cluster", "metrics", points],
    queryFn: () => clusterClient.getClusterMetrics({ points: points ?? 0 }),
  });
}
