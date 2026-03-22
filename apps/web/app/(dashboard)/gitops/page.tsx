"use client";

import { useState } from "react";
import { useGitOpsConfigs, useCreateGitOpsConfig, useDeleteGitOpsConfig, useSyncGitOpsConfig } from "@/lib/api/queries/gitops";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { RefreshCw, Plus, Trash2, GitBranch, AlertCircle } from "lucide-react";

export default function GitOpsPage() {
  const { data: configs, isLoading, error, refetch } = useGitOpsConfigs();
  const createMutation = useCreateGitOpsConfig();
  const deleteMutation = useDeleteGitOpsConfig();
  const syncMutation = useSyncGitOpsConfig();

  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [formData, setFormData] = useState({
    name: "",
    description: "",
    gitUrl: "",
    gitBranch: "main",
    gitPath: "/",
    syncInterval: 300,
    enabled: true,
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createMutation.mutateAsync(formData);
      setIsCreateOpen(false);
      refetch();
      setFormData({
        name: "",
        description: "",
        gitUrl: "",
        gitBranch: "main",
        gitPath: "/",
        syncInterval: 300,
        enabled: true,
      });
    } catch (err) {
      console.error("Failed to create GitOps config:", err);
    }
  };

  const handleSync = async (id: string) => {
    try {
      await syncMutation.mutateAsync({ id });
      refetch();
    } catch (err) {
      console.error("Failed to sync:", err);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Are you sure you want to delete this GitOps configuration?")) {
      return;
    }
    try {
      await deleteMutation.mutateAsync({ id });
      refetch();
    } catch (err) {
      console.error("Failed to delete:", err);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case "Healthy":
        return "bg-green-500";
      case "OutOfSync":
        return "bg-yellow-500";
      case "Failed":
        return "bg-red-500";
      default:
        return "bg-gray-500";
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <RefreshCw className="h-8 w-8 animate-spin" />
      </div>
    );
  }

  return (
    <div className="container mx-auto py-8">
      <div className="flex justify-between items-center mb-8">
        <div>
          <h1 className="text-3xl font-bold">GitOps</h1>
          <p className="text-muted-foreground">
            Manage Git-based infrastructure reconciliation
          </p>
        </div>
        <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Add GitOps Config
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-2xl">
            <DialogHeader>
              <DialogTitle>Add GitOps Configuration</DialogTitle>
              <DialogDescription>
                Connect a Git repository to automatically manage infrastructure
              </DialogDescription>
            </DialogHeader>
            <form onSubmit={handleSubmit}>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <Label htmlFor="name">Name</Label>
                  <Input
                    id="name"
                    value={formData.name}
                    onChange={(e) =>
                      setFormData({ ...formData, name: e.target.value })
                    }
                    placeholder="production-infra"
                    required
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="description">Description</Label>
                  <Input
                    id="description"
                    value={formData.description}
                    onChange={(e) =>
                      setFormData({ ...formData, description: e.target.value })
                    }
                    placeholder="Production infrastructure configuration"
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="gitUrl">Git Repository URL</Label>
                  <Input
                    id="gitUrl"
                    value={formData.gitUrl}
                    onChange={(e) =>
                      setFormData({ ...formData, gitUrl: e.target.value })
                    }
                    placeholder="https://github.com/org/infra.git"
                    required
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="grid gap-2">
                    <Label htmlFor="gitBranch">Branch</Label>
                    <Input
                      id="gitBranch"
                      value={formData.gitBranch}
                      onChange={(e) =>
                        setFormData({ ...formData, gitBranch: e.target.value })
                      }
                      placeholder="main"
                    />
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="gitPath">Path</Label>
                    <Input
                      id="gitPath"
                      value={formData.gitPath}
                      onChange={(e) =>
                        setFormData({ ...formData, gitPath: e.target.value })
                      }
                      placeholder="/"
                    />
                  </div>
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="syncInterval">Sync Interval (seconds)</Label>
                  <Input
                    id="syncInterval"
                    type="number"
                    value={formData.syncInterval}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        syncInterval: parseInt(e.target.value),
                      })
                    }
                    min={60}
                    step={60}
                  />
                </div>
              </div>
              <DialogFooter>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setIsCreateOpen(false)}
                >
                  Cancel
                </Button>
                <Button type="submit" disabled={createMutation.isPending}>
                  {createMutation.isPending ? "Creating..." : "Create"}
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      {error && (
        <Alert variant="destructive" className="mb-6">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            Failed to load GitOps configurations
          </AlertDescription>
        </Alert>
      )}

      <Card>
        <CardHeader>
          <CardTitle>GitOps Configurations</CardTitle>
          <CardDescription>
            Manage your Git-based infrastructure reconciliation
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Repository</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Last Sync</TableHead>
                <TableHead>Next Sync</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {configs && configs.length > 0 ? (
                configs.map((config) => (
                  <TableRow key={config.id}>
                    <TableCell className="font-medium">{config.name}</TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <GitBranch className="h-4 w-4" />
                        <span className="text-sm">{config.gitUrl}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant="secondary"
                        className={`${getStatusColor(config.status)} text-white`}
                      >
                        {config.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm">
                      {config.lastSync || "Never"}
                    </TableCell>
                    <TableCell className="text-sm">
                      {config.nextSync || "Pending"}
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleSync(config.id)}
                          disabled={syncMutation.isPending}
                        >
                          <RefreshCw className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleDelete(config.id)}
                          disabled={deleteMutation.isPending}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-8">
                    <p className="text-muted-foreground">
                      No GitOps configurations yet. Add one to get started.
                    </p>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}
