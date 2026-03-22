import { createConnectTransport } from "@connectrpc/connect-web";
import { createClient, type Interceptor } from "@connectrpc/connect";
import { ClusterService } from "../gen/lab/v1/cluster_pb";
import { NodeService } from "../gen/lab/v1/node_pb";
import { VmService } from "../gen/lab/v1/vm_pb";
import { ContainerService } from "../gen/lab/v1/container_pb";
import { StackService } from "../gen/lab/v1/stack_pb";
import { IsoService } from "../gen/lab/v1/iso_pb";
import { AuthService } from "../gen/lab/v1/auth_pb";
import { SnapshotService } from "../gen/lab/v1/snapshot_pb";
import { BackupService } from "../gen/lab/v1/backup_pb";
import { StorageService } from "../gen/lab/v1/storage_pb";
import { NetworkService } from "../gen/lab/v1/network_pb";
import { FirewallService } from "../gen/lab/v1/network_pb";
import { TaskService } from "../gen/lab/v1/task_pb";
import { ProxyService } from "../gen/lab/v1/proxy_pb";
import { GitOpsService } from "../gen/lab/v1/gitops_pb";
import { authClient as authManager } from "../auth/auth-client";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

// Interceptor to handle auth errors
const authInterceptor: Interceptor = (next) => async (req) => {
  const token = authManager.getAccessToken();
  if (token) {
    req.header.set("Authorization", `Bearer ${token}`);
  }

  try {
    return await next(req);
  } catch (error) {
    // Handle ConnectRPC auth errors
    // Check if error is a ConnectError by checking for code property
    if (error && typeof error === "object" && "code" in error) {
      const connectError = error as { code: string };
      if (
        connectError.code === "unauthenticated" ||
        connectError.code === "permission_denied"
      ) {
        if (typeof window !== "undefined") {
          authManager.clearTokens();
          window.location.href = `/login?from=${encodeURIComponent(window.location.pathname)}`;
        }
      }
    }
    // Handle network errors like "Failed to fetch" - likely auth/session expired
    if (error instanceof TypeError && error.message === "Failed to fetch") {
      if (typeof window !== "undefined") {
        authManager.clearTokens();
        window.location.href = `/login?from=${encodeURIComponent(window.location.pathname)}`;
      }
    }
    throw error;
  }
};

const transport = createConnectTransport({
  baseUrl: API_BASE_URL,
  useBinaryFormat: false,
  interceptors: [authInterceptor],
});

// Auth service client (doesn't need auth headers)
export const authClient = createClient(AuthService, transport);

// Base clients without auth (use authenticated clients below for protected routes)
export const clusterClient = createClient(ClusterService, transport);
export const nodeClient = createClient(NodeService, transport);
export const vmClient = createClient(VmService, transport);
export const containerClient = createClient(ContainerService, transport);
export const stackClient = createClient(StackService, transport);
export const isoClient = createClient(IsoService, transport);
export const snapshotClient = createClient(SnapshotService, transport);
export const backupClient = createClient(BackupService, transport);
export const storageClient = createClient(StorageService, transport);
export const networkClient = createClient(NetworkService, transport);
export const firewallClient = createClient(FirewallService, transport);
export const taskClient = createClient(TaskService, transport);
export const proxyClient = createClient(ProxyService, transport);
export const gitOpsClient = createClient(GitOpsService, transport);
