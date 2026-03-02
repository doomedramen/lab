import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { authClient } from "../client"
import type { SetupMFAResponse, CreateAPIKeyResponse, RevokeOtherSessionsResponse } from "@/lib/gen/lab/v1/auth_pb"

// ---------------------------------------------------------------------------
// Password / email change
// ---------------------------------------------------------------------------

interface UpdateProfileOptions {
  onSuccess?: () => void
}

export function useUpdateProfile(options: UpdateProfileOptions = {}) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      email,
      currentPassword,
      newPassword,
    }: {
      email?: string
      currentPassword?: string
      newPassword?: string
    }) =>
      authClient.updateCurrentUser({
        email: email ?? "",
        currentPassword: currentPassword ?? "",
        newPassword: newPassword ?? "",
      }),
    onSuccess: () => {
      toast.success("Profile updated successfully")
      queryClient.invalidateQueries({ queryKey: ["current-user"] })
      options.onSuccess?.()
    },
    onError: (error: Error) => {
      toast.error(`Failed to update profile: ${error.message}`)
    },
  })
}

// ---------------------------------------------------------------------------
// MFA management
// ---------------------------------------------------------------------------

interface MFAOptions {
  onSuccess?: () => void
}

export function useSetupMFA() {
  return useMutation<SetupMFAResponse, Error, void>({
    mutationFn: () => authClient.setupMFA({}),
    onError: (error: Error) => {
      toast.error(`Failed to set up MFA: ${error.message}`)
    },
  })
}

export function useEnableMFA(options: MFAOptions = {}) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (mfaCode: string) => authClient.enableMFA({ mfaCode }),
    onSuccess: () => {
      toast.success("Two-factor authentication enabled")
      queryClient.invalidateQueries({ queryKey: ["current-user"] })
      options.onSuccess?.()
    },
    onError: (error: Error) => {
      toast.error(`Failed to enable MFA: ${error.message}`)
    },
  })
}

export function useDisableMFA(options: MFAOptions = {}) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (mfaCode: string) => authClient.disableMFA({ mfaCode }),
    onSuccess: () => {
      toast.success("Two-factor authentication disabled")
      queryClient.invalidateQueries({ queryKey: ["current-user"] })
      options.onSuccess?.()
    },
    onError: (error: Error) => {
      toast.error(`Failed to disable MFA: ${error.message}`)
    },
  })
}

// ---------------------------------------------------------------------------
// API key management
// ---------------------------------------------------------------------------

interface APIKeyOptions {
  onSuccess?: () => void
}

export function useCreateAPIKey(options: APIKeyOptions = {}) {
  const queryClient = useQueryClient()
  return useMutation<
    CreateAPIKeyResponse,
    Error,
    { name: string; permissions?: string[]; expiresAt?: string }
  >({
    mutationFn: ({ name, permissions, expiresAt }) =>
      authClient.createAPIKey({
        name,
        permissions: permissions ?? [],
        expiresAt: expiresAt ?? "",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["api-keys"] })
      options.onSuccess?.()
    },
    onError: (error: Error) => {
      toast.error(`Failed to create API key: ${error.message}`)
    },
  })
}

export function useRevokeAPIKey(options: APIKeyOptions = {}) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => authClient.revokeAPIKey({ id }),
    onSuccess: () => {
      toast.success("API key revoked")
      queryClient.invalidateQueries({ queryKey: ["api-keys"] })
      options.onSuccess?.()
    },
    onError: (error: Error) => {
      toast.error(`Failed to revoke API key: ${error.message}`)
    },
  })
}

// ---------------------------------------------------------------------------
// Session management
// ---------------------------------------------------------------------------

interface SessionOptions {
  onSuccess?: () => void
}

export function useRevokeSession(options: SessionOptions = {}) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (sessionId: string) => authClient.revokeSession({ sessionId }),
    onSuccess: () => {
      toast.success("Session revoked")
      queryClient.invalidateQueries({ queryKey: ["sessions"] })
      options.onSuccess?.()
    },
    onError: (error: Error) => {
      toast.error(`Failed to revoke session: ${error.message}`)
    },
  })
}

export function useRevokeOtherSessions(options: SessionOptions = {}) {
  const queryClient = useQueryClient()
  return useMutation<RevokeOtherSessionsResponse, Error, void>({
    mutationFn: () => authClient.revokeOtherSessions({}),
    onSuccess: (res) => {
      toast.success(`${res.revokedCount} other session(s) revoked`)
      queryClient.invalidateQueries({ queryKey: ["sessions"] })
      options.onSuccess?.()
    },
    onError: (error: Error) => {
      toast.error(`Failed to revoke other sessions: ${error.message}`)
    },
  })
}
