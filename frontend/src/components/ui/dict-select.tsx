import { Label } from "@/components/ui/label"
import { cn } from "@/lib/utils"
import type { DictItem } from "@/lib/types"

export function DictSelect({
  id,
  label,
  value,
  items,
  onChange,
  allowEmpty,
  emptyLabel = "—",
  className,
}: {
  id: string
  label?: string
  value: string
  items: DictItem[]
  onChange: (value: string) => void
  allowEmpty?: boolean
  emptyLabel?: string
  className?: string
}) {
  return (
    <div className={cn("grid gap-1.5", className)}>
      {label ? <Label htmlFor={id}>{label}</Label> : null}
      <select
        id={id}
        className="h-9 rounded-md border border-input bg-card px-3 text-sm"
        value={value}
        onChange={(e) => onChange(e.target.value)}
      >
        {allowEmpty ? <option value="">{emptyLabel}</option> : null}
        {items.map((it) => (
          <option key={it.value} value={it.value}>
            {it.label}
          </option>
        ))}
      </select>
    </div>
  )
}
