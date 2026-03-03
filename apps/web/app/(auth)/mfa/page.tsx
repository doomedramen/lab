"use client"

import { useState, useEffect } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/lib/auth"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Checkbox } from "@/components/ui/checkbox"

export default function MFASetupPage() {
  const router = useRouter()
  const { setupMFA, enableMFA, user, isAuthenticated } = useAuth()
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)
  const [mfaData, setMfaData] = useState<{
    secret: string
    qrCodeUrl: string
    manualKey: string
    backupCodes: string[]
  } | null>(null)
  const [mfaCode, setMfaCode] = useState("")
  const [hasSavedBackupCodes, setHasSavedBackupCodes] = useState(false)

  // Redirect if not authenticated or MFA already enabled
  useEffect(() => {
    if (!isAuthenticated) {
      router.push("/login")
    } else if (user?.mfaEnabled) {
      router.push("/dashboard")
    }
  }, [isAuthenticated, user, router])

  const handleSetup = async () => {
    setIsLoading(true)
    setError(null)

    try {
      const data = await setupMFA()
      setMfaData(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to setup MFA")
    } finally {
      setIsLoading(false)
    }
  }

  const handleEnable = async () => {
    if (!mfaCode || mfaCode.length !== 6) {
      setError("Please enter a valid 6-digit code")
      return
    }

    if (!hasSavedBackupCodes) {
      setError("Please save your backup codes before enabling MFA")
      return
    }

    setIsLoading(true)
    setError(null)

    try {
      await enableMFA(mfaCode)
      setSuccess(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Invalid code. Please try again.")
    } finally {
      setIsLoading(false)
    }
  }

  const handleDownloadBackupCodes = () => {
    if (!mfaData) return

    const content = `Lab Backup Codes\n================\n\nSave these codes in a secure location. Each code can only be used once.\n\n${mfaData.backupCodes.join("\n")}\n`
    const blob = new Blob([content], { type: "text/plain" })
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = "lab-backup-codes.txt"
    a.click()
    URL.revokeObjectURL(url)
  }

  if (success) {
    return (
      <div className="flex min-h-screen items-center justify-center p-4">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle className="text-2xl font-bold">MFA Enabled!</CardTitle>
            <CardDescription>
              Two-factor authentication has been successfully enabled on your account.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Alert>
              <AlertDescription>
                You will now need to enter a code from your authenticator app when logging in.
              </AlertDescription>
            </Alert>
          </CardContent>
          <CardFooter>
            <Button className="w-full" onClick={() => router.push("/dashboard")}>
              Continue to Dashboard
            </Button>
          </CardFooter>
        </Card>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="text-2xl font-bold">Setup Two-Factor Authentication</CardTitle>
          <CardDescription>
            Protect your account with an authenticator app
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {!mfaData ? (
            <div className="text-center space-y-4">
              <p className="text-muted-foreground">
                Two-factor authentication adds an extra layer of security to your account.
                You will need to enter a code from an authenticator app when logging in.
              </p>
              <Button onClick={handleSetup} disabled={isLoading} className="w-full">
                {isLoading ? "Setting up..." : "Get Started"}
              </Button>
            </div>
          ) : (
            <>
              <div className="space-y-4">
                <div className="text-center">
                  <p className="text-sm font-medium mb-2">
                    Scan this QR code with your authenticator app
                  </p>
                  <div className="bg-white p-4 rounded-lg inline-block">
                    {/* QR Code would be rendered here using a library like qrcode.react */}
                    <div className="w-48 h-48 bg-gray-200 flex items-center justify-center text-gray-500 text-sm">
                      QR Code: {mfaData.qrCodeUrl}
                    </div>
                  </div>
                </div>

                <div className="space-y-2">
                  <Label>Manual Entry Key</Label>
                  <div className="flex gap-2">
                    <Input value={mfaData.manualKey} readOnly className="font-mono" />
                    <Button
                      variant="outline"
                      onClick={() => navigator.clipboard.writeText(mfaData.manualKey)}
                    >
                      Copy
                    </Button>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Enter this key manually if you can&apos;t scan the QR code
                  </p>
                </div>

                <div className="space-y-2">
                  <Label>Backup Codes</Label>
                  <div className="bg-muted p-3 rounded-md font-mono text-sm grid grid-cols-2 gap-2">
                    {mfaData.backupCodes.map((code, i) => (
                      <div key={i}>{code}</div>
                    ))}
                  </div>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      onClick={handleDownloadBackupCodes}
                      className="flex-1"
                    >
                      Download
                    </Button>
                    <Button
                      variant="outline"
                      onClick={() => navigator.clipboard.writeText(mfaData.backupCodes.join("\n"))}
                    >
                      Copy All
                    </Button>
                  </div>
                  <div className="flex items-center space-x-2">
                    <Checkbox
                      id="saved"
                      checked={hasSavedBackupCodes}
                      onCheckedChange={(checked) => setHasSavedBackupCodes(checked as boolean)}
                    />
                    <Label htmlFor="saved" className="text-sm">
                      I have saved these backup codes
                    </Label>
                  </div>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="code">Authentication Code</Label>
                  <Input
                    id="code"
                    type="text"
                    inputMode="numeric"
                    pattern="[0-9]*"
                    maxLength={6}
                    placeholder="123456"
                    value={mfaCode}
                    onChange={(e) => setMfaCode(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">
                    Enter the 6-digit code from your authenticator app
                  </p>
                </div>
              </div>
            </>
          )}
        </CardContent>
        {mfaData && (
          <CardFooter>
            <Button onClick={handleEnable} disabled={isLoading || !hasSavedBackupCodes} className="w-full">
              {isLoading ? "Enabling..." : "Enable Two-Factor Authentication"}
            </Button>
          </CardFooter>
        )}
      </Card>
    </div>
  )
}
