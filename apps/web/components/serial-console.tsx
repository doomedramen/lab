"use client";

import { useEffect, useRef, useState } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { AttachAddon } from "@xterm/addon-attach";
import { Button } from "@/components/ui/button";
import { AlertCircle, RefreshCw } from "lucide-react";

import "@xterm/xterm/css/xterm.css";

interface SerialConsoleProps {
  websocketUrl: string;
}

export default function SerialConsole({ websocketUrl }: SerialConsoleProps) {
  const terminalRef = useRef<HTMLDivElement>(null);
  const terminalInstance = useRef<Terminal | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState("Connecting...");

  useEffect(() => {
    if (!terminalRef.current) return;

    // Initialize terminal
    const term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily:
        '"Cascadia Code", "Fira Code", Consolas, "Courier New", monospace',
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
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(terminalRef.current);
    fitAddon.fit();

    terminalInstance.current = term;

    // Connect to WebSocket
    const connectWebSocket = () => {
      try {
        const ws = new WebSocket(websocketUrl);
        ws.binaryType = "arraybuffer";

        ws.onopen = () => {
          setIsConnected(true);
          setConnectionStatus("Connected");
          setError(null);

          // Attach terminal to WebSocket
          const attachAddon = new AttachAddon(ws);
          term.loadAddon(attachAddon);
          wsRef.current = ws;
        };

        ws.onclose = () => {
          setIsConnected(false);
          if (connectionStatus === "Connected") {
            setConnectionStatus("Disconnected");
          }
        };

        ws.onerror = () => {
          setError(
            "Failed to connect to serial console. The VM may not have serial console enabled.",
          );
          setConnectionStatus("Connection failed");
        };

        ws.onmessage = (event) => {
          // Data is handled by AttachAddon
          if (typeof event.data === "string") {
            term.write(event.data);
          }
        };
      } catch (err) {
        setError("Failed to create WebSocket connection");
        setConnectionStatus("Connection failed");
      }
    };

    connectWebSocket();

    // Handle resize
    const handleResize = () => fitAddon.fit();
    window.addEventListener("resize", handleResize);

    // Cleanup
    return () => {
      window.removeEventListener("resize", handleResize);
      if (wsRef.current) {
        wsRef.current.close();
      }
      if (terminalInstance.current) {
        terminalInstance.current.dispose();
      }
    };
  }, [websocketUrl]);

  const handleRetry = () => {
    setError(null);
    setConnectionStatus("Connecting...");
    setIsConnected(false);

    // Reload the component by forcing a re-render
    window.location.reload();
  };

  return (
    <div className="relative w-full h-full bg-black flex flex-col">
      {error && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/90 text-white z-10">
          <div className="text-center p-6 max-w-md">
            <AlertCircle className="size-12 text-red-500 mx-auto mb-4" />
            <h3 className="text-lg font-semibold mb-2">Connection Error</h3>
            <p className="text-sm text-gray-400 mb-4">{error}</p>
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
            <p className="text-xs text-gray-500 mt-2 font-mono">
              {websocketUrl}
            </p>
          </div>
        </div>
      )}

      {/* Terminal header */}
      <div className="flex items-center justify-between px-4 py-2 bg-zinc-900 border-b border-zinc-800">
        <div className="text-xs text-zinc-400 font-mono">Serial Console</div>
      </div>

      {/* Terminal container */}
      <div ref={terminalRef} className="flex-1 overflow-hidden" />
    </div>
  );
}
