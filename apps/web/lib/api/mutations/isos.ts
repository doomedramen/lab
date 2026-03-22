import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { isoClient } from "../client";
import type { GetISODownloadProgressResponse } from "../../gen/lab/v1/iso_pb";

export function useISOMutations() {
  const queryClient = useQueryClient();

  const deleteISO = useMutation({
    mutationFn: (id: string) => isoClient.deleteISO({ id }),
    onSuccess: () => {
      toast.success("ISO deleted successfully");
      queryClient.invalidateQueries({ queryKey: ["isos"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete ISO: ${error.message}`);
    },
  });

  return {
    deleteISO,
    isDeleting: deleteISO.isPending,
  };
}

export function useISODownload() {
  const queryClient = useQueryClient();

  const downloadISO = useMutation({
    mutationFn: (data: { url: string; filename?: string }) =>
      isoClient.downloadISO({ url: data.url, filename: data.filename || "" }),
    onSuccess: (res) => {
      toast.success(`ISO download started: ${res.filename}`);
      queryClient.invalidateQueries({ queryKey: ["iso-downloads"] });
      queryClient.invalidateQueries({ queryKey: ["isos"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to start ISO download: ${error.message}`);
    },
  });

  return {
    downloadISO,
    isDownloading: downloadISO.isPending,
  };
}

export function useISODownloadProgress(filename: string | undefined) {
  return useQuery<GetISODownloadProgressResponse>({
    queryKey: ["iso-download-progress", filename],
    queryFn: () =>
      isoClient.getISODownloadProgress({ filename: filename || "" }),
    select: (res) => res,
    enabled: !!filename,
    // Stop polling once the download has reached a terminal state.
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (status === "complete" || status === "error") return false;
      return 1000;
    },
  });
}

/**
 * Polls the server for all active/recent ISO download progress entries.
 * Used on page load to restore in-progress downloads that may have been started
 * in another tab or before a page refresh.
 */
export function useAllISODownloadProgress() {
  return useQuery({
    queryKey: ["iso-downloads"],
    queryFn: () => isoClient.listISODownloadProgress({}),
    select: (res) => res.downloads,
    // Stop polling once there are no more active downloads.
    refetchInterval: (query) => {
      const downloads = query.state.data;
      if (!downloads || !Array.isArray(downloads) || downloads.length === 0)
        return false;
      const hasActive = downloads.some((d) => d.status === "downloading");
      return hasActive ? 2000 : false;
    },
  });
}
