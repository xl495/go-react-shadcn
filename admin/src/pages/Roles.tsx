import { useState } from "react"
import { toast } from "sonner"
import { Can } from "@/components/auth/Can"
import { useAuth } from "@/providers/auth"
import { permLabel, roleDesc, roleLabel, translateApiError, useI18n } from "@/providers/i18n"
import { P } from "@/constants/perms"
import {
  PAGE_SIZE,
  PICKER_PAGE_SIZE,
  useAssignRolePermissions,
  useCreateRole,
  useDeleteRole,
  usePermissions,
  useRoles,
  useUpdateRole,
} from "@/hooks/queries"
import { ConfirmAlert, EmptyState, PaginationBar, TableSkeleton } from "@/components/feedback"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import type { Role } from "@/types"

const SCOPES = ["all", "dept_sub", "dept", "self"] as const

export function RolesPage() {
  const { can } = useAuth()
  const { t } = useI18n()
  const [page, setPage] = useState(1)
  const { data: rolesPage, isLoading, error: rolesError } = useRoles({ page, pageSize: PAGE_SIZE })
  const { data: permsPage } = usePermissions({ pageSize: PICKER_PAGE_SIZE })
  const roles = rolesPage?.items ?? []
  const perms = permsPage?.items ?? []
  const createRole = useCreateRole()
  const updateRole = useUpdateRole()
  const deleteRole = useDeleteRole()
  const assignPerms = useAssignRolePermissions()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState("")
  const [code, setCode] = useState("")
  const [description, setDescription] = useState("")
  const [dataScope, setDataScope] = useState("self")
  const [permissionIds, setPermissionIds] = useState<number[]>([])
  const [pending, setPending] = useState<Role | null>(null)

  function toggle(id: number) {
    setPermissionIds((cur) => (cur.includes(id) ? cur.filter((x) => x !== id) : [...cur, id]))
  }

  async function create() {
    try {
      await createRole.mutateAsync({ name, code, description, dataScope, permissionIds })
      toast.success(t("app.saved"))
      setOpen(false)
      setName("")
      setCode("")
      setDescription("")
      setDataScope("self")
      setPermissionIds([])
    } catch (e) {
      toast.error(translateApiError(e, t) || t("roles.createFailed"))
    }
  }

  return (
    <div className="mx-auto max-w-5xl space-y-4">
      <div className="flex items-end justify-between gap-4">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{t("roles.title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("roles.subtitle")}</p>
        </div>
        <Can perm={P.roleCreate}>
          <Button onClick={() => setOpen(true)}>{t("roles.create")}</Button>
        </Can>
      </div>
      {rolesError ? <p className="text-sm text-destructive">{translateApiError(rolesError, t)}</p> : null}
      {isLoading ? <TableSkeleton rows={4} cols={3} /> : null}
      {!isLoading && roles.length === 0 ? <EmptyState /> : null}
      <div className="space-y-4">
        {roles.map((role) => {
          const assigned = new Set((role.permissions ?? []).map((p) => p.id))
          return (
            <div key={role.id} className="rounded-xl border bg-card p-5">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <h2 className="text-base font-semibold">{roleLabel(role.code, role.name, t)}</h2>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {role.code}
                    {role.description || roleDesc(role.code, "", t)
                      ? ` · ${roleDesc(role.code, role.description ?? "", t)}`
                      : ""}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <Can perm={P.roleUpdate}>
                    <label className="flex items-center gap-2 text-xs">
                      <span className="text-muted-foreground">{t("roles.dataScope")}</span>
                      <select
                        className="h-8 rounded-md border border-input bg-card px-2"
                        value={role.dataScope || "self"}
                        onChange={(e) => {
                          updateRole.mutate(
                            { id: role.id, body: { dataScope: e.target.value } },
                            {
                              onSuccess: () => toast.success(t("app.saved")),
                              onError: (err) => toast.error(translateApiError(err, t)),
                            },
                          )
                        }}
                      >
                        {SCOPES.map((s) => (
                          <option key={s} value={s}>
                            {t(`roles.scope.${s}`)}
                          </option>
                        ))}
                      </select>
                    </label>
                  </Can>
                  <Can perm={P.roleDelete}>
                    <Button variant="ghost" size="sm" onClick={() => setPending(role)}>
                      {t("app.delete")}
                    </Button>
                  </Can>
                </div>
              </div>
              <div className="mt-4 space-y-3">
                {(["menu", "button", "api"] as const).map((kind) => {
                  const items = perms.filter((p) => (p.kind || "api") === kind)
                  if (items.length === 0) return null
                  return (
                    <div key={kind}>
                      <p className="mb-2 text-[11px] tracking-wider text-muted-foreground uppercase">
                        {t(`kinds.${kind}`)}
                      </p>
                      <div className="flex flex-wrap gap-3">
                        {items.map((p) => {
                          const checked = assigned.has(p.id)
                          return (
                            <label
                              key={p.id}
                              className="flex items-center gap-2 rounded-md border px-2 py-1 text-xs"
                            >
                              {can(P.rolePerms) ? (
                                <Checkbox
                                  checked={checked}
                                  onCheckedChange={() => {
                                    const next = checked
                                      ? [...assigned].filter((id) => id !== p.id)
                                      : [...assigned, p.id]
                                    assignPerms.mutate(
                                      { id: role.id, permissionIds: next },
                                      {
                                        onError: (e) =>
                                          toast.error(translateApiError(e, t) || t("roles.assignFailed")),
                                      },
                                    )
                                  }}
                                />
                              ) : null}
                              <span>{permLabel(p.code, p.name, t)}</span>
                              <Badge variant={p.kind === "button" ? "default" : "muted"}>
                                {p.kind === "button" ? t("kinds.button") : `${p.method} ${p.path}`}
                              </Badge>
                            </label>
                          )
                        })}
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          )
        })}
      </div>
      <PaginationBar
        page={page}
        pageSize={PAGE_SIZE}
        total={rolesPage?.total ?? 0}
        onPageChange={setPage}
      />

      <ConfirmAlert
        open={!!pending}
        onOpenChange={(next) => {
          if (!next) setPending(null)
        }}
        title={t("app.delete")}
        description={
          pending ? t("roles.confirmDelete", { name: roleLabel(pending.code, pending.name, t) }) : ""
        }
        onConfirm={() => {
          if (!pending) return
          deleteRole.mutate(pending.id, {
            onSuccess: () => toast.success(t("app.saved")),
            onError: (e) => toast.error(translateApiError(e, t) || t("roles.deleteFailed")),
          })
          setPending(null)
        }}
      />

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{t("roles.create")}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-3">
            <div className="grid gap-1.5">
              <Label htmlFor="rn">{t("app.name")}</Label>
              <Input id="rn" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="rc">{t("app.code")}</Label>
              <Input id="rc" value={code} onChange={(e) => setCode(e.target.value)} />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="rd">{t("app.description")}</Label>
              <Input id="rd" value={description} onChange={(e) => setDescription(e.target.value)} />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="rs">{t("roles.dataScope")}</Label>
              <select
                id="rs"
                className="h-9 rounded-md border border-input bg-card px-3 text-sm"
                value={dataScope}
                onChange={(e) => setDataScope(e.target.value)}
              >
                {SCOPES.map((s) => (
                  <option key={s} value={s}>
                    {t(`roles.scope.${s}`)}
                  </option>
                ))}
              </select>
            </div>
            <div className="grid gap-2">
              <Label>{t("roles.permissions")}</Label>
              {perms.map((p) => (
                <label key={p.id} className="flex items-center gap-2 text-sm">
                  <Checkbox
                    checked={permissionIds.includes(p.id)}
                    onCheckedChange={() => toggle(p.id)}
                  />
                  {permLabel(p.code, p.name, t)}
                  <span className="text-muted-foreground">
                    {p.method} {p.path}
                  </span>
                </label>
              ))}
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
