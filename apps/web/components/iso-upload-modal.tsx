"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import Uppy from "@uppy/core";
import Tus from "@uppy/tus";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Progress } from "@/components/ui/progress";
import { Button } from "@/components/ui/button";
import { Upload, CheckCircle, AlertCircle } from "lucide-react";

interface ISOUploadModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}

interface UploadProgress {
  fileId: string;
  fileName: string;
  progress: number;
  status: "uploading" | "complete" | "error";
  error?: string;
}

export function ISOUploadModal({
  open,
  onOpenChange,
  onSuccess,
}: ISOUploadModalProps) {
  const [uploads, setUploads] = useState<UploadProgress[]>([]);
  const uppyRef = useRef<Uppy | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (open && !uppyRef.current) {
      // Get API URL for Tus endpoint
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

      uppyRef.current = new Uppy({
        restrictions: {
          allowedFileTypes: [".iso", "application/octet-stream"],
          maxFileSize: 50 * 1024 * 1024 * 1024, // 50 GB
        },
        autoProceed: true,
      }).use(Tus, {
        endpoint: `${apiUrl}/tus/files/`,
        chunkSize: 50 * 1024 * 1024, // 50 MB chunks
        retryDelays: [0, 1000, 3000, 5000],
      });

      uppyRef.current.on("file-added", (file) => {
        setUploads((prev) => [
          ...prev,
          {
            fileId: file.id,
            fileName: file.name,
            progress: 0,
            status: "uploading",
          },
        ]);
      });

      uppyRef.current.on("upload-progress", (file, progress) => {
        if (file && progress.bytesTotal) {
          const percent = Math.round(
            (progress.bytesUploaded / progress.bytesTotal) * 100,
          );
          setUploads((prev) =>
            prev.map((u) =>
              u.fileId === file.id ? { ...u, progress: percent } : u,
            ),
          );
        }
      });

      uppyRef.current.on("upload-success", (file) => {
        if (file) {
          setUploads((prev) =>
            prev.map((u) =>
              u.fileId === file.id
                ? { ...u, status: "complete", progress: 100 }
                : u,
            ),
          );
          onSuccess?.();
        }
      });

      uppyRef.current.on("upload-error", (file, error) => {
        if (file) {
          setUploads((prev) =>
            prev.map((u) =>
              u.fileId === file.id
                ? { ...u, status: "error", error: String(error) }
                : u,
            ),
          );
        }
      });
    }

    return () => {
      if (!open && uppyRef.current) {
        uppyRef.current.destroy();
        uppyRef.current = null;
        setUploads([]);
      }
    };
  }, [open, onSuccess]);

  const handleClose = () => {
    onOpenChange(false);
  };

  const handleFiles = useCallback((files: FileList | null) => {
    if (!files || !uppyRef.current) return;

    Array.from(files).forEach((file) => {
      // Only accept .iso files
      if (
        file.name.endsWith(".iso") ||
        file.type === "application/octet-stream"
      ) {
        uppyRef.current?.addFile({
          name: file.name,
          type: file.type,
          data: file,
        });
      }
    });
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      handleFiles(e.dataTransfer.files);
    },
    [handleFiles],
  );

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
  }, []);

  const uploadingFiles = uploads.filter((u) => u.status === "uploading");
  const isUploading = uploadingFiles.length > 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Upload className="size-5" />
            Upload ISO Image
          </DialogTitle>
          <DialogDescription>
            Drag and drop ISO files to upload. Files are uploaded using
            resumable Tus protocol.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Drag and drop area */}
          <div
            onDrop={handleDrop}
            onDragOver={handleDragOver}
            onClick={() => inputRef.current?.click()}
            className="rounded-lg border-2 border-dashed p-8 text-center cursor-pointer transition-colors hover:border-muted-foreground/50 border-muted-foreground/25"
          >
            <input
              ref={inputRef}
              type="file"
              accept=".iso"
              multiple
              onChange={(e) => handleFiles(e.target.files)}
              className="hidden"
            />
            <Upload className="size-10 mx-auto mb-3 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">
              Drag and drop ISO files here, or click to browse
            </p>
          </div>

          {/* Upload progress */}
          {uploads.length > 0 && (
            <div className="space-y-3">
              <h4 className="text-sm font-medium">Upload Progress</h4>
              <div className="space-y-2">
                {uploads.map((upload) => (
                  <div key={upload.fileId} className="space-y-1">
                    <div className="flex items-center justify-between text-sm">
                      <span className="truncate font-medium">
                        {upload.fileName}
                      </span>
                      <div className="flex items-center gap-2">
                        {upload.status === "complete" && (
                          <CheckCircle className="size-4 text-success" />
                        )}
                        {upload.status === "error" && (
                          <AlertCircle className="size-4 text-destructive" />
                        )}
                        {upload.status === "uploading" && (
                          <span className="text-muted-foreground">
                            {upload.progress}%
                          </span>
                        )}
                      </div>
                    </div>
                    {upload.status === "uploading" && (
                      <Progress value={upload.progress} className="h-1.5" />
                    )}
                    {upload.status === "error" && (
                      <p className="text-xs text-destructive">{upload.error}</p>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Actions */}
          <div className="flex justify-end gap-2">
            <Button
              variant="outline"
              onClick={handleClose}
              disabled={isUploading}
            >
              {isUploading ? "Uploading..." : "Close"}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
