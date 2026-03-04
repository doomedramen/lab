"use client"

import { useState } from "react"
import { StatusBadge } from "@/components/lab-shared"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Layers, Plus } from "lucide-react"
import Link from "next/link"
import { useStacks } from "@/lib/api/queries"
import { Shimmer } from "@/components/shimmer"
import { ErrorDisplay } from "@/components/error-display"
import type { Stack } from "@/lib/gen/lab/v1/stack_pb"
import { StackStatus } from "@/lib/gen/lab/v1/stack_pb"
import { stackStatusToString } from "@/lib/api/enum-helpers"
import { CreateStackModal } from "@/components/create-stack-modal"

// Template data for shimmer
const templateStacks = [
  {
    id: "stack-1",
    name: "web-app",
    compose: "",
    env: "",
    status: StackStatus.RUNNING,
    containers: [
      { serviceName: "web", containerName: "web-app-web-1", containerId: "abc", image: "nginx", status: "Up 2 hours", state: "running", ports: ["80/tcp"] },
      { serviceName: "db", containerName: "web-app-db-1", containerId: "def", image: "postgres", status: "Up 2 hours", state: "running", ports: [] },
    ],
    createdAt: "2025-06-15T00:00:00Z",
  },
  {
    id: "stack-2",
    name: "monitoring",
    compose: "",
    env: "",
    status: StackStatus.STOPPED,
    containers: [],
    createdAt: "2025-04-20T00:00:00Z",
  },
  {
    id: "stack-3",
    name: "database-cluster",
    compose: "",
    env: "",
    status: StackStatus.PARTIALLY_RUNNING,
    containers: [
      { serviceName: "primary", containerName: "db-primary-1", containerId: "ghi", image: "postgres", status: "Up 1 day", state: "running", ports: ["5432/tcp"] },
      { serviceName: "replica", containerName: "db-replica-1", containerId: "jkl", image: "postgres", status: "Exited (1)", state: "exited", ports: [] },
    ],
    createdAt: "2025-05-10T00:00:00Z",
  },
] as unknown as Stack[]

function StacksContent({ stacks }: { stacks: Stack[] }) {
  const [createOpen, setCreateOpen] = useState(false)
  const running = stacks.filter((s) => s.status === StackStatus.RUNNING).length
  const partial = stacks.filter((s) => s.status === StackStatus.PARTIALLY_RUNNING).length
  const stopped = stacks.filter((s) => s.status === StackStatus.STOPPED).length

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-foreground text-balance" data-testid="stacks-page-title">Stacks</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {stacks.length} total — {running} running, {partial} partial, {stopped} stopped
          </p>
        </div>
        <Button size="sm" className="gap-1.5" onClick={() => setCreateOpen(true)} data-testid="create-stack-button">
          <Plus className="size-4" />
          New Stack
        </Button>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3" data-testid="stacks-list">
        {stacks.map((stack) => {
          const runningContainers = stack.containers.filter((c) => c.state === "running").length
          const totalContainers = stack.containers.length
          return (
            <Link key={stack.id} href={`/stacks/view?id=${stack.id}`} data-testid={`stack-card-${stack.id}`}>
              <Card className="hover:bg-secondary/20 transition-colors cursor-pointer h-full">
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <CardTitle className="flex items-center gap-2 text-sm font-medium">
                      <Layers className="size-4 text-primary" />
                      {stack.name}
                    </CardTitle>
                    <StatusBadge status={stackStatusToString(stack.status)} />
                  </div>
                </CardHeader>
                <CardContent className="space-y-3">
                  {/* Container count */}
                  <div className="flex items-center gap-2 text-xs">
                    <span className="text-muted-foreground">Containers:</span>
                    <span className="text-foreground font-medium">
                      {totalContainers === 0
                        ? "none"
                        : `${runningContainers}/${totalContainers} running`}
                    </span>
                  </div>

                  {/* Container list */}
                  {stack.containers.length > 0 && (
                    <div className="space-y-1">
                      {stack.containers.slice(0, 4).map((container) => (
                        <div key={container.containerName} className="flex items-center gap-2 text-xs">
                          <span
                            className={`inline-block size-1.5 rounded-full shrink-0 ${
                              container.state === "running" ? "bg-success" : "bg-muted-foreground"
                            }`}
                          />
                          <span className="text-muted-foreground truncate">{container.serviceName}</span>
                        </div>
                      ))}
                      {stack.containers.length > 4 && (
                        <p className="text-xs text-muted-foreground">+{stack.containers.length - 4} more</p>
                      )}
                    </div>
                  )}

                  {/* Created date */}
                  <div className="pt-1 border-t border-border text-xs text-muted-foreground">
                    Created {stack.createdAt ? new Date(stack.createdAt).toLocaleDateString() : "—"}
                  </div>
                </CardContent>
              </Card>
            </Link>
          )
        })}
      </div>

      <CreateStackModal open={createOpen} onOpenChange={setCreateOpen} />
    </div>
  )
}

export default function StacksPage() {
  const { data: stacks, isLoading, error, refetch } = useStacks()

  if (error) {
    return (
      <div className="p-6">
        <ErrorDisplay message={error.message} onRetry={() => refetch()} className="h-[50vh]" />
      </div>
    )
  }

  return (
    <Shimmer loading={isLoading} templateProps={{ stacks: templateStacks }}>
      <StacksContent stacks={stacks || templateStacks} />
    </Shimmer>
  )
}
