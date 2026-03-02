import { createClient, Transport } from "@connectrpc/connect"
import { create } from "@bufbuild/protobuf"
import { AuthService, RegisterRequestSchema, LoginRequestSchema, CreateAPIKeyRequestSchema } from "../gen/lab/v1/auth_pb"

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"

/**
 * AuthClient provides authentication methods with automatic token management
 */
export class AuthClient {
  private transport: Transport
  private client: ReturnType<typeof createClient<typeof AuthService>>
  private accessToken: string | null = null
  private refreshToken: string | null = null

  constructor(transport?: Transport) {
    const self = this
    this.transport =
      transport ??
      (() => {
        // Lazy import to avoid SSR issues
        const { createConnectTransport } = require("@connectrpc/connect-web")
        return createConnectTransport({
          baseUrl: API_BASE_URL,
          useBinaryFormat: false,
          interceptors: [
            (next: any) => async (req: any) => {
              if (self.accessToken) {
                req.header.set("Authorization", `Bearer ${self.accessToken}`)
              }
              return next(req)
            },
          ],
        })
      })()

    this.client = createClient(AuthService, this.transport)
  }

  /**
   * Set tokens (called after login or token refresh)
   */
  setTokens(accessToken: string, refreshToken: string) {
    this.accessToken = accessToken
    this.refreshToken = refreshToken

    // Store in localStorage for persistence
    if (typeof window !== "undefined") {
      localStorage.setItem("lab_access_token", accessToken)
      localStorage.setItem("lab_refresh_token", refreshToken)
      // Non-httpOnly flag cookie for middleware to read
      document.cookie = "lab_auth=1; path=/; max-age=28800; SameSite=Strict"
    }
  }

  /**
   * Load tokens from localStorage
   */
  loadTokens() {
    if (typeof window === "undefined") {
      return false
    }

    this.accessToken = localStorage.getItem("lab_access_token")
    this.refreshToken = localStorage.getItem("lab_refresh_token")

    // Restore flag cookie if tokens exist but cookie is absent (e.g. after browser restart)
    if (this.accessToken && this.refreshToken) {
      const hasCookie = document.cookie.split(";").some((c) => c.trim().startsWith("lab_auth=1"))
      if (!hasCookie) {
        document.cookie = "lab_auth=1; path=/; max-age=28800; SameSite=Strict"
      }
    }

    return !!(this.accessToken && this.refreshToken)
  }

  /**
   * Clear tokens (called on logout)
   */
  clearTokens() {
    this.accessToken = null
    this.refreshToken = null

    if (typeof window !== "undefined") {
      localStorage.removeItem("lab_access_token")
      localStorage.removeItem("lab_refresh_token")
      document.cookie = "lab_auth=; path=/; max-age=0; SameSite=Strict"
    }
  }

  /**
   * Get the current access token
   */
  getAccessToken(): string | null {
    return this.accessToken
  }

  /**
   * Register a new user
   */
  async register(email: string, password: string) {
    const response = await this.client.register(
      create(RegisterRequestSchema, { email, password })
    )

    if (response.accessToken && response.refreshToken) {
      this.setTokens(response.accessToken, response.refreshToken)
    }

    return response
  }

  /**
   * Login with email and password
   */
  async login(email: string, password: string, mfaCode?: string) {
    const response = await this.client.login(
      create(LoginRequestSchema, { email, password, mfaCode })
    )

    if (response.accessToken && response.refreshToken) {
      this.setTokens(response.accessToken, response.refreshToken)
    }

    return response
  }

  /**
   * Logout the current user
   */
  async logout() {
    try {
      await this.client.logout({})
    } catch (error) {
      // Ignore errors on logout
    } finally {
      this.clearTokens()
    }
  }

  /**
   * Refresh the access token
   */
  async refreshAccessToken(): Promise<boolean> {
    if (!this.refreshToken) {
      return false
    }

    try {
      const response = await this.client.refreshToken({
        refreshToken: this.refreshToken,
      })

      if (response.accessToken && response.refreshToken) {
        this.setTokens(response.accessToken, response.refreshToken)
        return true
      }

      return false
    } catch (error) {
      this.clearTokens()
      return false
    }
  }

  /**
   * Get the current user
   */
  async getCurrentUser() {
    return this.client.getCurrentUser({})
  }

  /**
   * Setup MFA for the current user
   */
  async setupMFA() {
    return this.client.setupMFA({})
  }

  /**
   * Enable MFA for the current user
   */
  async enableMFA(mfaCode: string) {
    return this.client.enableMFA({ mfaCode })
  }

  /**
   * Disable MFA for the current user
   */
  async disableMFA(mfaCode: string) {
    return this.client.disableMFA({ mfaCode })
  }

  /**
   * Create a new API key
   */
  async createAPIKey(name: string, permissions: string[], expiresAt?: string) {
    return this.client.createAPIKey(
      create(CreateAPIKeyRequestSchema, { name, permissions, expiresAt })
    )
  }

  /**
   * List all API keys for the current user
   */
  async listAPIKeys() {
    return this.client.listAPIKeys({})
  }

  /**
   * Revoke an API key
   */
  async revokeAPIKey(id: string) {
    return this.client.revokeAPIKey({ id })
  }

  /**
   * Create an authenticated transport for other services
   */
  createAuthenticatedTransport() {
    const { createConnectTransport } = require("@connectrpc/connect-web")

    return createConnectTransport({
      baseUrl: API_BASE_URL,
      useBinaryFormat: false,
      interceptors: [
        (next: any) => async (req: any) => {
          if (this.accessToken) {
            req.header.set("Authorization", `Bearer ${this.accessToken}`)
          }
          return next(req)
        },
      ],
    })
  }
}

// Export singleton instance
export const authClient = new AuthClient()
// Eagerly restore tokens from localStorage so they're available before the
// first React render — prevents the race where TanStack Query fires a request
// before AuthProvider's useEffect has had a chance to call loadTokens().
// loadTokens() is a no-op on the server (guards with typeof window check).
authClient.loadTokens()
