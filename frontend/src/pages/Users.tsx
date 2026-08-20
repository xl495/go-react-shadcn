import { useEffect, useMemo, useState } from "react"
import { Link } from "react-router-dom"
import { useQueryClient } from "@tanstack/react-query"
import { Can } from "@/components/auth/Can"
import { api } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { DICT, useDict } from "@/lib/dict"
import { formatDateTime } from "@/lib/format"
import { roleLabel, translateApiError, useI18n } from "@/lib/i18n"
import { P } from "@/lib/perms"
import { useRoles, useUsers } from "@/lib/queries"
import { Avatar } from "@/components/ui/avatar"
import { AvatarField } from "@/components/ui/avatar-field"
import { Badge } from "@/components/ui/badge"
import { DictSelect } from "@/components/ui/dict-select"
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import type { User } from "@/lib/types"

const emptyForm = {
  username: "",
  password: "",
  nickname: "",
  email: "",
  phone: "",
  gender: "",
  department: "",
  title: "",
  remark: "",
  status: "active",
}

export function UsersPage() {
  const { can } = useAuth()
  const { t } = useI18n()
  const qc = useQueryClient()
  const genderDict = useDict(DICT.gender)
  const statusDict = useDict(DICT.userStatus)
  const deptDict = useDict(DICT.department)
  const { data: usersPage, error: usersError } = useUsers({ pageSize: 500 })
  const needRoles = can(P.roleList) || can(P.userRoles) || can(P.userCreate) || can(P.userUpdate)
  const { data: rolesPage } = useRoles({ pageSize: 500 }, needRoles)
  const users = usersPage?.items ?? []
  const roles = rolesPage?.items ?? []
  const [error, setError] = useState("")
  const [query, setQuery] = useState("")
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<User | null>(null)
  const [form, setForm] = useState(emptyForm)
  const [roleIds, setRoleIds] = useState<number[]>([])

  useEffect(() => {
    if (usersError) setError(translateApiError(usersError as Error, t))
  }, [usersError, t])

  async function reload() {
    await qc.invalidateQueries({ queryKey: ["users"] })
    if (needRoles) await qc.invalidateQueries({ queryKey: ["roles"] })
  }

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return users
    return users.filter((u) =>
      [
        u.username,
        u.nickname,
        u.email,
        u.phone,
        u.title,
        u.department,
        genderDict.label(u.gender),
        deptDict.label(u.department),
        statusDict.label(u.status),
      ]
        .join(" ")
        .toLowerCase()
        .includes(q),
    )
  }, [users, query, genderDict.items, deptDict.items, statusDict.items])

  function openCreate() {
    setEditing(null)
    setForm(emptyForm)
    setRoleIds([])
    setOpen(true)
  }

  function openEdit(u: User) {
    setEditing(u)
    setForm({
      username: u.username,
      password: "",
      nickname: u.nickname ?? "",
      email: u.email ?? "",
      phone: u.phone ?? "",
      gender: u.gender ?? "",
      department: u.department ?? "",
      title: u.title ?? "",
      remark: u.remark ?? "",
      status: u.status || "active",
    })
    setRoleIds(u.roles.map((r) => r.id))
    setOpen(true)
  }

  function toggle(id: number) {
    setRoleIds((cur) => (cur.includes(id) ? cur.filter((x) => x !== id) : [...cur, id]))
  }

  async function submit() {
    setError("")
    try {
      if (editing) {
        await api.updateUser(editing.id, {
          nickname: form.nickname,
          email: form.email,
          phone: form.phone,
          gender: form.gender,
          department: form.department,
          title: form.title,
          remark: form.remark,
          status: form.status,
          password: form.password || undefined,
        })
        if (can(P.userRoles)) {
          await api.assignUserRoles(editing.id, roleIds)
        }
      } else {
        await api.createUser({
          username: form.username,
          password: form.password,
          nickname: form.nickname,
          email: form.email,
          phone: form.phone,
          gender: form.gender,
          department: form.department,
          title: form.title,
          remark: form.remark,
          status: form.status,
          roleIds,
        })
      }
      setOpen(false)
      await reload()
    } catch (e) {
      setError(translateApiError(e, t))
    }
  }

  async function remove(user: User) {
    if (!confirm(t("users.confirmDelete", { name: user.username }))) return
    try {
      await api.deleteUser(user.id)
      await reload()
    } catch (e) {
      setError(translateApiError(e, t))
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{t("users.title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("users.subtitle")}</p>
        </div>
        <div className="flex items-center gap-2">
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t("users.search")}
            className="w-56"
          />
          <Can perm={P.userCreate}>
            <Button onClick={openCreate}>{t("users.create")}</Button>
          </Can>
        </div>
      </div>
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      <div className="rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>{t("users.avatar")}</TableHead>
              <TableHead>{t("users.account")}</TableHead>
              <TableHead>{t("users.nickname")}</TableHead>
              <TableHead>{t("users.phone")}</TableHead>
              <TableHead>{t("users.email")}</TableHead>
              <TableHead>{t("users.gender")}</TableHead>
              <TableHead>{t("users.department")}</TableHead>
              <TableHead>{t("users.jobTitle")}</TableHead>
              <TableHead>{t("users.roles")}</TableHead>
              <TableHead>{t("app.status")}</TableHead>
              <TableHead>{t("users.lastLogin")}</TableHead>
              <TableHead>{t("users.createdAt")}</TableHead>
              <TableHead className="text-right">{t("app.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filtered.map((u) => (
              <TableRow key={u.id}>
                <TableCell className="tabular-nums text-muted-foreground">{u.id}</TableCell>
                <TableCell>
                  <Avatar name={u.nickname || u.username} src={u.avatar} />
                </TableCell>
                <TableCell className="font-medium">{u.username}</TableCell>
                <TableCell>{u.nickname || "—"}</TableCell>
                <TableCell>{u.phone || "—"}</TableCell>
                <TableCell>{u.email || "—"}</TableCell>
                <TableCell>{genderDict.label(u.gender)}</TableCell>
                <TableCell>{deptDict.label(u.department)}</TableCell>
                <TableCell>{u.title || "—"}</TableCell>
                <TableCell>
                  <div className="flex flex-wrap gap-1">
                    {u.roles.map((r) => (
                      <Badge key={r.id} variant="muted">
                        {roleLabel(r.code, r.name, t)}
                      </Badge>
                    ))}
                  </div>
                </TableCell>
                <TableCell>
                  <Badge variant={u.status === "active" ? "default" : "muted"}>
                    {statusDict.label(u.status)}
                  </Badge>
                </TableCell>
                <TableCell className="whitespace-nowrap text-xs">
                  {formatDateTime(u.lastLoginAt)}
                  {u.lastLoginIp ? (
                    <span className="mt-0.5 block text-muted-foreground">{u.lastLoginIp}</span>
                  ) : null}
                </TableCell>
                <TableCell className="whitespace-nowrap text-xs">{formatDateTime(u.createdAt)}</TableCell>
                <TableCell className="text-right">
                  <div className="flex justify-end gap-1">
                    <Button asChild variant="ghost" size="sm">
                      <Link to={`/users/${u.id}`}>{t("users.detail")}</Link>
                    </Button>
                    <Can perm={P.userUpdate}>
                      <Button variant="ghost" size="sm" onClick={() => openEdit(u)}>
                        {t("app.edit")}
                      </Button>
                    </Can>
                    <Can perm={P.userDelete}>
                      <Button variant="ghost" size="sm" onClick={() => remove(u)}>
                        {t("app.delete")}
                      </Button>
                    </Can>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{editing ? t("users.edit") : t("users.create")}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-3 sm:grid-cols-2">
            {editing ? (
              <div className="sm:col-span-2">
                <AvatarField
                  name={editing.nickname || editing.username}
                  src={editing.avatar}
                  onFile={async (file) => {
                    const next = await api.uploadUserAvatar(editing.id, file)
                    setEditing(next)
                    await reload()
                  }}
                />
              </div>
            ) : null}
            <div className="grid gap-1.5">
              <Label htmlFor="nu">{t("login.username")}</Label>
              <Input
                id="nu"
                value={form.username}
                disabled={!!editing}
                onChange={(e) => setForm((f) => ({ ...f, username: e.target.value }))}
              />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="np">{t("login.password")}</Label>
              <Input
                id="np"
                type="password"
                placeholder={editing ? t("users.passwordKeep") : ""}
                value={form.password}
                onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
              />
            </div>
            {(
              [
                ["nickname", "users.nickname"],
                ["phone", "users.phone"],
                ["email", "users.email"],
                ["title", "users.jobTitle"],
                ["remark", "users.remark"],
              ] as const
            ).map(([key, label]) => (
              <div key={key} className="grid gap-1.5">
                <Label htmlFor={key}>{t(label)}</Label>
                <Input
                  id={key}
                  value={form[key]}
                  onChange={(e) => setForm((f) => ({ ...f, [key]: e.target.value }))}
                />
              </div>
            ))}
            <DictSelect
              id="gd"
              label={t("users.gender")}
              value={form.gender}
              items={genderDict.items}
              allowEmpty
              emptyLabel={t("users.genderUnset")}
              onChange={(value) => setForm((f) => ({ ...f, gender: value }))}
            />
            <DictSelect
              id="dept"
              label={t("users.department")}
              value={form.department}
              items={deptDict.items}
              allowEmpty
              emptyLabel={t("app.optional")}
              onChange={(value) => setForm((f) => ({ ...f, department: value }))}
            />
            <DictSelect
              id="st"
              label={t("app.status")}
              value={form.status}
              items={statusDict.items}
              onChange={(value) => setForm((f) => ({ ...f, status: value }))}
            />
            {(can(P.userRoles) || !editing) && (
              <div className="grid gap-2 sm:col-span-2">
                <Label>{t("users.roles")}</Label>
                {roles.map((r) => (
                  <label key={r.id} className="flex items-center gap-2 text-sm">
                    <Checkbox checked={roleIds.includes(r.id)} onCheckedChange={() => toggle(r.id)} />
                    {roleLabel(r.code, r.name, t)}
                    <span className="text-muted-foreground">({r.code})</span>
                  </label>
                ))}
              </div>
            )}
          </div>
          <DialogFooter>
            <Button onClick={() => void submit()}>{editing ? t("app.save") : t("app.create")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
