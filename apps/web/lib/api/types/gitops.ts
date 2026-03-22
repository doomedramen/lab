/**
 * GitOps configuration for managing infrastructure from Git
 */
export interface GitOpsConfig {
  id: string;
  name: string;
  description: string;
  gitUrl: string;
  gitBranch: string;
  gitPath: string;
  syncInterval: number; // seconds
  lastSync?: string;
  lastSyncHash?: string;
  status: GitOpsStatus;
  statusMessage: string;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
  nextSync?: string;
  syncRetries: number;
  maxSyncRetries: number;
}

export type GitOpsStatus = "Healthy" | "OutOfSync" | "Failed" | "Pending";

/**
 * GitOps resource parsed from manifest
 */
export interface GitOpsResource {
  id: string;
  configId: string;
  kind: string; // VirtualMachine, Container, Network, etc.
  name: string;
  namespace: string;
  manifestPath: string;
  manifestHash: string;
  status: GitOpsStatus;
  statusMessage: string;
  lastApplied?: string;
  lastDiff?: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * Sync log entry
 */
export interface GitOpsSyncLog {
  id: string;
  configId: string;
  startTime: string;
  endTime?: string;
  duration: number; // seconds
  status: GitOpsStatus;
  message: string;
  commitHash: string;
  resourcesScanned: number;
  resourcesCreated: number;
  resourcesUpdated: number;
  resourcesDeleted: number;
  resourcesFailed: number;
}

/**
 * Input for creating a GitOps config
 */
export interface CreateGitOpsConfigInput {
  name: string;
  description?: string;
  gitUrl: string;
  gitBranch?: string;
  gitPath?: string;
  syncInterval?: number;
  enabled?: boolean;
  maxSyncRetries?: number;
}
