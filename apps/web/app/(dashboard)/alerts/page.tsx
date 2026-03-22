"use client";

import { useState, Suspense } from "react";
import {
  useAlerts,
  useAlertRules,
  useNotificationChannels,
  useAlertMutations,
  AlertRuleType,
  NotificationChannelType,
  AlertSeverity,
  AlertStatus,
} from "@/lib/api/queries/alerts";
import {
  TabsPersistent,
  TabsList,
  TabsTrigger,
  TabsContent,
} from "@/components/tabs-persistent";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Shimmer } from "@/components/shimmer";
import {
  RefreshCw,
  Bell,
  BellOff,
  Plus,
  Trash2,
  Edit,
  Check,
  X,
  AlertTriangle,
  Info,
  AlertCircle,
  Mail,
  Webhook,
  Settings,
} from "lucide-react";
import { formatDistanceToNow } from "date-fns";

// --- Template Data for Shimmer ---

const templateAlerts = Array.from({ length: 5 }).map((_, i) => ({
  id: `alert-template-${i}`,
  severity: AlertSeverity.WARNING,
  ruleName: "High CPU Usage",
  message: "CPU usage exceeded 80% on node-01",
  entityName: "node-01",
  status: AlertStatus.OPEN,
  firedAt: new Date().toISOString(),
})) as any;

const templateRules = Array.from({ length: 3 }).map((_, i) => ({
  id: `rule-template-${i}`,
  name: "Loading Rule...",
  type: AlertRuleType.STORAGE_POOL_USAGE,
  threshold: 80,
  channelId: "channel-1",
  enabled: true,
})) as any;

const templateChannels = Array.from({ length: 2 }).map((_, i) => ({
  id: `channel-template-${i}`,
  name: "Loading Channel...",
  type: NotificationChannelType.EMAIL,
  enabled: true,
  config: {},
})) as any;

// Helper functions
function alertRuleTypeToString(type: AlertRuleType): string {
  const labels: Record<AlertRuleType, string> = {
    [AlertRuleType.UNSPECIFIED]: "Unknown",
    [AlertRuleType.STORAGE_POOL_USAGE]: "Storage Pool Usage",
    [AlertRuleType.VM_STOPPED]: "VM Stopped",
    [AlertRuleType.BACKUP_FAILED]: "Backup Failed",
    [AlertRuleType.NODE_OFFLINE]: "Node Offline",
    [AlertRuleType.CPU_USAGE]: "CPU Usage",
    [AlertRuleType.MEMORY_USAGE]: "Memory Usage",
  };
  return labels[type] || "Unknown";
}

function channelTypeToString(type: NotificationChannelType): string {
  const labels: Record<NotificationChannelType, string> = {
    [NotificationChannelType.UNSPECIFIED]: "Unknown",
    [NotificationChannelType.EMAIL]: "Email",
    [NotificationChannelType.WEBHOOK]: "Webhook",
  };
  return labels[type] || "Unknown";
}

function severityToBadge(severity: AlertSeverity) {
  const config: Record<
    AlertSeverity,
    {
      label: string;
      variant: "default" | "secondary" | "destructive" | "outline";
      icon: React.ReactNode;
    }
  > = {
    [AlertSeverity.UNSPECIFIED]: {
      label: "Unknown",
      variant: "secondary",
      icon: <Info className="h-3 w-3" />,
    },
    [AlertSeverity.INFO]: {
      label: "Info",
      variant: "outline",
      icon: <Info className="h-3 w-3" />,
    },
    [AlertSeverity.WARNING]: {
      label: "Warning",
      variant: "default",
      icon: <AlertTriangle className="h-3 w-3" />,
    },
    [AlertSeverity.CRITICAL]: {
      label: "Critical",
      variant: "destructive",
      icon: <AlertCircle className="h-3 w-3" />,
    },
  };
  const { label, variant, icon } =
    config[severity] || config[AlertSeverity.UNSPECIFIED];
  return (
    <Badge variant={variant} className="gap-1">
      {icon}
      {label}
    </Badge>
  );
}

