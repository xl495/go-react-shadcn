import { useState } from "react"
import { Link } from "react-router-dom"
import { toast } from "sonner"
import { Can } from "@/components/auth/Can"
import { api } from "@/api/client"
import { useAuth } from "@/providers/auth"
import { DICT, useDict } from "@/hooks/dict"
import { useUserListParams } from "@/hooks/list-params"
import { formatDateTime } from "@/utils/format"
import { roleLabel, translateApiError, useI18n } from "@/providers/i18n"
import { P } from "@/constants/perms"
import {
  PAGE_SIZE,
  PICKER_PAGE_SIZE,
  useAssignUserRoles,
  useCreateUser,
  useDeleteUser,
  useRoles,
  useUpdateUser,
  useUsers,
} from "@/hooks/queries"
import { ConfirmAlert, EmptyTableRow, PaginationBar, TableSkeleton } from "@/components/feedback"
import { Avatar } from "@/components/ui/avatar"
import { AvatarField } from "@/components/ui/avatar-field"
import { Badge } from "@/components/ui/badge"
import { FilterForm, SearchField, SearchSubmitButton, useSyncedDraft } from "@/components/SearchField"
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
import { Switch } from "@/components/ui/switch"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { TimezoneSelect } from "@/components/ui/timezone-select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import type { User } from "@/types"

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
  timezone: "Asia/Shanghai",
  marketingOptIn: true,
  status: "active",
}

export function UsersPage() {
  return <UserDirectory kind="admin" />
}

export function WebUsersPage() {
  return <UserDirectory kind="web" />
}

