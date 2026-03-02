import Link from "next/link"
import { Card, CardContent, CardHeader, CardTitle } from "@workspace/ui/components/card"
import { StatusBadge, TagList } from "@workspace/components/lab-shared"
import { Cpu, MemoryStick, HardDrive } from "lucide-react"
import { cn } from "@/lib/utils"

export interface Service {
  name: string
  type: "vm" | "container"
  status: string
}

export interface EntityCardProps {
  id: string
  href: string
  name: string
  description: string
  status: string
  node: string
  tags: string[]
  runningServices: number
  totalServices: number
  totalCpu: number
  totalMemory: number
  totalDisk: number
  icon?: React.ReactNode
  className?: string
}

export function EntityCard({
  id,
  href,
  name,
  description,
  status,
  node,
  tags,
  runningServices,
  totalServices,
  totalCpu,
  totalMemory,
  totalDisk,
  icon,
  className,
}: EntityCardProps) {
  return (
    <Link href={href}>
      <Card className={cn("hover:bg-secondary/20 transition-colors cursor-pointer h-full", className)}>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-2 text-sm font-medium">
              {icon}
              {name}
            </CardTitle>
            <StatusBadge status={status} />
          </div>
          <p className="text-xs text-muted-foreground mt-1">{description}</p>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center gap-2 text-xs">
            <span className="text-muted-foreground">Services:</span>
            <span className="text-foreground font-medium">
              {runningServices}/{totalServices} running
            </span>
          </div>

          <div className="flex items-center gap-4 pt-2 border-t border-border text-xs text-muted-foreground">
            <span className="flex items-center gap-1">
              <Cpu className="size-3" />
              {totalCpu}c
            </span>
            <span className="flex items-center gap-1">
              <MemoryStick className="size-3" />
              {totalMemory} GB
            </span>
            <span className="flex items-center gap-1">
              <HardDrive className="size-3" />
              {totalDisk} GB
            </span>
          </div>

          <div className="flex items-center justify-between">
            <div className="flex items-center gap-1 text-xs text-muted-foreground">
              <span className="text-[11px]">{node}</span>
            </div>
            <TagList tags={tags.slice(0, 2)} />
          </div>
        </CardContent>
      </Card>
    </Link>
  )
}
