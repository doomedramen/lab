"use client"

import { useState } from "react"
import { VncScreen } from "react-vnc"
import { Button } from "@workspace/ui/components/button"
import { AlertCircle, RefreshCw, Info } from "lucide-react"

interface VNCConsoleProps {
  websocketUrl: string
  onConnect?: () => void
}

// VNCConsole wraps react-vnc's VncScreen component. It must be rendered client-side only
// (use next/dynamic with { ssr: false } at the import site).
export default function VNCConsole({ websocketUrl, onConnect }: VNCConsoleProps) {
  const [error, setError] = useState<string | null>(null)
  const [retryCount, setRetryCount] = useState(0)
  const [isConnected, setIsConnected] = useState(false)
  const [connectionStatus, setConnectionStatus] = useState("Connecting...")

  const handleDisconnect = () => {
    // Don't show error if we were connected (normal disconnect)
    if (isConnected) {
      setConnectionStatus("Disconnected")
      return
    }
    // Only show error if we never connected (connection failed)
    if (retryCount < 3) {
      setRetryCount((prev) => prev + 1)
      setConnectionStatus(`Reconnecting... (attempt ${retryCount + 1}/3)`)
    } else {
      setError("Failed to connect to VNC server. The VM may not be running or VNC is not enabled.")
    }
  }

  const handleConnect = () => {
    setError(null)
    setRetryCount(0)
    setIsConnected(true)
    setConnectionStatus("Connected")
    onConnect?.()
  }

  const handleSecurityFailure = () => {
    setError("Authentication failed. Please try again.")
  }

  const handleRetry = () => {
    setError(null)
    setRetryCount(0)
    setConnectionStatus("Connecting...")
    setIsConnected(false)
    // Force remount by changing key
  }

  return (
    <div className="relative w-full h-full bg-black">
      {error && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/90 text-white z-10">
          <div className="text-center p-6 max-w-md">
            <AlertCircle className="size-12 text-red-500 mx-auto mb-4" />
            <h3 className="text-lg font-semibold mb-2">Connection Error</h3>
            <p className="text-sm text-gray-400 mb-4">{error}</p>
            <div className="text-xs text-gray-500 mb-4 space-y-1">
              <p>Common causes:</p>
              <ul className="text-left list-disc list-inside">
                <li>VM is not running</li>
                <li>VNC is not enabled in VM config</li>
                <li>API server can't reach VNC port</li>
              </ul>
            </div>
            <div className="text-xs text-gray-500 mb-4 font-mono break-all">
              WebSocket: {websocketUrl}
            </div>
            <div className="flex gap-2 justify-center">
              <Button
                variant="outline"
                size="sm"
                onClick={handleRetry}
                className="gap-1"
              >
                <RefreshCw className="size-3" />
                Retry
              </Button>
            </div>
          </div>
        </div>
      )}
      
      {!isConnected && !error && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/90 text-white z-10">
          <div className="text-center">
            <RefreshCw className="size-8 animate-spin mx-auto mb-2" />
            <p className="text-sm">{connectionStatus}</p>
            <p className="text-xs text-gray-500 mt-2 font-mono">{websocketUrl}</p>
          </div>
        </div>
      )}

      <VncScreen
        key={`${websocketUrl}-${retryCount}`}
        url={websocketUrl}
        scaleViewport
        resizeSession
        background="#000000"
        className="w-full h-full"
        style={{ width: "100%", height: "100%" }}
        onConnect={handleConnect}
        onDisconnect={handleDisconnect}
        onSecurityFailure={handleSecurityFailure}
        retryDuration={5000}
      />
    </div>
  )
}
