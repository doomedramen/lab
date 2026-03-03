"use client"

import Link from "next/link"
import { usePathname, useRouter } from "next/navigation"
import {
  Server,
  Monitor,
  Box,
  Layers,
  LayoutDashboard,
  ChevronDown,
  ChevronRight,
  HardDrive,
  Container,
  Cpu,
  Search,
  Disc,
  LogOut,
  Network,
  Shield,
  Settings,
  ClipboardList,
  X,
  Globe,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { useNodes, useVMs, useContainers } from "@/lib/api/queries"
import { nodeStatusToString, vmStatusToString, containerStatusToString } from "@/lib/api/enum-helpers"
import { ScrollArea, ScrollBar } from "@workspace/ui/components/scroll-area"
import { useState, useCallback } from "react"
import { ThemeToggle } from "@/components/theme-toggle"
import { useAuth } from "@/lib/auth"

const navItems = [
  { label: "Dashboard", href: "/", icon: LayoutDashboard },
  { label: "Hosts", href: "/hosts", icon: Server },
  { label: "Virtual Machines", href: "/vms", icon: Monitor },
  { label: "Containers", href: "/containers", icon: Box },
  { label: "Stacks", href: "/stacks", icon: Layers },
  { label: "ISO Images", href: "/isos", icon: Disc },
  { label: "Storage", href: "/storage", icon: HardDrive },
  { label: "Networks", href: "/networks", icon: Network },
  { label: "Firewall", href: "/firewall", icon: Shield },
  { label: "Proxy", href: "/proxy", icon: Globe },
  { label: "Audit Logs", href: "/audit", icon: ClipboardList },
  { label: "Settings", href: "/settings", icon: Settings },
]

function StatusDot({ status }: { status: string }) {
  const color =
    status === "online" || status === "running"
      ? "bg-success"
      : status === "maintenance" || status === "paused" || status === "frozen" || status === "suspended"
        ? "bg-warning"
        : "bg-muted-foreground"
  return <span className={cn("inline-block size-2 rounded-full shrink-0", color)} />
}

function ResourceTree({ searchQuery = "" }: { searchQuery?: string }) {
  const pathname = usePathname()
  const [expandedNodes, setExpandedNodes] = useState<string[]>([])
  const { data: nodes, isLoading: nodesLoading } = useNodes()
  const { data: vms, isLoading: vmsLoading } = useVMs()
  const { data: containers, isLoading: containersLoading } = useContainers()

  const isLoading = nodesLoading || vmsLoading || containersLoading
  const q = searchQuery.toLowerCase().trim()

  const toggleNode = (name: string) => {
    setExpandedNodes((prev) =>
      prev.includes(name) ? prev.filter((n) => n !== name) : [...prev, name]
    )
  }

  if (isLoading) {
    return (
      <div className="flex flex-col gap-0.5 py-1">
        {templateNodes.map((node) => (
          <div key={node.id} className="space-y-1">
            <div className="h-7 w-32 rounded-md bg-sidebar-accent/60 animate-pulse" />
          </div>
        ))}
      </div>
    )
  }

  if (!nodes || nodes.length === 0) {
    return (
      <div className="px-2 py-1 text-xs text-muted-foreground">
        No nodes found
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-0.5 py-1">
      {nodes.map((node) => {
        const allNodeVms = vms?.filter((v) => v.node === node.name) || []
        const allNodeCts = containers?.filter((c) => c.node === node.name) || []

        // Filter by search query
        const nodeVms = q
          ? allNodeVms.filter(
              (v) =>
                v.name.toLowerCase().includes(q) ||
                String(v.vmid).includes(q)
            )
          : allNodeVms
        const nodeCts = q
          ? allNodeCts.filter(
              (c) =>
                c.name.toLowerCase().includes(q) ||
                String(c.ctid).includes(q)
            )
          : allNodeCts

        // When searching, include a node if its name matches or it has matching children
        const nodeNameMatches = q && node.name.toLowerCase().includes(q)
        const hasMatchingChildren = nodeVms.length > 0 || nodeCts.length > 0
        if (q && !nodeNameMatches && !hasMatchingChildren) {
          return null
        }

        // Auto-expand when searching
        const isExpanded =
          q ? (nodeNameMatches || hasMatchingChildren) : expandedNodes.includes(node.name)

        return (
          <div key={node.id}>
            <button
              onClick={() => !q && toggleNode(node.name)}
              className="flex w-full items-center gap-1.5 rounded-md px-2 py-1 text-xs text-muted-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-foreground transition-colors"
            >
              {isExpanded ? (
                <ChevronDown className="size-3 shrink-0" />
              ) : (
                <ChevronRight className="size-3 shrink-0" />
              )}
              <Server className="size-3 shrink-0" />
              <StatusDot status={nodeStatusToString(node.status)} />
              <span className="whitespace-nowrap">{node.name}</span>
            </button>

            {isExpanded && (
              <div className="ml-4 flex flex-col gap-0.5 py-0.5">
                {nodeVms.map((vm) => (
                  <Link
                    key={vm.id}
                    href={`/vms/${vm.vmid}`}
                    className={cn(
                      "flex items-center gap-1.5 rounded-md px-2 py-0.5 text-[11px] transition-colors",
                      pathname === `/vms/${vm.vmid}`
                        ? "bg-sidebar-accent text-sidebar-accent-foreground"
                        : "text-muted-foreground hover:bg-sidebar-accent/40 hover:text-sidebar-foreground"
                    )}
                  >
                    <HardDrive className="size-3 shrink-0" />
                    <StatusDot status={vmStatusToString(vm.status)} />
                    <span className="whitespace-nowrap">
                      {vm.vmid} ({vm.name})
                    </span>
                  </Link>
                ))}
                {nodeCts.map((ct) => (
                  <Link
                    key={ct.id}
                    href={`/containers/${ct.ctid}`}
                    className={cn(
                      "flex items-center gap-1.5 rounded-md px-2 py-0.5 text-[11px] transition-colors",
                      pathname === `/containers/${ct.ctid}`
                        ? "bg-sidebar-accent text-sidebar-accent-foreground"
                        : "text-muted-foreground hover:bg-sidebar-accent/40 hover:text-sidebar-foreground"
                    )}
                  >
                    <Container className="size-3 shrink-0" />
                    <StatusDot status={containerStatusToString(ct.status)} />
                    <span className="whitespace-nowrap">
                      {ct.ctid} ({ct.name})
                    </span>
                  </Link>
                ))}
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}

export function AppSidebar() {
  const pathname = usePathname()
  const router = useRouter()
  const { logout } = useAuth()
  const [searchQuery, setSearchQuery] = useState("")

  const handleLogout = async () => {
    await logout()
    router.push("/login")
  }

  const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchQuery(e.target.value)
  }, [])

  const clearSearch = useCallback(() => {
    setSearchQuery("")
  }, [])

  return (
    <aside className="flex h-screen w-52 flex-col border-r border-border bg-sidebar text-sidebar-foreground shrink-0">
      {/* Header */}
      <div className="flex items-center gap-2.5 border-b border-sidebar-border px-4 py-3">
        <div className="flex size-8 items-center justify-center rounded-md bg-primary">
          <Cpu className="size-4 text-primary-foreground" />
        </div>
        <div className="flex flex-col">
          <span className="text-sm font-semibold text-sidebar-foreground">Lab</span>
          <span className="text-[11px] text-muted-foreground">Datacenter</span>
        </div>
      </div>

      {/* Search */}
      <div className="px-3 py-2">
        <div className="flex items-center gap-2 rounded-md bg-sidebar-accent px-2.5 py-1.5">
          <Search className="size-3.5 text-muted-foreground shrink-0" />
          <input
            type="text"
            value={searchQuery}
            onChange={handleSearchChange}
            placeholder="Search cluster..."
            className="flex-1 bg-transparent text-xs text-sidebar-foreground placeholder:text-muted-foreground outline-none min-w-0"
            aria-label="Search nodes, VMs, and containers"
          />
          {searchQuery && (
            <button
              onClick={clearSearch}
              className="text-muted-foreground hover:text-sidebar-foreground transition-colors"
              aria-label="Clear search"
            >
              <X className="size-3" />
            </button>
          )}
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex flex-col gap-0.5 px-2 py-1">
        {navItems.map((item) => {
          const isActive = pathname === item.href || (item.href !== "/" && pathname.startsWith(item.href))
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm transition-colors",
                isActive
                  ? "bg-sidebar-accent text-sidebar-accent-foreground font-medium"
                  : "text-muted-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-foreground"
              )}
            >
              <item.icon className="size-4 shrink-0" />
              {item.label}
            </Link>
          )
        })}
      </nav>

      {/* Resource tree */}
      <div className="mt-2 border-t border-sidebar-border px-2 pt-2">
        <span className="px-2.5 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
          Resource Tree
        </span>
      </div>

      <ScrollArea className="flex-1 h-0 px-2 py-1">
        <ResourceTree searchQuery={searchQuery} />
        <ScrollBar orientation="horizontal" />
        <ScrollBar orientation="vertical" />
      </ScrollArea>

      {/* Footer */}
      <div className="border-t border-sidebar-border px-3 py-2.5 text-[11px] text-muted-foreground">
        <div className="flex items-center justify-between">
          <span>Lab v1.0.0</span>
          <div className="flex items-center gap-2">
            <span className="flex items-center gap-1">
              <StatusDot status="online" />
              Cluster OK
            </span>
            <ThemeToggle />
            <button
              onClick={handleLogout}
              data-testid="logout-button"
              className="flex items-center gap-1 rounded p-0.5 hover:text-foreground transition-colors"
              title="Sign out"
            >
              <LogOut className="size-3.5" />
            </button>
          </div>
        </div>
      </div>
    </aside>
  )
}
