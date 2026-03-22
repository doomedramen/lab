import { useQuery } from "@tanstack/react-query";
import { vmClient } from "../client";
import type { VMLogLevel } from "../../gen/lab/v1/vm_pb";
import { VMLogLevel as VMLogLevelEnum } from "../../gen/lab/v1/vm_pb";

export function useVMs(node?: string) {
  return useQuery({
    queryKey: ["vms", { node }],
    queryFn: () => vmClient.listVMs({ node: node ?? "" }),
    select: (res) => res.vms,
  });
}

export function useVM(vmid: string | undefined) {
  return useQuery({
    queryKey: ["vms", vmid],
    queryFn: () => vmClient.getVM({ vmid: parseInt(vmid!, 10) }),
    select: (res) => res.vm,
    enabled: !!vmid,
  });
}

export interface GetVMLogsOptions {
  vmid: number;
  limit?: number;
  minLevel?: VMLogLevel;
  source?: string;
  startTime?: string;
  endTime?: string;
}

export function useVMLogs(options: GetVMLogsOptions) {
  return useQuery({
    queryKey: ["vm-logs", options],
    queryFn: () =>
      vmClient.getVMLogs({
        vmid: options.vmid,
        limit: options.limit ?? 100,
        minLevel: options.minLevel,
        source: options.source ?? "",
        startTime: options.startTime ?? "",
        endTime: options.endTime ?? "",
      }),
    select: (res) => res,
    enabled: !!options.vmid,
    refetchInterval: 5000, // Auto-refresh every 5 seconds
  });
}

export function useVMDiagnostics(vmid: number | undefined) {
  return useQuery({
    queryKey: ["vm-diagnostics", vmid],
    queryFn: () => vmClient.getVMDiagnostics({ vmid: vmid! }),
    select: (res) => res.diagnostics,
    enabled: !!vmid,
  });
}

export function useGuestNetworkInterfaces(vmid: number | undefined) {
  return useQuery({
    queryKey: ["vm-guest-network", vmid],
    queryFn: () => vmClient.getVMGuestNetworkInterfaces({ vmid: vmid! }),
    select: (res) => ({
      interfaces: res.interfaces,
      agentConnected: res.agentConnected,
    }),
    enabled: !!vmid,
    refetchInterval: 30000, // Auto-refresh every 30 seconds
    retry: false, // Don't retry if agent is not available
  });
}

export function usePCIDevices() {
  return useQuery({
    queryKey: ["pci-devices"],
    queryFn: () => vmClient.listPCIDevices({}),
    select: (res) => ({
      devices: res.devices,
      iommuAvailable: res.iommuAvailable,
      vfioAvailable: res.vfioAvailable,
    }),
    staleTime: 60000, // Cache for 1 minute (devices don't change often)
  });
}
