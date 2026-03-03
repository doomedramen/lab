"use client"

import { useState } from "react"
import { Download, Loader2, X } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { useISODownload } from "@/lib/api/mutations/isos"

interface ISODownloadModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: (filename: string) => void
  prefillUrl?: string
}

export function ISODownloadModal({
  open,
  onOpenChange,
  onSuccess,
  prefillUrl,
}: ISODownloadModalProps) {
  const [url, setUrl] = useState(prefillUrl || "")
  const [filename, setFilename] = useState("")
  const { downloadISO, isDownloading } = useISODownload()

  const handleSubmit = () => {
    if (!url.trim()) return

    const finalFilename = filename.trim() || url.trim().split("/").pop() || "downloaded.iso"
    
    downloadISO.mutate(
      { url: url.trim(), filename: finalFilename },
      {
        onSuccess: () => {
          setUrl("")
          setFilename("")
          onSuccess?.(finalFilename)
          onOpenChange(false)
        },
      }
    )
  }

  const handleClose = () => {
    setUrl("")
    setFilename("")
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent data-testid="iso-download-modal" className="max-w-lg">
        <DialogHeader>
          <DialogTitle data-testid="iso-download-modal-title">Download ISO Image</DialogTitle>
          <DialogDescription data-testid="iso-download-modal-description">
            Download an ISO image from a URL. The ISO will be downloaded in the background and
            will be available for VM creation once complete.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="iso-url">ISO URL *</Label>
            <Input
              id="iso-url"
              data-testid="iso-url-input"
              placeholder="https://example.com/ubuntu.iso"
              value={url}
              onChange={(e) => {
                setUrl(e.target.value)
                // Auto-extract filename from URL
                if (!filename) {
                  const extracted = e.target.value.split("/").pop() || ""
                  setFilename(extracted)
                }
              }}
              disabled={isDownloading}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="iso-filename">Filename (optional)</Label>
            <Input
              id="iso-filename"
              data-testid="iso-filename-input"
              placeholder="ubuntu.iso"
              value={filename}
              onChange={(e) => setFilename(e.target.value)}
              disabled={isDownloading}
            />
            <p className="text-xs text-muted-foreground">
              Leave empty to use the filename from the URL
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose} disabled={isDownloading}>
            Cancel
          </Button>
          <Button
            data-testid="iso-download-submit"
            onClick={handleSubmit}
            disabled={!url.trim() || isDownloading}
          >
            {isDownloading ? (
              <>
                <Loader2 className="size-4 animate-spin mr-2" />
                Starting Download...
              </>
            ) : (
              <>
                <Download className="size-4 mr-2" />
                Download ISO
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
