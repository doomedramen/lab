import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { gitOpsClient } from "../client";
import type {
  GitOpsConfig,
  CreateGitOpsConfigInput,
} from "../types/gitops";

/**
 * Hook to fetch all GitOps configurations
 */
export function useGitOpsConfigs() {
  return useQuery({
    queryKey: ["gitops", "configs"],
    queryFn: async () => {
      const response = await gitOpsClient.listGitOpsConfigs({});
      return response.configs as GitOpsConfig[];
    },
  });
}

/**
 * Hook to fetch a single GitOps configuration
 */
export function useGitOpsConfig(id: string) {
  return useQuery({
    queryKey: ["gitops", "config", id],
    queryFn: async () => {
      const response = await gitOpsClient.getGitOpsConfig({ id });
      return response.config as GitOpsConfig;
    },
    enabled: !!id,
  });
}

/**
 * Hook to create a new GitOps configuration
 */
export function useCreateGitOpsConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: CreateGitOpsConfigInput) => {
      const response = await gitOpsClient.createGitOpsConfig(input);
      return response.config as GitOpsConfig;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["gitops", "configs"] });
    },
  });
}

/**
 * Hook to update a GitOps configuration
 */
export function useUpdateGitOpsConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { id: string } & Partial<CreateGitOpsConfigInput>) => {
      const response = await gitOpsClient.updateGitOpsConfig(input);
      return response.config as GitOpsConfig;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["gitops", "configs"] });
    },
  });
}

/**
 * Hook to delete a GitOps configuration
 */
export function useDeleteGitOpsConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id }: { id: string }) => {
      await gitOpsClient.deleteGitOpsConfig({ id });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["gitops", "configs"] });
    },
  });
}

/**
 * Hook to trigger a manual sync
 */
export function useSyncGitOpsConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id }: { id: string }) => {
      const response = await gitOpsClient.syncGitOpsConfig({ id });
      return response.syncLog;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["gitops", "configs"] });
    },
  });
}

/**
 * Hook to fetch GitOps resources
 */
export function useGitOpsResources(configId?: string, kind?: string, status?: string) {
  return useQuery({
    queryKey: ["gitops", "resources", configId, kind, status],
    queryFn: async () => {
      const response = await gitOpsClient.listGitOpsResources({
        configId,
        kind,
        status,
      });
      return response.resources;
    },
    enabled: !!configId,
  });
}

/**
 * Hook to fetch sync logs
 */
export function useGitOpsSyncLogs(configId: string, limit = 10) {
  return useQuery({
    queryKey: ["gitops", "sync-logs", configId, limit],
    queryFn: async () => {
      const response = await gitOpsClient.getGitOpsSyncLogs({ configId, limit });
      return response.logs;
    },
    enabled: !!configId,
  });
}
