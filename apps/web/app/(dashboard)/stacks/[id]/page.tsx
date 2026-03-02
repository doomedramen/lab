"use client"

import { use, useState } from "react"
import dynamic from "next/dynamic"
import { notFound, useRouter } from "next/navigation"
import Link from "next/link"
import { StatusBadge } from "@workspace/components/lab-shared"
import { Shimmer } from "@workspace/components/shimmer"
import { Card, CardContent, CardHeader, CardTitle } from "@workspace/ui/components/card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@workspace/ui/components/tabs"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@workspace/ui/components/table"
import { Badge } from "@workspace/ui/components/badge"
import { Button } from "@workspace/ui/components/button"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@workspace/ui/components/alert-dialog"
import { useStack } from "@/lib/api/queries"
import { useStackMutations } from "@/lib/api/mutations"
import { ErrorDisplay } from "@/components/error-display"
import { ContainerBash } from "@/components/container-bash"
import { StackLogs } from "@/components/stack-logs"
import type { Stack } from "@/lib/gen/lab/v1/stack_pb"
import { StackStatus } from "@/lib/gen/lab/v1/stack_pb"
import { stackStatusToString } from "@/lib/api/enum-helpers"
import {
  ArrowLeft,
  Layers,
  Play,
  Square,
  RefreshCw,
  Download,
  Trash2,
  Terminal,
  Loader2,
} from "lucide-react"

const MonacoEditor = dynamic(() => import("@monaco-editor/react"), { ssr: false })

const templateStack = {
  id: "template",
  name: "Loading...",
  compose: "",
  env: "",
  status: StackStatus.STOPPED,
  containers: [],
  createdAt: new Date().toISOString(),
} as unknown as Stack

interface StackDetailContentProps {
  stack: Stack
}

