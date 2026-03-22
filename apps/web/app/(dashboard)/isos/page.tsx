"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { StatusBadge } from "@/components/lab-shared";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Disc,
  Upload,
  MoreHorizontal,
  Trash2,
  HardDrive,
  Server,
  Download,
  Loader2,
  RefreshCw,
  X,
  AlertCircle,
} from "lucide-react";
import { useISOs, useStoragePools } from "@/lib/api/queries";
import { useISOMutations } from "@/lib/api/mutations";
import {
  useISODownload,
  useISODownloadProgress,
  useAllISODownloadProgress,
} from "@/lib/api/mutations/isos";
import { ISOUploadModal } from "@/components/iso-upload-modal";
import { ISODownloadModal } from "@/components/iso-download-modal";
import { Shimmer } from "@/components/shimmer";
import { ErrorDisplay } from "@/components/error-display";
import { Progress } from "@/components/ui/progress";
import type { ISOImage } from "@/lib/gen/lab/v1/iso_pb";
import type { StoragePool } from "@/lib/gen/lab/v1/storage_pb";

const ACTIVE_DOWNLOADS_KEY = "lab:active-iso-downloads";

// Format bytes to human readable
function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

// Template data for shimmer
const templateISOs = [
  {
    id: "iso-1",
    name: "ubuntu-24.04-live-server-amd64.iso",
    size: 2833241088n,
    path: "/var/lib/libvirt/isos/ubuntu-24.04.iso",
    os: "Ubuntu",
    status: "available",
    createdAt: "2025-01-15T10:00:00Z",
  },
  {
    id: "iso-2",
    name: "rocky-9.4-x86_64-dvd.iso",
    size: 9529552896n,
    path: "/var/lib/libvirt/isos/rocky-9.4.iso",
    os: "Rocky Linux",
    status: "available",
    createdAt: "2025-02-01T14:30:00Z",
  },
] as unknown as ISOImage[];

const templatePools = [
  {
    id: "pool-1",
    name: "Local ISO Storage",
    type: "dir",
    capacity: 500000000000n,
    available: 350000000000n,
    used: 150000000000n,
    path: "/var/lib/libvirt/isos",
    status: "active",
  },
] as unknown as StoragePool[];

// Persist active download filenames in sessionStorage so they survive page refreshes.
function loadActiveDownloads(): string[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = sessionStorage.getItem(ACTIVE_DOWNLOADS_KEY);
    return raw ? (JSON.parse(raw) as string[]) : [];
  } catch {
    return [];
  }
}

function saveActiveDownloads(filenames: string[]) {
  if (typeof window === "undefined") return;
  try {
    sessionStorage.setItem(ACTIVE_DOWNLOADS_KEY, JSON.stringify(filenames));
  } catch {
    // ignore quota / privacy errors
  }
}

