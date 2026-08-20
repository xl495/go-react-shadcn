import { useEffect, useState } from "react"
import { Can } from "@/components/auth/Can"
import { api } from "@/api/client"
import { useAuth } from "@/providers/auth"
import { permLabel, translateApiError, useI18n } from "@/providers/i18n"
import { P } from "@/constants/perms"
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
import type { Permission } from "@/types"

const empty = { name: "", code: "", path: "", method: "GET", kind: "button", description: "" }

export function PermissionsPage() {
  const { can } = useAuth()
  const { t } = useI18n()
  const [items, setItems] = useState<Permission[]>([])
  const [error, setError] = useState("")
  const [open, setOpen] = useState(false)
  const [form, setForm] = useState(empty)

  async function reload() {
    setItems(await api.permissions())
  }

  useEffect(() => {
    reload().catch((e: Error) => setError(translateApiError(e, t)))
  }, [])

  async function create() {
    setError("")
    try {
      await api.createPermission(form)
      setOpen(false)
      setForm(empty)
      await reload()
    } catch (e) {
      setError(translateApiError(e, t) || t("permissions.createFailed"))
    }
  }

  async function remove(p: Permission) {
    if (!confirm(t("permissions.confirmDelete", { code: p.code }))) return
    try {
      await api.deletePermission(p.id)
      await reload()
    } catch (e) {
      setError(translateApiError(e, t) || t("permissions.deleteFailed"))
    }
  }

  return (
    <div className="mx-auto max-w-5xl">
      <div className="flex items-end justify-between gap-4">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{t("permissions.title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("permissions.subtitle")}</p>
        </div>
        <Can perm={P.permCreate}>
          <Button onClick={() => setOpen(true)}>{t("permissions.create")}</Button>
        </Can>
      </div>
      {error ? <p className="mt-4 text-sm text-destructive">{error}</p> : null}
      <div className="mt-6 rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("app.name")}</TableHead>
              <TableHead>{t("app.code")}</TableHead>
              <TableHead>{t("permissions.type")}</TableHead>
              <TableHead>{t("permissions.rule")}</TableHead>
              {can(P.permDelete) ? <TableHead className="text-right">{t("app.actions")}</TableHead> : null}
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((p) => (
              <TableRow key={p.id}>
                <TableCell className="font-medium">{permLabel(p.code, p.name, t)}</TableCell>
                <TableCell className="font-mono text-xs">{p.code}</TableCell>
                <TableCell>
                  <Badge variant={p.kind === "button" ? "default" : "muted"}>
                    {t(`kinds.${p.kind || "api"}`)}
                  </Badge>
                </TableCell>
                <TableCell>
                  <Badge variant="outline">
                    {p.method} {p.path}
                  </Badge>
                </TableCell>
                {can(P.permDelete) ? (
                  <TableCell className="text-right">
                    <Button variant="ghost" size="sm" onClick={() => remove(p)}>
                      {t("app.delete")}
                    </Button>
                  </TableCell>
                ) : null}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("permissions.create")}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-3">
            {(
              [
                ["name", "app.name", "permissions.phName"],
                ["code", "app.code", "permissions.phCode"],
                ["path", "permissions.path", "permissions.phPath"],
                ["method", "permissions.method", "permissions.phMethod"],
                ["description", "app.description", "app.optional"],
              ] as const
            ).map(([key, labelKey, phKey]) => (
              <div key={key} className="grid gap-1.5">
                <Label htmlFor={key}>{t(labelKey)}</Label>
                <Input
                  id={key}
                  placeholder={t(phKey)}
                  value={form[key]}
                  onChange={(e) => setForm((f) => ({ ...f, [key]: e.target.value }))}
                />
              </div>
            ))}
            <div className="grid gap-1.5">
              <Label htmlFor="kind">{t("permissions.type")}</Label>
              <select
                id="kind"
                className="h-9 rounded-md border border-input bg-card px-3 text-sm"
                value={form.kind}
                onChange={(e) => setForm((f) => ({ ...f, kind: e.target.value }))}
              >
                <option value="menu">{t("kinds.menu")}</option>
                <option value="button">{t("kinds.button")}</option>
                <option value="api">{t("kinds.api")}</option>
              </select>
            </div>
          </div>
          <DialogFooter>
            <Button onClick={() => void create()}>{t("app.create")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
