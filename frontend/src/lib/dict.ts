import { useEffect, useMemo, useState } from "react"
import { api } from "@/lib/api"
import type { DictItem } from "@/lib/types"

export const DICT = {
  userStatus: "sys_user_status",
  gender: "sys_gender",
  department: "sys_department",
  yesNo: "sys_yes_no",
} as const

export function dictLabel(items: DictItem[], value?: string | null, fallback = "—") {
  if (!value) return fallback
  return items.find((it) => it.value === value)?.label ?? value
}

export function useDict(code: string) {
  const [items, setItems] = useState<DictItem[]>([])
  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    let cancelled = false
    api
      .lookupDict(code)
      .then((pack) => {
        if (!cancelled) setItems(pack.items ?? [])
      })
      .catch(() => {
        if (!cancelled) setItems([])
      })
      .finally(() => {
        if (!cancelled) setLoaded(true)
      })
    return () => {
      cancelled = true
    }
  }, [code])

  const byValue = useMemo(() => {
    const map = new Map<string, string>()
    for (const it of items) map.set(it.value, it.label)
    return map
  }, [items])

  return { items, loaded, label: (value?: string | null) => (value ? byValue.get(value) ?? value : "—") }
}
