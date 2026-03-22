"use client";

import { useState } from "react";
import {
  Globe,
  Plus,
  Trash2,
  Edit,
  CheckCircle2,
  XCircle,
  Loader2,
  Lock,
  Activity,
  ArrowUpCircle,
  ArrowDownCircle,
  PauseCircle,
  Clock,
} from "lucide-react";
import { PageHeader } from "@/components/page-header";
import {
  useProxyHosts,
  useProxyStatus,
  useMonitors,
  useMonitorStats,
  useMonitorHistory,
} from "@/lib/api/queries/proxy";
import {
  useProxyMutations,
  useMonitorMutations,
} from "@/lib/api/mutations/proxy";
import { ProxySSLMode } from "@/lib/gen/lab/v1/proxy_pb";
import type { UptimeMonitor } from "@/lib/gen/lab/v1/proxy_pb";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { MetricAreaChart } from "@/components/metric-area-chart";

// ---- helpers ----

function sslModeLabel(mode: ProxySSLMode): string {
  switch (mode) {
    case ProxySSLMode.PROXY_SSL_MODE_NONE:
      return "HTTP Only";
    case ProxySSLMode.PROXY_SSL_MODE_SELF_SIGNED:
      return "Self-Signed";
    case ProxySSLMode.PROXY_SSL_MODE_ACME:
      return "ACME / Let's Encrypt";
    case ProxySSLMode.PROXY_SSL_MODE_CUSTOM:
      return "Custom Cert";
    default:
      return "Unknown";
  }
}

function sslModeBadgeVariant(
  mode: ProxySSLMode,
): "default" | "secondary" | "outline" | "destructive" {
  switch (mode) {
    case ProxySSLMode.PROXY_SSL_MODE_NONE:
      return "secondary";
    case ProxySSLMode.PROXY_SSL_MODE_SELF_SIGNED:
      return "outline";
    case ProxySSLMode.PROXY_SSL_MODE_ACME:
      return "default";
    case ProxySSLMode.PROXY_SSL_MODE_CUSTOM:
      return "default";
    default:
      return "secondary";
  }
}

function uptimeStatusBadge(status: string | undefined) {
  switch (status) {
    case "up":
      return (
        <Badge
          variant="default"
          className="gap-1 bg-green-600 hover:bg-green-600"
        >
          <ArrowUpCircle className="size-3" />
          Up
        </Badge>
      );
    case "down":
      return (
        <Badge variant="destructive" className="gap-1">
          <ArrowDownCircle className="size-3" />
          Down
        </Badge>
      );
    case "paused":
      return (
        <Badge variant="secondary" className="gap-1">
          <PauseCircle className="size-3" />
          Paused
        </Badge>
      );
    default:
      return (
        <Badge variant="secondary" className="gap-1">
          <Clock className="size-3" />
          Pending
        </Badge>
      );
  }
}

// ---- sub-components ----

interface StatusBadgeProps {
  id: string;
}
function StatusBadge({ id }: StatusBadgeProps) {
  const { data: status, isLoading } = useProxyStatus(id);
  if (isLoading)
    return <Loader2 className="size-4 animate-spin text-muted-foreground" />;
  if (!status) return <Badge variant="secondary">Unknown</Badge>;
  if (status.isRunning && status.backendReachable) {
    return (
      <Badge variant="default" className="gap-1">
        <CheckCircle2 className="size-3" />
        Online
      </Badge>
    );
  }
  if (!status.backendReachable) {
    return (
      <Badge variant="destructive" className="gap-1">
        <XCircle className="size-3" />
        Unreachable
      </Badge>
    );
  }
  return (
    <Badge variant="secondary" className="gap-1">
      <XCircle className="size-3" />
      Offline
    </Badge>
  );
}

interface MonitorStatsRowProps {
  monitor: UptimeMonitor;
}
function MonitorStatsRow({ monitor }: MonitorStatsRowProps) {
  const { data: stats, isLoading } = useMonitorStats(monitor.id);
  if (isLoading)
    return (
      <>
        <TableCell>
          <Loader2 className="size-4 animate-spin text-muted-foreground" />
        </TableCell>
        <TableCell>—</TableCell>
        <TableCell>—</TableCell>
      </>
    );
  return (
    <>
      <TableCell>{uptimeStatusBadge(stats?.status)}</TableCell>
      <TableCell className="text-sm text-muted-foreground">
        {stats ? `${stats.uptimePercent24h.toFixed(1)}%` : "—"}
      </TableCell>
      <TableCell className="text-sm text-muted-foreground font-mono">
        {stats?.avgResponseMs24h
          ? `${Math.round(stats.avgResponseMs24h)}ms`
          : "—"}
      </TableCell>
    </>
  );
}

