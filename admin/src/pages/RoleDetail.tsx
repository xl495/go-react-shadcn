import { useState } from "react"
import { Link, useParams } from "react-router-dom"
import { toast } from "sonner"
import { useAuth } from "@/providers/auth"
import { permLabel, roleDesc, roleLabel, translateApiError, useI18n } from "@/providers/i18n"
import { P } from "@/constants/perms"
import {
  useAllPermissions,
  useAssignRolePermissions,
  useRole,
  useUpdateRole,
} from "@/hooks/queries"
import { PageFallback } from "@/components/PageFallback"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { UnsavedGuard } from "@/hooks/unsaved"
import type { Permission, Role } from "@/types"

const SCOPES = ["all", "dept_sub", "dept", "self"] as const
const KINDS = ["menu", "button", "api"] as const

function assignedIds(role: Role) {
  if (role.permissionIds?.length) return role.permissionIds
  return (role.permissions ?? []).map((p) => p.id)
}

function sameIdSet(a: number[], b: number[]) {
  if (a.length !== b.length) return false
  const seen = new Set(a)
  return b.every((id) => seen.has(id))
}

export function RoleDetailPage() {
  const { id } = useParams()
  const { t } = useI18n()
  const roleId = Number(id)
  const { data: role, error, isLoading } = useRole(roleId)

  if (!roleId) {
    return <p className="text-sm text-destructive">{t("errors.40420")}</p>
  }
  if (isLoading) return <PageFallback />
  if (error) return <p className="text-sm text-destructive">{translateApiError(error as Error, t)}</p>
  if (!role) return null
  return <RoleEditor key={role.id} role={role} />
}

