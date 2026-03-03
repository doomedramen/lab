"use client"

import { useState } from "react"
import { Plus, Loader2, HardDrive, Database, Server, Folder } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { Switch } from "@/components/ui/switch"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { useStoragePoolMutations } from "@/lib/api/mutations/storage"
import type { StorageType } from "@/lib/gen/lab/v1/storage_pb"

interface CreateStoragePoolModalProps {
  trigger?: React.ReactNode
  onSuccess?: () => void
}

export function CreateStoragePoolModal({ trigger, onSuccess }: CreateStoragePoolModalProps) {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState("")
  const [type, setType] = useState<StorageType>(1) // DIR
  const [path, setPath] = useState("")
  const [description, setDescription] = useState("")
  const [enabled, setEnabled] = useState(true)

  const { createPool, isCreating } = useStoragePoolMutations({
    onCreateSuccess: () => {
      setOpen(false)
      setName("")
      setPath("")
      setDescription("")
      onSuccess?.()
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    if (!name.trim() || !path.trim()) {
      return
    }

    createPool.mutate({
      name: name.trim(),
      type,
      path: path.trim(),
      description: description.trim(),
      enabled,
    })
  }

  const getTypeLabel = (t: StorageType) => {
    const labels: Record<number, string> = {
      1: "Directory",
      2: "LVM",
      3: "ZFS",
      4: "NFS",
      5: "iSCSI",
      6: "Ceph",
      7: "GlusterFS",
    }
    return labels[t] || "Unknown"
  }

  const getTypeIcon = (t: StorageType) => {
    switch (t) {
      case 1: return <Folder className="w-4 h-4" />
      case 2: return <Database className="w-4 h-4" />
      case 3: return <Server className="w-4 h-4" />
      default: return <HardDrive className="w-4 h-4" />
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger || (
          <Button size="sm">
            <Plus className="w-4 h-4 mr-2" />
            Add Storage Pool
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="sm:max-w-[500px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Create Storage Pool</DialogTitle>
            <DialogDescription>
              Add a new storage pool for VM disks, ISOs, and backups.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="pool-name">Name</Label>
              <Input
                id="pool-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g., local-storage"
                required
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="pool-type">Type</Label>
              <Select value={String(type)} onValueChange={(v) => setType(Number(v) as StorageType)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="1">Directory</SelectItem>
                  <SelectItem value="2">LVM</SelectItem>
                  <SelectItem value="3">ZFS</SelectItem>
                  <SelectItem value="4">NFS</SelectItem>
                  <SelectItem value="5">iSCSI</SelectItem>
                  <SelectItem value="6">Ceph</SelectItem>
                  <SelectItem value="7">GlusterFS</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="pool-path">Path</Label>
              <Input
                id="pool-path"
                value={path}
                onChange={(e) => setPath(e.target.value)}
                placeholder={type === 1 ? "/var/lib/lab/storage" : type === 4 ? "nfs-server:/export" : "vg-name"}
                required
              />
              <p className="text-xs text-muted-foreground">
                {type === 1 && "Local directory path"}
                {type === 2 && "LVM volume group name"}
                {type === 3 && "ZFS pool/dataset name"}
                {type === 4 && "NFS export path (server:/path)"}
                {type === 5 && "iSCSI target"}
                {type === 6 && "Ceph pool name"}
                {type === 7 && "GlusterFS volume"}
              </p>
            </div>

            <Card>
              <CardContent className="pt-4">
                <div className="grid gap-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <Label htmlFor="pool-enabled" className="cursor-pointer">
                        Enabled
                      </Label>
                      <p className="text-xs text-muted-foreground">
                        Make pool available for use
                      </p>
                    </div>
                    <Switch
                      id="pool-enabled"
                      checked={enabled}
                      onCheckedChange={setEnabled}
                    />
                  </div>
                </div>
              </CardContent>
            </Card>

            <div className="grid gap-2">
              <Label htmlFor="pool-description">Description (optional)</Label>
              <Textarea
                id="pool-description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Describe this storage pool..."
                rows={2}
              />
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isCreating || !name.trim() || !path.trim()}>
              {isCreating && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
              {isCreating ? "Creating..." : "Create Pool"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
