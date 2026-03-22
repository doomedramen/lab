import { useQuery } from "@tanstack/react-query";
import { authClient } from "../client";

export function useCurrentUser() {
  return useQuery({
    queryKey: ["current-user"],
    queryFn: () => authClient.getCurrentUser({}),
    select: (res) => res.user,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

export function useAPIKeys() {
  return useQuery({
    queryKey: ["api-keys"],
    queryFn: () => authClient.listAPIKeys({}),
    select: (res) => res.apiKeys,
  });
}

export function useSessions() {
  return useQuery({
    queryKey: ["sessions"],
    queryFn: () => authClient.listSessions({}),
    select: (res) => res.sessions,
  });
}

export function useAuditLogs({
  userId,
  action,
  limit = 50,
  offset = 0,
}: {
  userId?: string;
  action?: string;
  limit?: number;
  offset?: number;
} = {}) {
  return useQuery({
    queryKey: ["audit-logs", userId, action, limit, offset],
    queryFn: () =>
      authClient.listAuditLogs({
        userId: userId ?? "",
        action: action ?? "",
        limit,
        offset,
      }),
    select: (res) => ({
      logs: res.logs,
      total: Number(res.total),
    }),
  });
}
