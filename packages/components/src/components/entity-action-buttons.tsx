import { Button } from "@workspace/ui/components/button"
import { ButtonGroup } from "@workspace/ui/components/button-group"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from "@workspace/ui/components/dropdown-menu"
import { Play, Square, RotateCcw, Pause, Terminal, Loader2, PowerOff, PlayCircle, Trash2, ChevronDown, Copy } from "lucide-react"

export type ConsoleType = "serial" | "vnc" | "websockify"

export interface EntityActionButtonsProps {
  status: "running" | "stopped" | "paused" | string
  variant?: "vm" | "container"
  onPlay?: () => void
  onPause?: () => void
  onStop?: () => void
  onShutdown?: () => void
  onResume?: () => void
  onReboot?: () => void
  onConsole?: (type: ConsoleType) => void
  onClone?: () => void
  onDelete?: () => void
  className?: string
  loading?: {
    start?: boolean
    stop?: boolean
    shutdown?: boolean
    pause?: boolean
    resume?: boolean
    reboot?: boolean
    clone?: boolean
    delete?: boolean
    console?: boolean
  }
}

export function EntityActionButtons({
  status,
  variant = "vm",
  onPlay,
  onPause,
  onStop,
  onShutdown,
  onResume,
  onReboot,
  onConsole,
  onClone,
  onDelete,
  className,
  loading,
}: EntityActionButtonsProps) {
  const isRunning = status === "running"
  const isPaused = status === "paused"
  const isStopped = !isRunning && !isPaused

  const handleConsoleClick = () => {
    // Main button always opens Serial console
    onConsole?.("serial")
  }

  const handleConsoleTypeSelect = (type: string) => {
    // Dropdown options open that specific console type
    onConsole?.(type as ConsoleType)
  }

  return (
    <div className={`flex items-center gap-2 flex-wrap ${className || ""}`} data-testid="entity-action-buttons">
      {isRunning && (
        <>
          {variant === "vm" && (
            <Button variant="outline" size="sm" className="gap-1.5" onClick={onPause} disabled={loading?.pause} data-testid="vm-pause-button">
              {loading?.pause ? <Loader2 className="size-3.5 animate-spin" /> : <Pause className="size-3.5" />}
              Pause
            </Button>
          )}
          {variant === "vm" && (
            <Button variant="outline" size="sm" className="gap-1.5" onClick={onShutdown} disabled={loading?.shutdown} data-testid="vm-shutdown-button">
              {loading?.shutdown ? <Loader2 className="size-3.5 animate-spin" /> : <PowerOff className="size-3.5" />}
              Shutdown
            </Button>
          )}
          <Button variant="outline" size="sm" className="gap-1.5" onClick={onStop} disabled={loading?.stop} data-testid="vm-stop-button">
            {loading?.stop ? <Loader2 className="size-3.5 animate-spin" /> : <Square className="size-3.5" />}
            Force Stop
          </Button>
          <Button variant="outline" size="sm" className="gap-1.5" onClick={onReboot} disabled={loading?.reboot} data-testid="vm-reboot-button">
            {loading?.reboot ? <Loader2 className="size-3.5 animate-spin" /> : <RotateCcw className="size-3.5" />}
            Reboot
          </Button>
        </>
      )}

      {isPaused && variant === "vm" && (
        <Button
          size="sm"
          className="gap-1.5 bg-success text-success-foreground hover:bg-success/90"
          onClick={onResume}
          disabled={loading?.resume}
          data-testid="vm-resume-button"
        >
          {loading?.resume ? <Loader2 className="size-3.5 animate-spin" /> : <PlayCircle className="size-3.5" />}
          Resume
        </Button>
      )}

      {isStopped && (
        <Button
          size="sm"
          className="gap-1.5 bg-success text-success-foreground hover:bg-success/90"
          onClick={onPlay}
          disabled={loading?.start}
          data-testid="vm-start-button"
        >
          {loading?.start ? <Loader2 className="size-3.5 animate-spin" /> : <Play className="size-3.5" />}
          Start
        </Button>
      )}

      {/* Console split button with dropdown */}
      <ButtonGroup className="gap-0">
        <Button
          variant="outline"
          size="sm"
          className="gap-1.5 rounded-r-none border-r-0"
          onClick={handleConsoleClick}
          disabled={!isRunning || loading?.console}
          data-testid="vm-console-button"
        >
          {loading?.console ? <Loader2 className="size-3.5 animate-spin" /> : <Terminal className="size-3.5" />}
          Console
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="outline"
              size="sm"
              className="rounded-l-none px-2"
              disabled={!isRunning || loading?.console}
              data-testid="vm-console-dropdown-button"
            >
              <ChevronDown className="size-3.5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-48">
            <DropdownMenuItem onClick={() => handleConsoleTypeSelect("serial")}>
              <Terminal className="size-4 mr-2" />
              Serial Console
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => handleConsoleTypeSelect("vnc")}>
              <Terminal className="size-4 mr-2" />
              noVNC (WebSocket)
            </DropdownMenuItem>
            <DropdownMenuItem disabled onClick={() => handleConsoleTypeSelect("websockify")}>
              <Terminal className="size-4 mr-2" />
              websockify
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <div className="px-2 py-1.5 text-xs text-muted-foreground">
              <p className="font-medium">Console Types:</p>
              <ul className="mt-1 space-y-0.5 list-disc list-inside">
                <li>Serial: Text-based, always available</li>
                <li>noVNC: GUI access via VNC</li>
              </ul>
            </div>
          </DropdownMenuContent>
        </DropdownMenu>
      </ButtonGroup>

      {/* Clone button - only for VMs */}
      {variant === "vm" && onClone && (
        <Button
          variant="outline"
          size="sm"
          className="gap-1.5"
          onClick={onClone}
          disabled={loading?.clone}
          data-testid="vm-clone-button"
        >
          {loading?.clone ? <Loader2 className="size-3.5 animate-spin" /> : <Copy className="size-3.5" />}
          Clone
        </Button>
      )}

      <Button
        variant="outline"
        size="sm"
        className="gap-1.5 text-destructive hover:text-destructive hover:bg-destructive/10"
        onClick={onDelete}
        disabled={isRunning || isPaused || loading?.delete}
        data-testid="vm-delete-button"
      >
        {loading?.delete ? <Loader2 className="size-3.5 animate-spin" /> : <Trash2 className="size-3.5" />}
        Delete
      </Button>
    </div>
  )
}
