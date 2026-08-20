import { useEffect, useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { Can } from "@/components/auth/Can"
import { api } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { permLabel, roleDesc, roleLabel, translateApiError, useI18n } from "@/lib/i18n"
import { P } from "@/lib/perms"
import { usePermissions, useRoles } from "@/lib/queries"
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
import type { Role } from "@/lib/types"

export function RolesPage() {
  const { can } = useAuth()
  const { t } = useI18n()
  const qc = useQueryClient()
  const { data: rolesPage, error: rolesError } = useRoles({ pageSize: 500 })
  const { data: perms = [] } = usePermissions()
  const roles = rolesPage?.items ?? []
  const [error, setError] = useState("")
  const [open, setOpen] = useState(false)
  const [name, setName] = useState("")
  const [code, setCode] = useState("")
  const [description, setDescription] = useState("")
  const [permissionIds, setPermissionIds] = useState<number[]>([])

  useEffect(() => {
    if (rolesError) setError(translateApiError(rolesError as Error, t))
  }, [rolesError, t])

  async function reload() {
    await qc.invalidateQueries({ queryKey: ["roles"] })
    await qc.invalidateQueries({ queryKey: ["permissions"] })
  }

  function toggle(id: number) {
    setPermissionIds((cur) => (cur.includes(id) ? cur.filter((x) => x !== id) : [...cur, id]))
  }

  async function createRole() {
    setError("")
    try {
      await api.createRole({ name, code, description, permissionIds })
      setOpen(false)
      setName("")
      setCode("")
      setDescription("")
      setPermissionIds([])
      await reload()
    } catch (e) {
      setError(translateApiError(e, t) || t("roles.createFailed"))
    }
  }

  async function savePerms(role: Role, next: number[]) {
    try {
      await api.assignRolePermissions(role.id, next)
      await reload()
    } catch (e) {
      setError(translateApiError(e, t) || t("roles.assignFailed"))
    }
  }

  async function remove(role: Role) {
    if (!confirm(t("roles.confirmDelete", { name: roleLabel(role.code, role.name, t) }))) return
    try {
      await api.deleteRole(role.id)
      await reload()
    } catch (e) {
      setError(translateApiError(e, t) || t("roles.deleteFailed"))
    }
  }

  return (
    <div className="mx-auto max-w-5xl">
      <div className="flex items-end justify-between gap-4">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{t("roles.title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("roles.subtitle")}</p>
        </div>
        <Can perm={P.roleCreate}>
          <Button onClick={() => setOpen(true)}>{t("roles.create")}</Button>
        </Can>
      </div>
      {error ? <p className="mt-4 text-sm text-destructive">{error}</p> : null}
      <div className="mt-6 space-y-4">
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
                      ? ` · ${roleDesc(role.code, role.description, t)}`
                      : ""}
                  </p>
                </div>
                <Can perm={P.roleDelete}>
                  <Button variant="ghost" size="sm" onClick={() => remove(role)}>
                    {t("app.delete")}
                  </Button>
                </Can>
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
                                    void savePerms(role, next)
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
            <Button onClick={() => void createRole()}>{t("app.create")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
