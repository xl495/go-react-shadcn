import { useEffect, useRef, useState } from "react"
import { Link } from "react-router-dom"
import { toast } from "sonner"
import { Can } from "@/components/auth/Can"
import { api } from "@/api/client"
import type { UserImportJob } from "@/api/users"
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
  useBatchUserStatus,
  useCreateUser,
  useDeleteUser,
  useImportUsers,
  useRoles,
  useUpdateUser,
  useUsers,
} from "@/hooks/queries"
import { ConfirmAlert, DesktopOnly, EmptyState, EmptyTableRow, ResourceTable, StackedCards } from "@/components/feedback"
import { Avatar } from "@/components/ui/avatar"
import { emptyUserForm, UserFormDialog, type UserFormValues } from "@/pages/users/form-dialog"
import { Badge } from "@/components/ui/badge"
import { FilterForm, SearchField, SearchSubmitButton, useSyncedDraft } from "@/components/SearchField"
import { DictSelect } from "@/components/ui/dict-select"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import type { User } from "@/types"

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
  const { data, isLoading, error, refetch } = useUsers({
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
  const batchStatus = useBatchUserStatus()
  const [picked, setPicked] = useState<number[]>([])
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
  const importUsers = useImportUsers()
  const assignRoles = useAssignUserRoles()
  const fileRef = useRef<HTMLInputElement>(null)
  const [importJob, setImportJob] = useState<UserImportJob | null>(null)
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<User | null>(null)
  const [form, setForm] = useState<UserFormValues>(emptyUserForm)
  const [roleIds, setRoleIds] = useState<number[]>([])
  const [pending, setPending] = useState<User | null>(null)
  const importBusy =
    importUsers.isPending || importJob?.status === "queued" || importJob?.status === "running"
  const mounted = useRef(true)
  useEffect(() => {
    mounted.current = true
    return () => {
      mounted.current = false
    }
  }, [])

  async function runImport(file: File) {
    try {
      const job = await importUsers.mutateAsync({ file, kind })
      if (!mounted.current) return
      setImportJob(job)
      let current = job
      let delay = 300
      let noticed = false
      const started = Date.now()
      try {
        while (current.status === "queued" || current.status === "running") {
          if (!noticed && Date.now() - started > 60_000) {
            toast.message(t("users.importStillRunning"))
            noticed = true
          }
          await new Promise((r) => setTimeout(r, delay))
          if (!mounted.current) return
          current = await api.importUserJob(current.id)
          if (!mounted.current) return
          setImportJob(current)
          delay = Math.min(2000, Math.round(delay * 1.25))
        }
      } catch (err) {
        if (mounted.current) toast.error(translateApiError(err, t))
        return
      }
      if (current.status === "failed") {
        toast.error(t("users.importFailed", { failed: current.failed }))
        return
      }
      if (current.status !== "done") {
        toast.message(t("users.importStillRunning"))
        return
      }
      void refetch()
      toast.success(t("users.importResult", { created: current.created, failed: current.failed }))
    } catch {
      // POST errors are toasted by the HTTP client.
    }
  }

  function openCreate() {
    setEditing(null)
    setForm(emptyUserForm)
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
      marketingOptIn: u.marketingOptIn ?? false,
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
            kind,
          },
        })
        if (can(P.userRoles)) {
          await assignRoles.mutateAsync({ id: editing.id, roleIds, kind })
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
    } catch {
      // API message is toasted by the HTTP client.
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
        <div className="flex flex-wrap gap-2">
          <Can perm={P.userExport}>
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                void api.exportUsers(kind).catch((err) => {
                  toast.error(translateApiError(err, t))
                })
              }}
            >
              {t("app.export")}
            </Button>
          </Can>
          <Can perm={P.userImport}>
            <input
              ref={fileRef}
              type="file"
              accept=".csv,text/csv"
              className="sr-only"
              onChange={(e) => {
                const file = e.target.files?.[0]
                e.target.value = ""
                if (!file) return
                void runImport(file)
              }}
            />
            <Button
              type="button"
              variant="outline"
              disabled={importBusy}
              onClick={() => fileRef.current?.click()}
            >
              {importBusy ? t("users.importing") : t("app.import")}
            </Button>
            {importJob ? (
              <span className="self-center text-xs text-muted-foreground">
                {t("users.importStatus", {
                  status: importJob.status,
                  created: importJob.created,
                  failed: importJob.failed,
                })}
              </span>
            ) : null}
          </Can>
          <Can perm={P.userUpdate}>
            <Button
              variant="outline"
              disabled={picked.length === 0 || batchStatus.isPending}
              onClick={() =>
                batchStatus.mutate(
                  { ids: picked, status: "disabled", kind },
                  {
                    onSuccess: () => {
                      toast.success(t("app.saved"))
                      setPicked([])
                    },
                  },
                )
              }
            >
              {t("users.batchDisable")}
            </Button>
            <Button
              variant="outline"
              disabled={picked.length === 0 || batchStatus.isPending}
              onClick={() =>
                batchStatus.mutate(
                  { ids: picked, status: "active", kind },
                  {
                    onSuccess: () => {
                      toast.success(t("app.saved"))
                      setPicked([])
                    },
                  },
                )
              }
            >
              {t("users.batchEnable")}
            </Button>
          </Can>
          <Can perm={P.userCreate}>
            <Button onClick={openCreate}>{isWeb ? t("webUsers.create") : t("users.create")}</Button>
          </Can>
        </div>
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
      <ResourceTable
        loading={isLoading}
        page={page}
        pageSize={PAGE_SIZE}
        total={data?.total ?? 0}
        onPageChange={(next) => void setParams({ page: next })}
      >
        <StackedCards>
          {users.length === 0 ? (
            <EmptyState />
          ) : (
            users.map((u) => (
              <div key={u.id} className="rounded-lg border bg-card p-3 space-y-2">
                <div className="flex items-center gap-2">
                  <Avatar name={u.nickname || u.username} src={u.avatar} />
                  <div className="min-w-0 flex-1">
                    <p className="truncate font-medium">{u.nickname || u.username}</p>
                    <p className="truncate text-xs text-muted-foreground">@{u.username}</p>
                  </div>
                  <Badge variant={u.status === "active" ? "default" : "muted"}>{statusDict.label(u.status)}</Badge>
                </div>
                <p className="text-xs text-muted-foreground">{u.email || u.phone || "—"}</p>
                <div className="flex flex-wrap justify-end gap-1">
                  <Button asChild variant="ghost" size="sm">
                    <Link to={`/users/${u.id}?kind=${kind}`}>{t("users.detail")}</Link>
                  </Button>
                  <Can perm={P.userUpdate}>
                    <Button variant="ghost" size="sm" onClick={() => openEdit(u)}>
                      {t("app.edit")}
                    </Button>
                  </Can>
                </div>
              </div>
            ))
          )}
        </StackedCards>
        <DesktopOnly>
          <Table>
            <TableCaption className="sr-only">{isWeb ? t("webUsers.title") : t("users.title")}</TableCaption>
            <TableHeader>
              <TableRow>
                <TableHead className="w-8" />
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
                <EmptyTableRow colSpan={isWeb ? 13 : 15} />
              ) : (
                users.map((u) => (
                  <TableRow key={u.id}>
                    <TableCell>
                      <input
                        type="checkbox"
                        checked={picked.includes(u.id)}
                        onChange={(e) =>
                          setPicked((cur) =>
                            e.target.checked ? [...cur, u.id] : cur.filter((id) => id !== u.id),
                          )
                        }
                      />
                    </TableCell>
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
                          <Link to={`/users/${u.id}?kind=${kind}`}>{t("users.detail")}</Link>
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
        </DesktopOnly>
      </ResourceTable>

      <ConfirmAlert
        open={!!pending}
        onOpenChange={(next) => {
          if (!next) setPending(null)
        }}
        title={t("app.delete")}
        description={pending ? t("users.confirmDelete", { name: pending.username }) : ""}
        onConfirm={() => {
          if (!pending) return
          deleteUser.mutate({ id: pending.id, kind }, {
            onSuccess: () => toast.success(t("app.saved")),
          })
          setPending(null)
        }}
      />

      <UserFormDialog
        open={open}
        onOpenChange={setOpen}
        kind={kind}
        editing={editing}
        form={form}
        setForm={setForm}
        roles={roles}
        roleIds={roleIds}
        onToggleRole={toggle}
        onSubmit={() => void submit()}
        onAvatar={setEditing}
        canAssignRoles={can(P.userRoles) || !editing}
        genderItems={genderDict.items}
        statusItems={statusDict.items}
        deptItems={deptDict.items}
      />
    </div>
  )
}
