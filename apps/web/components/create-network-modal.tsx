"use client"

import { useState } from "react"
import { Plus, Loader2, Network, Shield } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { Switch } from "@/components/ui/switch"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useNetworkMutations } from "@/lib/api/mutations/network"
import type { NetworkType } from "@/lib/gen/lab/v1/common_pb"

interface CreateNetworkModalProps {
  trigger?: React.ReactNode
  onSuccess?: () => void
}

export function CreateNetworkModal({ trigger, onSuccess }: CreateNetworkModalProps) {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState("")
  const [type, setType] = useState<NetworkType>(1) // BRIDGE
  const [bridgeName, setBridgeName] = useState("")
  const [vlanId, setVlanId] = useState("")
  const [subnet, setSubnet] = useState("")
  const [gateway, setGateway] = useState("")
  const [dhcpEnabled, setDhcpEnabled] = useState(false)
  const [dhcpRangeStart, setDhcpRangeStart] = useState("")
  const [dhcpRangeEnd, setDhcpRangeEnd] = useState("")
  const [dnsServers, setDnsServers] = useState("")
  const [isolated, setIsolated] = useState(false)
  const [mtu, setMtu] = useState("1500")
  const [description, setDescription] = useState("")

  const { createNetwork, isCreating } = useNetworkMutations({
    onCreateSuccess: () => {
      setOpen(false)
      resetForm()
      onSuccess?.()
    },
  })

  const resetForm = () => {
    setName("")
    setType(1)
    setBridgeName("")
    setVlanId("")
    setSubnet("")
    setGateway("")
    setDhcpEnabled(false)
    setDhcpRangeStart("")
    setDhcpRangeEnd("")
    setDnsServers("")
    setIsolated(false)
    setMtu("1500")
    setDescription("")
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    if (!name.trim()) {
      return
    }

    createNetwork.mutate({
      name: name.trim(),
      type,
      bridgeName: bridgeName.trim(),
      vlanId: vlanId ? parseInt(vlanId, 10) : 0,
      subnet: subnet.trim(),
      gateway: gateway.trim(),
      dhcpEnabled,
      dhcpRangeStart: dhcpRangeStart.trim(),
      dhcpRangeEnd: dhcpRangeEnd.trim(),
      dnsServers: dnsServers.trim(),
      isolated,
      mtu: parseInt(mtu, 10) || 1500,
      description: description.trim(),
    })
  }

  const getTypeLabel = (t: NetworkType) => {
    const labels: Record<number, string> = {
      0: "Unspecified",
      1: "Linux Bridge",
      2: "VLAN",
      3: "VXLAN",
      4: "Open vSwitch",
      5: "MACVLAN",
      6: "IPVLAN",
    }
    return labels[t] || "Unknown"
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger || (
          <Button size="sm">
            <Plus className="w-4 h-4 mr-2" />
            Add Network
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-y-auto">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Create Virtual Network</DialogTitle>
            <DialogDescription>
              Create a virtual network for VM and container connectivity.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label htmlFor="network-name">Name</Label>
                <Input
                  id="network-name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="e.g., vm-network"
                  required
                />
              </div>

              <div className="grid gap-2">
                <Label htmlFor="network-type">Type</Label>
                <Select value={String(type)} onValueChange={(v) => setType(Number(v) as NetworkType)}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="1">Linux Bridge</SelectItem>
                    <SelectItem value="2">VLAN</SelectItem>
                    <SelectItem value="3">VXLAN</SelectItem>
                    <SelectItem value="4">Open vSwitch</SelectItem>
                    <SelectItem value="5">MACVLAN</SelectItem>
                    <SelectItem value="6">IPVLAN</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="network-description">Description (optional)</Label>
              <Textarea
                id="network-description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Describe this network..."
                rows={2}
              />
            </div>

            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Network Configuration</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-4">
                {type === 1 && (
                  <div className="grid gap-2">
                    <Label htmlFor="network-bridge">Bridge Name</Label>
                    <Input
                      id="network-bridge"
                      value={bridgeName}
                      onChange={(e) => setBridgeName(e.target.value)}
                      placeholder="e.g., vmbr0"
                    />
                  </div>
                )}

                <div className="grid grid-cols-2 gap-4">
                  <div className="grid gap-2">
                    <Label htmlFor="network-vlan">VLAN ID (optional)</Label>
                    <Input
                      id="network-vlan"
                      type="number"
                      value={vlanId}
                      onChange={(e) => setVlanId(e.target.value)}
                      placeholder="0 = no VLAN"
                      min="0"
                      max="4094"
                    />
                  </div>

                  <div className="grid gap-2">
                    <Label htmlFor="network-mtu">MTU</Label>
                    <Input
                      id="network-mtu"
                      type="number"
                      value={mtu}
                      onChange={(e) => setMtu(e.target.value)}
                      min="576"
                      max="9000"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="grid gap-2">
                    <Label htmlFor="network-subnet">Subnet (CIDR)</Label>
                    <Input
                      id="network-subnet"
                      value={subnet}
                      onChange={(e) => setSubnet(e.target.value)}
                      placeholder="e.g., 192.168.1.0/24"
                    />
                  </div>

                  <div className="grid gap-2">
                    <Label htmlFor="network-gateway">Gateway</Label>
                    <Input
                      id="network-gateway"
                      value={gateway}
                      onChange={(e) => setGateway(e.target.value)}
                      placeholder="e.g., 192.168.1.1"
                    />
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-sm">DHCP Configuration</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Shield className="w-4 h-4 text-muted-foreground" />
                    <div>
                      <Label htmlFor="network-dhcp" className="cursor-pointer">
                        Enable DHCP
                      </Label>
                      <p className="text-xs text-muted-foreground">
                        Provide automatic IP addresses to VMs
                      </p>
                    </div>
                  </div>
                  <Switch
                    id="network-dhcp"
                    checked={dhcpEnabled}
                    onCheckedChange={setDhcpEnabled}
                  />
                </div>

                {dhcpEnabled && (
                  <div className="grid grid-cols-2 gap-4">
                    <div className="grid gap-2">
                      <Label htmlFor="dhcp-start">DHCP Range Start</Label>
                      <Input
                        id="dhcp-start"
                        value={dhcpRangeStart}
                        onChange={(e) => setDhcpRangeStart(e.target.value)}
                        placeholder="e.g., 192.168.1.100"
                      />
                    </div>

                    <div className="grid gap-2">
                      <Label htmlFor="dhcp-end">DHCP Range End</Label>
                      <Input
                        id="dhcp-end"
                        value={dhcpRangeEnd}
                        onChange={(e) => setDhcpRangeEnd(e.target.value)}
                        placeholder="e.g., 192.168.1.200"
                      />
                    </div>

                    <div className="grid gap-2 col-span-2">
                      <Label htmlFor="dns-servers">DNS Servers (comma-separated)</Label>
                      <Input
                        id="dns-servers"
                        value={dnsServers}
                        onChange={(e) => setDnsServers(e.target.value)}
                        placeholder="e.g., 8.8.8.8, 8.8.4.4"
                      />
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Security</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="flex items-center justify-between">
                  <div>
                    <Label htmlFor="network-isolated" className="cursor-pointer">
                      Isolated Network
                    </Label>
                    <p className="text-xs text-muted-foreground">
                      No external network access (VMs can only communicate with each other)
                    </p>
                  </div>
                  <Switch
                    id="network-isolated"
                    checked={isolated}
                    onCheckedChange={setIsolated}
                  />
                </div>
              </CardContent>
            </Card>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isCreating || !name.trim()}>
              {isCreating && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
              {isCreating ? "Creating..." : "Create Network"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
