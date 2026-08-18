import { useEffect, useState } from "react"
import { Can } from "@/components/auth/Can"
import { api } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { translateApiError, useI18n } from "@/lib/i18n"
import { P } from "@/lib/perms"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import type { DictItem, DictType } from "@/lib/types"

export function DictsPage() {
  const { can } = useAuth()
  const { t } = useI18n()
  const [types, setTypes] = useState<DictType[]>([])
  const [active, setActive] = useState<DictType | null>(null)
  const [items, setItems] = useState<DictItem[]>([])
  const [error, setError] = useState("")
  const [typeOpen, setTypeOpen] = useState(false)
  const [itemOpen, setItemOpen] = useState(false)
  const [typeForm, setTypeForm] = useState({ code: "", name: "", remark: "" })
  const [itemForm, setItemForm] = useState({ label: "", value: "", sort: "0", remark: "" })

  async function reloadTypes() {
    const rows = await api.dicts()
    setTypes(rows)
    if (active) {
      const next = rows.find((r) => r.id === active.id) ?? rows[0] ?? null
      setActive(next)
      if (next) setItems(await api.dictItems(next.id))
      else setItems([])
    } else if (rows[0]) {
      setActive(rows[0])
      setItems(await api.dictItems(rows[0].id))
    }
  }

  useEffect(() => {
    reloadTypes().catch((e: Error) => setError(translateApiError(e, t)))
  }, [])

  async function selectType(row: DictType) {
    setActive(row)
    try {
      setItems(await api.dictItems(row.id))
    } catch (e) {
      setError(translateApiError(e, t))
    }
  }

  async function createType() {
    setError("")
    try {
      await api.createDict(typeForm)
      setTypeOpen(false)
      setTypeForm({ code: "", name: "", remark: "" })
      await reloadTypes()
    } catch (e) {
      setError(translateApiError(e, t))
    }
  }

  async function removeType(row: DictType) {
    if (!confirm(t("dict.confirmDeleteType", { name: row.name }))) return
    try {
      await api.deleteDict(row.id)
      setActive(null)
      await reloadTypes()
    } catch (e) {
      setError(translateApiError(e, t))
    }
  }

  async function createItem() {
    if (!active) return
    setError("")
    try {
      await api.createDictItem(active.id, {
        label: itemForm.label,
        value: itemForm.value,
        sort: Number(itemForm.sort) || 0,
        remark: itemForm.remark,
      })
      setItemOpen(false)
      setItemForm({ label: "", value: "", sort: "0", remark: "" })
      setItems(await api.dictItems(active.id))
    } catch (e) {
      setError(translateApiError(e, t))
    }
  }

  async function removeItem(row: DictItem) {
    if (!confirm(t("dict.confirmDeleteItem", { name: row.label }))) return
    try {
      await api.deleteDictItem(row.id)
      if (active) setItems(await api.dictItems(active.id))
    } catch (e) {
      setError(translateApiError(e, t))
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-end justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{t("dict.title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("dict.subtitle")}</p>
        </div>
        <Can perm={P.dictCreate}>
          <Button onClick={() => setTypeOpen(true)}>{t("dict.createType")}</Button>
        </Can>
      </div>
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      <div className="grid gap-4 lg:grid-cols-[280px_1fr]">
        <div className="rounded-lg border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("dict.type")}</TableHead>
                {can(P.dictDelete) ? <TableHead className="text-right">{t("app.actions")}</TableHead> : null}
              </TableRow>
            </TableHeader>
            <TableBody>
              {types.map((row) => (
                <TableRow
                  key={row.id}
                  className={active?.id === row.id ? "bg-accent" : "cursor-pointer"}
                  onClick={() => void selectType(row)}
                >
                  <TableCell>
                    <div className="font-medium">{row.name}</div>
                    <div className="font-mono text-xs text-muted-foreground">{row.code}</div>
                  </TableCell>
                  {can(P.dictDelete) ? (
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation()
                          void removeType(row)
                        }}
                      >
                        {t("app.delete")}
                      </Button>
                    </TableCell>
                  ) : null}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium">{active ? active.name : t("dict.pickType")}</h3>
            {active ? (
              <Can perm={P.dictItemCreate}>
                <Button size="sm" onClick={() => setItemOpen(true)}>
                  {t("dict.createItem")}
                </Button>
              </Can>
            ) : null}
          </div>
          <div className="rounded-lg border bg-card">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("dict.label")}</TableHead>
                  <TableHead>{t("dict.value")}</TableHead>
                  <TableHead>{t("dict.sort")}</TableHead>
                  <TableHead>{t("app.status")}</TableHead>
                  {can(P.dictItemDelete) ? (
                    <TableHead className="text-right">{t("app.actions")}</TableHead>
                  ) : null}
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell>{row.label}</TableCell>
                    <TableCell className="font-mono text-xs">{row.value}</TableCell>
                    <TableCell>{row.sort}</TableCell>
                    <TableCell>
                      <Badge variant={row.status === "active" ? "default" : "muted"}>
                        {row.status === "active" ? t("app.active") : t("app.disabled")}
                      </Badge>
                    </TableCell>
                    {can(P.dictItemDelete) ? (
                      <TableCell className="text-right">
                        <Button variant="ghost" size="sm" onClick={() => void removeItem(row)}>
                          {t("app.delete")}
                        </Button>
                      </TableCell>
                    ) : null}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </div>
      </div>

      <Dialog open={typeOpen} onOpenChange={setTypeOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("dict.createType")}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-3">
            <div className="grid gap-1.5">
              <Label>{t("app.code")}</Label>
              <Input value={typeForm.code} onChange={(e) => setTypeForm((f) => ({ ...f, code: e.target.value }))} />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("app.name")}</Label>
              <Input value={typeForm.name} onChange={(e) => setTypeForm((f) => ({ ...f, name: e.target.value }))} />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("app.description")}</Label>
              <Input value={typeForm.remark} onChange={(e) => setTypeForm((f) => ({ ...f, remark: e.target.value }))} />
            </div>
          </div>
          <DialogFooter>
            <Button onClick={() => void createType()}>{t("app.create")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={itemOpen} onOpenChange={setItemOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("dict.createItem")}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-3">
            <div className="grid gap-1.5">
              <Label>{t("dict.label")}</Label>
              <Input value={itemForm.label} onChange={(e) => setItemForm((f) => ({ ...f, label: e.target.value }))} />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("dict.value")}</Label>
              <Input value={itemForm.value} onChange={(e) => setItemForm((f) => ({ ...f, value: e.target.value }))} />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("dict.sort")}</Label>
              <Input value={itemForm.sort} onChange={(e) => setItemForm((f) => ({ ...f, sort: e.target.value }))} />
            </div>
          </div>
          <DialogFooter>
            <Button onClick={() => void createItem()}>{t("app.create")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
