import { useQuery } from "@tanstack/react-query";
import { clusterClient } from "../client";
import type { MetricQuery } from "../types";

// Metrics hooks

export function useMetrics(query?: MetricQuery) {
  return useQuery({
    queryKey: ["metrics", query],
    queryFn: async () => {
      const response = await clusterClient.queryMetrics({
        nodeId: query?.node_id ?? "",
        resourceType: query?.resource_type ?? "",
        resourceId: query?.resource_id,
        startTime: BigInt(query?.start_time ?? 0),
        endTime: BigInt(query?.end_time ?? 0),
        aggregate: query?.aggregate ?? "",
        groupBy: query?.group_by ?? "",
        hostOnly: query?.host_only ?? false,
      });
      return {
        metrics: response.metrics.map((m) => ({
          time: m.time,
          value: m.value,
        })),
        count: response.count,
      };
    },
  });
}
