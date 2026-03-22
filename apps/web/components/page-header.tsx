import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { cn } from "@/lib/utils";

export interface PageHeaderProps {
  backHref: string;
  backLabel: string;
  title: string;
  subtitle?: string;
  icon?: React.ReactNode;
  badges?: React.ReactNode;
  className?: string;
}

export function PageHeader({
  backHref,
  backLabel,
  title,
  subtitle,
  icon,
  badges,
  className,
}: PageHeaderProps) {
  return (
    <div className="space-y-4">
      <Link
        href={backHref}
        className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
      >
        <ArrowLeft className="size-4" />
        {backLabel}
      </Link>

      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          {icon && (
            <div className="flex size-10 items-center justify-center rounded-lg bg-secondary">
              {icon}
            </div>
          )}
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-xl font-semibold text-foreground">{title}</h1>
              {badges}
            </div>
            {subtitle && (
              <p className="text-sm text-muted-foreground">{subtitle}</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
