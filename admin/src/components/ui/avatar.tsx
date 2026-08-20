import { cn } from "@/utils/cn"
import { initials } from "@/utils/format"

export function Avatar({
  name,
  src,
  className,
}: {
  name?: string
  src?: string | null
  className?: string
}) {
  return (
    <span
      className={cn(
        "relative inline-flex size-8 shrink-0 items-center justify-center overflow-hidden rounded-full bg-foreground text-xs font-medium text-background",
        className,
      )}
    >
      {src ? (
        <img src={src} alt={name || ""} className="size-full object-cover" />
      ) : (
        initials(name)
      )}
    </span>
  )
}
