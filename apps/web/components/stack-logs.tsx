"use client"

import { useEffect, useRef, useState } from "react"
import { Terminal } from "@xterm/xterm"
import { FitAddon } from "@xterm/addon-fit"
import { AlertCircle, RefreshCw } from "lucide-react"
import { useStackLogsToken } from "@/lib/api/queries"

import "@xterm/xterm/css/xterm.css"

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"

interface StackLogsProps {
  stackId: string
}

function LogsTerminal({ websocketUrl }: { websocketUrl: string }) {
  const terminalRef = useRef<HTMLDivElement>(null)
  const terminalInstance = useRef<Terminal | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [isConnected, setIsConnected] = useState(false)

  useEffect(() => {
    if (!terminalRef.current) return

    const term = new Terminal({
      cursorBlink: false,
      fontSize: 13,
      fontFamily: '"Cascadia Code", "Fira Code", Consolas, "Courier New", monospace',
      theme: {
        background: "#0a0a0a",
        foreground: "#d4d4d4",
        cursor: "#d4d4d4",
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
      disableStdin: true,
    })

    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(terminalRef.current)
    fitAddon.fit()
    terminalInstance.current = term

    const ws = new WebSocket(websocketUrl)

    ws.onopen = () => {
      setIsConnected(true)
      setError(null)
      wsRef.current = ws
    }

    ws.onmessage = (event) => {
      term.write(typeof event.data === "string" ? event.data : new Uint8Array(event.data))
    }

    ws.onclose = () => setIsConnected(false)

    ws.onerror = () => setError("Failed to connect to log stream.")

    const handleResize = () => fitAddon.fit()
    window.addEventListener("resize", handleResize)

    return () => {
      window.removeEventListener("resize", handleResize)
      ws.close()
      term.dispose()
    }
  }, [websocketUrl])

  if (error) {
    return (
      <div className="flex items-center justify-center h-full bg-[#0a0a0a] text-white">
        <div className="text-center p-6">
          <AlertCircle className="size-12 text-red-500 mx-auto mb-4" />
          <p className="text-sm text-gray-400">{error}</p>
        </div>
      </div>
    )
  }

  if (!isConnected) {
    return (
      <div className="flex items-center justify-center h-full bg-[#0a0a0a] text-white">
        <RefreshCw className="size-8 animate-spin" />
      </div>
    )
  }

  return <div ref={terminalRef} className="w-full h-full" />
}

export function StackLogs({ stackId }: StackLogsProps) {
  const [wsUrl, setWsUrl] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const getToken = useStackLogsToken()

  useEffect(() => {
    setWsUrl(null)
    setError(null)

    getToken.mutate(
      { stackId },
      {
        onSuccess: (res) => {
          const wsBase = API_BASE_URL.replace(/^http/, "ws")
          setWsUrl(`${wsBase}/ws/stack-logs?token=${res.token}`)
        },
        onError: (err) => {
          setError(err.message)
        },
      }
    )
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [stackId])

  if (error) {
    return (
      <div className="flex items-center justify-center h-64 bg-[#0a0a0a] rounded-lg text-white">
        <div className="text-center p-6">
          <AlertCircle className="size-12 text-red-500 mx-auto mb-4" />
          <p className="text-sm text-gray-400">{error}</p>
        </div>
      </div>
    )
  }

  if (!wsUrl) {
    return (
      <div className="flex items-center justify-center h-64 bg-[#0a0a0a] rounded-lg text-white">
        <RefreshCw className="size-8 animate-spin" />
      </div>
    )
  }

  return (
    <div className="bg-[#0a0a0a] rounded-lg overflow-hidden border border-border" style={{ height: 480 }}>
      <LogsTerminal websocketUrl={wsUrl} />
    </div>
  )
}
