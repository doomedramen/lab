"use client"

import { useEffect, useRef, useState } from "react"
import { Terminal } from "@xterm/xterm"
import { FitAddon } from "@xterm/addon-fit"
import { AttachAddon } from "@xterm/addon-attach"
import { Button } from "@workspace/ui/components/button"
import { AlertCircle, RefreshCw, Terminal as TerminalIcon } from "lucide-react"
import { useHostShellToken } from "@/lib/api/queries"

import "@xterm/xterm/css/xterm.css"

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"

interface HostShellProps {
  nodeId: string
  nodeName: string
}

function ShellTerminal({ websocketUrl }: { websocketUrl: string }) {
  const terminalRef = useRef<HTMLDivElement>(null)
  const terminalInstance = useRef<Terminal | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const [connectionStatus, setConnectionStatus] = useState("Connecting...")

  useEffect(() => {
    if (!terminalRef.current) return

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: '"Cascadia Code", "Fira Code", Consolas, "Courier New", monospace',
      theme: {
        background: "#0c0c0c",
        foreground: "#cccccc",
        cursor: "#ffffff",
        cursorAccent: "#000000",
        black: "#0c0c0c",
        red: "#c50f1f",
        green: "#13a10e",
        yellow: "#c19c00",
        blue: "#0037da",
        magenta: "#881798",
        cyan: "#3a96dd",
        white: "#cccccc",
        brightBlack: "#767676",
        brightRed: "#e74856",
        brightGreen: "#16c60c",
        brightYellow: "#f9f1a5",
        brightBlue: "#3b78ff",
        brightMagenta: "#b4009e",
        brightCyan: "#61d6d6",
        brightWhite: "#f2f2f2",
      },
      allowProposedApi: true,
    })

    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(terminalRef.current)
    fitAddon.fit()
    terminalInstance.current = term

    const ws = new WebSocket(websocketUrl)
    ws.binaryType = "arraybuffer"

    ws.onopen = () => {
      setIsConnected(true)
      setConnectionStatus("Connected")
      setError(null)
      const attachAddon = new AttachAddon(ws)
      term.loadAddon(attachAddon)
      wsRef.current = ws
    }

    ws.onclose = () => {
      setIsConnected(false)
      setConnectionStatus("Disconnected")
    }

    ws.onerror = () => {
      setError("Failed to connect to host shell. Please try again.")
      setConnectionStatus("Connection failed")
    }

    // Send resize messages
    const sendResize = () => {
      fitAddon.fit()
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ Width: term.cols, Height: term.rows }))
      }
    }

    const observer = new ResizeObserver(sendResize)
    if (terminalRef.current) observer.observe(terminalRef.current)
    window.addEventListener("resize", sendResize)

    return () => {
      window.removeEventListener("resize", sendResize)
      observer.disconnect()
      ws.close()
      term.dispose()
    }
  }, [websocketUrl])

  if (error) {
    return (
      <div className="flex items-center justify-center h-full bg-[#0c0c0c] text-white">
        <div className="text-center p-6 max-w-md">
          <AlertCircle className="size-12 text-red-500 mx-auto mb-4" />
          <h3 className="text-lg font-semibold mb-2">Connection Error</h3>
          <p className="text-sm text-gray-400">{error}</p>
        </div>
      </div>
    )
  }

  if (!isConnected) {
    return (
      <div className="flex items-center justify-center h-full bg-[#0c0c0c] text-white">
        <div className="text-center">
          <RefreshCw className="size-8 animate-spin mx-auto mb-2" />
          <p className="text-sm">{connectionStatus}</p>
        </div>
      </div>
    )
  }

  return <div ref={terminalRef} className="w-full h-full" />
}

export function HostShell({ nodeId, nodeName }: HostShellProps) {
  const [wsUrl, setWsUrl] = useState<string | null>(null)
  const [tokenError, setTokenError] = useState<string | null>(null)
  const getToken = useHostShellToken()

  useEffect(() => {
    // Request token on mount
    getToken.mutate(nodeId, {
      onSuccess: (res) => {
        const wsBase = API_BASE_URL.replace(/^http/, "ws")
        setWsUrl(`${wsBase}/ws/host-shell?token=${res.token}`)
      },
      onError: (err) => {
        setTokenError(err instanceof Error ? err.message : "Failed to get shell token")
      },
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [nodeId])

  if (tokenError) {
    return (
      <div className="flex items-center justify-center h-[400px] bg-[#0c0c0c] rounded-lg text-white">
        <div className="text-center p-6">
          <AlertCircle className="size-12 text-red-500 mx-auto mb-4" />
          <h3 className="text-lg font-semibold mb-2">Authentication Error</h3>
          <p className="text-sm text-gray-400">{tokenError}</p>
          <Button
            variant="outline"
            size="sm"
            className="mt-4"
            onClick={() => {
              setTokenError(null)
              getToken.mutate(nodeId, {
                onSuccess: (res) => {
                  const wsBase = API_BASE_URL.replace(/^http/, "ws")
                  setWsUrl(`${wsBase}/ws/host-shell?token=${res.token}`)
                },
                onError: (err) => {
                  setTokenError(err instanceof Error ? err.message : "Failed to get shell token")
                },
              })
            }}
          >
            Retry
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <TerminalIcon className="size-4" />
        <span>Shell access to {nodeName}</span>
        {wsUrl && (
          <span className="ml-auto flex items-center gap-1.5">
            <span className="size-2 rounded-full bg-green-500 animate-pulse" />
            Connected
          </span>
        )}
        {!wsUrl && !tokenError && (
          <span className="ml-auto flex items-center gap-1.5">
            <RefreshCw className="size-3 animate-spin" />
            Connecting...
          </span>
        )}
      </div>

      <div className="h-[400px] bg-[#0c0c0c] rounded-lg overflow-hidden border border-border">
        {!tokenError && !wsUrl && (
          <div className="flex items-center justify-center h-full text-white">
            <RefreshCw className="size-8 animate-spin" />
          </div>
        )}
        {!tokenError && wsUrl && <ShellTerminal websocketUrl={wsUrl} />}
      </div>

      <p className="text-xs text-muted-foreground">
        Use this terminal to run commands directly on the host system. Type <code className="px-1 py-0.5 bg-secondary rounded">exit</code> to close the session.
      </p>
    </div>
  )
}