function statusToBadge(status: AlertStatus) {
  const config: Record<
    AlertStatus,
    {
      label: string;
      variant: "default" | "secondary" | "destructive" | "outline";
    }
  > = {
    [AlertStatus.UNSPECIFIED]: { label: "Unknown", variant: "secondary" },
    [AlertStatus.OPEN]: { label: "Open", variant: "destructive" },
    [AlertStatus.ACKNOWLEDGED]: { label: "Acknowledged", variant: "default" },
    [AlertStatus.RESOLVED]: { label: "Resolved", variant: "outline" },
  };
  const { label, variant } = config[status] || config[AlertStatus.UNSPECIFIED];
  return <Badge variant={variant}>{label}</Badge>;
}

function AlertsTableBody({
  alerts,
  onAcknowledge,
  onResolve,
  acknowledgePending,
  resolvePending,
}: {
  alerts: any[];
  onAcknowledge: (id: string) => void;
  onResolve: (id: string) => void;
  acknowledgePending: boolean;
  resolvePending: boolean;
}) {
  if (alerts.length === 0) {
    return (
      <TableRow>
        <TableCell
          colSpan={7}
          className="text-center py-8 text-muted-foreground"
        >
          No alerts found
        </TableCell>
      </TableRow>
    );
  }

  return (
    <>
      {alerts.map((alert) => (
        <TableRow key={alert.id}>
          <TableCell>{severityToBadge(alert.severity)}</TableCell>
          <TableCell className="font-medium">{alert.ruleName}</TableCell>
          <TableCell className="max-w-[300px] truncate">
            {alert.message}
          </TableCell>
          <TableCell>
            {alert.entityName && (
              <span className="text-sm">{alert.entityName}</span>
            )}
          </TableCell>
          <TableCell>{statusToBadge(alert.status)}</TableCell>
          <TableCell className="text-sm text-muted-foreground">
            {formatDistanceToNow(new Date(alert.firedAt), { addSuffix: true })}
          </TableCell>
          <TableCell>
            <div className="flex gap-1">
              {alert.status === AlertStatus.OPEN && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => onAcknowledge(alert.id)}
                  disabled={acknowledgePending}
                >
                  <Check className="h-3 w-3" />
                </Button>
              )}
              {(alert.status === AlertStatus.OPEN ||
                alert.status === AlertStatus.ACKNOWLEDGED) && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => onResolve(alert.id)}
                  disabled={resolvePending}
                >
                  <X className="h-3 w-3" />
                </Button>
              )}
            </div>
          </TableCell>
        </TableRow>
      ))}
    </>
  );
}

// Alerts Tab Content
function AlertsTab() {
  const [statusFilter, setStatusFilter] = useState<AlertStatus>(
    AlertStatus.UNSPECIFIED,
  );
  const { data, isLoading, error, refetch } = useAlerts({
    status: statusFilter || undefined,
    openOnly: statusFilter === AlertStatus.OPEN,
  });
  const { acknowledgeAlert, resolveAlert } = useAlertMutations();

  if (error) {
    return (
      <div className="rounded-md border border-red-200 bg-red-50 p-4">
        <h3 className="text-sm font-medium text-red-800">
          Failed to load alerts
        </h3>
        <p className="mt-1 text-sm text-red-600">{error.message}</p>
        <Button
          variant="outline"
          size="sm"
          className="mt-2"
          onClick={() => refetch()}
        >
          <RefreshCw className="h-4 w-4 mr-2" />
          Retry
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <Select
          value={statusFilter.toString()}
          onValueChange={(v) => setStatusFilter(Number(v) as AlertStatus)}
        >
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="Filter by status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={AlertStatus.UNSPECIFIED.toString()}>
              All Alerts
            </SelectItem>
            <SelectItem value={AlertStatus.OPEN.toString()}>
              Open Only
            </SelectItem>
            <SelectItem value={AlertStatus.ACKNOWLEDGED.toString()}>
              Acknowledged
            </SelectItem>
            <SelectItem value={AlertStatus.RESOLVED.toString()}>
              Resolved
            </SelectItem>
          </SelectContent>
        </Select>
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          <RefreshCw className="h-4 w-4 mr-2" />
          Refresh
        </Button>
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Severity</TableHead>
              <TableHead>Rule</TableHead>
              <TableHead>Message</TableHead>
              <TableHead>Entity</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Fired</TableHead>
              <TableHead className="w-[150px]">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <Shimmer
              loading={isLoading}
              templateProps={{ alerts: templateAlerts }}
            >
              <AlertsTableBody
                alerts={data?.alerts || templateAlerts}
                onAcknowledge={(id) => acknowledgeAlert.mutate(id)}
                onResolve={(id) => resolveAlert.mutate(id)}
                acknowledgePending={acknowledgeAlert.isPending}
                resolvePending={resolveAlert.isPending}
              />
            </Shimmer>
          </TableBody>
        </Table>
      </div>
    </div>
  );
}

function RulesTableBody({
  rules,
  channels,
  onEdit,
  onDelete,
}: {
  rules: any[];
  channels?: any[];
  onEdit: (rule: any) => void;
  onDelete: (id: string) => void;
}) {
  if (rules.length === 0) {
    return (
      <TableRow>
        <TableCell
          colSpan={6}
          className="text-center py-8 text-muted-foreground"
        >
          No alert rules configured
        </TableCell>
      </TableRow>
    );
  }

  return (
    <>
      {rules.map((rule) => (
        <TableRow key={rule.id}>
          <TableCell className="font-medium">{rule.name}</TableCell>
          <TableCell>{alertRuleTypeToString(rule.type)}</TableCell>
          <TableCell>{rule.threshold ? `${rule.threshold}%` : "-"}</TableCell>
          <TableCell>
            {channels?.find((c) => c.id === rule.channelId)?.name ||
              rule.channelId}
          </TableCell>
          <TableCell>
            <Badge variant={rule.enabled ? "default" : "secondary"}>
              {rule.enabled ? "Enabled" : "Disabled"}
            </Badge>
          </TableCell>
          <TableCell>
            <div className="flex gap-1">
              <Button variant="ghost" size="sm" onClick={() => onEdit(rule)}>
                <Edit className="h-3 w-3" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onDelete(rule.id)}
              >
                <Trash2 className="h-3 w-3 text-destructive" />
              </Button>
            </div>
          </TableCell>
        </TableRow>
      ))}
    </>
  );
}

