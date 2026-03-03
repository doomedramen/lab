"use client"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Alert, AlertDescription } from "@/components/ui/alert"
import {
  AlertCircle,
  Server,
  Network,
  HardDrive,
  Terminal,
  Copy,
  Check,
  RefreshCw,
} from "lucide-react"
import { useState } from "react"

export interface DiagnosticsData {
  info?: {
    id: number
    name: string
    uuid: string
    osType: string
    state: string
    maxMemoryKb: number
    usedMemoryKb: number
    cpuCount: number
    autostart: string
    persistent: string
  }
  xmlConfig?: string
  networkInterfaces?: Array<{
    name: string
    macAddress: string
    protocol: string
    address: string
    prefix: number
  }>
  disks?: Array<{
    targetDev: string
    sourceFile: string
    driverType: string
    bus: string
  }>
  qemuMonitor?: {
    vncServer: string
    vncPort: number
    charDevices?: Array<{
      name: string
      sourcePath: string
    }>
  }
  host?: {
    hostname: string
    arch: string
    libvirtUri: string
    libvirtVersion: string
  }
}

interface VMDiagnosticsPanelProps {
  vmid: number
  data: DiagnosticsData
  onRefresh: () => void
  isLoading?: boolean
}

export function VMDiagnosticsPanel({ vmid, data, onRefresh, isLoading }: VMDiagnosticsPanelProps) {
  const [copiedXml, setCopiedXml] = useState(false)

  const copyToClipboard = async (text: string) => {
    await navigator.clipboard.writeText(text)
    setCopiedXml(true)
    setTimeout(() => setCopiedXml(false), 2000)
  }

  const formatBytes = (kb: number) => {
    if (kb >= 1048576) {
      return `${(kb / 1048576).toFixed(2)} GB`
    }
    if (kb >= 1024) {
      return `${(kb / 1024).toFixed(2)} MB`
    }
    return `${kb} KB`
  }

  return (
    <div className="space-y-4">
      {/* Info Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Domain ID</CardTitle>
            <Server className="size-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.info?.id ?? "N/A"}</div>
            <p className="text-xs text-muted-foreground">{data.info?.uuid?.slice(0, 8)}...</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">State</CardTitle>
            <Badge variant={data.info?.state === "running" ? "default" : "secondary"}>
              {data.info?.state ?? "unknown"}
            </Badge>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.info?.cpuCount ?? 0} vCPUs</div>
            <p className="text-xs text-muted-foreground">
              {formatBytes(data.info?.usedMemoryKb ?? 0)} / {formatBytes(data.info?.maxMemoryKb ?? 0)}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">VNC Console</CardTitle>
            <Terminal className="size-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {data.qemuMonitor?.vncPort ? (
              <>
                <div className="text-2xl font-bold">:{data.qemuMonitor.vncPort - 5900}</div>
                <p className="text-xs text-muted-foreground">
                  {data.qemuMonitor.vncServer}:{data.qemuMonitor.vncPort}
                </p>
              </>
            ) : (
              <div className="text-sm text-muted-foreground">Not configured</div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Host</CardTitle>
            <Server className="size-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.host?.hostname ?? "N/A"}</div>
            <p className="text-xs text-muted-foreground">{data.host?.arch ?? "unknown"}</p>
          </CardContent>
        </Card>
      </div>

      {/* Tabs for detailed info */}
      <Tabs defaultValue="overview" className="w-full">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="network">Network</TabsTrigger>
          <TabsTrigger value="storage">Storage</TabsTrigger>
          <TabsTrigger value="console">Console</TabsTrigger>
          <TabsTrigger value="xml">XML Config</TabsTrigger>
        </TabsList>

        {/* Overview Tab */}
        <TabsContent value="overview" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Domain Information</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              <div className="grid grid-cols-2 gap-2 text-sm">
                <div className="text-muted-foreground">Name:</div>
                <div className="font-mono">{data.info?.name}</div>

                <div className="text-muted-foreground">UUID:</div>
                <div className="font-mono">{data.info?.uuid}</div>

                <div className="text-muted-foreground">OS Type:</div>
                <div>{data.info?.osType}</div>

                <div className="text-muted-foreground">State:</div>
                <div>
                  <Badge variant={data.info?.state === "running" ? "default" : "secondary"}>
                    {data.info?.state}
                  </Badge>
                </div>

                <div className="text-muted-foreground">Persistent:</div>
                <div>{data.info?.persistent}</div>

                <div className="text-muted-foreground">Autostart:</div>
                <div>{data.info?.autostart}</div>

                <div className="text-muted-foreground">Memory:</div>
                <div>
                  {formatBytes(data.info?.usedMemoryKb ?? 0)} /{" "}
                  {formatBytes(data.info?.maxMemoryKb ?? 0)}
                </div>

                <div className="text-muted-foreground">CPUs:</div>
                <div>{data.info?.cpuCount} vCPU(s)</div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Network Tab */}
        <TabsContent value="network">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Network className="size-5" />
                Network Interfaces
              </CardTitle>
            </CardHeader>
            <CardContent>
              {data.networkInterfaces && data.networkInterfaces.length > 0 ? (
                <div className="space-y-2">
                  {data.networkInterfaces.map((iface, idx) => (
                    <div
                      key={idx}
                      className="flex items-center justify-between p-3 rounded-lg bg-muted/50"
                    >
                      <div className="space-y-1">
                        <div className="font-medium">{iface.name}</div>
                        <div className="text-sm text-muted-foreground">{iface.macAddress}</div>
                      </div>
                      <div className="text-right">
                        <div className="font-mono text-sm">
                          {iface.address}/{iface.prefix}
                        </div>
                        <div className="text-xs text-muted-foreground">{iface.protocol}</div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <Alert>
                  <AlertCircle className="size-4" />
                  <AlertDescription>
                    No network interfaces detected. This is normal for VMs using user-mode (NAT)
                    networking or without QEMU guest agent installed.
                  </AlertDescription>
                </Alert>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Storage Tab */}
        <TabsContent value="storage">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <HardDrive className="size-5" />
                Disk Devices
              </CardTitle>
            </CardHeader>
            <CardContent>
              {data.disks && data.disks.length > 0 ? (
                <div className="space-y-2">
                  {data.disks.map((disk, idx) => (
                    <div
                      key={idx}
                      className="flex items-center justify-between p-3 rounded-lg bg-muted/50"
                    >
                      <div className="space-y-1">
                        <div className="font-medium">{disk.targetDev}</div>
                        <div className="text-sm text-muted-foreground font-mono">
                          {disk.sourceFile}
                        </div>
                      </div>
                      <div className="text-right">
                        <Badge variant="outline">{disk.driverType}</Badge>
                        <div className="text-xs text-muted-foreground mt-1">{disk.bus}</div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-muted-foreground">No disk information available</div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Console Tab */}
        <TabsContent value="console">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Terminal className="size-5" />
                Console Devices
              </CardTitle>
            </CardHeader>
            <CardContent>
              {data.qemuMonitor?.charDevices && data.qemuMonitor.charDevices.length > 0 ? (
                <div className="space-y-2">
                  {data.qemuMonitor.charDevices.map((device, idx) => (
                    <div
                      key={idx}
                      className="flex items-center justify-between p-3 rounded-lg bg-muted/50"
                    >
                      <div className="font-medium">{device.name}</div>
                      {device.sourcePath ? (
                        <div className="font-mono text-sm text-muted-foreground">
                          {device.sourcePath}
                        </div>
                      ) : (
                        <div className="text-sm text-muted-foreground">No path configured</div>
                      )}
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-muted-foreground">No console devices configured</div>
              )}

              {data.qemuMonitor?.vncPort && (
                <div className="mt-4 p-3 rounded-lg bg-muted/50">
                  <div className="font-medium mb-1">VNC Server</div>
                  <div className="font-mono text-sm">
                    {data.qemuMonitor.vncServer}:{data.qemuMonitor.vncPort}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* XML Config Tab */}
        <TabsContent value="xml">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>Domain XML Configuration</CardTitle>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => copyToClipboard(data.xmlConfig ?? "")}
                  >
                    {copiedXml ? (
                      <Check className="size-4 mr-2" />
                    ) : (
                      <Copy className="size-4 mr-2" />
                    )}
                    {copiedXml ? "Copied" : "Copy"}
                  </Button>
                  <Button variant="outline" size="sm" onClick={onRefresh}>
                    <RefreshCw className="size-4 mr-2" />
                    Refresh
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <pre className="p-4 rounded-lg bg-muted/50 overflow-x-auto text-xs font-mono max-h-[600px] overflow-y-auto">
                {data.xmlConfig ?? "XML configuration not available"}
              </pre>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}

export default VMDiagnosticsPanel
