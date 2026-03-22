"use client";

import { useState, Suspense } from "react";
import {
  User,
  Lock,
  Shield,
  Key,
  Eye,
  EyeOff,
  Trash2,
  Plus,
  QrCode,
  CheckCircle2,
  AlertCircle,
  Monitor,
  LogOut,
} from "lucide-react";
import { PageHeader } from "@/components/page-header";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  TabsPersistent,
  TabsList,
  TabsTrigger,
  TabsContent,
} from "@/components/tabs-persistent";
import { useCurrentUser, useAPIKeys, useSessions } from "@/lib/api/queries";
import {
  useUpdateProfile,
  useSetupMFA,
  useEnableMFA,
  useDisableMFA,
  useCreateAPIKey,
  useRevokeAPIKey,
  useRevokeSession,
  useRevokeOtherSessions,
} from "@/lib/api/mutations";
import type { APIKey, Session } from "@/lib/gen/lab/v1/auth_pb";

function SettingsContent() {
  return (
    <div className="p-6 space-y-6">
      <PageHeader
        backHref="/"
        backLabel="Dashboard"
        title="Account Settings"
        subtitle="Manage your profile, security, and API access"
        icon={<User className="size-5 text-foreground" />}
      />

      <TabsPersistent defaultValue="profile" paramKey="settings-tab">
        <TabsList>
          <TabsTrigger value="profile">
            <User className="size-3.5 mr-1.5" />
            Profile
          </TabsTrigger>
          <TabsTrigger value="mfa">
            <Shield className="size-3.5 mr-1.5" />
            Two-Factor Auth
          </TabsTrigger>
          <TabsTrigger value="api-keys">
            <Key className="size-3.5 mr-1.5" />
            API Keys
          </TabsTrigger>
          <TabsTrigger value="sessions">
            <Monitor className="size-3.5 mr-1.5" />
            Sessions
          </TabsTrigger>
        </TabsList>

        <TabsContent value="profile" className="mt-4">
          <ProfileTab />
        </TabsContent>
        <TabsContent value="mfa" className="mt-4">
          <MFATab />
        </TabsContent>
        <TabsContent value="api-keys" className="mt-4">
          <APIKeysTab />
        </TabsContent>
        <TabsContent value="sessions" className="mt-4">
          <SessionsTab />
        </TabsContent>
      </TabsPersistent>
    </div>
  );
}

export default function SettingsPage() {
  return (
    <Suspense
      fallback={
        <div className="p-6">
          <div className="h-48 animate-pulse rounded-lg bg-muted" />
        </div>
      }
    >
      <SettingsContent />
    </Suspense>
  );
}

// ---------------------------------------------------------------------------
// Profile Tab — change email and password
// ---------------------------------------------------------------------------

