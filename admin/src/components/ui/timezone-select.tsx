import { TIMEZONES } from "@/constants/timezones"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

export function TimezoneSelect({
  id,
  value,
  onChange,
  className,
}: {
  id?: string
  value: string
  onChange: (value: string) => void
  className?: string
}) {
  const extra = value && !(TIMEZONES as readonly string[]).includes(value) ? [value] : []
  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger id={id} className={className}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {[...TIMEZONES, ...extra].map((tz) => (
          <SelectItem key={tz} value={tz}>
            {tz}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
