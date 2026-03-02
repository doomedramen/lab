"use client"

import { useEffect, useRef, useState } from "react"
import { Terminal } from "@xterm/xterm"
import { FitAddon } from "@xterm/addon-fit"
import { AttachAddon } from "@xterm/addon-attach"
import { Button } from "@workspace/ui/components/button"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@workspace/ui/components/dialog"
import { AlertCircle, RefreshCw } from "lucide-react"
import { useContainerToken } from "@/lib/api/queries"

import "@xterm/xterm/css/xterm.css"

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"

interface ContainerBashProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  stackId: string
  containerName: string
}

function BashTerminal({ websocketUrl }: { websocketUrl: string }) {
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
        background: "#000000",
        foreground: "#ffffff",
        cursor: "#ffffff",
        cursorAccent: "#000000",
        black: "#000000",
        red: "#cd3131",
        green: "#0dbc79",
        yellow: "#e5e510",
        blue: "#2472c8",
        magenta: "#bc3fbc",
        cyan: "#11a8cd",
        white: "#e5e5e5",
        brightBlack: "#666666",
        brightRed: "#f14c4c",
        brightGreen: "#23d18b",
        brightYellow: "#f5f543",
        brightBlue: "#3b8eea",
        brightMagenta: "#d670d6",
        brightCyan: "#29b8db",
        brightWhite: "#ffffff",
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
      setError("Failed to connect to container. The container may not be running.")
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
      <div className="flex items-center justify-center h-full bg-black text-white">
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
      <div className="flex items-center justify-center h-full bg-black text-white">
        <div className="text-center">
          <RefreshCw className="size-8 animate-spin mx-auto mb-2" />
          <p className="text-sm">{connectionStatus}</p>
        </div>
      </div>
    )
  }

  return <div ref={terminalRef} className="w-full h-full" />
}

export function ContainerBash({ open, onOpenChange, stackId, containerName }: ContainerBashProps) {
  const [wsUrl, setWsUrl] = useState<string | null>(null)
  const [tokenError, setTokenError] = useState<string | null>(null)
  const getToken = useContainerToken()

  useEffect(() => {
    if (!open) {
      setWsUrl(null)
      setTokenError(null)
      return
    }

    getToken.mutate(
      { stackId, containerName },
      {
        onSuccess: (res) => {
          const wsBase = API_BASE_URL.replace(/^http/, "ws")
          setWsUrl(`${wsBase}/ws/stack-bash?token=${res.token}`)
        },
        onError: (err) => {
          setTokenError(err.message)
        },
      }
    )
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, stackId, containerName])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl p-0 overflow-hidden" style={{ height: "70vh" }}>
        <div className="flex flex-col h-full">
          <DialogHeader className="px-4 py-2 bg-zinc-900 border-b border-zinc-800 shrink-0">
            <DialogTitle className="text-xs font-mono text-zinc-300">
              {containerName} — bash
            </DialogTitle>
          </DialogHeader>

          <div className="flex-1 bg-black overflow-hidden">
            {tokenError && (
              <div className="flex items-center justify-center h-full text-white">
                <div className="text-center p-6">
                  <AlertCircle className="size-12 text-red-500 mx-auto mb-4" />
                  <p className="text-sm text-gray-400">{tokenError}</p>
                  <Button
                    variant="outline"
                    size="sm"
                    className="mt-4"
                    onClick={() => onOpenChange(false)}
                  >
                    Close
                  </Button>
                </div>
              </div>
            )}
            {!tokenError && !wsUrl && (
              <div className="flex items-center justify-center h-full text-white">
                <RefreshCw className="size-8 animate-spin" />
              </div>
            )}
            {!tokenError && wsUrl && <BashTerminal websocketUrl={wsUrl} />}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
