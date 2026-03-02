"use client"

import { useState } from "react"
import { MoreVertical, Trash2, Shield, Plus, CheckCircle2, XCircle } from "lucide-react"
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
import { CreateFirewallRuleModal } from "./create-firewall-rule-modal"
import { useFirewallMutations } from "@/lib/api/mutations/network"
import type { FirewallRule } from "@/lib/gen/lab/v1/network_pb"

interface FirewallRulesListProps {
  rules?: FirewallRule[]
  isLoading?: boolean
}

export function FirewallRulesList({ rules, isLoading }: FirewallRulesListProps) {
  const { deleteRule, enableRule, disableRule, isDeleting } = useFirewallMutations()

  const getActionBadge = (action: number) => {
    const actionMap: Record<number, { label: string; variant: "default" | "destructive" | "secondary" | "outline" }> = {
      0: { label: "Unknown", variant: "secondary" },
      1: { label: "Accept", variant: "default" },
      2: { label: "Drop", variant: "destructive" },
      3: { label: "Reject", variant: "destructive" },
      4: { label: "Log", variant: "outline" },
    }
    const config = actionMap[action] ?? { label: "Unknown", variant: "secondary" as const }
    return <Badge variant={config.variant}>{config.label}</Badge>
  }

  const getDirectionBadge = (direction: number) => {
    const directionMap: Record<number, string> = {
      0: "Any",
      1: "Inbound",
      2: "Outbound",
      3: "Both",
    }
    return <Badge variant="outline">{directionMap[direction] ?? "Unknown"}</Badge>
  }

  const handleDelete = (ruleId: string) => {
    if (confirm("Are you sure you want to delete this firewall rule?")) {
      deleteRule.mutate(ruleId)
    }
  }

  const handleToggle = (rule: FirewallRule) => {
    if (rule.enabled) {
      disableRule.mutate(rule.id)
    } else {
      enableRule.mutate(rule.id)
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Shield className="w-6 h-6 animate-spin text-muted-foreground" />
        <span className="ml-2 text-muted-foreground">Loading firewall rules...</span>
      </div>
    )
  }

  if (!rules || rules.length === 0) {
    return (
      <div className="text-center py-8">
        <Shield className="w-12 h-12 mx-auto text-muted-foreground/50" />
        <h3 className="mt-4 text-lg font-medium">No firewall rules yet</h3>
        <p className="mt-2 text-muted-foreground">
          Create firewall rules to control network traffic
        </p>
        <div className="mt-4">
          <CreateFirewallRuleModal />
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <CreateFirewallRuleModal />
      </div>

      <div className="rounded-md border">
        <div className="grid grid-cols-12 gap-4 p-4 border-b bg-muted/50 text-sm font-medium">
          <div className="col-span-1">Priority</div>
          <div className="col-span-2">Name</div>
          <div className="col-span-1">Action</div>
          <div className="col-span-1">Direction</div>
          <div className="col-span-2">Source</div>
          <div className="col-span-2">Destination</div>
          <div className="col-span-1">Protocol</div>
          <div className="col-span-1">Status</div>
          <div className="col-span-1"></div>
        </div>

        <div className="divide-y">
          {rules.map((rule) => (
            <div key={rule.id} className="grid grid-cols-12 gap-4 p-4 items-center text-sm">
              <div className="col-span-1 font-mono">{rule.priority}</div>
              <div className="col-span-2 font-medium">{rule.name}</div>
              <div className="col-span-1">{getActionBadge(rule.action)}</div>
              <div className="col-span-1">{getDirectionBadge(rule.direction)}</div>
              <div className="col-span-2 truncate" title={rule.sourceCidr || "Any"}>
                {rule.sourceCidr || "Any"}
                {rule.sourcePort && <span className="text-muted-foreground">:{rule.sourcePort}</span>}
              </div>
              <div className="col-span-2 truncate" title={rule.destCidr || "Any"}>
                {rule.destCidr || "Any"}
                {rule.destPort && <span className="text-muted-foreground">:{rule.destPort}</span>}
              </div>
              <div className="col-span-1 uppercase">{rule.protocol || "Any"}</div>
              <div className="col-span-1">
                {rule.enabled ? (
                  <Badge variant="default" className="gap-1">
                    <CheckCircle2 className="w-3 h-3" />
                    Active
                  </Badge>
                ) : (
                  <Badge variant="secondary" className="gap-1">
                    <XCircle className="w-3 h-3" />
                    Disabled
                  </Badge>
                )}
              </div>
              <div className="col-span-1 flex justify-end">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                      <MoreVertical className="w-4 h-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem onClick={() => handleToggle(rule)}>
                      {rule.enabled ? (
                        <>
                          <XCircle className="w-4 h-4 mr-2" />
                          Disable
                        </>
                      ) : (
                        <>
                          <CheckCircle2 className="w-4 h-4 mr-2" />
                          Enable
                        </>
                      )}
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      onClick={() => handleDelete(rule.id)}
                      className="text-destructive"
                      disabled={isDeleting}
                    >
                      <Trash2 className="w-4 h-4 mr-2" />
                      Delete
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
