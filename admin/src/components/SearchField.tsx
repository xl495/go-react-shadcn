import { useState, type FormEvent, type ReactNode } from "react"
import { useI18n } from "@/providers/i18n"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { cn } from "@/utils/cn"

export function useSyncedDraft<T>(committed: T) {
  const [draft, setDraft] = useState(committed)
  const [seen, setSeen] = useState(committed)
  if (!Object.is(committed, seen)) {
    setSeen(committed)
    setDraft(committed)
  }
  return [draft, setDraft] as const
}

export function FilterForm({
  children,
  onSubmit,
  className,
}: {
  children: ReactNode
  onSubmit: () => void
  className?: string
}) {
  function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    onSubmit()
  }

  return (
    <form className={cn("flex flex-wrap items-end gap-2", className)} onSubmit={handleSubmit}>
      {children}
    </form>
  )
}

export function SearchField({
  id,
  label,
  value,
  onChange,
  placeholder,
  className,
  inputClassName,
}: {
  id: string
  label?: string
  value: string
  onChange: (value: string) => void
  placeholder?: string
  className?: string
  inputClassName?: string
}) {
  return (
    <div className={cn("grid gap-1.5", className)}>
      {label ? <Label htmlFor={id}>{label}</Label> : null}
      <Input
        id={id}
        value={value}
        placeholder={placeholder}
        className={inputClassName}
        onChange={(e) => onChange(e.target.value)}
      />
    </div>
  )
}

export function SearchSubmitButton() {
  const { t } = useI18n()
  return <Button type="submit">{t("app.search")}</Button>
}
