import { useMemo } from "react"
import { useQuery } from "@tanstack/react-query"
import { api } from "@/api/client"
import type { DictItem } from "@/types"

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
  const query = useQuery({
    queryKey: ["dicts", "lookup", code],
    queryFn: () => api.lookupDict(code),
  })
  const items = useMemo(() => query.data?.items ?? [], [query.data?.items])
  const byValue = useMemo(() => {
    const map = new Map<string, string>()
    for (const it of items) map.set(it.value, it.label)
    return map
  }, [items])

  return {
    items,
    loaded: query.isSuccess || query.isError,
    label: (value?: string | null) => (value ? (byValue.get(value) ?? value) : "—"),
  }
}
