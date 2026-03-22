"use client";

import React, {
  createContext,
  useContext,
  useEffect,
  useState,
  useCallback,
  useMemo,
} from "react";
import { authClient, AuthClient } from "./auth-client";
import type { User, APIKey } from "../../lib/gen/lab/v1/auth_pb";

export interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (email: string, password: string, mfaCode?: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshAccessToken: () => Promise<boolean>;
  setupMFA: () => Promise<{
    secret: string;
    qrCodeUrl: string;
    manualKey: string;
    backupCodes: string[];
  }>;
  enableMFA: (mfaCode: string) => Promise<void>;
  disableMFA: (mfaCode: string) => Promise<void>;
  createAPIKey: (
    name: string,
    permissions: string[],
    expiresAt?: string,
  ) => Promise<void>;
  listAPIKeys: () => Promise<APIKey[]>;
  revokeAPIKey: (id: string) => Promise<void>;
  error: string | null;
  clearError: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

interface AuthProviderProps {
  children: React.ReactNode;
  client?: AuthClient;
}

export function AuthProvider({
  children,
  client = authClient,
}: AuthProviderProps) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const clearError = useCallback(() => {
    setError(null);
  }, []);

  const loadUser = useCallback(async () => {
    // Load tokens from localStorage
    const hasTokens = client.loadTokens();
    if (!hasTokens) {
      setIsLoading(false);
      return;
    }

    try {
      const response = await client.getCurrentUser();
      setUser(response.user ?? null);
    } catch (err) {
      // Token might be expired, try to refresh
      const refreshed = await client.refreshAccessToken();
      if (refreshed) {
        try {
          const response = await client.getCurrentUser();
          setUser(response.user ?? null);
        } catch {
          setUser(null);
        }
      } else {
        setUser(null);
      }
    } finally {
      setIsLoading(false);
    }
  }, [client]);

  useEffect(() => {
    loadUser();
  }, [loadUser]);

  const login = useCallback(
    async (email: string, password: string, mfaCode?: string) => {
      setError(null);
      try {
        const response = await client.login(email, password, mfaCode);

        if (response.mfaRequired) {
          // MFA is required, throw special error
          throw new MFARequiredError(response.user ?? null);
        }

        setUser(response.user ?? null);
      } catch (err) {
        if (err instanceof MFARequiredError) {
          throw err;
        }
        setError(err instanceof Error ? err.message : "Login failed");
        throw err;
      }
    },
    [client],
  );

  const register = useCallback(
    async (email: string, password: string) => {
      setError(null);
      try {
        const response = await client.register(email, password);
        setUser(response.user ?? null);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Registration failed");
        throw err;
      }
    },
    [client],
  );

  const logout = useCallback(async () => {
    setError(null);
    try {
      await client.logout();
    } finally {
      setUser(null);
    }
  }, [client]);

  const refreshAccessToken = useCallback(async () => {
    return client.refreshAccessToken();
  }, [client]);

  const setupMFA = useCallback(async () => {
    setError(null);
    try {
      const response = await client.setupMFA();
      return {
        secret: response.secret,
        qrCodeUrl: response.qrCodeUrl,
        manualKey: response.manualKey,
        backupCodes: response.backupCodes,
      };
    } catch (err) {
      setError(err instanceof Error ? err.message : "MFA setup failed");
      throw err;
    }
  }, [client]);

  const enableMFA = useCallback(
    async (mfaCode: string) => {
      setError(null);
      try {
        await client.enableMFA(mfaCode);
        // Update local user state
        const response = await client.getCurrentUser();
        setUser(response.user ?? null);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to enable MFA");
        throw err;
      }
    },
    [client],
  );

  const disableMFA = useCallback(
    async (mfaCode: string) => {
      setError(null);
      try {
        await client.disableMFA(mfaCode);
        // Update local user state
        const response = await client.getCurrentUser();
        setUser(response.user ?? null);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to disable MFA");
        throw err;
      }
    },
    [client],
  );

  const createAPIKey = useCallback(
    async (name: string, permissions: string[], expiresAt?: string) => {
      setError(null);
      try {
        await client.createAPIKey(name, permissions, expiresAt);
      } catch (err) {
        setError(
          err instanceof Error ? err.message : "Failed to create API key",
        );
        throw err;
      }
    },
    [client],
  );

  const listAPIKeys = useCallback(async () => {
    setError(null);
    try {
      const response = await client.listAPIKeys();
      return response.apiKeys;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to list API keys");
      throw err;
    }
  }, [client]);

  const revokeAPIKey = useCallback(
    async (id: string) => {
      setError(null);
      try {
        await client.revokeAPIKey(id);
      } catch (err) {
        setError(
          err instanceof Error ? err.message : "Failed to revoke API key",
        );
        throw err;
      }
    },
    [client],
  );

  const value = useMemo<AuthContextType>(
    () => ({
      user,
      isLoading,
      isAuthenticated: !!user,
      login,
      register,
      logout,
      refreshAccessToken,
      setupMFA,
      enableMFA,
      disableMFA,
      createAPIKey,
      listAPIKeys,
      revokeAPIKey,
      error,
      clearError,
    }),
    [
      user,
      isLoading,
      login,
      register,
      logout,
      refreshAccessToken,
      setupMFA,
      enableMFA,
      disableMFA,
      createAPIKey,
      listAPIKeys,
      revokeAPIKey,
      error,
      clearError,
    ],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}

/**
 * MFARequiredError is thrown when login requires MFA code
 */
export class MFARequiredError extends Error {
  user: User | null;

  constructor(user: User | null) {
    super("MFA code required");
    this.name = "MFARequiredError";
    this.user = user;
  }
}
