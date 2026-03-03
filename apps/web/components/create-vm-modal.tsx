"use client"

import { useState, useEffect, useRef } from "react"
import { useRouter } from "next/navigation"
import { Plus, Loader2, Cpu, MemoryStick, HardDrive, Disc, Server, Tag, FileText, Settings2, Globe, Upload, AlertCircle } from "lucide-react"
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
import { Switch } from "@/components/ui/switch"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { useVMMutations } from "@/lib/api/mutations"
import { useISOs, usePCIDevices } from "@/lib/api/queries"
import { useVMTemplates } from "@/lib/api/queries/templates"
import type { Node } from "@/lib/gen/lab/v1/node_pb"
import type { VMTemplate } from "@/lib/gen/lab/v1/vm_pb"
import { osTypeFromString, networkTypeFromString, networkModelFromString } from "@/lib/api/enum-helpers"

// Preset configurations (for custom template)
const PRESETS = [
  { name: "Small", cpu: 1, memory: 2, disk: 20, description: "Basic workloads" },
  { name: "Medium", cpu: 2, memory: 4, disk: 40, description: "Standard workloads" },
  { name: "Large", cpu: 4, memory: 8, disk: 80, description: "Heavy workloads" },
  { name: "Custom", cpu: 0, memory: 0, disk: 0, description: "Custom configuration" },
]

interface CreateVMModalProps {
  nodes: Node[] | undefined
  trigger?: React.ReactNode
}