interface MonitorHistoryChartProps {
  monitorId: string;
}
function MonitorHistoryChart({ monitorId }: MonitorHistoryChartProps) {
  const { data: results } = useMonitorHistory(monitorId, 50);
  if (!results || results.length === 0) {
    return (
      <p className="text-xs text-muted-foreground py-2">No history yet.</p>
    );
  }
  // Oldest first for chart display
  const chartData = [...results].reverse().map((r) => ({
    time: new Date(r.checkedAt).toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    }),
    value: r.success ? Number(r.responseTimeMs) : 0,
  }));
  return (
    <MetricAreaChart
      data={chartData}
      color="oklch(0.6 0.15 220)"
      height={80}
      tooltipLabel="Response time"
      tooltipUnit="ms"
      showYAxis={false}
      showXAxis={false}
      showGrid={false}
    />
  );
}

// ---- default form states ----

const defaultProxyForm = {
  domain: "",
  targetUrl: "",
  sslMode: ProxySSLMode.PROXY_SSL_MODE_NONE as ProxySSLMode,
  basicAuthEnabled: false,
  basicAuthUser: "",
  basicAuthPassword: "",
  websocketSupport: true,
  enabled: true,
};

const defaultMonitorForm = {
  name: "",
  url: "",
  intervalSeconds: 60,
  timeoutSeconds: 10,
  expectedStatusCode: 200,
  enabled: true,
};

// ---- main page ----

