import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { cn } from "@/utils/cn"

const EMPTY = "__empty__"

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
  items: { value: string; label: string }[]
  onChange: (value: string) => void
  allowEmpty?: boolean
  emptyLabel?: string
  className?: string
}) {
  return (
    <div className={cn("grid gap-1.5", className)}>
      {label ? <Label htmlFor={id}>{label}</Label> : null}
      <Select
        value={allowEmpty && value === "" ? EMPTY : value}
        onValueChange={(next) => onChange(next === EMPTY ? "" : next)}
      >
        <SelectTrigger id={id}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {allowEmpty ? <SelectItem value={EMPTY}>{emptyLabel}</SelectItem> : null}
          {items.map((it) => (
            <SelectItem key={it.value} value={it.value}>
              {it.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