// Rules Tab Content
function RulesTab() {
  const { data: rules, isLoading, error, refetch } = useAlertRules();
  const { data: channels } = useNotificationChannels();
  const { createRule, updateRule, deleteRule } = useAlertMutations();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<string | null>(null);
  const [formData, setFormData] = useState({
    name: "",
    description: "",
    type: AlertRuleType.STORAGE_POOL_USAGE,
    threshold: 80,
    durationMinutes: 0,
    channelId: "",
    enabled: true,
  });

  const resetForm = () => {
    setFormData({
      name: "",
      description: "",
      type: AlertRuleType.STORAGE_POOL_USAGE,
      threshold: 80,
      durationMinutes: 0,
      channelId: "",
      enabled: true,
    });
    setEditingRule(null);
  };

  const handleOpenDialog = (rule?: any) => {
    if (rule) {
      setEditingRule(rule.id);
      setFormData({
        name: rule.name,
        description: rule.description,
        type: rule.type,
        threshold: rule.threshold ?? 80,
        durationMinutes: rule.durationMinutes,
        channelId: rule.channelId,
        enabled: rule.enabled,
      });
    } else {
      resetForm();
    }
    setDialogOpen(true);
  };

  const handleSubmit = () => {
    if (editingRule) {
      updateRule.mutate(
        {
          id: editingRule,
          name: formData.name,
          description: formData.description,
          threshold: formData.threshold,
          durationMinutes: formData.durationMinutes,
          channelId: formData.channelId,
          enabled: formData.enabled,
        },
        {
          onSuccess: () => {
            setDialogOpen(false);
            resetForm();
          },
        },
      );
    } else {
      createRule.mutate(
        {
          name: formData.name,
          description: formData.description,
          type: formData.type,
          threshold: formData.threshold,
          durationMinutes: formData.durationMinutes,
          channelId: formData.channelId,
          enabled: formData.enabled,
        },
        {
          onSuccess: () => {
            setDialogOpen(false);
            resetForm();
          },
        },
      );
    }
  };

  const handleDelete = (id: string) => {
    if (confirm("Are you sure you want to delete this alert rule?")) {
      deleteRule.mutate(id);
    }
  };

  if (error) {
    return (
      <div className="rounded-md border border-red-200 bg-red-50 p-4">
        <h3 className="text-sm font-medium text-red-800">
          Failed to load alert rules
        </h3>
        <p className="mt-1 text-sm text-red-600">{error.message}</p>
        <Button
          variant="outline"
          size="sm"
          className="mt-2"
          onClick={() => refetch()}
        >
          <RefreshCw className="h-4 w-4 mr-2" />
          Retry
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">Alert Rules</h3>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
          <Button size="sm" onClick={() => handleOpenDialog()}>
            <Plus className="h-4 w-4 mr-2" />
            Add Rule
          </Button>
        </div>
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>Threshold</TableHead>
              <TableHead>Channel</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="w-[100px]">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <Shimmer
              loading={isLoading}
              templateProps={{ rules: templateRules }}
            >
              <RulesTableBody
                rules={rules || templateRules}
                channels={channels}
                onEdit={handleOpenDialog}
                onDelete={handleDelete}
              />
            </Shimmer>
          </TableBody>
        </Table>
      </div>

      {/* Rule Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editingRule ? "Edit Alert Rule" : "Create Alert Rule"}
            </DialogTitle>
            <DialogDescription>
              Configure when alerts should be triggered and how to notify.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="rule-name">Name</Label>
              <Input
                id="rule-name"
                value={formData.name}
                onChange={(e) =>
                  setFormData({ ...formData, name: e.target.value })
                }
                placeholder="e.g., High CPU Usage"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="rule-type">Type</Label>
              <Select
                value={formData.type.toString()}
                onValueChange={(v) =>
                  setFormData({ ...formData, type: Number(v) as AlertRuleType })
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem
                    value={AlertRuleType.STORAGE_POOL_USAGE.toString()}
                  >
                    Storage Pool Usage
                  </SelectItem>
                  <SelectItem value={AlertRuleType.VM_STOPPED.toString()}>
                    VM Stopped
                  </SelectItem>
                  <SelectItem value={AlertRuleType.BACKUP_FAILED.toString()}>
                    Backup Failed
                  </SelectItem>
                  <SelectItem value={AlertRuleType.NODE_OFFLINE.toString()}>
                    Node Offline
                  </SelectItem>
                  <SelectItem value={AlertRuleType.CPU_USAGE.toString()}>
                    CPU Usage
                  </SelectItem>
                  <SelectItem value={AlertRuleType.MEMORY_USAGE.toString()}>
                    Memory Usage
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
            {[
              AlertRuleType.STORAGE_POOL_USAGE,
              AlertRuleType.CPU_USAGE,
              AlertRuleType.MEMORY_USAGE,
            ].includes(formData.type) && (
              <div className="space-y-2">
                <Label htmlFor="threshold">Threshold (%)</Label>
                <Input
                  id="threshold"
                  type="number"
                  value={formData.threshold}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      threshold: Number(e.target.value),
                    })
                  }
                  min={0}
                  max={100}
                />
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="channel">Notification Channel</Label>
              <Select
                value={formData.channelId}
                onValueChange={(v) =>
                  setFormData({ ...formData, channelId: v })
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select a channel" />
                </SelectTrigger>
                <SelectContent>
                  {channels
                    ?.filter((c) => c.enabled)
                    .map((channel) => (
                      <SelectItem key={channel.id} value={channel.id}>
                        {channel.name} ({channelTypeToString(channel.type)})
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center justify-between">
              <Label htmlFor="enabled">Enabled</Label>
              <Switch
                id="enabled"
                checked={formData.enabled}
                onCheckedChange={(checked) =>
                  setFormData({ ...formData, enabled: checked })
                }
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={
                !formData.name ||
                !formData.channelId ||
                createRule.isPending ||
                updateRule.isPending
              }
            >
              {editingRule ? "Update" : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function ChannelsGrid({
  channels,
  onEdit,
  onDelete,
}: {
  channels: any[];
  onEdit: (channel: any) => void;
  onDelete: (id: string) => void;
}) {
  if (channels.length === 0) {
    return (
      <Card className="col-span-2">
        <CardContent className="py-8 text-center text-muted-foreground">
          No notification channels configured
        </CardContent>
      </Card>
    );
  }

  return (
    <>
      {channels.map((channel) => (
        <Card key={channel.id}>
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-base flex items-center gap-2">
                {channel.type === NotificationChannelType.EMAIL ? (
                  <Mail className="h-4 w-4" />
                ) : (
                  <Webhook className="h-4 w-4" />
                )}
                {channel.name}
              </CardTitle>
              <div className="flex gap-1">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onEdit(channel)}
                >
                  <Edit className="h-3 w-3" />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onDelete(channel.id)}
                >
                  <Trash2 className="h-3 w-3 text-destructive" />
                </Button>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">
                {channelTypeToString(channel.type)}
              </span>
              <Badge variant={channel.enabled ? "default" : "secondary"}>
                {channel.enabled ? "Enabled" : "Disabled"}
              </Badge>
            </div>
          </CardContent>
        </Card>
      ))}
    </>
  );
}

// Channels Tab Content
function ChannelsTab() {
  const {
    data: channels,
    isLoading,
    error,
    refetch,
  } = useNotificationChannels();
  const { createChannel, updateChannel, deleteChannel } = useAlertMutations();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingChannel, setEditingChannel] = useState<string | null>(null);
  const [formData, setFormData] = useState({
    name: "",
    type: NotificationChannelType.EMAIL,
    config: {} as Record<string, string>,
    enabled: true,
  });

  const resetForm = () => {
    setFormData({
      name: "",
      type: NotificationChannelType.EMAIL,
      config: {},
      enabled: true,
    });
    setEditingChannel(null);
  };

  const handleOpenDialog = (channel?: any) => {
    if (channel) {
      setEditingChannel(channel.id);
      setFormData({
        name: channel.name,
        type: channel.type,
        config: channel.config,
        enabled: channel.enabled,
      });
    } else {
      resetForm();
    }
    setDialogOpen(true);
  };

  const handleSubmit = () => {
    if (editingChannel) {
      updateChannel.mutate(
        {
          id: editingChannel,
          name: formData.name,
          config: formData.config,
          enabled: formData.enabled,
        },
        {
          onSuccess: () => {
            setDialogOpen(false);
            resetForm();
          },
        },
      );
    } else {
      createChannel.mutate(
        {
          name: formData.name,
          type: formData.type,
          config: formData.config,
        },
        {
          onSuccess: () => {
            setDialogOpen(false);
            resetForm();
          },
        },
      );
    }
  };

  const handleDelete = (id: string) => {
    if (confirm("Are you sure you want to delete this notification channel?")) {
      deleteChannel.mutate(id);
    }
  };

  if (error) {
    return (
      <div className="rounded-md border border-red-200 bg-red-50 p-4">
        <h3 className="text-sm font-medium text-red-800">
          Failed to load notification channels
        </h3>
        <p className="mt-1 text-sm text-red-600">{error.message}</p>
        <Button
          variant="outline"
          size="sm"
          className="mt-2"
          onClick={() => refetch()}
        >
          <RefreshCw className="h-4 w-4 mr-2" />
          Retry
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">Notification Channels</h3>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
          <Button size="sm" onClick={() => handleOpenDialog()}>
            <Plus className="h-4 w-4 mr-2" />
            Add Channel
          </Button>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Shimmer
          loading={isLoading}
          templateProps={{ channels: templateChannels }}
        >
          <ChannelsGrid
            channels={channels || templateChannels}
            onEdit={handleOpenDialog}
            onDelete={handleDelete}
          />
        </Shimmer>
      </div>

      {/* Channel Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editingChannel
                ? "Edit Notification Channel"
                : "Create Notification Channel"}
            </DialogTitle>
            <DialogDescription>
              Configure where alert notifications should be sent.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="channel-name">Name</Label>
              <Input
                id="channel-name"
                value={formData.name}
                onChange={(e) =>
                  setFormData({ ...formData, name: e.target.value })
                }
                placeholder="e.g., Admin Email"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="channel-type">Type</Label>
              <Select
                value={formData.type.toString()}
                onValueChange={(v) =>
                  setFormData({
                    ...formData,
                    type: Number(v) as NotificationChannelType,
                    config: {},
                  })
                }
                disabled={!!editingChannel}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={NotificationChannelType.EMAIL.toString()}>
                    Email
                  </SelectItem>
                  <SelectItem
                    value={NotificationChannelType.WEBHOOK.toString()}
                  >
                    Webhook
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            {formData.type === NotificationChannelType.EMAIL && (
              <>
                <div className="space-y-2">
                  <Label htmlFor="smtp-host">SMTP Host</Label>
                  <Input
                    id="smtp-host"
                    value={formData.config.smtp_host || ""}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        config: {
                          ...formData.config,
                          smtp_host: e.target.value,
                        },
                      })
                    }
                    placeholder="smtp.example.com"
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="smtp-port">SMTP Port</Label>
                    <Input
                      id="smtp-port"
                      type="number"
                      value={formData.config.smtp_port || "587"}
                      onChange={(e) =>
                        setFormData({
                          ...formData,
                          config: {
                            ...formData.config,
                            smtp_port: e.target.value,
                          },
                        })
                      }
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="smtp-user">SMTP User</Label>
                    <Input
                      id="smtp-user"
                      value={formData.config.smtp_user || ""}
                      onChange={(e) =>
                        setFormData({
                          ...formData,
                          config: {
                            ...formData.config,
                            smtp_user: e.target.value,
                          },
                        })
                      }
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="smtp-pass">SMTP Password</Label>
                  <Input
                    id="smtp-pass"
                    type="password"
                    value={formData.config.smtp_pass || ""}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        config: {
                          ...formData.config,
                          smtp_pass: e.target.value,
                        },
                      })
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="from-address">From Address</Label>
                  <Input
                    id="from-address"
                    type="email"
                    value={formData.config.from_address || ""}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        config: {
                          ...formData.config,
                          from_address: e.target.value,
                        },
                      })
                    }
                    placeholder="alerts@example.com"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="to-addresses">To Addresses</Label>
                  <Input
                    id="to-addresses"
                    value={formData.config.to_addresses || ""}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        config: {
                          ...formData.config,
                          to_addresses: e.target.value,
                        },
                      })
                    }
                    placeholder="admin@example.com, ops@example.com"
                  />
                </div>
              </>
            )}

            {formData.type === NotificationChannelType.WEBHOOK && (
              <>
                <div className="space-y-2">
                  <Label htmlFor="webhook-url">URL</Label>
                  <Input
                    id="webhook-url"
                    value={formData.config.url || ""}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        config: { ...formData.config, url: e.target.value },
                      })
                    }
                    placeholder="https://hooks.example.com/alert"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="webhook-method">Method</Label>
                  <Select
                    value={formData.config.method || "POST"}
                    onValueChange={(v) =>
                      setFormData({
                        ...formData,
                        config: { ...formData.config, method: v },
                      })
                    }
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="POST">POST</SelectItem>
                      <SelectItem value="PUT">PUT</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </>
            )}

            <div className="flex items-center justify-between">
              <Label htmlFor="channel-enabled">Enabled</Label>
              <Switch
                id="channel-enabled"
                checked={formData.enabled}
                onCheckedChange={(checked) =>
                  setFormData({ ...formData, enabled: checked })
                }
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={
                !formData.name ||
                createChannel.isPending ||
                updateChannel.isPending
              }
            >
              {editingChannel ? "Update" : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function AlertsPageContent() {
  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-foreground flex items-center gap-2">
            <Bell className="h-6 w-6" />
            Alerts
          </h1>
          <p className="text-sm text-muted-foreground mt-1">
            Monitor and manage alerts and notifications
          </p>
        </div>
      </div>

      {/* Tabs */}
      <TabsPersistent
        defaultValue="alerts"
        className="flex flex-col"
        paramKey="alerts-tab"
      >
        <TabsList>
          <TabsTrigger value="alerts">
            <AlertTriangle className="h-4 w-4 mr-2" />
            Alerts
          </TabsTrigger>
          <TabsTrigger value="rules">
            <Settings className="h-4 w-4 mr-2" />
            Rules
          </TabsTrigger>
          <TabsTrigger value="channels">
            <Bell className="h-4 w-4 mr-2" />
            Channels
          </TabsTrigger>
        </TabsList>

        <TabsContent value="alerts" className="mt-4">
          <AlertsTab />
        </TabsContent>
        <TabsContent value="rules" className="mt-4">
          <RulesTab />
        </TabsContent>
        <TabsContent value="channels" className="mt-4">
          <ChannelsTab />
        </TabsContent>
      </TabsPersistent>
    </div>
  );
}

export default function AlertsPage() {
  return (
    <Suspense fallback={null}>
      <AlertsPageContent />
    </Suspense>
  );
}