function ISOsContent({
  isos,
  pools,
}: {
  isos: ISOImage[];
  pools: StoragePool[];
}) {
  const { deleteISO } = useISOMutations();

  const [deleteDialog, setDeleteDialog] = useState<{
    open: boolean;
    iso: ISOImage | null;
    error: string | null;
  }>({
    open: false,
    iso: null,
    error: null,
  });
  const [uploadModalOpen, setUploadModalOpen] = useState(false);
  const [downloadModalOpen, setDownloadModalOpen] = useState(false);

  // Active downloads are persisted in sessionStorage so they survive page refreshes.
  const [activeDownloads, setActiveDownloads] =
    useState<string[]>(loadActiveDownloads);

  const addActiveDownload = useCallback((filename: string) => {
    setActiveDownloads((prev) => {
      const next = prev.includes(filename) ? prev : [...prev, filename];
      saveActiveDownloads(next);
      return next;
    });
  }, []);

  const removeActiveDownload = useCallback((filename: string) => {
    setActiveDownloads((prev) => {
      const next = prev.filter((f) => f !== filename);
      saveActiveDownloads(next);
      return next;
    });
  }, []);

  // Seed active downloads from the server on page load so that downloads started
  // in another tab or before a page refresh are auto-discovered.
  const { data: serverDownloads } = useAllISODownloadProgress();
  useEffect(() => {
    if (!serverDownloads) return;
    for (const d of serverDownloads) {
      if (d.status === "downloading") {
        addActiveDownload(d.filename);
      }
    }
  }, [serverDownloads, addActiveDownload]);

  const isDeleting = deleteISO.isPending;

  const handleDeleteConfirm = () => {
    if (!deleteDialog.iso) return;
    const id = deleteDialog.iso.id;
    deleteISO.mutate(id, {
      onSuccess: () => {
        setDeleteDialog({ open: false, iso: null, error: null });
      },
      onError: (err) => {
        setDeleteDialog((prev) => ({ ...prev, error: err.message }));
      },
    });
  };

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1
            className="text-2xl font-semibold text-foreground"
            data-testid="isos-page-title"
          >
            ISO Images
          </h1>
          <p className="text-sm text-muted-foreground mt-1">
            {isos.length} ISO{isos.length !== 1 ? "s" : ""} available
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            data-testid="download-iso-button"
            onClick={() => setDownloadModalOpen(true)}
          >
            <Download className="size-4 mr-2" />
            Download ISO
          </Button>
          <Button onClick={() => setUploadModalOpen(true)}>
            <Upload className="size-4 mr-2" />
            Upload ISO
          </Button>
        </div>
      </div>

      {/* Active Downloads */}
      {activeDownloads.length > 0 && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-sm font-medium">
              <Loader2 className="size-4 text-primary animate-spin" />
              Active Downloads
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {activeDownloads.map((filename) => (
              <ActiveDownloadItem
                key={filename}
                filename={filename}
                onComplete={() => removeActiveDownload(filename)}
                onDismiss={() => removeActiveDownload(filename)}
              />
            ))}
          </CardContent>
        </Card>
      )}

      {/* Storage Pool Info */}
      {pools.length > 0 && (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {pools.map((pool) => {
            const capacity = Number(pool.capacityBytes);
            const used = Number(pool.usedBytes);
            const usedPercent = capacity > 0 ? (used / capacity) * 100 : 0;
            return (
              <Card key={pool.id}>
                <CardHeader className="pb-2">
                  <CardTitle className="flex items-center gap-2 text-sm font-medium">
                    <HardDrive className="size-4 text-primary" />
                    {pool.name}
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-2">
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-muted-foreground">Used</span>
                    <span className="font-medium">
                      {formatBytes(used)} / {formatBytes(capacity)}
                    </span>
                  </div>
                  <div className="h-2 bg-secondary rounded-full overflow-hidden">
                    <div
                      className="h-full bg-primary transition-all"
                      style={{ width: `${usedPercent}%` }}
                    />
                  </div>
                  <div className="flex items-center justify-between text-xs text-muted-foreground">
                    <span className="flex items-center gap-1">
                      <Server className="size-3" />
                      {pool.path}
                    </span>
                    <StatusBadge status={String(pool.status)} />
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}

      {/* ISO Table */}
      <Card>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>OS</TableHead>
              <TableHead>Size</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Created</TableHead>
              <TableHead className="w-12"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isos.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={6}
                  className="text-center py-8 text-muted-foreground"
                >
                  No ISO images found. Upload one to get started.
                </TableCell>
              </TableRow>
            ) : (
              isos.map((iso) => (
                <TableRow key={iso.id}>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Disc className="size-4 text-muted-foreground" />
                      <span className="font-medium">{iso.name}</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <span className="text-muted-foreground">
                      {iso.os || "Unknown"}
                    </span>
                  </TableCell>
                  <TableCell>
                    <span className="text-muted-foreground">
                      {formatBytes(Number(iso.size))}
                    </span>
                  </TableCell>
                  <TableCell>
                    <StatusBadge status={iso.status} />
                  </TableCell>
                  <TableCell>
                    <span className="text-muted-foreground text-sm">
                      {new Date(iso.createdAt).toLocaleDateString()}
                    </span>
                  </TableCell>
                  <TableCell>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" className="size-8">
                          <MoreHorizontal className="size-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem
                          className="text-destructive"
                          onClick={() =>
                            setDeleteDialog({ open: true, iso, error: null })
                          }
                        >
                          <Trash2 className="size-4 mr-2" />
                          Delete
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </Card>

      {/* Upload Modal */}
      <ISOUploadModal
        open={uploadModalOpen}
        onOpenChange={setUploadModalOpen}
        onSuccess={() => {
          window.location.reload();
        }}
      />

      {/* Download Modal */}
      <ISODownloadModal
        open={downloadModalOpen}
        onOpenChange={setDownloadModalOpen}
        onSuccess={(filename) => {
          addActiveDownload(filename);
        }}
      />

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={deleteDialog.open}
        onOpenChange={(open) => {
          // Prevent closing while a deletion is in-flight.
          if (isDeleting) return;
          setDeleteDialog({ open, iso: deleteDialog.iso, error: null });
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete ISO</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete "{deleteDialog.iso?.name}"? This
              action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          {deleteDialog.error && (
            <Alert variant="destructive">
              <AlertCircle className="size-4" />
              <AlertDescription>{deleteDialog.error}</AlertDescription>
            </Alert>
          )}
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() =>
                setDeleteDialog({ open: false, iso: null, error: null })
              }
              disabled={isDeleting}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDeleteConfirm}
              disabled={isDeleting}
            >
              {isDeleting ? (
                <>
                  <Loader2 className="size-4 animate-spin mr-2" />
                  Deleting…
                </>
              ) : (
                "Delete"
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

// Active download item component
function ActiveDownloadItem({
  filename,
  onComplete,
  onDismiss,
}: {
  filename: string;
  onComplete: () => void;
  onDismiss: () => void;
}) {
  const { data: progress, isLoading } = useISODownloadProgress(filename);
  const { downloadISO } = useISODownload();

  // Track whether this was ever in-progress to distinguish "not started yet" from "completed".
  const wasDownloadingRef = useRef(false);
  useEffect(() => {
    if (progress?.status === "downloading") {
      wasDownloadingRef.current = true;
    }
  }, [progress?.status]);

  if (isLoading || !progress) {
    return (
      <div className="space-y-2">
        <div className="flex items-center justify-between text-sm">
          <span className="font-medium">{filename}</span>
          <Loader2 className="size-4 animate-spin text-muted-foreground" />
        </div>
        <Progress value={0} className="h-2" />
      </div>
    );
  }

  const isComplete = progress.status === "complete";
  const isError = progress.status === "error";

  if (isComplete) {
    // Defer state update to avoid calling during render.
    setTimeout(onComplete, 0);
    return null;
  }

  if (isError) {
    return (
      <div className="space-y-2">
        <div className="flex items-center justify-between text-sm">
          <div className="flex items-center gap-2 min-w-0">
            <AlertCircle className="size-4 text-destructive shrink-0" />
            <span className="font-medium truncate">{filename}</span>
            <span className="text-xs text-destructive shrink-0">
              {progress.error || "Download failed"}
            </span>
          </div>
          <div className="flex items-center gap-1 shrink-0 ml-2">
            <Button
              variant="ghost"
              size="sm"
              className="h-7 text-xs"
              onClick={() =>
                downloadISO.mutate({ url: progress.url, filename })
              }
              disabled={downloadISO.isPending}
            >
              <RefreshCw className="size-3 mr-1" />
              Retry
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="size-7"
              onClick={onDismiss}
              title="Dismiss"
            >
              <X className="size-3" />
            </Button>
          </div>
        </div>
        <Progress
          value={progress.percent}
          className="h-2 [&>div]:bg-destructive"
        />
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between text-sm">
        <span className="font-medium">{filename}</span>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span>
            {formatBytes(Number(progress.downloaded))} /{" "}
            {formatBytes(Number(progress.total))}
          </span>
          {progress.estimatedTime && <span>ETA: {progress.estimatedTime}</span>}
          <span>{Math.round(progress.percent)}%</span>
        </div>
      </div>
      <Progress value={progress.percent} className="h-2" />
    </div>
  );
}

export default function ISOsPage() {
  const {
    data: isos,
    isLoading: isosLoading,
    error: isosError,
    refetch: refetchISOs,
  } = useISOs();
  const { data: poolsData, isLoading: poolsLoading } = useStoragePools();
  const pools = poolsData?.pools;

  if (isosError) {
    return (
      <div className="p-6">
        <ErrorDisplay
          message={isosError.message}
          onRetry={() => refetchISOs()}
          className="h-[50vh]"
        />
      </div>
    );
  }

  const isLoading = isosLoading || poolsLoading;

  return (
    <Shimmer
      loading={isLoading}
      templateProps={{ isos: templateISOs, pools: templatePools }}
    >
      <ISOsContent isos={isos || templateISOs} pools={pools || templatePools} />
    </Shimmer>
  );
}
