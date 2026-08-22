import { useState } from "react"
import { Link, useNavigate } from "react-router-dom"
import { toast } from "sonner"
import { Can } from "@/components/auth/Can"
import { roleDesc, roleLabel, translateApiError, useI18n } from "@/providers/i18n"
import { P } from "@/constants/perms"
import { useSearchPageParams } from "@/hooks/list-params"
import { PAGE_SIZE, useCopyRole, useCreateRole, useDeleteRole, useRoles } from "@/hooks/queries"
import { ConfirmAlert, DesktopOnly, EmptyState, EmptyTableRow, ResourceTable, StackedCards } from "@/components/feedback"
import { FilterForm, SearchField, SearchSubmitButton, useSyncedDraft } from "@/components/SearchField"
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
import type { Role } from "@/types"

const SCOPES = ["all", "dept_sub", "dept", "self"] as const

export function RolesPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const [{ page, q }, setParams] = useSearchPageParams()
  const [draftQ, setDraftQ] = useSyncedDraft(q)
  const { data: rolesPage, isLoading, error: rolesError } = useRoles({ page, pageSize: PAGE_SIZE, q: q || undefined })
  const roles = rolesPage?.items ?? []
  const createRole = useCreateRole()
  const copyRole = useCopyRole()
  const deleteRole = useDeleteRole()
  const [open, setOpen] = useState(false)
  const [copying, setCopying] = useState<Role | null>(null)
  const [copyName, setCopyName] = useState("")
  const [copyCode, setCopyCode] = useState("")
  const [name, setName] = useState("")
  const [code, setCode] = useState("")
  const [description, setDescription] = useState("")
  const [dataScope, setDataScope] = useState("self")
  const [pending, setPending] = useState<Role | null>(null)

  async function create() {
    try {
      const role = await createRole.mutateAsync({ name, code, description, dataScope, permissionIds: [] })
      toast.success(t("app.saved"))
      setOpen(false)
      setName("")
      setCode("")
      setDescription("")
      setDataScope("self")
      navigate(`/roles/${role.id}`)
    } catch {
      // API message is toasted by the HTTP client.
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{t("roles.title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("roles.subtitle")}</p>
        </div>
        <Can perm={P.roleCreate}>
          <Button onClick={() => setOpen(true)}>{t("roles.create")}</Button>
        </Can>
      </div>
      <FilterForm onSubmit={() => void setParams({ q: draftQ.trim(), page: 1 })}>
        <SearchField
          id="role-q"
          label={t("app.search")}
          value={draftQ}
          placeholder={t("roles.search")}
          inputClassName="w-64"
          onChange={setDraftQ}
        />
        <SearchSubmitButton />
        {q || draftQ ? (
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              setDraftQ("")
              void setParams({ q: "", page: 1 })
            }}
          >
            {t("app.resetFilters")}
          </Button>
        ) : null}
      </FilterForm>
      {rolesError ? <p className="text-sm text-destructive">{translateApiError(rolesError, t)}</p> : null}
      <ResourceTable
        loading={isLoading}
        page={page}
        pageSize={PAGE_SIZE}
        total={rolesPage?.total ?? 0}
        onPageChange={(next) => void setParams({ page: next })}
      >
        <StackedCards>
          {roles.length === 0 ? (
            <EmptyState />
          ) : (
            roles.map((role) => (
              <div key={role.id} className="rounded-lg border p-3 space-y-2">
                <div className="flex items-center justify-between gap-2">
                  <p className="truncate font-medium">{roleLabel(role.code, role.name, t)}</p>
                  <Badge variant="muted">{t(`roles.scope.${role.dataScope || "self"}`)}</Badge>
                </div>
                <p className="font-mono text-xs text-muted-foreground">{role.code}</p>
                <div className="flex justify-end gap-1">
                  <Button asChild variant="ghost" size="sm">
                    <Link to={`/roles/${role.id}`}>{t("roles.detail")}</Link>
                  </Button>
                  <Can perm={P.roleCopy}>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        setCopying(role)
                        setCopyName(`${role.name} copy`)
                        setCopyCode(`${role.code}-copy`)
                      }}
                    >
                      {t("roles.copy")}
                    </Button>
                  </Can>
                </div>
              </div>
            ))
          )}
        </StackedCards>
        <DesktopOnly>
        <Table>
          <TableCaption className="sr-only">{t("roles.title")}</TableCaption>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>{t("app.name")}</TableHead>
              <TableHead>{t("app.code")}</TableHead>
              <TableHead>{t("roles.dataScope")}</TableHead>
              <TableHead>{t("roles.permissions")}</TableHead>
              <TableHead className="text-right">{t("app.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {roles.length === 0 ? (
              <EmptyTableRow colSpan={6} />
            ) : (
              roles.map((role) => (
                <TableRow key={role.id}>
                  <TableCell className="tabular-nums text-muted-foreground">{role.id}</TableCell>
                  <TableCell>
                    <div className="font-medium">{roleLabel(role.code, role.name, t)}</div>
                    {role.description || roleDesc(role.code, "", t) ? (
                      <p className="mt-0.5 text-xs text-muted-foreground">
                        {roleDesc(role.code, role.description ?? "", t)}
                      </p>
                    ) : null}
                  </TableCell>
                  <TableCell className="font-mono text-xs">{role.code}</TableCell>
                  <TableCell>
                    <Badge variant="muted">{t(`roles.scope.${role.dataScope || "self"}`)}</Badge>
                  </TableCell>
                  <TableCell className="tabular-nums">{role.permissionIds?.length ?? 0}</TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-1">
                  <Button asChild variant="ghost" size="sm">
                    <Link to={`/roles/${role.id}`}>{t("roles.detail")}</Link>
                  </Button>
                  <Can perm={P.roleCopy}>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        setCopying(role)
                        setCopyName(`${role.name} copy`)
                        setCopyCode(`${role.code}-copy`)
                      }}
                    >
                      {t("roles.copy")}
                    </Button>
                  </Can>
                  <Can perm={P.roleDelete}>
                        <Button variant="ghost" size="sm" onClick={() => setPending(role)}>
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
        description={
          pending ? t("roles.confirmDelete", { name: roleLabel(pending.code, pending.name, t) }) : ""
        }
        onConfirm={() => {
          if (!pending) return
          deleteRole.mutate(pending.id, {
            onSuccess: () => toast.success(t("app.saved")),
          })
          setPending(null)
        }}
      />

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
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
              <Select value={dataScope} onValueChange={setDataScope}>
                <SelectTrigger id="rs">
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
          </div>
          <DialogFooter>
            <Button onClick={() => void create()}>{t("app.create")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!copying} onOpenChange={(next) => { if (!next) setCopying(null) }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("roles.copy")}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-3">
            <div className="grid gap-1.5">
              <Label htmlFor="cn">{t("app.name")}</Label>
              <Input id="cn" value={copyName} onChange={(e) => setCopyName(e.target.value)} />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="cc">{t("app.code")}</Label>
              <Input id="cc" value={copyCode} onChange={(e) => setCopyCode(e.target.value)} />
            </div>
          </div>
          <DialogFooter>
            <Button
              onClick={() => {
                if (!copying) return
                copyRole.mutate(
                  { id: copying.id, body: { name: copyName, code: copyCode } },
                  {
                    onSuccess: (role) => {
                      toast.success(t("app.saved"))
                      setCopying(null)
                      navigate(`/roles/${role.id}`)
                    },
                  },
                )
              }}
            >
              {t("roles.copy")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