function UserDirectory({ kind }: { kind: "admin" | "web" }) {
  const isWeb = kind === "web"
  const { can } = useAuth()
  const { t } = useI18n()
  const genderDict = useDict(DICT.gender)
  const statusDict = useDict(DICT.userStatus)
  const deptDict = useDict(DICT.department)
  const [{ q, page, gender, status, department, roleId }, setParams] = useUserListParams()
  const [draftQ, setDraftQ] = useSyncedDraft(q)
  const [draftGender, setDraftGender] = useSyncedDraft(gender)
  const [draftStatus, setDraftStatus] = useSyncedDraft(status)
  const [draftDepartment, setDraftDepartment] = useSyncedDraft(department)
  const [draftRoleId, setDraftRoleId] = useSyncedDraft(roleId)
  const { data, isLoading, error } = useUsers({
    page,
    pageSize: PAGE_SIZE,
    q: q || undefined,
    gender: gender || undefined,
    status: status || undefined,
    department: isWeb ? undefined : department || undefined,
    roleId: roleId ?? undefined,
    kind,
  })
  const needRoles = can(P.roleList) || can(P.userRoles) || can(P.userCreate) || can(P.userUpdate)
  const { data: rolesPage } = useRoles({ pageSize: PICKER_PAGE_SIZE }, needRoles)
  const users = data?.items ?? []
  const roles = rolesPage?.items ?? []
  const filtered = Boolean(q || gender || status || department || roleId)
  const draftFiltered = Boolean(
    draftQ !== q ||
      draftGender !== gender ||
      draftStatus !== status ||
      draftDepartment !== department ||
      draftRoleId !== roleId,
  )

  function searchUsers() {
    void setParams({
      q: draftQ.trim(),
      gender: draftGender,
      status: draftStatus,
      department: draftDepartment,
      roleId: draftRoleId,
      page: 1,
    })
  }

  function resetUsers() {
    setDraftQ("")
    setDraftGender("")
    setDraftStatus("")
    setDraftDepartment("")
    setDraftRoleId(null)
    void setParams({ q: "", page: 1, gender: "", status: "", department: "", roleId: null })
  }
  const createUser = useCreateUser()
  const updateUser = useUpdateUser()
  const deleteUser = useDeleteUser()
  const assignRoles = useAssignUserRoles()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<User | null>(null)
  const [form, setForm] = useState(emptyForm)
  const [roleIds, setRoleIds] = useState<number[]>([])
  const [pending, setPending] = useState<User | null>(null)

  function openCreate() {
    setEditing(null)
    setForm(emptyForm)
    const member = roles.find((r) => r.code === "member")
    setRoleIds(isWeb && member ? [member.id] : [])
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
      timezone: u.timezone || "Asia/Shanghai",
      marketingOptIn: u.marketingOptIn ?? true,
      status: u.status || "active",
    })
    setRoleIds((u.roles ?? []).map((r) => r.id))
    setOpen(true)
  }

  function toggle(id: number) {
    setRoleIds((cur) => (cur.includes(id) ? cur.filter((x) => x !== id) : [...cur, id]))
  }

  async function submit() {
    try {
      if (editing) {
        await updateUser.mutateAsync({
          id: editing.id,
          body: {
            nickname: form.nickname,
            email: form.email,
            phone: form.phone,
            gender: form.gender,
            department: form.department,
            title: form.title,
            remark: form.remark,
            timezone: form.timezone,
            marketingOptIn: form.marketingOptIn,
            status: form.status,
            password: form.password || undefined,
          },
        })
        if (can(P.userRoles)) {
          await assignRoles.mutateAsync({ id: editing.id, roleIds })
        }
      } else {
        await createUser.mutateAsync({
          username: form.username,
          password: form.password,
          nickname: form.nickname,
          email: form.email,
          phone: form.phone,
          gender: form.gender,
          department: isWeb ? "" : form.department,
          title: isWeb ? "" : form.title,
          remark: form.remark,
          timezone: form.timezone,
          marketingOptIn: form.marketingOptIn,
          status: form.status,
          kind,
          roleIds,
        })
      }
      toast.success(t("app.saved"))
      setOpen(false)
    } catch (e) {
      toast.error(translateApiError(e, t))
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">
            {isWeb ? t("webUsers.title") : t("users.title")}
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            {isWeb ? t("webUsers.subtitle") : t("users.subtitle")}
          </p>
        </div>
        <Can perm={P.userCreate}>
          <Button onClick={openCreate}>{isWeb ? t("webUsers.create") : t("users.create")}</Button>
        </Can>
      </div>
      <FilterForm onSubmit={searchUsers}>
        <SearchField
          id="user-q"
          label={t("app.search")}
          value={draftQ}
          placeholder={isWeb ? t("webUsers.search") : t("users.search")}
          inputClassName="w-64"
          onChange={setDraftQ}
        />
        <DictSelect
          id="user-gender"
          className="w-36"
          label={t("users.gender")}
          value={draftGender}
          items={genderDict.items}
          allowEmpty
          emptyLabel={t("app.all")}
          onChange={setDraftGender}
        />
        <DictSelect
          id="user-status"
          className="w-36"
          label={t("app.status")}
          value={draftStatus}
          items={statusDict.items}
          allowEmpty
          emptyLabel={t("app.all")}
          onChange={setDraftStatus}
        />
        {!isWeb ? (
        <DictSelect
          id="user-dept"
          className="w-36"
          label={t("users.department")}
          value={draftDepartment}
          items={deptDict.items}
          allowEmpty
          emptyLabel={t("app.all")}
          onChange={setDraftDepartment}
        />
        ) : null}
        {can(P.roleList) ? (
          <div className="grid w-40 gap-1.5">
            <Label htmlFor="user-role">{t("users.roles")}</Label>
            <Select
              value={draftRoleId == null ? "__all__" : String(draftRoleId)}
              onValueChange={(value) => setDraftRoleId(value === "__all__" ? null : Number(value))}
            >
              <SelectTrigger id="user-role">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__all__">{t("app.all")}</SelectItem>
                {roles.map((r) => (
                  <SelectItem key={r.id} value={String(r.id)}>
                    {roleLabel(r.code, r.name, t)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        ) : null}
        <SearchSubmitButton />
        {filtered || draftFiltered ? (
          <Button type="button" variant="outline" onClick={resetUsers}>
            {t("app.resetFilters")}
          </Button>
        ) : null}
      </FilterForm>
      {error ? <p className="text-sm text-destructive">{translateApiError(error, t)}</p> : null}
      <div className="rounded-lg border bg-card">
        {isLoading ? (
          <TableSkeleton rows={8} cols={6} />
        ) : (
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
                {!isWeb ? <TableHead>{t("users.department")}</TableHead> : null}
                {!isWeb ? <TableHead>{t("users.jobTitle")}</TableHead> : null}
                <TableHead>{t("users.roles")}</TableHead>
                <TableHead>{t("app.status")}</TableHead>
                <TableHead>{t("users.lastLogin")}</TableHead>
                <TableHead>{t("users.createdAt")}</TableHead>
                <TableHead className="text-right">{t("app.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.length === 0 ? (
                <EmptyTableRow colSpan={isWeb ? 12 : 14} />
              ) : (
                users.map((u) => (
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
                    {!isWeb ? <TableCell>{deptDict.label(u.department)}</TableCell> : null}
                    {!isWeb ? <TableCell>{u.title || "—"}</TableCell> : null}
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {(u.roles ?? []).map((r) => (
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
                          <Button variant="ghost" size="sm" onClick={() => setPending(u)}>
                            {t("app.delete")}
                          </Button>
                        </Can>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        )}
      </div>
      <PaginationBar
        page={page}
        pageSize={PAGE_SIZE}
        total={data?.total ?? 0}
        onPageChange={(next) => void setParams({ page: next })}
      />

      <ConfirmAlert
        open={!!pending}
        onOpenChange={(next) => {
          if (!next) setPending(null)
        }}
        title={t("app.delete")}
        description={pending ? t("users.confirmDelete", { name: pending.username }) : ""}
        onConfirm={() => {
          if (!pending) return
          deleteUser.mutate(pending.id, {
            onSuccess: () => toast.success(t("app.saved")),
            onError: (e) => toast.error(translateApiError(e, t)),
          })
          setPending(null)
        }}
      />

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-lg">
          <DialogHeader>
          <DialogTitle>{editing ? t("users.edit") : isWeb ? t("webUsers.create") : t("users.create")}</DialogTitle>
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
                    toast.success(t("app.saved"))
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
            {(isWeb
              ? ([
                  ["nickname", "users.nickname"],
                  ["phone", "users.phone"],
                  ["email", "users.email"],
                  ["remark", "users.remark"],
                ] as const)
              : ([
                  ["nickname", "users.nickname"],
                  ["phone", "users.phone"],
                  ["email", "users.email"],
                  ["title", "users.jobTitle"],
                  ["remark", "users.remark"],
                ] as const)
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
            {!isWeb ? (
            <DictSelect
              id="dept"
              label={t("users.department")}
              value={form.department}
              items={deptDict.items}
              allowEmpty
              emptyLabel={t("app.optional")}
              onChange={(value) => setForm((f) => ({ ...f, department: value }))}
            />
            ) : null}
            <DictSelect
              id="st"
              label={t("app.status")}
              value={form.status}
              items={statusDict.items}
              onChange={(value) => setForm((f) => ({ ...f, status: value }))}
            />
            <div className="grid gap-1.5">
              <Label htmlFor="utz">{t("users.timezone")}</Label>
              <TimezoneSelect
                id="utz"
                value={form.timezone}
                onChange={(timezone) => setForm((f) => ({ ...f, timezone }))}
              />
            </div>
            <div className="flex items-center gap-2 text-sm sm:col-span-2">
              <Switch
                id="umkt"
                checked={form.marketingOptIn}
                onCheckedChange={(checked) => setForm((f) => ({ ...f, marketingOptIn: checked }))}
              />
              <Label htmlFor="umkt" className="font-normal">
                {t("users.marketingOptIn")}
              </Label>
            </div>
            {(can(P.userRoles) || !editing) && (
              <div className="grid gap-2 sm:col-span-2">
                <Label>{t("users.roles")}</Label>
                {roles.map((r) => {
                  const id = `ur-${r.id}`
                  return (
                    <div key={r.id} className="flex items-center gap-2 text-sm">
                      <Checkbox
                        id={id}
                        checked={roleIds.includes(r.id)}
                        onCheckedChange={() => toggle(r.id)}
                      />
                      <label htmlFor={id} className="cursor-pointer">
                        {roleLabel(r.code, r.name, t)}
                        <span className="ml-1 text-muted-foreground">({r.code})</span>
                      </label>
                    </div>
                  )
                })}
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