function StackDetailContent({ stack }: StackDetailContentProps) {
  const router = useRouter()
  const { startStack, stopStack, restartStack, updateImages, downStack, deleteStack, updateStack,
    isStarting, isStopping, isRestarting, isUpdatingImages, isDown, isDeleting, isUpdating } = useStackMutations()

  const [bashOpen, setBashOpen] = useState(false)
  const [bashContainer, setBashContainer] = useState<{ id: string; name: string } | null>(null)
  const [confirmDown, setConfirmDown] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [compose, setCompose] = useState(stack.compose)
  const [env, setEnv] = useState(stack.env)

  const openBash = (containerId: string, containerName: string) => {
    setBashContainer({ id: containerId, name: containerName })
    setBashOpen(true)
  }

  const handleDelete = () => {
    deleteStack.mutate(stack.id, {
      onSuccess: () => router.push("/stacks"),
    })
  }

  const handleDown = () => {
    downStack.mutate(stack.id, {
      onSuccess: () => setConfirmDown(false),
    })
  }

  const isAnyActionLoading = isStarting || isStopping || isRestarting || isUpdatingImages || isDown || isDeleting

  return (
    <div className="p-6 space-y-6">
      <Link
        href="/stacks"
        className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
      >
        <ArrowLeft className="size-4" />
        Back to Stacks
      </Link>

      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <div className="flex size-10 items-center justify-center rounded-lg bg-secondary">
            <Layers className="size-5 text-foreground" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-xl font-semibold text-foreground" data-testid="stack-detail-name">{stack.name}</h1>
              <StatusBadge status={stackStatusToString(stack.status)} data-testid="stack-detail-status" />
            </div>
            <p className="text-xs text-muted-foreground font-mono" data-testid="stack-detail-id">{stack.id}</p>
          </div>
        </div>

        <div className="flex items-center gap-2 flex-wrap justify-end">
          {stack.status !== StackStatus.RUNNING && (
            <Button
              size="sm"
              className="gap-1.5 bg-success text-success-foreground hover:bg-success/90"
              onClick={() => startStack.mutate(stack.id)}
              disabled={isAnyActionLoading}
              data-testid="stack-start-button"
            >
              {isStarting ? <Loader2 className="size-3.5 animate-spin" /> : <Play className="size-3.5" />}
              Start
            </Button>
          )}
          {stack.status === StackStatus.RUNNING && (
            <Button
              size="sm"
              variant="outline"
              className="gap-1.5"
              onClick={() => restartStack.mutate(stack.id)}
              disabled={isAnyActionLoading}
              data-testid="stack-restart-button"
            >
              {isRestarting ? <Loader2 className="size-3.5 animate-spin" /> : <RefreshCw className="size-3.5" />}
              Restart
            </Button>
          )}
          <Button
            size="sm"
            variant="outline"
            className="gap-1.5"
            onClick={() => updateImages.mutate(stack.id)}
            disabled={isAnyActionLoading}
          >
            {isUpdatingImages ? <Loader2 className="size-3.5 animate-spin" /> : <Download className="size-3.5" />}
            Update
          </Button>
          {stack.status !== StackStatus.STOPPED && (
            <Button
              size="sm"
              variant="outline"
              className="gap-1.5"
              onClick={() => stopStack.mutate(stack.id)}
              disabled={isAnyActionLoading}
              data-testid="stack-stop-button"
            >
              {isStopping ? <Loader2 className="size-3.5 animate-spin" /> : <Square className="size-3.5" />}
              Stop
            </Button>
          )}
          <Button
            size="sm"
            variant="outline"
            className="gap-1.5 text-destructive hover:text-destructive"
            onClick={() => setConfirmDown(true)}
            disabled={isAnyActionLoading}
            data-testid="stack-down-button"
          >
            {isDown ? <Loader2 className="size-3.5 animate-spin" /> : <Square className="size-3.5" />}
            Down
          </Button>
          <Button
            size="sm"
            variant="outline"
            className="gap-1.5 text-destructive hover:text-destructive"
            onClick={() => setConfirmDelete(true)}
            disabled={isAnyActionLoading}
            data-testid="stack-delete-button"
          >
            {isDeleting ? <Loader2 className="size-3.5 animate-spin" /> : <Trash2 className="size-3.5" />}
            Delete
          </Button>
        </div>
      </div>

      <Tabs defaultValue="containers" className="flex flex-col">
        <TabsList>
          <TabsTrigger value="containers">
            Containers ({stack.containers.length})
          </TabsTrigger>
          <TabsTrigger value="compose">docker-compose.yml</TabsTrigger>
          <TabsTrigger value="env">.env</TabsTrigger>
          <TabsTrigger value="logs">Logs</TabsTrigger>
        </TabsList>

        {/* Containers tab */}
        <TabsContent value="containers" className="mt-4">
          {stack.containers.length === 0 ? (
            <div className="rounded-lg border border-border bg-secondary/10 p-8 text-center">
              <p className="text-sm text-muted-foreground">No containers. Start the stack to bring up containers.</p>
            </div>
          ) : (
            <div className="rounded-lg border border-border bg-card overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow className="bg-secondary/30 hover:bg-secondary/30">
                    <TableHead>Service</TableHead>
                    <TableHead>Image</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Ports</TableHead>
                    <TableHead className="w-16"></TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {stack.containers.map((container) => (
                    <TableRow key={container.containerName}>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <span
                            className={`inline-block size-2 rounded-full shrink-0 ${
                              container.state === "running" ? "bg-success" : "bg-muted-foreground"
                            }`}
                          />
                          <span className="font-medium text-foreground">{container.serviceName}</span>
                          <span className="text-xs text-muted-foreground font-mono">{container.containerName}</span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant="secondary" className="font-mono text-[11px]">
                          {container.image}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <StatusBadge status={container.state === "running" ? "running" : "stopped"} />
                        <span className="ml-2 text-xs text-muted-foreground">{container.status}</span>
                      </TableCell>
                      <TableCell>
                        {container.ports.length > 0 ? (
                          <div className="flex flex-wrap gap-1">
                            {container.ports.map((p, i) => (
                              <Badge key={i} variant="outline" className="font-mono text-[10px]">
                                {p}
                              </Badge>
                            ))}
                          </div>
                        ) : (
                          <span className="text-xs text-muted-foreground">—</span>
                        )}
                      </TableCell>
                      <TableCell>
                        {container.state === "running" && (
                          <Button
                            size="sm"
                            variant="outline"
                            className="gap-1 h-7 px-2 text-xs"
                            onClick={() => openBash(container.containerName, container.containerName)}
                          >
                            <Terminal className="size-3" />
                            Bash
                          </Button>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </TabsContent>

        {/* Compose tab */}
        <TabsContent value="compose" className="mt-4 space-y-3">
          <div className="border border-border rounded-md overflow-hidden">
            <MonacoEditor
              height="480px"
              language="yaml"
              value={compose}
              onChange={(val) => setCompose(val ?? "")}
              theme="vs-dark"
              options={{
                minimap: { enabled: false },
                fontSize: 13,
                lineNumbers: "on",
                scrollBeyondLastLine: false,
                automaticLayout: true,
                tabSize: 2,
              }}
            />
          </div>
          <div className="flex justify-end">
            <Button
              size="sm"
              onClick={() => updateStack.mutate({ id: stack.id, compose, env: stack.env })}
              disabled={isUpdating}
            >
              {isUpdating ? <Loader2 className="size-4 mr-2 animate-spin" /> : null}
              Save Compose
            </Button>
          </div>
        </TabsContent>

        {/* Env tab */}
        <TabsContent value="env" className="mt-4 space-y-3">
          <div className="border border-border rounded-md overflow-hidden">
            <MonacoEditor
              height="480px"
              language="plaintext"
              value={env}
              onChange={(val) => setEnv(val ?? "")}
              theme="vs-dark"
              options={{
                minimap: { enabled: false },
                fontSize: 13,
                lineNumbers: "on",
                scrollBeyondLastLine: false,
                automaticLayout: true,
                tabSize: 2,
              }}
            />
          </div>
          <div className="flex justify-end">
            <Button
              size="sm"
              onClick={() => updateStack.mutate({ id: stack.id, compose: stack.compose, env })}
              disabled={isUpdating}
            >
              {isUpdating ? <Loader2 className="size-4 mr-2 animate-spin" /> : null}
              Save Env
            </Button>
          </div>
        </TabsContent>

        {/* Logs tab */}
        <TabsContent value="logs" className="mt-4">
          <StackLogs stackId={stack.id} />
        </TabsContent>
      </Tabs>

      {/* Container bash dialog */}
      {bashContainer && (
        <ContainerBash
          open={bashOpen}
          onOpenChange={setBashOpen}
          stackId={stack.id}
          containerName={bashContainer.name}
        />
      )}

      {/* Down confirmation */}
      <AlertDialog open={confirmDown} onOpenChange={setConfirmDown}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Stop and remove containers?</AlertDialogTitle>
            <AlertDialogDescription>
              This will run <code className="font-mono text-sm">docker compose down</code>, stopping and removing all
              containers for <strong>{stack.name}</strong>. The compose file will be preserved. You can start it again
              later.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={handleDown}
              data-testid="stack-down-confirm"
            >
              {isDown ? <Loader2 className="size-4 mr-2 animate-spin" /> : null}
              Down Stack
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Delete confirmation */}
      <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete stack?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete the <strong>{stack.name}</strong> stack folder, including
              the compose file and all configuration. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={handleDelete}
              data-testid="stack-delete-confirm"
            >
              {isDeleting ? <Loader2 className="size-4 mr-2 animate-spin" /> : null}
              Delete Stack
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

export default function StackDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params)
  const { data: stack, isLoading, error, refetch } = useStack(id)

  if (error) {
    return (
      <div className="p-6">
        <ErrorDisplay message={error.message} onRetry={() => refetch()} className="h-[50vh]" />
      </div>
    )
  }

  if (!isLoading && !stack) return notFound()

  return (
    <Shimmer loading={isLoading} templateProps={{ stack: templateStack }}>
      <StackDetailContent stack={stack || templateStack} />
    </Shimmer>
  )
}
