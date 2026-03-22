import { useQuery } from "@tanstack/react-query";
import { networkClient, firewallClient } from "../client";

export function useNetworks(type?: number, status?: number) {
  return useQuery({
    queryKey: ["networks", { type, status }],
    queryFn: () =>
      networkClient.listNetworks({
        type: type ?? 0,
        status: status ?? 0,
      }),
    select: (res) => ({
      networks: res.networks,
      total: res.total,
    }),
  });
}

export function useNetwork(id: string | undefined) {
  return useQuery({
    queryKey: ["network", id],
    queryFn: () => networkClient.getNetwork({ id: id! }),
    select: (res) => res.network,
    enabled: !!id,
  });
}

export function useNetworkInterfaces(
  networkId?: string,
  entityId?: number,
  entityType?: string,
) {
  return useQuery({
    queryKey: ["network-interfaces", { networkId, entityId, entityType }],
    queryFn: () =>
      networkClient.listVmNetworkInterfaces({
        networkId: networkId ?? "",
        entityId: entityId ?? 0,
        entityType: entityType ?? "",
      }),
    select: (res) => ({
      interfaces: res.interfaces,
      total: res.total,
    }),
  });
}

export function useBridges() {
  return useQuery({
    queryKey: ["bridges"],
    queryFn: () => networkClient.listBridges({}),
    select: (res) => ({
      bridges: res.bridges,
      total: res.total,
    }),
  });
}

export function useFirewallRules(
  scopeType?: string,
  scopeId?: string,
  enabledOnly?: boolean,
) {
  return useQuery({
    queryKey: ["firewall-rules", { scopeType, scopeId, enabledOnly }],
    queryFn: () =>
      firewallClient.listFirewallRules({
        scopeType: scopeType ?? "",
        scopeId: scopeId ?? "",
        enabledOnly: enabledOnly ?? false,
      }),
    select: (res) => ({
      rules: res.rules,
      total: res.total,
    }),
  });
}

export function useFirewallStatus(scopeType?: string, scopeId?: string) {
  return useQuery({
    queryKey: ["firewall-status", { scopeType, scopeId }],
    queryFn: () =>
      firewallClient.getFirewallStatus({
        scopeType: scopeType ?? "",
        scopeId: scopeId ?? "",
      }),
    select: (res) => res,
  });
}
