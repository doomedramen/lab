"use client"

import { useState } from "react"
import { MoreVertical, Trash2, RefreshCw, Network, Plus, Activity, Shield } from "lucide-react"
import { Button } from "@workspace/ui/components/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from "@workspace/ui/components/dropdown-menu"
import { Badge } from "@workspace/ui/components/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@workspace/ui/components/card"
import { CreateNetworkModal } from "./create-network-modal"
import { useNetworkMutations } from "@/lib/api/mutations/network"
import type { VirtualNetwork } from "@/lib/gen/lab/v1/network_pb"

interface NetworkListProps {
  networks?: VirtualNetwork[]
  isLoading?: boolean
}

export function NetworkList({ networks, isLoading }: NetworkListProps) {
  const { deleteNetwork, isDeleting } = useNetworkMutations()

  const getTypeBadge = (type: number) => {
    const typeMap: Record<number, string> = {
      0: "Unspecified",
      1: "Bridge",
      2: "VLAN",
      3: "VXLAN",
      4: "OVS",
      5: "MACVLAN",
      6: "IPVLAN",
    }
    return <Badge variant="outline">{typeMap[type] ?? "Unknown"}</Badge>
  }

  const getStatusBadge = (status: number) => {
    const statusMap: Record<number, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
      0: { label: "Unknown", variant: "secondary" },
      1: { label: "Active", variant: "default" },
      2: { label: "Inactive", variant: "secondary" },
      3: { label: "Error", variant: "destructive" },
    }
    const config = statusMap[status] ?? { label: "Unknown", variant: "secondary" as const }
    return <Badge variant={config.variant}>{config.label}</Badge>
  }

  const handleDelete = (networkId: string, interfaceCount: number) => {
    const message = interfaceCount > 0
      ? `This network has ${interfaceCount} interface(s). Deleting will remove all interfaces. Are you sure?`
      : "Are you sure you want to delete this network? This action cannot be undone."
    
    if (confirm(message)) {
      deleteNetwork.mutate({ id: networkId, force: true })
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <RefreshCw className="w-6 h-6 animate-spin text-muted-foreground" />
        <span className="ml-2 text-muted-foreground">Loading networks...</span>
      </div>
    )
  }

  if (!networks || networks.length === 0) {
    return (
      <div className="text-center py-8">
        <Network className="w-12 h-12 mx-auto text-muted-foreground/50" />
        <h3 className="mt-4 text-lg font-medium">No networks yet</h3>
        <p className="mt-2 text-muted-foreground">
          Create a virtual network for VM and container connectivity
        </p>
        <div className="mt-4">
          <CreateNetworkModal />
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <CreateNetworkModal />
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {networks.map((network) => (
          <Card key={network.id}>
            <CardHeader className="pb-2">
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-2">
                  {getTypeBadge(network.type)}
                  {getStatusBadge(network.status)}
                </div>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                      <MoreVertical className="w-4 h-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem>
                      <RefreshCw className="w-4 h-4 mr-2" />
                      Refresh
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      onClick={() => handleDelete(network.id, network.interfaceCount)}
                      className="text-destructive"
                      disabled={isDeleting}
                    >
                      <Trash2 className="w-4 h-4 mr-2" />
                      Delete
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
              <CardTitle className="text-lg mt-2">{network.name}</CardTitle>
            </CardHeader>
            <CardContent>
              {network.description && (
                <p className="text-sm text-muted-foreground mb-3">{network.description}</p>
              )}

              <div className="grid grid-cols-2 gap-2 text-sm mb-3">
                <div>
                  <p className="text-muted-foreground">Bridge</p>
                  <p className="font-medium">{network.bridgeName || "N/A"}</p>
                </div>
                <div>
                  <p className="text-muted-foreground">VLAN ID</p>
                  <p className="font-medium">{network.vlanId || "None"}</p>
                </div>
                <div>
                  <p className="text-muted-foreground">Subnet</p>
                  <p className="font-medium">{network.subnet || "N/A"}</p>
                </div>
                <div>
                  <p className="text-muted-foreground">Gateway</p>
                  <p className="font-medium">{network.gateway || "N/A"}</p>
                </div>
              </div>

              <div className="flex items-center justify-between pt-3 border-t">
                <div className="flex items-center gap-2 text-sm">
                  <Activity className="w-4 h-4 text-muted-foreground" />
                  <span>{network.interfaceCount} interfaces</span>
                </div>
                {network.dhcpEnabled && (
                  <Badge variant="secondary" className="gap-1">
                    <Shield className="w-3 h-3" />
                    DHCP
                  </Badge>
                )}
              </div>

              <div className="mt-2 text-xs text-muted-foreground">
                MTU: {network.mtu} | {network.isolated ? "Isolated" : "External access"}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
