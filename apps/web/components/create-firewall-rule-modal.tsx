"use client"

import { useState } from "react"
import { Plus, Loader2, Shield } from "lucide-react"
import { Button } from "@workspace/ui/components/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@workspace/ui/components/dialog"
import { Input } from "@workspace/ui/components/input"
import { Label } from "@workspace/ui/components/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@workspace/ui/components/select"
import { Textarea } from "@workspace/ui/components/textarea"
import { Switch } from "@workspace/ui/components/switch"
import { Card, CardContent, CardHeader, CardTitle } from "@workspace/ui/components/card"
import { useFirewallMutations } from "@/lib/api/mutations/network"
import type { FirewallAction, FirewallDirection } from "@/lib/gen/lab/v1/network_pb"

interface CreateFirewallRuleModalProps {
  trigger?: React.ReactNode
  onSuccess?: () => void
}

export function CreateFirewallRuleModal({ trigger, onSuccess }: CreateFirewallRuleModalProps) {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState("")
  const [priority, setPriority] = useState("100")
  const [action, setAction] = useState<FirewallAction>(1) // ACCEPT
  const [direction, setDirection] = useState<FirewallDirection>(3) // BOTH
  const [sourceCidr, setSourceCidr] = useState("")
  const [destCidr, setDestCidr] = useState("")
  const [protocol, setProtocol] = useState("")
  const [sourcePort, setSourcePort] = useState("")
  const [destPort, setDestPort] = useState("")
  const [log, setLog] = useState(false)
  const [description, setDescription] = useState("")

  const { createRule, isCreating } = useFirewallMutations({
    onCreateSuccess: () => {
      setOpen(false)
      resetForm()
      onSuccess?.()
    },
  })

  const resetForm = () => {
    setName("")
    setPriority("100")
    setAction(1)
    setDirection(3)
    setSourceCidr("")
    setDestCidr("")
    setProtocol("")
    setSourcePort("")
    setDestPort("")
    setLog(false)
    setDescription("")
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    if (!name.trim()) {
      return
    }

    createRule.mutate({
      name: name.trim(),
      priority: parseInt(priority, 10) || 100,
      action,
      direction,
      sourceCidr: sourceCidr.trim() || "",
      destCidr: destCidr.trim() || "",
      protocol: protocol.toLowerCase() || "any",
      sourcePort: sourcePort.trim() || "",
      destPort: destPort.trim() || "",
      log,
      description: description.trim(),
    })
  }

  const getActionLabel = (a: FirewallAction) => {
    const labels: Record<number, string> = {
      0: "Unknown",
      1: "Accept",
      2: "Drop",
      3: "Reject",
      4: "Log",
    }
    return labels[a] || "Unknown"
  }

  const getDirectionLabel = (d: FirewallDirection) => {
    const labels: Record<number, string> = {
      0: "Any",
      1: "Inbound",
      2: "Outbound",
      3: "Both",
    }
    return labels[d] || "Unknown"
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger || (
          <Button size="sm">
            <Plus className="w-4 h-4 mr-2" />
            Add Rule
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-y-auto">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Create Firewall Rule</DialogTitle>
            <DialogDescription>
              Define a firewall rule to control network traffic.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label htmlFor="rule-name">Name</Label>
                <Input
                  id="rule-name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="e.g., Allow SSH"
                  required
                />
              </div>

              <div className="grid gap-2">
                <Label htmlFor="rule-priority">Priority</Label>
                <Input
                  id="rule-priority"
                  type="number"
                  value={priority}
                  onChange={(e) => setPriority(e.target.value)}
                  min="1"
                  max="9999"
                />
                <p className="text-xs text-muted-foreground">Lower = higher priority</p>
              </div>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="rule-description">Description (optional)</Label>
              <Textarea
                id="rule-description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Describe this rule..."
                rows={2}
              />
            </div>

            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Rule Configuration</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="grid gap-2">
                    <Label htmlFor="rule-action">Action</Label>
                    <Select value={String(action)} onValueChange={(v) => setAction(Number(v) as FirewallAction)}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="1">Accept</SelectItem>
                        <SelectItem value="2">Drop</SelectItem>
                        <SelectItem value="3">Reject</SelectItem>
                        <SelectItem value="4">Log</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="grid gap-2">
                    <Label htmlFor="rule-direction">Direction</Label>
                    <Select value={String(direction)} onValueChange={(v) => setDirection(Number(v) as FirewallDirection)}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="1">Inbound</SelectItem>
                        <SelectItem value="2">Outbound</SelectItem>
                        <SelectItem value="3">Both</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="grid gap-2">
                    <Label htmlFor="rule-protocol">Protocol</Label>
                    <Select value={protocol || "any"} onValueChange={setProtocol}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="any">Any</SelectItem>
                        <SelectItem value="tcp">TCP</SelectItem>
                        <SelectItem value="udp">UDP</SelectItem>
                        <SelectItem value="icmp">ICMP</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="grid gap-2">
                    <Label htmlFor="rule-interface">Interface (optional)</Label>
                    <Input
                      id="rule-interface"
                      value=""
                      onChange={() => {}}
                      placeholder="e.g., eth0"
                      disabled
                    />
                    <p className="text-xs text-muted-foreground">Not yet implemented</p>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Source & Destination</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="grid gap-2">
                    <Label htmlFor="rule-source">Source CIDR</Label>
                    <Input
                      id="rule-source"
                      value={sourceCidr}
                      onChange={(e) => setSourceCidr(e.target.value)}
                      placeholder="0.0.0.0/0 = Any"
                    />
                  </div>

                  <div className="grid gap-2">
                    <Label htmlFor="rule-source-port">Source Port</Label>
                    <Input
                      id="rule-source-port"
                      value={sourcePort}
                      onChange={(e) => setSourcePort(e.target.value)}
                      placeholder="e.g., 80 or 80:443"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="grid gap-2">
                    <Label htmlFor="rule-dest">Destination CIDR</Label>
                    <Input
                      id="rule-dest"
                      value={destCidr}
                      onChange={(e) => setDestCidr(e.target.value)}
                      placeholder="0.0.0.0/0 = Any"
                    />
                  </div>

                  <div className="grid gap-2">
                    <Label htmlFor="rule-dest-port">Destination Port</Label>
                    <Input
                      id="rule-dest-port"
                      value={destPort}
                      onChange={(e) => setDestPort(e.target.value)}
                      placeholder="e.g., 22"
                    />
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Options</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Shield className="w-4 h-4 text-muted-foreground" />
                    <div>
                      <Label htmlFor="rule-log" className="cursor-pointer">
                        Log Matched Packets
                      </Label>
                      <p className="text-xs text-muted-foreground">
                        Log all packets that match this rule
                      </p>
                    </div>
                  </div>
                  <Switch
                    id="rule-log"
                    checked={log}
                    onCheckedChange={setLog}
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
              {isCreating ? "Creating..." : "Create Rule"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
