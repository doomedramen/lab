import Link from "next/link";
import { Server, Monitor, Box } from "lucide-react";
import { StatusBadge } from "@/components/lab-shared";
import { cn } from "@/lib/utils";

export interface NodeListItemProps {
  id: string;
  href: string;
  name: string;
  ip: string;
  status: string;
  cpuUsage: number;
  memoryUsed: number;
  memoryTotal: number;
  vms?: number;
  containers?: number;
  type?: "host" | "vm" | "container";
  className?: string;
}

export function NodeListItem({
  id,
  href,
  name,
  ip,
  status,
  cpuUsage,
  memoryUsed,
  memoryTotal,
  vms,
  containers,
  type = "host",
  className,
}: NodeListItemProps) {
  const Icon = type === "vm" ? Monitor : type === "container" ? Box : Server;

  return (
    <Link
      href={href}
      className={cn(
        "flex items-center gap-4 rounded-md border border-border bg-secondary/30 px-4 py-3 hover:bg-secondary/60 transition-colors",
        className,
      )}
    >
      <Icon className="size-5 text-muted-foreground shrink-0" />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-foreground">{name}</span>
          <StatusBadge status={status} />
        </div>
        <span className="text-xs text-muted-foreground">{ip}</span>
      </div>
      <div className="hidden md:flex items-center gap-6">
        <div className="text-center">
          <div className="text-xs text-muted-foreground">CPU</div>
          <div className="text-sm font-medium text-foreground">{cpuUsage}%</div>
        </div>
        <div className="text-center">
          <div className="text-xs text-muted-foreground">Memory</div>
          <div className="text-sm font-medium text-foreground">
            {Math.round(memoryUsed)}/{memoryTotal}{" "}
            {type === "host" ? "GB" : "GB"}
          </div>
        </div>
        {vms !== undefined && (
          <div className="text-center">
            <div className="text-xs text-muted-foreground">VMs</div>
            <div className="text-sm font-medium text-foreground">{vms}</div>
          </div>
        )}
        {containers !== undefined && (
          <div className="text-center">
            <div className="text-xs text-muted-foreground">CTs</div>
            <div className="text-sm font-medium text-foreground">
              {containers}
            </div>
          </div>
        )}
      </div>
    </Link>
  );
}