export default function ProxyPage() {
  return (
    <div className="p-6 space-y-6">
      <PageHeader
        backHref="/"
        backLabel="Dashboard"
        title="Reverse Proxy"
        subtitle="Manage domain-based proxy hosts with optional HTTPS and uptime monitoring"
        icon={<Globe className="size-5 text-foreground" />}
      />

      <Tabs defaultValue="hosts">
        <TabsList>
          <TabsTrigger value="hosts">Proxy Hosts</TabsTrigger>
          <TabsTrigger value="monitors">
            <Activity className="size-4 mr-1.5" />
            Uptime Monitors
          </TabsTrigger>
        </TabsList>

        <TabsContent value="hosts" className="space-y-6 mt-4">
          <ProxyHostsTab />
        </TabsContent>

        <TabsContent value="monitors" className="space-y-6 mt-4">
          <UptimeMonitorsTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}

// ---- Proxy Hosts Tab ----

function ProxyHostsTab() {
  const { data, isLoading } = useProxyHosts();
  const {
    createProxyHost,
    updateProxyHost,
    deleteProxyHost,
    isCreating,
    isUpdating,
    isDeleting,
  } = useProxyMutations();

  const [createOpen, setCreateOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [form, setForm] = useState({ ...defaultProxyForm });

  function openCreate() {
    setForm({ ...defaultProxyForm });
    setCreateOpen(true);
  }

  function openEdit(hostId: string) {
    const host = data?.proxyHosts.find((h) => h.id === hostId);
    if (!host) return;
    setForm({
      domain: host.domain,
      targetUrl: host.targetUrl,
      sslMode: host.sslMode,
      basicAuthEnabled: host.basicAuthEnabled,
      basicAuthUser: host.basicAuthUser,
      basicAuthPassword: "",
      websocketSupport: host.websocketSupport,
      enabled: host.enabled,
    });
    setEditTarget(hostId);
  }

  function handleCreate() {
    createProxyHost.mutate(
      {
        domain: form.domain,
        targetUrl: form.targetUrl,
        sslMode: form.sslMode,
        basicAuthEnabled: form.basicAuthEnabled,
        basicAuthUser: form.basicAuthUser,
        basicAuthPassword: form.basicAuthPassword,
        websocketSupport: form.websocketSupport,
        enabled: form.enabled,
        customRequestHeaders: {},
        customResponseHeaders: {},
      },
      { onSuccess: () => setCreateOpen(false) },
    );
  }

  function handleUpdate() {
    if (!editTarget) return;
    updateProxyHost.mutate(
      {
        id: editTarget,
        domain: form.domain,
        targetUrl: form.targetUrl,
        sslMode: form.sslMode,
        basicAuthEnabled: form.basicAuthEnabled,
        basicAuthUser: form.basicAuthUser,
        basicAuthPassword: form.basicAuthPassword,
        websocketSupport: form.websocketSupport,
        enabled: form.enabled,
        customRequestHeaders: {},
        customResponseHeaders: {},
      },
      { onSuccess: () => setEditTarget(null) },
    );
  }

  function handleDelete() {
    if (!deleteTarget) return;
    deleteProxyHost.mutate(
      { id: deleteTarget },
      { onSuccess: () => setDeleteTarget(null) },
    );
  }

  const hosts = data?.proxyHosts ?? [];

  return (
    <>
      {/* Summary cards */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Total Hosts</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data?.total ?? 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Enabled</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {hosts.filter((h) => h.enabled).length}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">TLS Hosts</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {
                hosts.filter(
                  (h) =>
                    h.sslMode !== ProxySSLMode.PROXY_SSL_MODE_NONE &&
                    h.sslMode !== ProxySSLMode.PROXY_SSL_MODE_UNSPECIFIED,
                ).length
              }
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Host table */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>Proxy Hosts</CardTitle>
          <Button size="sm" onClick={openCreate}>
            <Plus className="size-4 mr-1" />
            Add Host
          </Button>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-8 text-muted-foreground">
              <Loader2 className="size-5 animate-spin mr-2" />
              Loading proxy hosts…
            </div>
          ) : hosts.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground gap-3">
              <Globe className="size-10 opacity-30" />
              <p className="text-sm">No proxy hosts configured.</p>
              <Button size="sm" variant="outline" onClick={openCreate}>
                <Plus className="size-4 mr-1" />
                Add your first host
              </Button>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Domain</TableHead>
                  <TableHead>Target</TableHead>
                  <TableHead>SSL</TableHead>
                  <TableHead>Auth</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="w-20" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {hosts.map((host) => (
                  <TableRow key={host.id}>
                    <TableCell className="font-medium">{host.domain}</TableCell>
                    <TableCell className="text-muted-foreground text-sm font-mono">
                      {host.targetUrl}
                    </TableCell>
                    <TableCell>
                      <Badge variant={sslModeBadgeVariant(host.sslMode)}>
                        {sslModeLabel(host.sslMode)}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {host.basicAuthEnabled ? (
                        <Lock className="size-4 text-muted-foreground" />
                      ) : (
                        <span className="text-muted-foreground text-xs">—</span>
                      )}
                    </TableCell>
                    <TableCell>
                      {host.enabled ? (
                        <StatusBadge id={host.id} />
                      ) : (
                        <Badge variant="secondary">Disabled</Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1">
                        <Button
                          size="icon"
                          variant="ghost"
                          className="size-8"
                          onClick={() => openEdit(host.id)}
                        >
                          <Edit className="size-4" />
                        </Button>
                        <Button
                          size="icon"
                          variant="ghost"
                          className="size-8 text-destructive hover:text-destructive"
                          onClick={() => setDeleteTarget(host.id)}
                        >
                          <Trash2 className="size-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Create dialog */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Add Proxy Host</DialogTitle>
            <DialogDescription>
              Configure a new domain-based reverse proxy rule.
            </DialogDescription>
          </DialogHeader>
          <ProxyHostForm form={form} onChange={setForm} />
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleCreate} disabled={isCreating}>
              {isCreating && <Loader2 className="size-4 animate-spin mr-2" />}
              Create
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit dialog */}
      <Dialog
        open={!!editTarget}
        onOpenChange={(o) => !o && setEditTarget(null)}
      >
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Edit Proxy Host</DialogTitle>
            <DialogDescription>
              Update the proxy host configuration.
            </DialogDescription>
          </DialogHeader>
          <ProxyHostForm form={form} onChange={setForm} />
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditTarget(null)}>
              Cancel
            </Button>
            <Button onClick={handleUpdate} disabled={isUpdating}>
              {isUpdating && <Loader2 className="size-4 animate-spin mr-2" />}
              Save Changes
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete confirmation dialog */}
      <Dialog
        open={!!deleteTarget}
        onOpenChange={(o) => !o && setDeleteTarget(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Proxy Host</DialogTitle>
            <DialogDescription>
              This will remove the proxy host and its associated certificate.
              This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={isDeleting}
            >
              {isDeleting && <Loader2 className="size-4 animate-spin mr-2" />}
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

// ---- Uptime Monitors Tab ----

function UptimeMonitorsTab() {
  const { data, isLoading } = useMonitors();
  const {
    createMonitor,
    updateMonitor,
    deleteMonitor,
    isCreating,
    isUpdating,
    isDeleting,
  } = useMonitorMutations();

  const [createOpen, setCreateOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [expandedHistory, setExpandedHistory] = useState<string | null>(null);
  const [form, setForm] = useState({ ...defaultMonitorForm });

  function openCreate() {
    setForm({ ...defaultMonitorForm });
    setCreateOpen(true);
  }

  function openEdit(m: UptimeMonitor) {
    setForm({
      name: m.name,
      url: m.url,
      intervalSeconds: m.intervalSeconds,
      timeoutSeconds: m.timeoutSeconds,
      expectedStatusCode: m.expectedStatusCode,
      enabled: m.enabled,
    });
    setEditTarget(m.id);
  }

  function handleCreate() {
    createMonitor.mutate(
      {
        name: form.name,
        url: form.url,
        intervalSeconds: form.intervalSeconds,
        timeoutSeconds: form.timeoutSeconds,
        expectedStatusCode: form.expectedStatusCode,
        enabled: form.enabled,
      },
      { onSuccess: () => setCreateOpen(false) },
    );
  }

  function handleUpdate() {
    if (!editTarget) return;
    updateMonitor.mutate(
      {
        id: editTarget,
        name: form.name,
        url: form.url,
        intervalSeconds: form.intervalSeconds,
        timeoutSeconds: form.timeoutSeconds,
        expectedStatusCode: form.expectedStatusCode,
        enabled: form.enabled,
      },
      { onSuccess: () => setEditTarget(null) },
    );
  }

  function handleDelete() {
    if (!deleteTarget) return;
    deleteMonitor.mutate(
      { id: deleteTarget },
      { onSuccess: () => setDeleteTarget(null) },
    );
  }

  const monitors = data?.monitors ?? [];

  return (
    <>
      {/* Summary cards */}
      <UptimeMonitorSummary monitors={monitors} />

      {/* Monitor table */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>Monitors</CardTitle>
          <Button size="sm" onClick={openCreate}>
            <Plus className="size-4 mr-1" />
            Add Monitor
          </Button>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-8 text-muted-foreground">
              <Loader2 className="size-5 animate-spin mr-2" />
              Loading monitors…
            </div>
          ) : monitors.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground gap-3">
              <Activity className="size-10 opacity-30" />
              <p className="text-sm">No uptime monitors configured.</p>
              <Button size="sm" variant="outline" onClick={openCreate}>
                <Plus className="size-4 mr-1" />
                Add your first monitor
              </Button>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>URL</TableHead>
                  <TableHead>Interval</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Uptime 24h</TableHead>
                  <TableHead>Avg Response</TableHead>
                  <TableHead className="w-28" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {monitors.map((m) => (
                  <>
                    <TableRow key={m.id}>
                      <TableCell className="font-medium">{m.name}</TableCell>
                      <TableCell className="text-muted-foreground text-sm font-mono max-w-xs truncate">
                        {m.url}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {formatInterval(m.intervalSeconds)}
                      </TableCell>
                      <MonitorStatsRow monitor={m} />
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <Button
                            size="icon"
                            variant="ghost"
                            className="size-8"
                            title="Response time history"
                            onClick={() =>
                              setExpandedHistory(
                                expandedHistory === m.id ? null : m.id,
                              )
                            }
                          >
                            <Activity className="size-4" />
                          </Button>
                          <Button
                            size="icon"
                            variant="ghost"
                            className="size-8"
                            onClick={() => openEdit(m)}
                          >
                            <Edit className="size-4" />
                          </Button>
                          <Button
                            size="icon"
                            variant="ghost"
                            className="size-8 text-destructive hover:text-destructive"
                            onClick={() => setDeleteTarget(m.id)}
                          >
                            <Trash2 className="size-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                    {expandedHistory === m.id && (
                      <TableRow key={`${m.id}-history`}>
                        <TableCell
                          colSpan={7}
                          className="bg-muted/30 px-4 py-3"
                        >
                          <p className="text-xs text-muted-foreground mb-2 font-medium">
                            Response time — last 50 checks
                          </p>
                          <MonitorHistoryChart monitorId={m.id} />
                        </TableCell>
                      </TableRow>
                    )}
                  </>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Create dialog */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Add Uptime Monitor</DialogTitle>
            <DialogDescription>
              Configure a URL to check for availability.
            </DialogDescription>
          </DialogHeader>
          <MonitorForm form={form} onChange={setForm} />
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleCreate} disabled={isCreating}>
              {isCreating && <Loader2 className="size-4 animate-spin mr-2" />}
              Create
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit dialog */}
      <Dialog
        open={!!editTarget}
        onOpenChange={(o) => !o && setEditTarget(null)}
      >
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Edit Monitor</DialogTitle>
            <DialogDescription>
              Update the monitor configuration.
            </DialogDescription>
          </DialogHeader>
          <MonitorForm form={form} onChange={setForm} />
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditTarget(null)}>
              Cancel
            </Button>
            <Button onClick={handleUpdate} disabled={isUpdating}>
              {isUpdating && <Loader2 className="size-4 animate-spin mr-2" />}
              Save Changes
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete confirmation */}
      <Dialog
        open={!!deleteTarget}
        onOpenChange={(o) => !o && setDeleteTarget(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Monitor</DialogTitle>
            <DialogDescription>
              This will remove the monitor and all its history. This action
              cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={isDeleting}
            >
              {isDeleting && <Loader2 className="size-4 animate-spin mr-2" />}
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

// ---- summary cards that load stats per monitor ----

function UptimeMonitorSummary({ monitors }: { monitors: UptimeMonitor[] }) {
  return (
    <div className="grid gap-4 md:grid-cols-3">
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">Total Monitors</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{monitors.length}</div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">Enabled</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">
            {monitors.filter((m) => m.enabled).length}
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">Checking Every</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">
            {monitors.length === 0
              ? "—"
              : formatInterval(
                  Math.min(
                    ...monitors
                      .filter((m) => m.enabled)
                      .map((m) => m.intervalSeconds),
                  ),
                )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function formatInterval(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${seconds / 60}m`;
  return `${seconds / 3600}h`;
}

// ---- ProxyHostForm sub-component ----

interface ProxyFormState {
  domain: string;
  targetUrl: string;
  sslMode: ProxySSLMode;
  basicAuthEnabled: boolean;
  basicAuthUser: string;
  basicAuthPassword: string;
  websocketSupport: boolean;
  enabled: boolean;
}

interface ProxyHostFormProps {
  form: ProxyFormState;
  onChange: (f: ProxyFormState) => void;
}

function ProxyHostForm({ form, onChange }: ProxyHostFormProps) {
  function set<K extends keyof ProxyFormState>(
    key: K,
    value: ProxyFormState[K],
  ) {
    onChange({ ...form, [key]: value });
  }

  return (
    <div className="space-y-4">
      <div className="space-y-1.5">
        <Label htmlFor="domain">Domain</Label>
        <Input
          id="domain"
          placeholder="app.example.com"
          value={form.domain}
          onChange={(e) => set("domain", e.target.value)}
        />
      </div>

      <div className="space-y-1.5">
        <Label htmlFor="targetUrl">Target URL</Label>
        <Input
          id="targetUrl"
          placeholder="http://192.168.1.100:3000"
          value={form.targetUrl}
          onChange={(e) => set("targetUrl", e.target.value)}
        />
      </div>

      <div className="space-y-1.5">
        <Label>SSL Mode</Label>
        <Select
          value={String(form.sslMode)}
          onValueChange={(v) => set("sslMode", Number(v) as ProxySSLMode)}
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={String(ProxySSLMode.PROXY_SSL_MODE_NONE)}>
              HTTP Only
            </SelectItem>
            <SelectItem value={String(ProxySSLMode.PROXY_SSL_MODE_SELF_SIGNED)}>
              Self-Signed Certificate
            </SelectItem>
            <SelectItem value={String(ProxySSLMode.PROXY_SSL_MODE_ACME)}>
              ACME / Let's Encrypt
            </SelectItem>
            <SelectItem value={String(ProxySSLMode.PROXY_SSL_MODE_CUSTOM)}>
              Custom Certificate
            </SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="flex items-center justify-between">
        <div>
          <Label>WebSocket Support</Label>
          <p className="text-xs text-muted-foreground">
            Forward WebSocket upgrade headers
          </p>
        </div>
        <Switch
          checked={form.websocketSupport}
          onCheckedChange={(v) => set("websocketSupport", v)}
        />
      </div>

      <div className="flex items-center justify-between">
        <div>
          <Label>Basic Authentication</Label>
          <p className="text-xs text-muted-foreground">
            Protect this host with a username and password
          </p>
        </div>
        <Switch
          checked={form.basicAuthEnabled}
          onCheckedChange={(v) => set("basicAuthEnabled", v)}
        />
      </div>

      {form.basicAuthEnabled && (
        <div className="space-y-3 rounded-md border p-3">
          <div className="space-y-1.5">
            <Label htmlFor="basicAuthUser">Username</Label>
            <Input
              id="basicAuthUser"
              value={form.basicAuthUser}
              onChange={(e) => set("basicAuthUser", e.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="basicAuthPassword">Password</Label>
            <Input
              id="basicAuthPassword"
              type="password"
              placeholder="Leave blank to keep existing password"
              value={form.basicAuthPassword}
              onChange={(e) => set("basicAuthPassword", e.target.value)}
            />
          </div>
        </div>
      )}

      <div className="flex items-center justify-between">
        <div>
          <Label>Enabled</Label>
          <p className="text-xs text-muted-foreground">
            Route traffic through this proxy host
          </p>
        </div>
        <Switch
          checked={form.enabled}
          onCheckedChange={(v) => set("enabled", v)}
        />
      </div>
    </div>
  );
}

// ---- MonitorForm sub-component ----

interface MonitorFormState {
  name: string;
  url: string;
  intervalSeconds: number;
  timeoutSeconds: number;
  expectedStatusCode: number;
  enabled: boolean;
}

interface MonitorFormProps {
  form: MonitorFormState;
  onChange: (f: MonitorFormState) => void;
}

function MonitorForm({ form, onChange }: MonitorFormProps) {
  function set<K extends keyof MonitorFormState>(
    key: K,
    value: MonitorFormState[K],
  ) {
    onChange({ ...form, [key]: value });
  }

  return (
    <div className="space-y-4">
      <div className="space-y-1.5">
        <Label htmlFor="monitorName">Name</Label>
        <Input
          id="monitorName"
          placeholder="My Service"
          value={form.name}
          onChange={(e) => set("name", e.target.value)}
        />
      </div>

      <div className="space-y-1.5">
        <Label htmlFor="monitorUrl">URL</Label>
        <Input
          id="monitorUrl"
          placeholder="https://example.com/health"
          value={form.url}
          onChange={(e) => set("url", e.target.value)}
        />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1.5">
          <Label>Check Interval</Label>
          <Select
            value={String(form.intervalSeconds)}
            onValueChange={(v) => set("intervalSeconds", Number(v))}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="30">Every 30 seconds</SelectItem>
              <SelectItem value="60">Every 1 minute</SelectItem>
              <SelectItem value="300">Every 5 minutes</SelectItem>
              <SelectItem value="900">Every 15 minutes</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="expectedStatus">Expected Status</Label>
          <Input
            id="expectedStatus"
            type="number"
            min={100}
            max={599}
            value={form.expectedStatusCode}
            onChange={(e) => set("expectedStatusCode", Number(e.target.value))}
          />
        </div>
      </div>

      <div className="space-y-1.5">
        <Label htmlFor="timeoutSeconds">Timeout (seconds)</Label>
        <Input
          id="timeoutSeconds"
          type="number"
          min={1}
          max={60}
          value={form.timeoutSeconds}
          onChange={(e) => set("timeoutSeconds", Number(e.target.value))}
        />
      </div>

      <div className="flex items-center justify-between">
        <div>
          <Label>Enabled</Label>
          <p className="text-xs text-muted-foreground">
            Actively check this URL
          </p>
        </div>
        <Switch
          checked={form.enabled}
          onCheckedChange={(v) => set("enabled", v)}
        />
      </div>
    </div>
  );
}
