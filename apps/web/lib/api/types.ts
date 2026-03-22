// Cluster summary — returned by useClusterSummary (transformed from proto GetClusterSummaryResponse)
export interface ClusterSummary {
  nodes: { total: number; running: number };
  vms: { total: number; running: number };
  containers: { total: number; running: number };
  stacks: { total: number; running: number };
  cpu: { cores: number; avgUsage: number };
  memory: { used: number; total: number };
  disk: { used: number; total: number };
}

// Chart data point — used by dashboard metric charts
export interface MetricPoint {
  time: string;
  value: number;
}

// SQLite metrics REST API
export interface Metric {
  id: number;
  ts: number;
  node_id: string;
  resource_type: string;
  resource_id?: string;
  value: number;
  unit: string;
}

export interface MetricQuery {
  node_id?: string;
  resource_type?: string;
  resource_id?: string;
  start_time?: number;
  end_time?: number;
  aggregate?: string;
  group_by?: string;
  host_only?: boolean;
}

// SQLite events REST API
export interface Event {
  id: number;
  ts: number;
  node_id: string;
  resource_id?: string;
  event_type: string;
  severity: string;
  message: string;
  metadata?: any;
}

export interface EventQuery {
  node_id?: string;
  resource_id?: string;
  event_type?: string;
  severity?: string;
  start_time?: number;
  end_time?: number;
  limit?: number;
  offset?: number;
}
