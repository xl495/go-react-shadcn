import { useMemo, useState } from "react"
import { toast } from "sonner"
import { Can } from "@/components/auth/Can"
import { useAuth } from "@/providers/auth"
import { useI18n } from "@/providers/i18n"
import { P } from "@/constants/perms"
import { ConfirmAlert, EmptyTableRow, ResourceTable } from "@/components/feedback"
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
import { Switch } from "@/components/ui/switch"
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import type { NavMenu } from "@/api/menus"
import {
  useCreateNavMenu,
  useDeleteNavMenu,
  useNavMenus,
  useReorderNavMenus,
  useUpdateNavMenu,
} from "@/hooks/queries"

type Flat = { row: NavMenu; depth: number }

function flatten(nodes: NavMenu[], depth = 0): Flat[] {
  const out: Flat[] = []
  for (const n of nodes) {
    out.push({ row: n, depth })
    if (n.children?.length) out.push(...flatten(n.children, depth + 1))
  }
  return out
}

const emptyForm = {
  name: "",
  code: "",
  routePath: "",
  component: "",
  icon: "",
  permCode: "",
  parentId: "",
  sort: "0",
  hidden: false,
}

export function MenusPage() {
  const { can } = useAuth()
  const { t } = useI18n()
  const [audience, setAudience] = useState<"admin" | "web">("admin")
  const { data, isLoading, error } = useNavMenus(audience)
  const rows = flatten(data ?? [])
  const createMenu = useCreateNavMenu()
  const updateMenu = useUpdateNavMenu()
  const deleteMenu = useDeleteNavMenu()
  const reorder = useReorderNavMenus()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<NavMenu | null>(null)
  const [form, setForm] = useState(emptyForm)
  const [pending, setPending] = useState<NavMenu | null>(null)

  const parents = useMemo(
    () => rows.filter((r) => !editing || r.row.id !== editing.id).map((r) => r.row),
    [rows, editing],
  )

  function openCreate() {
    setEditing(null)
    setForm(emptyForm)
    setOpen(true)
  }

  function openEdit(row: NavMenu) {
    setEditing(row)
    setForm({
      name: row.name,
      code: row.code,
      routePath: row.routePath,
      component: row.component,
      icon: row.icon,
      permCode: row.permCode,
      parentId: row.parentId ? String(row.parentId) : "",
      sort: String(row.sort ?? 0),
      hidden: row.hidden,
    })
    setOpen(true)
  }

  async function onSave() {
    const body = {
      audience,
      name: form.name,
      code: form.code,
      routePath: form.routePath,
      component: form.component,
      icon: form.icon,
      permCode: form.permCode,
      parentId: form.parentId ? Number(form.parentId) : null,
      sort: Number(form.sort) || 0,
      hidden: form.hidden,
    }
    try {
      if (editing) {
        await updateMenu.mutateAsync({ id: editing.id, body })
      } else {
        await createMenu.mutateAsync(body)
      }
      toast.success(t("app.saved"))
      setOpen(false)
    } catch {
      // toasted
    }
  }

  async function move(row: NavMenu, dir: -1 | 1) {
    const siblings = rows.filter((r) => (r.row.parentId ?? 0) === (row.parentId ?? 0)).map((r) => r.row)
    const idx = siblings.findIndex((s) => s.id === row.id)
    const swap = siblings[idx + dir]
    if (!swap) return
    try {
      await reorder.mutateAsync([
        { id: row.id, sort: swap.sort, parentId: row.parentId ?? null },
        { id: swap.id, sort: row.sort, parentId: swap.parentId ?? null },
      ])
    } catch {
      // toasted
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{t("menus.title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("menus.subtitle")}</p>
        </div>
        <div className="flex items-center gap-2">
          <Button type="button" variant={audience === "admin" ? "default" : "outline"} onClick={() => setAudience("admin")}>
            {t("menus.admin")}
          </Button>
          <Button type="button" variant={audience === "web" ? "default" : "outline"} onClick={() => setAudience("web")}>
            {t("menus.web")}
          </Button>
          <Can perm={P.menuCreate}>
            <Button type="button" onClick={openCreate}>
              {t("app.create")}
            </Button>
          </Can>
        </div>
      </div>
      {error ? <p className="text-sm text-destructive">{String((error as Error).message)}</p> : null}
      <ResourceTable
        loading={isLoading}
        page={1}
        pageSize={Math.max(rows.length, 1)}
        total={rows.length}
        onPageChange={() => undefined}
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("app.name")}</TableHead>
              <TableHead>{t("app.code")}</TableHead>
              <TableHead>{t("menus.route")}</TableHead>
              <TableHead>{t("menus.sort")}</TableHead>
              <TableHead className="text-right">{t("app.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.length === 0 ? (
              <EmptyTableRow colSpan={5} />
            ) : (
              rows.map(({ row, depth }) => (
                <TableRow key={row.id}>
                  <TableCell>
                    <span style={{ paddingLeft: depth * 16 }} className="inline-flex items-center gap-2">
                      {row.name}
                      {row.hidden ? <Badge variant="muted">{t("menus.hidden")}</Badge> : null}
                      {row.isSystem ? <Badge variant="outline">{t("menus.system")}</Badge> : null}
                    </span>
                  </TableCell>
                  <TableCell className="font-mono text-xs">{row.code}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">{row.routePath}</TableCell>
                  <TableCell>{row.sort}</TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-1">
                      <Can perm={P.menuUpdate}>
                        <Button type="button" size="sm" variant="ghost" onClick={() => void move(row, -1)}>
                          {t("menus.up")}
                        </Button>
                        <Button type="button" size="sm" variant="ghost" onClick={() => void move(row, 1)}>
                          {t("menus.down")}
                        </Button>
                        <Button type="button" size="sm" variant="outline" onClick={() => openEdit(row)}>
                          {t("app.edit")}
                        </Button>
                      </Can>
                      <Can perm={P.menuDelete}>
                        <Button
                          type="button"
                          size="sm"
                          variant="ghost"
                          disabled={row.isSystem}
                          onClick={() => setPending(row)}
                        >
                          {t("app.delete")}
                        </Button>
                      </Can>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
          <TableCaption className="sr-only">{t("menus.title")}</TableCaption>
        </Table>
      </ResourceTable>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editing ? t("app.edit") : t("app.create")}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-3">
            {(
              [
                ["name", "app.name"],
                ["code", "app.code"],
                ["routePath", "menus.route"],
                ["component", "menus.component"],
                ["icon", "menus.icon"],
                ["permCode", "menus.perm"],
                ["sort", "menus.sort"],
              ] as const
            ).map(([key, label]) => (
              <div key={key} className="grid gap-1.5">
                <Label htmlFor={key}>{t(label)}</Label>
                <Input
                  id={key}
                  value={form[key]}
                  disabled={key === "code" && !!editing}
                  onChange={(e) => setForm((p) => ({ ...p, [key]: e.target.value }))}
                />
              </div>
            ))}
            <div className="grid gap-1.5">
              <Label htmlFor="parent">{t("menus.parent")}</Label>
              <select
                id="parent"
                className="h-9 rounded-md border bg-background px-2 text-sm"
                value={form.parentId}
                onChange={(e) => setForm((p) => ({ ...p, parentId: e.target.value }))}
              >
                <option value="">{t("app.optional")}</option>
                {parents.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex items-center gap-2">
              <Switch id="hidden" checked={form.hidden} onCheckedChange={(hidden) => setForm((p) => ({ ...p, hidden }))} />
              <Label htmlFor="hidden" className="font-normal">
                {t("menus.hidden")}
              </Label>
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              {t("app.cancel")}
            </Button>
            <Button type="button" onClick={() => void onSave()} disabled={!can(editing ? P.menuUpdate : P.menuCreate)}>
              {t("app.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmAlert
        open={!!pending}
        onOpenChange={(next) => {
          if (!next) setPending(null)
        }}
        title={t("app.delete")}
        description={pending ? t("menus.confirmDelete", { name: pending.name }) : ""}
        onConfirm={() => {
          if (!pending) return
          deleteMenu.mutate(pending.id, {
            onSuccess: () => toast.success(t("app.saved")),
          })
          setPending(null)
        }}
      />
    </div>
  )
}
