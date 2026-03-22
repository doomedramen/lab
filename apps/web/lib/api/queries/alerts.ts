import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  AlertService,
  type NotificationChannel,
  type AlertRule,
  type Alert,
  AlertRuleType,
  NotificationChannelType,
  AlertSeverity,
  AlertStatus,
} from "../../gen/lab/v1/alert_pb";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

const alertClient = createClient(
  AlertService,
  createConnectTransport({
    baseUrl: API_URL,
    useBinaryFormat: true,
  }),
);

// Re-export types for convenience
export type { NotificationChannel, AlertRule, Alert };
export { AlertRuleType, NotificationChannelType, AlertSeverity, AlertStatus };

// --- Notification Channels ---

export function useNotificationChannels() {
  return useQuery({
    queryKey: ["notificationChannels"],
    queryFn: async () => {
      const res = await alertClient.listNotificationChannels({});
      return res.channels;
    },
  });
}

export function useNotificationChannel(id: string) {
  return useQuery({
    queryKey: ["notificationChannels", id],
    queryFn: async () => {
      const res = await alertClient.getNotificationChannel({ id });
      return res.channel;
    },
    enabled: !!id,
  });
}

// --- Alert Rules ---

export function useAlertRules(enabledOnly = false) {
  return useQuery({
    queryKey: ["alertRules", { enabledOnly }],
    queryFn: async () => {
      const res = await alertClient.listAlertRules({ enabledOnly });
      return res.rules;
    },
  });
}

export function useAlertRule(id: string) {
  return useQuery({
    queryKey: ["alertRules", id],
    queryFn: async () => {
      const res = await alertClient.getAlertRule({ id });
      return res.rule;
    },
    enabled: !!id,
  });
}

// --- Fired Alerts ---

export interface AlertFilter {
  status?: AlertStatus;
  severity?: AlertSeverity;
  ruleId?: string;
  entityType?: string;
  entityId?: string;
  openOnly?: boolean;
}

export function useAlerts(filter: AlertFilter = {}) {
  return useQuery({
    queryKey: ["alerts", filter],
    queryFn: async () => {
      const res = await alertClient.listAlerts({
        status: filter.status ?? AlertStatus.UNSPECIFIED,
        severity: filter.severity ?? AlertSeverity.UNSPECIFIED,
        ruleId: filter.ruleId ?? "",
        entityType: filter.entityType ?? "",
        entityId: filter.entityId ?? "",
        openOnly: filter.openOnly ?? false,
      });
      return { alerts: res.alerts, total: res.total };
    },
  });
}

export function useAlert(id: string) {
  return useQuery({
    queryKey: ["alerts", id],
    queryFn: async () => {
      const res = await alertClient.getAlert({ id });
      return res.alert;
    },
    enabled: !!id,
  });
}

// --- Mutations ---

export function useAlertMutations() {
  const queryClient = useQueryClient();

  // Notification Channels
  const createChannel = useMutation({
    mutationFn: (params: {
      name: string;
      type: NotificationChannelType;
      config: Record<string, string>;
    }) =>
      alertClient.createNotificationChannel({
        name: params.name,
        type: params.type,
        config: params.config,
      }),
    onSuccess: () => {
      toast.success("Notification channel created");
      queryClient.invalidateQueries({ queryKey: ["notificationChannels"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to create channel: ${error.message}`);
    },
  });

  const updateChannel = useMutation({
    mutationFn: (params: {
      id: string;
      name?: string;
      config?: Record<string, string>;
      enabled?: boolean;
    }) =>
      alertClient.updateNotificationChannel({
        id: params.id,
        name: params.name,
        config: params.config,
        enabled: params.enabled,
      }),
    onSuccess: () => {
      toast.success("Notification channel updated");
      queryClient.invalidateQueries({ queryKey: ["notificationChannels"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to update channel: ${error.message}`);
    },
  });

  const deleteChannel = useMutation({
    mutationFn: (id: string) => alertClient.deleteNotificationChannel({ id }),
    onSuccess: () => {
      toast.success("Notification channel deleted");
      queryClient.invalidateQueries({ queryKey: ["notificationChannels"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete channel: ${error.message}`);
    },
  });

  // Alert Rules
  const createRule = useMutation({
    mutationFn: (params: {
      name: string;
      description: string;
      type: AlertRuleType;
      threshold?: number;
      durationMinutes: number;
      entityType?: string;
      entityId?: string;
      channelId: string;
      enabled: boolean;
    }) =>
      alertClient.createAlertRule({
        name: params.name,
        description: params.description,
        type: params.type,
        threshold: params.threshold,
        durationMinutes: params.durationMinutes,
        entityType: params.entityType ?? "",
        entityId: params.entityId ?? "",
        channelId: params.channelId,
        enabled: params.enabled,
      }),
    onSuccess: () => {
      toast.success("Alert rule created");
      queryClient.invalidateQueries({ queryKey: ["alertRules"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to create rule: ${error.message}`);
    },
  });

  const updateRule = useMutation({
    mutationFn: (params: {
      id: string;
      name?: string;
      description?: string;
      threshold?: number;
      durationMinutes?: number;
      entityType?: string;
      entityId?: string;
      channelId?: string;
      enabled?: boolean;
    }) =>
      alertClient.updateAlertRule({
        id: params.id,
        name: params.name,
        description: params.description,
        threshold: params.threshold,
        durationMinutes: params.durationMinutes,
        entityType: params.entityType,
        entityId: params.entityId,
        channelId: params.channelId,
        enabled: params.enabled,
      }),
    onSuccess: () => {
      toast.success("Alert rule updated");
      queryClient.invalidateQueries({ queryKey: ["alertRules"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to update rule: ${error.message}`);
    },
  });

  const deleteRule = useMutation({
    mutationFn: (id: string) => alertClient.deleteAlertRule({ id }),
    onSuccess: () => {
      toast.success("Alert rule deleted");
      queryClient.invalidateQueries({ queryKey: ["alertRules"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete rule: ${error.message}`);
    },
  });

  // Alerts
  const acknowledgeAlert = useMutation({
    mutationFn: (id: string) => alertClient.acknowledgeAlert({ id }),
    onSuccess: () => {
      toast.success("Alert acknowledged");
      queryClient.invalidateQueries({ queryKey: ["alerts"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to acknowledge alert: ${error.message}`);
    },
  });

  const resolveAlert = useMutation({
    mutationFn: (id: string) => alertClient.resolveAlert({ id }),
    onSuccess: () => {
      toast.success("Alert resolved");
      queryClient.invalidateQueries({ queryKey: ["alerts"] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to resolve alert: ${error.message}`);
    },
  });

  return {
    // Channels
    createChannel,
    updateChannel,
    deleteChannel,
    isCreatingChannel: createChannel.isPending,
    isUpdatingChannel: updateChannel.isPending,
    isDeletingChannel: deleteChannel.isPending,
    // Rules
    createRule,
    updateRule,
    deleteRule,
    isCreatingRule: createRule.isPending,
    isUpdatingRule: updateRule.isPending,
    isDeletingRule: deleteRule.isPending,
    // Alerts
    acknowledgeAlert,
    resolveAlert,
    isAcknowledging: acknowledgeAlert.isPending,
    isResolving: resolveAlert.isPending,
  };
}
