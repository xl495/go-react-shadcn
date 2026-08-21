import { cn } from "@/utils/cn"

export function GraMark({ className }: { className?: string }) {
  return (
    <img
      src="/gra-mark.png"
      alt=""
      width={32}
      height={32}
      className={cn("rounded-md", className)}
    />
  )
}

export function GraWord({ className }: { className?: string }) {
  return <span className={cn("font-display tracking-tight", className)}>gra</span>
}
