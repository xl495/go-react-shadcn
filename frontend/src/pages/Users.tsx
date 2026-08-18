import { useEffect, useMemo, useState } from "react"
import { Link } from "react-router-dom"
import { Can } from "@/components/auth/Can"
import { api } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { formatDateTime } from "@/lib/format"
import { roleLabel, translateApiError, useI18n } from "@/lib/i18n"
import { P } from "@/lib/perms"
import { Avatar } from "@/components/ui/avatar"
import { AvatarField } from "@/components/ui/avatar-field"
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import type { Role, User } from "@/lib/types"

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
  const [users, setUsers] = useState<User[]>([])
  const [roles, setRoles] = useState<Role[]>([])
  const [error, setError] = useState("")
  const [query, setQuery] = useState("")
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<User | null>(null)
  const [form, setForm] = useState(emptyForm)
  const [roleIds, setRoleIds] = useState<number[]>([])

  async function reload() {
    const u = await api.users()
    setUsers(u)
    if (can(P.roleList) || can(P.userRoles) || can(P.userCreate) || can(P.userUpdate)) {
      try {
        setRoles(await api.roles())
      } catch {
        setRoles([])
      }
    }
  }

  useEffect(() => {
    reload().catch((e: Error) => setError(translateApiError(e, t)))
  }, [])

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return users
    return users.filter((u) =>
      [u.username, u.nickname, u.email, u.phone, u.department, u.title]
        .join(" ")
        .toLowerCase()
        .includes(q),
    )
  }, [users, query])

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
                <TableCell>{genderLabel(u.gender, t)}</TableCell>
                <TableCell>{u.department || "—"}</TableCell>
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
                    {u.status === "active" ? t("app.active") : t("app.disabled")}
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
                ["department", "users.department"],
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
            <div className="grid gap-1.5">
              <Label htmlFor="gd">{t("users.gender")}</Label>
              <select
                id="gd"
                className="h-9 rounded-md border border-input bg-card px-3 text-sm"
                value={form.gender}
                onChange={(e) => setForm((f) => ({ ...f, gender: e.target.value }))}
              >
                <option value="">{t("users.genderUnset")}</option>
                <option value="male">{t("users.genderMale")}</option>
                <option value="female">{t("users.genderFemale")}</option>
                <option value="other">{t("users.genderOther")}</option>
              </select>
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="st">{t("app.status")}</Label>
              <select
                id="st"
                className="h-9 rounded-md border border-input bg-card px-3 text-sm"
                value={form.status}
                onChange={(e) => setForm((f) => ({ ...f, status: e.target.value }))}
              >
                <option value="active">{t("app.active")}</option>
                <option value="disabled">{t("app.disabled")}</option>
              </select>
            </div>
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

function genderLabel(value: string, t: (key: string) => string) {
  if (value === "male") return t("users.genderMale")
  if (value === "female") return t("users.genderFemale")
  if (value === "other") return t("users.genderOther")
  return "—"
}