function ProfileTab() {
  const { data: user, isLoading } = useCurrentUser();
  const updateProfile = useUpdateProfile();

  const [email, setEmail] = useState("");
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showCurrent, setShowCurrent] = useState(false);
  const [showNew, setShowNew] = useState(false);
  const [passwordError, setPasswordError] = useState("");

  if (isLoading)
    return <div className="h-48 animate-pulse rounded-lg bg-muted" />;

  const handleEmailChange = (e: React.FormEvent) => {
    e.preventDefault();
    if (!email) return;
    updateProfile.mutate({ email });
    setEmail("");
  };

  const handlePasswordChange = (e: React.FormEvent) => {
    e.preventDefault();
    setPasswordError("");
    if (newPassword !== confirmPassword) {
      setPasswordError("New passwords do not match");
      return;
    }
    if (newPassword.length < 8) {
      setPasswordError("Password must be at least 8 characters");
      return;
    }
    updateProfile.mutate(
      { currentPassword, newPassword },
      {
        onSuccess: () => {
          setCurrentPassword("");
          setNewPassword("");
          setConfirmPassword("");
        },
      },
    );
  };

  return (
    <div className="space-y-6">
      {/* Current info */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Account Information</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <div className="flex items-center gap-3">
            <span className="text-muted-foreground w-20">Email</span>
            <span className="font-medium">{user?.email ?? "—"}</span>
          </div>
          <div className="flex items-center gap-3">
            <span className="text-muted-foreground w-20">Role</span>
            <Badge variant="secondary" className="text-xs capitalize">
              {user?.role?.toString().replace("USER_ROLE_", "").toLowerCase() ??
                "—"}
            </Badge>
          </div>
          <div className="flex items-center gap-3">
            <span className="text-muted-foreground w-20">Member since</span>
            <span>
              {user?.createdAt
                ? new Date(user.createdAt).toLocaleDateString()
                : "—"}
            </span>
          </div>
        </CardContent>
      </Card>

      {/* Change email */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Change Email</CardTitle>
          <CardDescription>Update your email address</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleEmailChange} className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="new-email">New email address</Label>
              <Input
                id="new-email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder={user?.email ?? "email@example.com"}
                className="max-w-sm"
              />
            </div>
            <Button type="submit" disabled={!email || updateProfile.isPending}>
              Update email
            </Button>
          </form>
        </CardContent>
      </Card>

      {/* Change password */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Change Password</CardTitle>
          <CardDescription>
            Use a strong password with at least 8 characters
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handlePasswordChange} className="space-y-4 max-w-sm">
            <div className="space-y-1.5">
              <Label htmlFor="current-pw">Current password</Label>
              <div className="relative">
                <Input
                  id="current-pw"
                  type={showCurrent ? "text" : "password"}
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  className="pr-9"
                />
                <button
                  type="button"
                  onClick={() => setShowCurrent((v) => !v)}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                >
                  {showCurrent ? (
                    <EyeOff className="size-3.5" />
                  ) : (
                    <Eye className="size-3.5" />
                  )}
                </button>
              </div>
            </div>

            <Separator />

            <div className="space-y-1.5">
              <Label htmlFor="new-pw">New password</Label>
              <div className="relative">
                <Input
                  id="new-pw"
                  type={showNew ? "text" : "password"}
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  className="pr-9"
                />
                <button
                  type="button"
                  onClick={() => setShowNew((v) => !v)}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                >
                  {showNew ? (
                    <EyeOff className="size-3.5" />
                  ) : (
                    <Eye className="size-3.5" />
                  )}
                </button>
              </div>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="confirm-pw">Confirm new password</Label>
              <Input
                id="confirm-pw"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
              />
            </div>

            {passwordError && (
              <p className="text-xs text-destructive flex items-center gap-1">
                <AlertCircle className="size-3" />
                {passwordError}
              </p>
            )}

            <Button
              type="submit"
              disabled={
                !currentPassword ||
                !newPassword ||
                !confirmPassword ||
                updateProfile.isPending
              }
            >
              Change password
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

// ---------------------------------------------------------------------------
// MFA Tab — setup / enable / disable two-factor auth
// ---------------------------------------------------------------------------

function MFATab() {
  const { data: user, isLoading } = useCurrentUser();
  const setupMFA = useSetupMFA();
  const enableMFA = useEnableMFA();
  const disableMFA = useDisableMFA();

  const [setupData, setSetupData] = useState<{
    secret: string;
    qrCodeUrl: string;
    manualKey: string;
    backupCodes: string[];
  } | null>(null);
  const [verifyCode, setVerifyCode] = useState("");
  const [disableCode, setDisableCode] = useState("");
  const [showDisableDialog, setShowDisableDialog] = useState(false);
  const [showBackupCodes, setShowBackupCodes] = useState(false);

  if (isLoading)
    return <div className="h-48 animate-pulse rounded-lg bg-muted" />;

  const mfaEnabled = user?.mfaEnabled ?? false;

  const handleSetup = () => {
    setupMFA.mutate(undefined, {
      onSuccess: (res) => {
        setSetupData({
          secret: res.secret,
          qrCodeUrl: res.qrCodeUrl,
          manualKey: res.manualKey,
          backupCodes: res.backupCodes,
        });
      },
    });
  };

  const handleEnable = (e: React.FormEvent) => {
    e.preventDefault();
    if (!verifyCode) return;
    enableMFA.mutate(verifyCode, {
      onSuccess: () => {
        setSetupData(null);
        setVerifyCode("");
        setShowBackupCodes(true);
      },
    });
  };

  const handleDisable = () => {
    disableMFA.mutate(disableCode, {
      onSuccess: () => {
        setShowDisableDialog(false);
        setDisableCode("");
      },
    });
  };

  return (
    <div className="space-y-6">
      {/* Status */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Two-Factor Authentication</CardTitle>
          <CardDescription>
            Add an extra layer of security to your account with TOTP
            authenticator apps
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-3">
            {mfaEnabled ? (
              <CheckCircle2 className="size-5 text-success shrink-0" />
            ) : (
              <AlertCircle className="size-5 text-muted-foreground shrink-0" />
            )}
            <div>
              <p className="text-sm font-medium">
                {mfaEnabled
                  ? "Two-factor authentication is enabled"
                  : "Two-factor authentication is disabled"}
              </p>
              <p className="text-xs text-muted-foreground">
                {mfaEnabled
                  ? "Your account is protected with an authenticator app."
                  : "Enable 2FA to secure your account."}
              </p>
            </div>
            <div className="ml-auto">
              {mfaEnabled ? (
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => setShowDisableDialog(true)}
                >
                  Disable 2FA
                </Button>
              ) : (
                <Button
                  size="sm"
                  onClick={handleSetup}
                  disabled={setupMFA.isPending || setupData !== null}
                >
                  <QrCode className="size-3.5 mr-1.5" />
                  Set up 2FA
                </Button>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Setup flow */}
      {setupData && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Scan QR Code</CardTitle>
            <CardDescription>
              Scan this QR code with your authenticator app (Google
              Authenticator, Authy, 1Password, etc.)
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-col items-start gap-4">
              {/* QR code rendered as img from otpauth URL */}
              <div className="bg-white p-3 rounded-lg border">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  src={`https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(setupData.qrCodeUrl)}`}
                  alt="QR code for authenticator app"
                  width={200}
                  height={200}
                />
              </div>
              <div className="text-sm">
                <p className="text-muted-foreground mb-1">
                  Can&apos;t scan? Enter this key manually:
                </p>
                <code className="bg-muted px-2 py-1 rounded text-xs font-mono">
                  {setupData.manualKey}
                </code>
              </div>
            </div>

            <Separator />

            <form onSubmit={handleEnable} className="space-y-3">
              <div className="space-y-1.5">
                <Label htmlFor="verify-code">
                  Enter the 6-digit code from your app to verify
                </Label>
                <Input
                  id="verify-code"
                  type="text"
                  inputMode="numeric"
                  pattern="[0-9]{6}"
                  maxLength={6}
                  value={verifyCode}
                  onChange={(e) =>
                    setVerifyCode(e.target.value.replace(/\D/g, ""))
                  }
                  placeholder="000000"
                  className="max-w-[10rem] tracking-widest text-center font-mono"
                />
              </div>
              <div className="flex gap-2">
                <Button
                  type="submit"
                  disabled={verifyCode.length !== 6 || enableMFA.isPending}
                >
                  Verify and enable
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => {
                    setSetupData(null);
                    setVerifyCode("");
                  }}
                >
                  Cancel
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      {/* Backup codes shown after successful enable */}
      {showBackupCodes && setupData && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Save Your Backup Codes</CardTitle>
            <CardDescription>
              Store these codes somewhere safe. Each can be used once if you
              lose access to your authenticator.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-1.5 font-mono text-sm bg-muted rounded-md p-3">
              {setupData.backupCodes.map((code) => (
                <span key={code}>{code}</span>
              ))}
            </div>
            <Button
              variant="outline"
              size="sm"
              className="mt-3"
              onClick={() => setShowBackupCodes(false)}
            >
              Done
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Disable dialog */}
      <Dialog open={showDisableDialog} onOpenChange={setShowDisableDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Disable Two-Factor Authentication</DialogTitle>
            <DialogDescription>
              Enter your current TOTP code or a backup code to disable 2FA.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3 py-2">
            <div className="space-y-1.5">
              <Label htmlFor="disable-code">Authentication code</Label>
              <Input
                id="disable-code"
                type="text"
                inputMode="numeric"
                value={disableCode}
                onChange={(e) =>
                  setDisableCode(e.target.value.replace(/\D/g, ""))
                }
                placeholder="000000"
                className="max-w-[10rem] tracking-widest text-center font-mono"
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setShowDisableDialog(false);
                setDisableCode("");
              }}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              disabled={disableCode.length < 6 || disableMFA.isPending}
              onClick={handleDisable}
            >
              Disable 2FA
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ---------------------------------------------------------------------------
// API Keys Tab — list, create, revoke
// ---------------------------------------------------------------------------

function APIKeysTab() {
  const { data: keys, isLoading } = useAPIKeys();
  const createKey = useCreateAPIKey();
  const revokeKey = useRevokeAPIKey();

  const [showCreate, setShowCreate] = useState(false);
  const [newKeyName, setNewKeyName] = useState("");
  const [newKeyExpiry, setNewKeyExpiry] = useState("");
  const [createdRawKey, setCreatedRawKey] = useState<string | null>(null);
  const [copiedKey, setCopiedKey] = useState(false);

  if (isLoading)
    return <div className="h-48 animate-pulse rounded-lg bg-muted" />;

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newKeyName) return;
    createKey.mutate(
      { name: newKeyName, expiresAt: newKeyExpiry || undefined },
      {
        onSuccess: (res) => {
          setCreatedRawKey(res.rawKey);
          setNewKeyName("");
          setNewKeyExpiry("");
          setShowCreate(false);
        },
      },
    );
  };

  const copyKey = async () => {
    if (!createdRawKey) return;
    await navigator.clipboard.writeText(createdRawKey);
    setCopiedKey(true);
    setTimeout(() => setCopiedKey(false), 2000);
  };

  return (
    <div className="space-y-6">
      {/* One-time key reveal */}
      {createdRawKey && (
        <Card className="border-success/30 bg-success/5">
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <CheckCircle2 className="size-4 text-success" />
              API Key Created
            </CardTitle>
            <CardDescription>
              Copy this key now — it will never be shown again.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center gap-2">
              <code className="flex-1 bg-muted px-3 py-2 rounded-md text-xs font-mono break-all">
                {createdRawKey}
              </code>
              <Button size="sm" variant="outline" onClick={copyKey}>
                {copiedKey ? "Copied!" : "Copy"}
              </Button>
            </div>
            <Button
              size="sm"
              variant="ghost"
              onClick={() => setCreatedRawKey(null)}
            >
              I&apos;ve saved it
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Create key form */}
      {showCreate ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Create New API Key</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleCreate} className="space-y-4 max-w-sm">
              <div className="space-y-1.5">
                <Label htmlFor="key-name">Key name</Label>
                <Input
                  id="key-name"
                  value={newKeyName}
                  onChange={(e) => setNewKeyName(e.target.value)}
                  placeholder="e.g. Automation script"
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="key-expiry">Expiry date (optional)</Label>
                <Input
                  id="key-expiry"
                  type="date"
                  value={newKeyExpiry}
                  onChange={(e) => setNewKeyExpiry(e.target.value)}
                />
              </div>
              <div className="flex gap-2">
                <Button
                  type="submit"
                  disabled={!newKeyName || createKey.isPending}
                >
                  Create key
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => setShowCreate(false)}
                >
                  Cancel
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      ) : (
        <div className="flex justify-end">
          <Button onClick={() => setShowCreate(true)}>
            <Plus className="size-4 mr-1.5" />
            New API key
          </Button>
        </div>
      )}

      {/* Keys list */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Your API Keys</CardTitle>
          <CardDescription>
            API keys let you authenticate with the Lab API without a password.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {!keys || keys.length === 0 ? (
            <p className="text-sm text-muted-foreground py-4 text-center">
              No API keys yet. Create one to get started.
            </p>
          ) : (
            <div className="divide-y">
              {keys.map((key: APIKey) => (
                <div key={key.id} className="flex items-center gap-3 py-3">
                  <Key className="size-4 text-muted-foreground shrink-0" />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate">{key.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {key.prefix}…
                      {key.lastUsedAt
                        ? ` · Last used ${new Date(key.lastUsedAt).toLocaleDateString()}`
                        : " · Never used"}
                      {key.expiresAt &&
                        ` · Expires ${new Date(key.expiresAt).toLocaleDateString()}`}
                    </p>
                  </div>
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="size-7 text-muted-foreground hover:text-destructive"
                      >
                        <Trash2 className="size-3.5" />
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Revoke API key?</AlertDialogTitle>
                        <AlertDialogDescription>
                          This will immediately revoke &quot;{key.name}&quot;.
                          Any scripts or applications using it will stop
                          working.
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction
                          onClick={() => revokeKey.mutate(key.id)}
                          className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                        >
                          Revoke key
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Sessions Tab — view and manage active sessions
// ---------------------------------------------------------------------------

function SessionsTab() {
  const { data: sessions, isLoading } = useSessions();
  const revokeSession = useRevokeSession();
  const revokeOtherSessions = useRevokeOtherSessions();

  const [showRevokeAllDialog, setShowRevokeAllDialog] = useState(false);
  const [sessionToRevoke, setSessionToRevoke] = useState<Session | null>(null);

  if (isLoading)
    return <div className="h-48 animate-pulse rounded-lg bg-muted" />;

  const currentSession = sessions?.find((s: Session) => s.isCurrent);
  const otherSessions = sessions?.filter((s: Session) => !s.isCurrent) ?? [];
  const otherSessionCount = otherSessions.length;

  const formatDateTime = (dateStr: string) => {
    if (!dateStr) return "Unknown";
    const date = new Date(dateStr);
    return date.toLocaleString(undefined, {
      dateStyle: "medium",
      timeStyle: "short",
    });
  };

  const formatRelativeTime = (dateStr: string) => {
    if (!dateStr) return "Unknown";
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return "Just now";
    if (diffMins < 60)
      return `${diffMins} minute${diffMins !== 1 ? "s" : ""} ago`;
    if (diffHours < 24)
      return `${diffHours} hour${diffHours !== 1 ? "s" : ""} ago`;
    return `${diffDays} day${diffDays !== 1 ? "s" : ""} ago`;
  };

  const handleRevokeSession = (session: Session) => {
    revokeSession.mutate(session.id, {
      onSuccess: () => {
        setSessionToRevoke(null);
      },
    });
  };

  const handleRevokeOtherSessions = () => {
    revokeOtherSessions.mutate(undefined, {
      onSuccess: () => {
        setShowRevokeAllDialog(false);
      },
    });
  };

  const getDeviceIcon = (userAgent: string) => {
    const ua = userAgent.toLowerCase();
    if (
      ua.includes("mobile") ||
      ua.includes("android") ||
      ua.includes("iphone")
    ) {
      return "📱";
    }
    if (ua.includes("tablet") || ua.includes("ipad")) {
      return "📱";
    }
    return "💻";
  };

  const getBrowserInfo = (userAgent: string) => {
    const ua = userAgent.toLowerCase();
    if (ua.includes("firefox")) return "Firefox";
    if (ua.includes("edg")) return "Edge";
    if (ua.includes("chrome")) return "Chrome";
    if (ua.includes("safari")) return "Safari";
    if (ua.includes("opera") || ua.includes("opr")) return "Opera";
    return "Browser";
  };

  const getOSInfo = (userAgent: string) => {
    const ua = userAgent.toLowerCase();
    if (ua.includes("windows")) return "Windows";
    if (ua.includes("mac os")) return "macOS";
    if (ua.includes("linux")) return "Linux";
    if (ua.includes("android")) return "Android";
    if (ua.includes("iphone") || ua.includes("ipad")) return "iOS";
    return "Unknown OS";
  };

  return (
    <div className="space-y-6">
      {/* Current session */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Current Session</CardTitle>
          <CardDescription>
            This is the device you&apos;re currently using
          </CardDescription>
        </CardHeader>
        <CardContent>
          {currentSession ? (
            <div className="flex items-start gap-3 py-2">
              <div className="text-2xl shrink-0">
                {getDeviceIcon(currentSession.userAgent)}
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <p className="text-sm font-medium">
                    {currentSession.deviceName ||
                      `${getBrowserInfo(currentSession.userAgent)} on ${getOSInfo(currentSession.userAgent)}`}
                  </p>
                  <Badge
                    variant="outline"
                    className="text-xs bg-success/10 text-success border-success/20"
                  >
                    Current
                  </Badge>
                </div>
                <p className="text-xs text-muted-foreground mt-1">
                  {currentSession.ipAddress &&
                    `IP: ${currentSession.ipAddress}`}
                  {currentSession.ipAddress &&
                    currentSession.userAgent &&
                    " · "}
                  {getBrowserInfo(currentSession.userAgent)} ·{" "}
                  {getOSInfo(currentSession.userAgent)}
                </p>
                <p className="text-xs text-muted-foreground">
                  Signed in {formatRelativeTime(currentSession.issuedAt)}
                </p>
              </div>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground py-2">
              Unable to load current session information
            </p>
          )}
        </CardContent>
      </Card>

      {/* Other sessions */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-base">Other Sessions</CardTitle>
              <CardDescription>
                Devices where you&apos;re currently logged in
              </CardDescription>
            </div>
            {otherSessionCount > 0 && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowRevokeAllDialog(true)}
                className="text-destructive border-destructive/30 hover:bg-destructive/10"
              >
                <LogOut className="size-3.5 mr-1.5" />
                Sign out all others
              </Button>
            )}
          </div>
        </CardHeader>
        <CardContent>
          {otherSessionCount === 0 ? (
            <p className="text-sm text-muted-foreground py-4 text-center">
              No other active sessions
            </p>
          ) : (
            <div className="divide-y">
              {otherSessions.map((session: Session) => (
                <div key={session.id} className="flex items-start gap-3 py-3">
                  <div className="text-2xl shrink-0">
                    {getDeviceIcon(session.userAgent)}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium">
                      {session.deviceName ||
                        `${getBrowserInfo(session.userAgent)} on ${getOSInfo(session.userAgent)}`}
                    </p>
                    <p className="text-xs text-muted-foreground mt-1">
                      {session.ipAddress && `IP: ${session.ipAddress}`}
                      {session.ipAddress && session.userAgent && " · "}
                      {getBrowserInfo(session.userAgent)} ·{" "}
                      {getOSInfo(session.userAgent)}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      Signed in {formatDateTime(session.issuedAt)}
                      {" · "}
                      Last active {formatRelativeTime(session.lastSeenAt)}
                    </p>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setSessionToRevoke(session)}
                    className="text-muted-foreground hover:text-destructive shrink-0"
                  >
                    <LogOut className="size-3.5 mr-1" />
                    Revoke
                  </Button>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Revoke single session dialog */}
      <AlertDialog
        open={!!sessionToRevoke}
        onOpenChange={(open) => !open && setSessionToRevoke(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Revoke Session?</AlertDialogTitle>
            <AlertDialogDescription>
              {sessionToRevoke && (
                <>
                  This will sign out the device at{" "}
                  {sessionToRevoke.ipAddress || "unknown IP"}. They will need to
                  log in again to access the account.
                </>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setSessionToRevoke(null)}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={() =>
                sessionToRevoke && handleRevokeSession(sessionToRevoke)
              }
              disabled={revokeSession.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Revoke session
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Revoke all other sessions dialog */}
      <AlertDialog
        open={showRevokeAllDialog}
        onOpenChange={setShowRevokeAllDialog}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Sign Out All Other Sessions?</AlertDialogTitle>
            <AlertDialogDescription>
              This will sign out {otherSessionCount} other device
              {otherSessionCount !== 1 ? "s" : ""}. You will stay signed in on
              this device.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleRevokeOtherSessions}
              disabled={revokeOtherSessions.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Sign out all other sessions
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