export function CreateVMModal({ nodes, trigger }: CreateVMModalProps) {
  const router = useRouter()
  const [open, setOpen] = useState(false)
  const { data: templates } = useVMTemplates()
  const { data: isos } = useISOs()
  const { data: pciData } = usePCIDevices()

  // Form state - node MUST be declared before selectedNode
  const [node, setNode] = useState("")
  const [name, setName] = useState("")
  const [selectedTemplate, setSelectedTemplate] = useState<string>("")
  const [arch, setArch] = useState<"x86_64" | "aarch64">("x86_64")
  const [hostArch, setHostArch] = useState<"x86_64" | "aarch64" | null>(null)

  // ISO configuration - either uploaded ISO or URL
  const [isoSource, setIsoSource] = useState<"uploaded" | "url">("uploaded")
  const [selectedIsoId, setSelectedIsoId] = useState("")
  const [isoUrl, setIsoUrl] = useState("")
  const [isoName, setIsoName] = useState("")
  const [isUrlValid, setIsUrlValid] = useState<boolean | null>(null)
  const [isUrlChecking, setIsUrlChecking] = useState(false)
  const [urlError, setUrlError] = useState("")

  // Resources
  const [preset, setPreset] = useState("Medium")
  const [cpuCores, setCpuCores] = useState(2)
  const [memory, setMemory] = useState(4)
  const [disk, setDisk] = useState(40)
  const [cpuError, setCpuError] = useState("")
  const [memoryError, setMemoryError] = useState("")
  const [diskError, setDiskError] = useState("")

  // Advanced settings
  const [description, setDescription] = useState("")
  const [tags, setTags] = useState("")
  const [startOnBoot, setStartOnBoot] = useState(true)
  const [nestedVirt, setNestedVirt] = useState(false)
  const [agent, setAgent] = useState(true)
  const [networkType, setNetworkType] = useState<"user" | "bridge">("user")
  const [networkBridge, setNetworkBridge] = useState("")
  const [networkModel, setNetworkModel] = useState<"virtio" | "e1000" | "rtl8139">("virtio")
  const [tpm, setTpm] = useState(false)
  const [secureBoot, setSecureBoot] = useState(false)
  const [selectedPCIDevices, setSelectedPCIDevices] = useState<string[]>([])
  const [bootOrder, setBootOrder] = useState<string[]>(["hd", "cdrom"])

  const { createVM, isCreating } = useVMMutations({
    onCreateSuccess: (res) => {
      setOpen(false)
      if (res.vm?.vmid) {
        router.push(`/vms/${res.vm.vmid}`)
      } else {
        resetForm()
      }
    },
  })

  // Get selected node (declared after node state)
  const selectedNode = nodes?.find((n) => n.name === node)
  const selectedTemplateData = templates?.find((t) => t.id === selectedTemplate)

  // Detect host architecture when node changes
  useEffect(() => {
    if (selectedNode?.arch) {
      const detectedArch = selectedNode.arch === "aarch64" ? "aarch64" : "x86_64"
      setHostArch(detectedArch)
      setArch(detectedArch)
      // Reset template selection when host changes (template may not be compatible)
      if (selectedTemplate) {
        const template = templates?.find((t) => t.id === selectedTemplate)
        if (template && !template.isoUrl) {
          // Template doesn't support this architecture, reset selection
          setSelectedTemplate("")
        }
      }
    }
  }, [selectedNode, templates, selectedTemplate])

  // Update form when template is selected
  useEffect(() => {
    if (!selectedTemplate || !templates || templates.length === 0) {
      return
    }
    
    const template = templates.find((t) => t.id === selectedTemplate)
    if (!template) {
      return
    }
    
    // Apply template resources
    setCpuCores(template.cpuCores || 2)
    setMemory(template.memoryGb || 4)
    setDisk(template.diskGb || 40)
    setPreset("Custom")
    
    // Apply template ISO configuration based on host architecture
    if (template.isoUrl) {
      // Use the URL from the template (already filtered by arch)
      setIsoSource("url")
      setIsoUrl(template.isoUrl)
      // Extract filename from URL or use template's isoName
      const urlParts = template.isoUrl.split('/')
      setIsoName(urlParts[urlParts.length - 1] || template.isoName || "")
      setSelectedIsoId("")
      setIsUrlValid(null) // Reset validation for new URL
      setUrlError("")
    } else {
      // Template without URL (like Windows) - user must provide ISO
      setIsoSource("uploaded")
      setIsoUrl("")
      setIsoName(template.isoName || "")
      setSelectedIsoId("")
    }
  }, [selectedTemplate, templates])

  // Validate ISO URL when it changes (debounced, with AbortController to prevent stale results).
  const urlAbortRef = useRef<AbortController | null>(null)
  useEffect(() => {
    if (isoSource !== "url" || !isoUrl.trim()) {
      setIsUrlValid(null)
      setUrlError("")
      return
    }

    setIsUrlChecking(true)
    setUrlError("")

    const controller = new AbortController()
    urlAbortRef.current = controller

    const timer = setTimeout(async () => {
      try {
        // no-cors: we cannot read the response status, but a non-throw means the server responded.
        await fetch(isoUrl, { method: "HEAD", mode: "no-cors", signal: controller.signal })
        setIsUrlValid(true)
      } catch (error) {
        if ((error as Error).name === "AbortError") return
        setIsUrlValid(false)
        setUrlError("Unable to reach this URL. Please check if it's accessible.")
      } finally {
        if (!controller.signal.aborted) {
          setIsUrlChecking(false)
        }
      }
    }, 500)

    return () => {
      clearTimeout(timer)
      controller.abort()
    }
  }, [isoUrl, isoSource])

  // Update values when preset changes
  const handlePresetChange = (value: string) => {
    setPreset(value)
    const presetConfig = PRESETS.find((p) => p.name === value)
    if (presetConfig && value !== "Custom") {
      setCpuCores(presetConfig.cpu)
      setMemory(presetConfig.memory)
      setDisk(presetConfig.disk)
    }
  }

  const handleSubmit = () => {
    if (!name.trim() || !node) {
      return
    }

    // Validate ISO selection
    let isoPath = ""
    let isoUrlValue = ""
    let isoNameValue = ""

    if (isoSource === "uploaded") {
      const selectedIso = isos?.find((iso) => iso.id === selectedIsoId)
      if (!selectedIso) {
        return // No ISO selected
      }
      isoPath = selectedIso.path
      isoNameValue = selectedIso.name
    } else {
      // URL source - use architecture-specific URL
      if (!isoUrl.trim()) {
        return // No URL provided
      }
      isoUrlValue = isoUrl.trim()
      // Extract or use provided ISO name
      isoNameValue = isoName || isoUrl.split("/").pop() || "downloaded.iso"
    }

    // Get OS from template or default
    const template = templates?.find((t) => t.id === selectedTemplate)
    const osConfig = template?.os || { osType: "linux", version: "" }

    createVM.mutate({
      name: name.trim(),
      node,
      cpuCores,
      memoryGb: memory,
      diskGb: disk,
      iso: isoPath,
      isoUrl: isoUrlValue,
      isoName: isoNameValue,
      os: {
        osType: osTypeFromString(String(osConfig.osType || "linux")),
        version: osConfig.version || "",
      },
      arch: arch,
      description: description || "",
      tags: tags ? tags.split(",").map((t) => t.trim()).filter(Boolean) : [],
      startOnBoot,
      nestedVirt,
      agent,
      network: [{
        type: networkTypeFromString(networkType),
        model: networkModelFromString(networkModel),
        bridge: networkType === "bridge" ? networkBridge : "",
        vlan: 0,
      }],
      tpm,
      secureBoot,
      pciDeviceAddresses: selectedPCIDevices,
      bootOrder,
    })
  }

  const resetForm = () => {
    setName("")
    setNode(nodes?.[0]?.name || "")
    setSelectedTemplate("")
    setHostArch(null)
    setArch("x86_64")
    setIsoSource("uploaded")
    setSelectedIsoId("")
    setIsoUrl("")
    setIsoName("")
    setIsUrlValid(null)
    setUrlError("")
    setCpuCores(2)
    setMemory(4)
    setDisk(40)
    setCpuError("")
    setMemoryError("")
    setDiskError("")
    setDescription("")
    setTags("")
    setStartOnBoot(true)
    setNestedVirt(false)
    setAgent(true)
    setNetworkType("user")
    setNetworkBridge("")
    setNetworkModel("virtio")
    setTpm(false)
    setSecureBoot(false)
    setSelectedPCIDevices([])
    setBootOrder(["hd", "cdrom"])
    setPreset("Medium")
  }

  // Filter templates based on host architecture support
  const compatibleTemplates = templates?.filter((t) => {
    if (!hostArch) return true // Show all if no host selected yet
    // Check if template has URL for this architecture
    if (hostArch === "x86_64") {
      return t.isoUrl.includes("amd64") || t.isoUrl.includes("x86_64")
    } else {
      return t.isoUrl.includes("arm64") || t.isoUrl.includes("aarch64")
    }
  }) || []
  
  const incompatibleTemplates = templates?.filter((t) => {
    if (!hostArch) return false
    if (hostArch === "x86_64") {
      return !t.isoUrl.includes("amd64") && !t.isoUrl.includes("x86_64")
    } else {
      return !t.isoUrl.includes("arm64") && !t.isoUrl.includes("aarch64")
    }
  }) || []

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger || (
          <Button data-testid="create-vm-button" className="gap-1.5">
            <Plus className="size-4" />
            Create VM
          </Button>
        )}
      </DialogTrigger>
      <DialogContent data-testid="create-vm-modal" className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Create Virtual Machine</DialogTitle>
          <DialogDescription>
            Select a template or configure your virtual machine manually.
            {!hostArch && node && (
              <span className="text-amber-600 block mt-1">
                ⚠️ No templates available for this host's architecture
              </span>
            )}
            {!node && (
              <span className="text-muted-foreground block mt-1">
                Select a node first to see compatible templates
              </span>
            )}
          </DialogDescription>
        </DialogHeader>

        <Tabs defaultValue="template" className="mt-4">
          <TabsList className="grid grid-cols-4">
            <TabsTrigger value="template">Template</TabsTrigger>
            <TabsTrigger value="basic" disabled={!selectedTemplate}>Basic</TabsTrigger>
            <TabsTrigger value="resources" disabled={!selectedTemplate}>Resources</TabsTrigger>
            <TabsTrigger value="advanced" disabled={!selectedTemplate}>Advanced</TabsTrigger>
          </TabsList>

          {/* Template Tab - Node selection MUST happen here first */}
          <TabsContent value="template" className="space-y-4 mt-4">
            {/* Node Selection - REQUIRED FIRST */}
            <div className="space-y-2">
              <Label htmlFor="template-node">Select Target Node *</Label>
              <p className="text-xs text-muted-foreground">
                You must select a node first. Available templates will be filtered by the node's architecture.
              </p>
              <Select value={node} onValueChange={(newNode) => {
                setNode(newNode)
                setSelectedTemplate("") // Reset template when node changes
              }}>
                <SelectTrigger id="template-node" data-testid="vm-template-node-select" className="w-full">
                  <SelectValue placeholder="Select a node" />
                </SelectTrigger>
                <SelectContent>
                  {nodes?.map((n) => (
                    <SelectItem key={n.id} value={n.name}>
                      <div className="flex items-center gap-2">
                        <Server className="size-4 text-muted-foreground" />
                        <span>{n.name}</span>
                        <span className="text-xs text-muted-foreground">
                          ({n.arch || "x86_64"}, {n.cpu?.cores ?? 0} cores, {(n.memory?.total ?? 0).toFixed(2)} GB RAM)
                        </span>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {!node && (
                <div className="p-3 rounded-lg bg-amber-500/10 border border-amber-500/20">
                  <p className="text-sm text-amber-600 font-medium">⚠️ Node selection required</p>
                  <p className="text-xs text-amber-600/80">Select a node above to see compatible VM templates</p>
                </div>
              )}
            </div>

            {/* Template Selection - only shown after node is selected */}
            {node && (
              <div className="space-y-2">
                <Label htmlFor="template-select">Select a Template</Label>
                <p className="text-xs text-muted-foreground">
                  Templates provide pre-configured VMs with optimal settings for your node's architecture ({hostArch}).
                </p>
                <Select value={selectedTemplate} onValueChange={setSelectedTemplate}>
                  <SelectTrigger id="template-select" data-testid="vm-template-select" className="w-full">
                    <SelectValue placeholder="Select a template" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="custom">Custom Configuration</SelectItem>
                    {compatibleTemplates.map((template) => (
                      <SelectItem key={template.id} value={template.id}>
                        <div className="flex items-center gap-2">
                          <span>{template.icon}</span>
                          <span>{template.name}</span>
                          <span className="text-xs text-success ml-auto">✓ Compatible</span>
                        </div>
                      </SelectItem>
                    ))}
                    {incompatibleTemplates.length > 0 && (
                      <>
                        <SelectItem value="incompatible-header" disabled>
                          ────────────────
                        </SelectItem>
                        {incompatibleTemplates.map((template) => (
                          <SelectItem key={template.id} value={template.id} disabled>
                            <div className="flex items-center gap-2 text-muted-foreground">
                              <span>{template.icon}</span>
                              <span>{template.name}</span>
                              <span className="text-xs ml-auto">Not available for {hostArch}</span>
                            </div>
                          </SelectItem>
                        ))}
                      </>
                    )}
                  </SelectContent>
                </Select>
              </div>
            )}

            {/* Template Info Card */}
            {selectedTemplateData && selectedTemplate !== "custom" && (
              <Card className="mt-4">
                <CardContent className="p-4 space-y-3">
                  <div className="flex items-center gap-3">
                    <span className="text-3xl">{selectedTemplateData.icon}</span>
                    <div>
                      <div className="font-medium">{selectedTemplateData.name}</div>
                      <div className="text-xs text-muted-foreground">{selectedTemplateData.description}</div>
                    </div>
                  </div>
                  <div className="flex flex-wrap gap-2 pt-2">
                    <Badge variant="secondary" className="gap-1">
                      <Cpu className="size-3" />
                      {selectedTemplateData.cpuCores} CPU
                    </Badge>
                    <Badge variant="secondary" className="gap-1">
                      <MemoryStick className="size-3" />
                      {selectedTemplateData.memoryGb} GB RAM
                    </Badge>
                    <Badge variant="secondary" className="gap-1">
                      <HardDrive className="size-3" />
                      {selectedTemplateData.diskGb} GB disk
                    </Badge>
                    <Badge variant="secondary" className="gap-1">
                      <Settings2 className="size-3" />
                      {selectedTemplateData.arch === "x86_64" ? "Intel/AMD" : "ARM"}
                    </Badge>
                  </div>
                  {selectedTemplateData.isoUrl ? (
                    <div className="flex items-start gap-2 text-xs text-success">
                      <Globe className="size-3 mt-0.5" />
                      <div>
                        <div>ISO will be downloaded for {hostArch === "aarch64" ? "ARM" : "Intel/AMD"}:</div>
                        <div className="font-mono break-all">{isoUrl}</div>
                      </div>
                    </div>
                  ) : (
                    <div className="flex items-center gap-2 text-xs text-amber-600">
                      <Disc className="size-3" />
                      You must upload an ISO for this template
                    </div>
                  )}
                </CardContent>
              </Card>
            )}

            {/* Custom option info */}
            {selectedTemplate === "custom" && (
              <Card className="mt-4">
                <CardContent className="p-4">
                  <div className="flex items-center gap-3">
                    <Settings2 className="size-6 text-muted-foreground" />
                    <div>
                      <div className="font-medium">Custom Configuration</div>
                      <div className="text-xs text-muted-foreground">
                        Manually configure all settings including ISO source
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )}
          </TabsContent>

          {/* Basic Tab */}
          <TabsContent value="basic" className="space-y-4 mt-4">
            {/* Name */}
            <div className="space-y-2">
              <Label htmlFor="name">VM Name *</Label>
              <Input
                id="name"
                data-testid="vm-name-input"
                placeholder="my-ubuntu-vm"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>

            {/* ISO Source Selection */}
            <div className="space-y-2">
              <Label>ISO Source</Label>
              <div className="flex gap-2">
                <Button
                  type="button"
                  variant={isoSource === "uploaded" ? "default" : "outline"}
                  size="sm"
                  onClick={() => {
                    setIsoSource("uploaded")
                    setIsUrlValid(null)
                    setUrlError("")
                  }}
                  className="flex-1"
                >
                  <Upload className="size-4 mr-2" />
                  Uploaded ISO
                </Button>
                <Button
                  type="button"
                  variant={isoSource === "url" ? "default" : "outline"}
                  size="sm"
                  onClick={() => {
                    setIsoSource("url")
                    setIsUrlValid(null)
                    setUrlError("")
                  }}
                  className="flex-1"
                >
                  <Globe className="size-4 mr-2" />
                  Download from URL
                </Button>
              </div>
            </div>

            {/* ISO Selection based on source */}
            {isoSource === "uploaded" ? (
              <div className="space-y-2">
                <Label htmlFor="iso-select">Select ISO Image *</Label>
                <Select value={selectedIsoId} onValueChange={setSelectedIsoId}>
                  <SelectTrigger id="iso-select" data-testid="vm-iso-select">
                    <SelectValue placeholder="Select an ISO" />
                  </SelectTrigger>
                  <SelectContent>
                    {isos && isos.length > 0 ? (
                      isos.map((iso) => (
                        <SelectItem key={iso.id} value={iso.id}>
                          <div className="flex items-center gap-2">
                            <Disc className="size-4 text-muted-foreground" />
                            <span>{iso.name}</span>
                            <span className="text-xs text-muted-foreground">
                              ({(Number(iso.size) / 1024 / 1024 / 1024).toFixed(2)} GB)
                            </span>
                          </div>
                        </SelectItem>
                      ))
                    ) : (
                      <SelectItem value="none" disabled>
                        No ISOs uploaded - go to ISOs page to upload
                      </SelectItem>
                    )}
                  </SelectContent>
                </Select>
                {isos && isos.length === 0 && (
                  <p className="text-xs text-amber-600">
                    No ISO images available. Please upload an ISO first or use a template with auto-download.
                  </p>
                )}
              </div>
            ) : (
              <div className="space-y-2">
                <Label htmlFor="iso-url">ISO Download URL *</Label>
                <div className="flex gap-2">
                  <Input
                    id="iso-url"
                    data-testid="vm-iso-url-input"
                    placeholder="https://example.com/ubuntu.iso"
                    value={isoUrl}
                    onChange={(e) => {
                      setIsoUrl(e.target.value)
                      setIsUrlValid(null)
                      setUrlError("")
                    }}
                    className="flex-1"
                  />
                  {isoUrl.trim() && (
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setIsUrlChecking(true)
                        setUrlError("")
                        fetch(isoUrl, { method: "HEAD", mode: "no-cors" })
                          .then(() => {
                            setIsUrlValid(true)
                          })
                          .catch(() => {
                            setIsUrlValid(false)
                            setUrlError("Unable to reach this URL.")
                          })
                          .finally(() => {
                            setIsUrlChecking(false)
                          })
                      }}
                      disabled={isUrlChecking}
                    >
                      {isUrlChecking ? (
                        <Loader2 className="size-4 animate-spin" />
                      ) : isUrlValid === true ? (
                        <span className="text-success">✓</span>
                      ) : isUrlValid === false ? (
                        <span className="text-destructive">✗</span>
                      ) : (
                        "Check"
                      )}
                    </Button>
                  )}
                </div>
                {isUrlValid === true && (
                  <p className="text-xs text-success flex items-center gap-1">
                    <span className="inline-block w-1.5 h-1.5 rounded-full bg-success" />
                    URL is accessible
                  </p>
                )}
                {isUrlValid === false && (
                  <p className="text-xs text-destructive flex items-center gap-1">
                    <span className="inline-block w-1.5 h-1.5 rounded-full bg-destructive" />
                    {urlError || "URL is not accessible"}
                  </p>
                )}
                {isUrlChecking && (
                  <p className="text-xs text-muted-foreground flex items-center gap-1">
                    <Loader2 className="size-3 animate-spin" />
                    Checking URL...
                  </p>
                )}
                <p className="text-xs text-muted-foreground">
                  The ISO will be downloaded to the server and stored for future use.
                </p>
              </div>
            )}

            {/* Description */}
            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <div className="relative">
                <FileText className="absolute left-3 top-3 size-4 text-muted-foreground" />
                <Input
                  id="description"
                  data-testid="vm-description-input"
                  placeholder="Purpose of this VM..."
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="pl-9"
                />
              </div>
            </div>

            {/* Tags */}
            <div className="space-y-2">
              <Label htmlFor="tags">Tags</Label>
              <div className="relative">
                <Tag className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
                <Input
                  id="tags"
                  data-testid="vm-tags-input"
                  placeholder="production, web, database"
                  value={tags}
                  onChange={(e) => setTags(e.target.value)}
                  className="pl-9"
                />
              </div>
              <p className="text-xs text-muted-foreground">Comma-separated tags</p>
            </div>
          </TabsContent>

          {/* Resources Tab */}
          <TabsContent value="resources" className="space-y-4 mt-4">
            {/* Architecture Info (read-only, based on selected node) */}
            <div className="space-y-2">
              <Label>Architecture</Label>
              <div className="flex items-center gap-3 p-3 rounded-lg border border-border bg-secondary/50">
                <Settings2 className="size-5 text-muted-foreground" />
                <div className="flex-1">
                  <div className="font-medium">
                    {hostArch === "aarch64" ? "ARM 64-bit (aarch64)" : "Intel/AMD (x86_64)"}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    {hostArch ? `Detected from host: ${node || "select a node"}` : "Select a node to detect architecture"}
                  </div>
                </div>
                <Badge variant="outline" className="text-xs">
                  {hostArch || "Unknown"}
                </Badge>
              </div>
              <p className="text-xs text-muted-foreground">
                Architecture is automatically detected from the selected node. VMs run best with native architecture.
              </p>
            </div>

            {/* Preset Selector */}
            <div className="space-y-2">
              <Label>Resource Preset</Label>
              <div className="grid grid-cols-4 gap-2">
                {PRESETS.map((p) => (
                  <button
                    key={p.name}
                    type="button"
                    onClick={() => handlePresetChange(p.name)}
                    className={`flex flex-col items-center p-3 rounded-lg border transition-colors ${
                      preset === p.name
                        ? "border-primary bg-primary/10 text-primary"
                        : "border-border hover:border-primary/50 hover:bg-secondary/50"
                    }`}
                  >
                    <span className="font-medium">{p.name}</span>
                    {p.name !== "Custom" && (
                      <span className="text-xs text-muted-foreground mt-1">
                        {p.cpu}c / {p.memory}GB / {p.disk}GB
                      </span>
                    )}
                  </button>
                ))}
              </div>
            </div>

            {/* CPU */}
            <div className="space-y-2">
              <Label htmlFor="cpu">CPU Cores</Label>
              <div className="flex items-center gap-4">
                <Cpu className="size-4 text-muted-foreground shrink-0" />
                <Input
                  id="cpu"
                  data-testid="vm-cpu-input"
                  type="number"
                  min={1}
                  max={selectedNode?.cpu?.cores ?? 64}
                  value={cpuCores}
                  onChange={(e) => {
                    setPreset("Custom")
                    const val = parseInt(e.target.value)
                    const clamped = isNaN(val) ? 1 : val
                    setCpuCores(clamped)
                    setCpuError(clamped < 1 ? "Must be at least 1 core" : "")
                  }}
                  className={`flex-1 ${cpuError ? "border-destructive" : ""}`}
                />
                <span className="text-sm text-muted-foreground">cores</span>
              </div>
              {cpuError && <p className="text-xs text-destructive">{cpuError}</p>}
              {selectedNode && (
                <p className="text-xs text-muted-foreground">
                  Available: {selectedNode.cpu?.cores ?? 0} cores on {selectedNode.name}
                </p>
              )}
            </div>

            {/* Memory */}
            <div className="space-y-2">
              <Label htmlFor="memory">Memory</Label>
              <div className="flex items-center gap-4">
                <MemoryStick className="size-4 text-muted-foreground shrink-0" />
                <Input
                  id="memory"
                  data-testid="vm-memory-input"
                  type="number"
                  min={1}
                  max={selectedNode?.memory?.total ?? 256}
                  value={memory}
                  onChange={(e) => {
                    setPreset("Custom")
                    const val = parseInt(e.target.value)
                    const clamped = isNaN(val) ? 1 : val
                    setMemory(clamped)
                    setMemoryError(clamped < 1 ? "Must be at least 1 GB" : "")
                  }}
                  className={`flex-1 ${memoryError ? "border-destructive" : ""}`}
                />
                <span className="text-sm text-muted-foreground">GB</span>
              </div>
              {memoryError && <p className="text-xs text-destructive">{memoryError}</p>}
              {selectedNode && (
                <p className="text-xs text-muted-foreground">
                  Available: {(selectedNode.memory?.total ?? 0).toFixed(2)} GB on {selectedNode.name}
                </p>
              )}
            </div>

            {/* Disk */}
            <div className="space-y-2">
              <Label htmlFor="disk">Disk Size</Label>
              <div className="flex items-center gap-4">
                <HardDrive className="size-4 text-muted-foreground shrink-0" />
                <Input
                  id="disk"
                  data-testid="vm-disk-input"
                  type="number"
                  min={10}
                  max={2000}
                  value={disk}
                  onChange={(e) => {
                    setPreset("Custom")
                    const val = parseInt(e.target.value)
                    const clamped = isNaN(val) ? 10 : val
                    setDisk(clamped)
                    setDiskError(clamped < 10 ? "Must be at least 10 GB" : "")
                  }}
                  className={`flex-1 ${diskError ? "border-destructive" : ""}`}
                />
                <span className="text-sm text-muted-foreground">GB</span>
              </div>
              {diskError && <p className="text-xs text-destructive">{diskError}</p>}
            </div>

            {/* Summary */}
            <div className="mt-4 p-4 rounded-lg bg-secondary/50 border border-border">
              <h4 className="text-sm font-medium mb-2">Configuration Summary</h4>
              <div className="flex flex-wrap gap-2">
                <Badge variant="secondary" className="gap-1">
                  <Cpu className="size-3" /> {cpuCores} cores
                </Badge>
                <Badge variant="secondary" className="gap-1">
                  <MemoryStick className="size-3" /> {memory} GB RAM
                </Badge>
                <Badge variant="secondary" className="gap-1">
                  <HardDrive className="size-3" /> {disk} GB disk
                </Badge>
              </div>
            </div>
          </TabsContent>

          {/* Advanced Tab */}
          <TabsContent value="advanced" className="space-y-4 mt-4">
            {/* Start on Boot */}
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="startOnBoot">Start on Boot</Label>
                <p className="text-xs text-muted-foreground">
                  Automatically start this VM when the host boots
                </p>
              </div>
              <Switch id="startOnBoot" checked={startOnBoot} onCheckedChange={setStartOnBoot} />
            </div>

            {/* Nested Virtualization */}
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="nestedVirt">Nested Virtualization</Label>
                <p className="text-xs text-muted-foreground">
                  Allow running VMs inside this VM (Docker, etc.)
                </p>
              </div>
              <Switch id="nestedVirt" checked={nestedVirt} onCheckedChange={setNestedVirt} />
            </div>

            {/* QEMU Agent */}
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="agent">QEMU Agent</Label>
                <p className="text-xs text-muted-foreground">
                  Enable for better host-guest communication
                </p>
              </div>
              <Switch id="agent" checked={agent} onCheckedChange={setAgent} />
            </div>

            {/* TPM 2.0 */}
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="tpm">TPM 2.0 Device</Label>
                <p className="text-xs text-muted-foreground">
                  Virtual TPM 2.0 device (required for Windows 11)
                </p>
              </div>
              <Switch
                id="tpm"
                checked={tpm}
                onCheckedChange={setTpm}
                disabled={arch === "aarch64"}
              />
            </div>
            {arch === "aarch64" && (
              <p className="text-xs text-amber-600 -mt-2">
                TPM is not supported on ARM architecture
              </p>
            )}

            {/* Secure Boot */}
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="secureBoot">Secure Boot</Label>
                <p className="text-xs text-muted-foreground">
                  UEFI Secure Boot (requires OVMF firmware)
                </p>
              </div>
              <Switch
                id="secureBoot"
                checked={secureBoot}
                onCheckedChange={setSecureBoot}
                disabled={arch === "aarch64"}
              />
            </div>
            {arch === "aarch64" && (
              <p className="text-xs text-amber-600 -mt-2">
                Secure Boot is not supported on ARM architecture
              </p>
            )}

            {/* PCI Passthrough */}
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>PCI Device Passthrough</Label>
                  <p className="text-xs text-muted-foreground">
                    Pass through host PCI devices (GPU, etc.) to this VM
                  </p>
                </div>
              </div>

              {/* IOMMU Status */}
              {pciData && (
                <div className="flex gap-2">
                  <Badge variant={pciData.iommuAvailable ? "default" : "destructive"} className="text-xs">
                    IOMMU: {pciData.iommuAvailable ? "Available" : "Not Available"}
                  </Badge>
                  <Badge variant={pciData.vfioAvailable ? "default" : "secondary"} className="text-xs">
                    VFIO: {pciData.vfioAvailable ? "Loaded" : "Not Loaded"}
                  </Badge>
                </div>
              )}

              {!pciData?.iommuAvailable && (
                <div className="p-3 rounded-lg bg-amber-500/10 border border-amber-500/20 text-amber-700 dark:text-amber-400 text-xs">
                  <div className="flex items-start gap-2">
                    <AlertCircle className="size-4 shrink-0 mt-0.5" />
                    <div>
                      IOMMU not detected. Enable IOMMU in BIOS and add <code className="px-1 py-0.5 bg-amber-500/20 rounded">intel_iommu=on</code> or <code className="px-1 py-0.5 bg-amber-500/20 rounded">amd_iommu=on</code> to kernel parameters.
                    </div>
                  </div>
                </div>
              )}

              {/* PCI Device Selection */}
              {pciData?.devices && pciData.devices.length > 0 ? (
                <div className="border rounded-lg divide-y max-h-48 overflow-y-auto">
                  {Object.entries(
                    pciData.devices.reduce((groups, device) => {
                      const group = device.iommuGroup ?? -1
                      if (!groups[group]) groups[group] = []
                      groups[group].push(device)
                      return groups
                    }, {} as Record<number, typeof pciData.devices>)
                  ).map(([group, devices]) => (
                    <div key={group} className="p-2">
                      <div className="text-xs text-muted-foreground mb-1">
                        IOMMU Group {group !== "-1" ? group : "Unknown"}
                      </div>
                      {devices.map((device) => (
                        <label
                          key={device.address}
                          className="flex items-center gap-2 py-1 px-2 hover:bg-secondary/50 rounded cursor-pointer"
                        >
                          <input
                            type="checkbox"
                            checked={selectedPCIDevices.includes(device.address)}
                            onChange={(e) => {
                              if (e.target.checked) {
                                setSelectedPCIDevices([...selectedPCIDevices, device.address])
                              } else {
                                setSelectedPCIDevices(selectedPCIDevices.filter((a) => a !== device.address))
                              }
                            }}
                            className="size-4"
                          />
                          <div className="flex-1 min-w-0">
                            <div className="text-sm font-medium truncate">
                              {device.productName || device.className || "Unknown Device"}
                            </div>
                            <div className="text-xs text-muted-foreground flex gap-2">
                              <span className="font-mono">{device.address}</span>
                              {device.vendorName && <span>{device.vendorName}</span>}
                              {device.driver && <span className="text-amber-600">[{device.driver}]</span>}
                            </div>
                          </div>
                        </label>
                      ))}
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-xs text-muted-foreground">
                  No PCI devices available or unable to detect devices.
                </p>
              )}

              {/* Selected devices summary */}
              {selectedPCIDevices.length > 0 && (
                <div className="flex flex-wrap gap-1">
                  {selectedPCIDevices.map((addr) => {
                    const device = pciData?.devices?.find((d) => d.address === addr)
                    return (
                      <Badge key={addr} variant="secondary" className="text-xs">
                        {device?.productName || addr}
                        <button
                          type="button"
                          className="ml-1 hover:text-destructive"
                          onClick={() => setSelectedPCIDevices(selectedPCIDevices.filter((a) => a !== addr))}
                        >
                          ×
                        </button>
                      </Badge>
                    )
                  })}
                </div>
              )}
            </div>

            {/* Boot Order */}
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Boot Order</Label>
                  <p className="text-xs text-muted-foreground">
                    Device boot priority (first device has highest priority)
                  </p>
                </div>
              </div>

              <div className="flex flex-wrap gap-2">
                {bootOrder.map((device, index) => (
                  <Badge key={device} variant="secondary" className="gap-1.5">
                    <span className="text-xs text-muted-foreground">{index + 1}.</span>
                    {device === "hd" && "Hard Disk"}
                    {device === "cdrom" && "CD-ROM"}
                    {device === "network" && "Network (PXE)"}
                    <button
                      type="button"
                      className="ml-1 hover:text-destructive"
                      onClick={() => setBootOrder(bootOrder.filter((d) => d !== device))}
                    >
                      ×
                    </button>
                  </Badge>
                ))}
              </div>

              <div className="flex gap-2">
                <Select
                  value=""
                  onValueChange={(v) => {
                    if (v && !bootOrder.includes(v)) {
                      setBootOrder([...bootOrder, v])
                    }
                  }}
                >
                  <SelectTrigger className="w-[180px]">
                    <SelectValue placeholder="Add boot device..." />
                  </SelectTrigger>
                  <SelectContent>
                    {!bootOrder.includes("hd") && (
                      <SelectItem value="hd">Hard Disk</SelectItem>
                    )}
                    {!bootOrder.includes("cdrom") && (
                      <SelectItem value="cdrom">CD-ROM</SelectItem>
                    )}
                    {!bootOrder.includes("network") && (
                      <SelectItem value="network">Network (PXE)</SelectItem>
                    )}
                  </SelectContent>
                </Select>
                {bootOrder.length > 1 && (
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      // Move last device up
                      const idx = bootOrder.length - 1
                      if (idx > 0) {
                        const newOrder = [...bootOrder]
                        const temp = newOrder[idx - 1]!
                        newOrder[idx - 1] = newOrder[idx]!
                        newOrder[idx] = temp
                        setBootOrder(newOrder)
                      }
                    }}
                  >
                    ↑
                  </Button>
                )}
              </div>
            </div>

            {/* Network */}
            <div className="space-y-2">
              <Label>Network Interface</Label>
              <div className="grid grid-cols-2 gap-2">
                <Select value={networkType} onValueChange={(v) => setNetworkType(v as "user" | "bridge")}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="user">User (NAT)</SelectItem>
                    <SelectItem value="bridge">Bridge</SelectItem>
                  </SelectContent>
                </Select>
                <Select value={networkModel} onValueChange={(v) => setNetworkModel(v as "virtio" | "e1000" | "rtl8139")}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="virtio">VirtIO</SelectItem>
                    <SelectItem value="e1000">Intel E1000</SelectItem>
                    <SelectItem value="rtl8139">RTL8139</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              {networkType === "bridge" && (
                <Input
                  placeholder="Bridge name (e.g. br0, vmbr0)"
                  value={networkBridge}
                  onChange={(e) => setNetworkBridge(e.target.value)}
                />
              )}
            </div>
          </TabsContent>
        </Tabs>

        {createVM.error && (
          <Alert variant="destructive" className="mt-4">
            <AlertCircle className="size-4" />
            <AlertDescription>{createVM.error.message}</AlertDescription>
          </Alert>
        )}

        <DialogFooter className="mt-4">
          <Button variant="outline" onClick={() => setOpen(false)} disabled={isCreating}>
            Cancel
          </Button>
          <Button
            data-testid="vm-create-submit"
            onClick={handleSubmit}
            disabled={
              !name.trim() ||
              !node ||
              isCreating ||
              !!cpuError ||
              !!memoryError ||
              !!diskError ||
              (isoSource === "uploaded" && !selectedIsoId) ||
              (isoSource === "url" && (!isoUrl.trim() || isUrlValid === false || isUrlChecking))
            }
          >
            {isCreating ? (
              <>
                <Loader2 className="size-4 animate-spin mr-2" />
                Creating VM...
              </>
            ) : (
              <>
                <Plus className="size-4 mr-2" />
                Create VM
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
