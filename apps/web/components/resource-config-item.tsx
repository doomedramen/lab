import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";

export interface ResourceConfigItemProps {
  icon: React.ReactNode;
  label: string;
  value: string;
  detail?: string;
  badge?: string;
  className?: string;
}

export function ResourceConfigItem({
  icon,
  label,
  value,
  detail,
  badge,
  className,
}: ResourceConfigItemProps) {
  return (
    <div
      className={cn(
        "flex items-center gap-3 rounded-md border border-border bg-secondary/30 px-4 py-3",
        className,
      )}
    >
      {icon}
      <div className="flex-1">
        <div className="text-sm font-medium text-foreground">{label}</div>
        {detail && (
          <div className="text-xs text-muted-foreground">{detail}</div>
        )}
      </div>
      {badge ? (
        <Badge variant="secondary" className="text-[11px]">
          {badge}
        </Badge>
      ) : (
        <span className="text-sm font-medium text-foreground">{value}</span>
      )}
    </div>
  );
}
