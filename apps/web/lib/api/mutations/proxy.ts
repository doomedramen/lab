import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { proxyClient } from "../client";

interface UseProxyMutationsOptions {
  onCreateSuccess?: () => void;
  onUpdateSuccess?: () => void;
  onDeleteSuccess?: () => void;
}

export function useProxyMutations({
  onCreateSuccess,
  onUpdateSuccess,
  onDeleteSuccess,
}: UseProxyMutationsOptions = {}) {
  const queryClient = useQueryClient();

  const createProxyHost = useMutation({
    mutationFn: (data: Parameters<typeof proxyClient.createProxyHost>[0]) =>
      proxyClient.createProxyHost(data),
    onSuccess: () => {
      toast.success("Proxy host created successfully");
      queryClient.invalidateQueries({ queryKey: ["proxy-hosts"] });
      onCreateSuccess?.();
    },
    onError: (error: Error) => {
      toast.error(`Failed to create proxy host: ${error.message}`);
    },
  });

  const updateProxyHost = useMutation({
    mutationFn: (data: Parameters<typeof proxyClient.updateProxyHost>[0]) =>
      proxyClient.updateProxyHost(data),
    onSuccess: () => {
      toast.success("Proxy host updated successfully");
      queryClient.invalidateQueries({ queryKey: ["proxy-hosts"] });
      onUpdateSuccess?.();
    },
    onError: (error: Error) => {
      toast.error(`Failed to update proxy host: ${error.message}`);
    },
  });

  const deleteProxyHost = useMutation({
    mutationFn: ({ id }: { id: string }) => proxyClient.deleteProxyHost({ id }),
    onSuccess: () => {
      toast.success("Proxy host deleted successfully");
      queryClient.invalidateQueries({ queryKey: ["proxy-hosts"] });
      onDeleteSuccess?.();
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete proxy host: ${error.message}`);
    },
  });

  const uploadCert = useMutation({
    mutationFn: (data: Parameters<typeof proxyClient.uploadCert>[0]) =>
      proxyClient.uploadCert(data),
    onSuccess: () => {
      toast.success("Certificate uploaded successfully");
      queryClient.invalidateQueries({ queryKey: ["proxy-hosts"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to upload certificate: ${error.message}`);
    },
  });

  return {
    createProxyHost,
    updateProxyHost,
    deleteProxyHost,
    uploadCert,
    isCreating: createProxyHost.isPending,
    isUpdating: updateProxyHost.isPending,
    isDeleting: deleteProxyHost.isPending,
    isUploading: uploadCert.isPending,
  };
}

interface UseMonitorMutationsOptions {
  onCreateSuccess?: () => void;
  onUpdateSuccess?: () => void;
  onDeleteSuccess?: () => void;
}

export function useMonitorMutations({
  onCreateSuccess,
  onUpdateSuccess,
  onDeleteSuccess,
}: UseMonitorMutationsOptions = {}) {
  const queryClient = useQueryClient();

  const createMonitor = useMutation({
    mutationFn: (data: Parameters<typeof proxyClient.createMonitor>[0]) =>
      proxyClient.createMonitor(data),
    onSuccess: () => {
      toast.success("Monitor created successfully");
      queryClient.invalidateQueries({ queryKey: ["uptime-monitors"] });
      onCreateSuccess?.();
    },
    onError: (error: Error) => {
      toast.error(`Failed to create monitor: ${error.message}`);
    },
  });

  const updateMonitor = useMutation({
    mutationFn: (data: Parameters<typeof proxyClient.updateMonitor>[0]) =>
      proxyClient.updateMonitor(data),
    onSuccess: () => {
      toast.success("Monitor updated successfully");
      queryClient.invalidateQueries({ queryKey: ["uptime-monitors"] });
      onUpdateSuccess?.();
    },
    onError: (error: Error) => {
      toast.error(`Failed to update monitor: ${error.message}`);
    },
  });

  const deleteMonitor = useMutation({
    mutationFn: ({ id }: { id: string }) => proxyClient.deleteMonitor({ id }),
    onSuccess: () => {
      toast.success("Monitor deleted successfully");
      queryClient.invalidateQueries({ queryKey: ["uptime-monitors"] });
      onDeleteSuccess?.();
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete monitor: ${error.message}`);
    },
  });

  return {
    createMonitor,
    updateMonitor,
    deleteMonitor,
    isCreating: createMonitor.isPending,
    isUpdating: updateMonitor.isPending,
    isDeleting: deleteMonitor.isPending,
  };
}
