import { useQuery } from "@tanstack/react-query";
import { proxyClient } from "../client";

export function useProxyHosts() {
  return useQuery({
    queryKey: ["proxy-hosts"],
    queryFn: () => proxyClient.listProxyHosts({}),
    select: (res) => ({
      proxyHosts: res.proxyHosts,
      total: res.total,
    }),
  });
}

export function useProxyHostsByTarget(ip: string | undefined) {
  return useQuery({
    queryKey: ["proxy-hosts-by-target", ip],
    queryFn: () => proxyClient.listProxyHostsByTarget({ targetIp: ip ?? "" }),
    select: (res) => ({
      proxyHosts: res.proxyHosts,
      total: res.total,
    }),
    enabled: !!ip,
  });
}

export function useProxyHost(id: string | undefined) {
  return useQuery({
    queryKey: ["proxy-host", id],
    queryFn: () => proxyClient.getProxyHost({ id: id! }),
    select: (res) => res.proxyHost,
    enabled: !!id,
  });
}

export function useProxyStatus(id: string | undefined) {
  return useQuery({
    queryKey: ["proxy-status", id],
    queryFn: () => proxyClient.getProxyStatus({ id: id! }),
    select: (res) => res.status,
    enabled: !!id,
    refetchInterval: 30_000, // poll every 30 s
  });
}

export function useMonitors() {
  return useQuery({
    queryKey: ["uptime-monitors"],
    queryFn: () => proxyClient.listMonitors({}),
    select: (res) => ({ monitors: res.monitors, total: res.total }),
  });
}

export function useMonitor(id: string | undefined) {
  return useQuery({
    queryKey: ["uptime-monitor", id],
    queryFn: () => proxyClient.getMonitor({ id: id! }),
    select: (res) => res.monitor,
    enabled: !!id,
  });
}

export function useMonitorStats(id: string | undefined) {
  return useQuery({
    queryKey: ["uptime-monitor-stats", id],
    queryFn: () => proxyClient.getMonitorStats({ id: id! }),
    select: (res) => res.stats,
    enabled: !!id,
    refetchInterval: 60_000, // poll every 60 s
  });
}

export function useMonitorHistory(id: string | undefined, limit = 100) {
  return useQuery({
    queryKey: ["uptime-monitor-history", id, limit],
    queryFn: () => proxyClient.getMonitorHistory({ id: id!, limit }),
    select: (res) => res.results,
    enabled: !!id,
    refetchInterval: 60_000,
  });
}
