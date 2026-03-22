export interface ConfigItem {
  label: string;
  value: string;
}

export interface ConfigListProps {
  items: ConfigItem[];
  className?: string;
}

export function ConfigList({ items, className }: ConfigListProps) {
  return (
    <div className="space-y-3">
      {items.map((item) => (
        <div
          key={item.label}
          className="flex items-center justify-between text-sm"
        >
          <span className="text-muted-foreground">{item.label}</span>
          <span className="text-foreground font-medium">{item.value}</span>
        </div>
      ))}
    </div>
  );
}