function RoleEditor({ role }: { role: Role }) {
  const { can } = useAuth()
  const { t } = useI18n()
  const { data: perms = [] } = useAllPermissions()
  const updateRole = useUpdateRole()
  const assignPerms = useAssignRolePermissions()
  const [name, setName] = useState(role.name)
  const [description, setDescription] = useState(role.description ?? "")
  const [dataScope, setDataScope] = useState(role.dataScope || "self")
  const [permissionIds, setPermissionIds] = useState(() => assignedIds(role))
  const [permFilter, setPermFilter] = useState("")
  const [openKinds, setOpenKinds] = useState<Record<string, boolean>>({ menu: true, button: false, api: false })

  const baseline = assignedIds(role)
  const dirtyPerms = !sameIdSet(permissionIds, baseline)
  const dirtyMeta =
    name !== role.name || description !== (role.description ?? "") || dataScope !== (role.dataScope || "self")
  const assigned = new Set(permissionIds)
  const needle = permFilter.trim().toLowerCase()
  const editablePerms = can(P.rolePerms) && !assignPerms.isPending

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{roleLabel(role.code, role.name, t)}</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            {role.code}
            {roleDesc(role.code, role.description ?? "", t)
              ? ` · ${roleDesc(role.code, role.description ?? "", t)}`
              : ""}
          </p>
        </div>
        <Button asChild variant="outline" size="sm">
          <Link to="/roles">{t("roles.backToList")}</Link>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("roles.detail")}</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 sm:grid-cols-2">
          <div className="grid gap-1.5">
            <Label htmlFor="role-name">{t("app.name")}</Label>
            <Input
              id="role-name"
              value={name}
              disabled={!can(P.roleUpdate)}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="role-code">{t("app.code")}</Label>
            <Input id="role-code" value={role.code} disabled />
          </div>
          <div className="grid gap-1.5 sm:col-span-2">
            <Label htmlFor="role-desc">{t("app.description")}</Label>
            <Input
              id="role-desc"
              value={description}
              disabled={!can(P.roleUpdate)}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="role-scope">{t("roles.dataScope")}</Label>
            <Select value={dataScope} onValueChange={setDataScope} disabled={!can(P.roleUpdate)}>
              <SelectTrigger id="role-scope">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {SCOPES.map((s) => (
                  <SelectItem key={s} value={s}>
                    {t(`roles.scope.${s}`)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          {can(P.roleUpdate) ? (
            <div className="flex items-end">
              <Button
                disabled={!dirtyMeta || updateRole.isPending}
                onClick={() =>
                  updateRole.mutate(
                    { id: role.id, body: { name, description, dataScope } },
                    {
                      onSuccess: () => toast.success(t("app.saved")),
                    },
                  )
                }
              >
                {t("app.save")}
              </Button>
            </div>
          ) : null}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-3">
          <CardTitle className="text-base">{t("roles.permissions")}</CardTitle>
          {can(P.rolePerms) && dirtyPerms ? (
            <Button
              size="sm"
              disabled={assignPerms.isPending}
              onClick={() =>
                assignPerms.mutate(
                  { id: role.id, permissionIds },
                  {
                    onSuccess: () => toast.success(t("app.saved")),
                  },
                )
              }
            >
              {t("roles.savePerms")}
            </Button>
          ) : null}
        </CardHeader>
        <CardContent className="space-y-3">
          {!can(P.rolePerms) ? (
            <p className="text-xs text-muted-foreground">{t("roles.readonlyHint")}</p>
          ) : null}
          <div className="grid max-w-sm gap-1.5">
            <Label htmlFor="perm-q">{t("roles.filterPerms")}</Label>
            <Input
              id="perm-q"
              value={permFilter}
              placeholder={t("roles.filterPerms")}
              onChange={(e) => setPermFilter(e.target.value)}
            />
          </div>
          {KINDS.map((kind) => (
            <PermKindBlock
              key={kind}
              kind={kind}
              items={perms}
              needle={needle}
              expanded={needle ? true : openKinds[kind] !== false}
              onToggle={() => setOpenKinds((cur) => ({ ...cur, [kind]: !expandedFor(cur, kind, needle) }))}
              roleId={role.id}
              assigned={assigned}
              editable={editablePerms}
              onTogglePerm={(permId, want) => {
                if (!can(P.rolePerms)) return
                setPermissionIds((cur) => (want ? [...cur, permId] : cur.filter((item) => item !== permId)))
              }}
            />
          ))}
        </CardContent>
      </Card>
      <UnsavedGuard dirty={dirtyMeta || dirtyPerms} />
    </div>
  )
}

function expandedFor(openKinds: Record<string, boolean>, kind: string, needle: string) {
  if (needle) return true
  return openKinds[kind] !== false
}

function PermKindBlock({
  kind,
  items,
  needle,
  expanded,
  onToggle,
  roleId,
  assigned,
  editable,
  onTogglePerm,
}: {
  kind: (typeof KINDS)[number]
  items: Permission[]
  needle: string
  expanded: boolean
  onToggle: () => void
  roleId: number
  assigned: Set<number>
  editable: boolean
  onTogglePerm: (id: number, want: boolean) => void
}) {
  const { t } = useI18n()
  const filtered = items.filter((p) => {
    if ((p.kind || "api") !== kind) return false
    if (!needle) return true
    return `${p.code} ${p.name} ${p.path} ${p.method}`.toLowerCase().includes(needle)
  })
  if (filtered.length === 0) return null
  return (
    <div>
      <button
        type="button"
        className="mb-2 text-[11px] tracking-wider text-muted-foreground uppercase hover:text-foreground"
        onClick={onToggle}
      >
        {t(`kinds.${kind}`)} ({filtered.length})
      </button>
      {expanded ? (
        <div className="flex max-h-72 flex-wrap gap-3 overflow-y-auto pr-1">
          {filtered.map((p) => {
            const checked = assigned.has(p.id)
            const boxId = `rp-${roleId}-${p.id}`
            return (
              <div key={p.id} className="flex items-center gap-2 rounded-md border px-2 py-1 text-xs">
                <Checkbox
                  id={boxId}
                  checked={checked}
                  disabled={!editable}
                  onCheckedChange={(value) => {
                    const want = value === true
                    if (want === checked) return
                    onTogglePerm(p.id, want)
                  }}
                />
                <label htmlFor={boxId} className={editable ? "cursor-pointer" : "cursor-default"}>
                  {permLabel(p.code, p.name, t)}
                </label>
                <Badge variant={p.kind === "button" ? "default" : "muted"}>
                  {p.kind === "button" ? t("kinds.button") : `${p.method} ${p.path}`}
                </Badge>
              </div>
            )
          })}
        </div>
      ) : null}
    </div>
  )
}
