import { toast } from "sonner"
import { Can } from "@/components/auth/Can"
import { useAuth } from "@/providers/auth"
import { permLabel, translateApiError, useI18n } from "@/providers/i18n"
import { P } from "@/constants/perms"
import { DICT, useDict } from "@/hooks/dict"
import { usePermissionListParams } from "@/hooks/list-params"
import { PAGE_SIZE, useCreatePermission, useDeletePermission, usePermissions } from "@/hooks/queries"
import { ConfirmAlert, EmptyTableRow, PaginationBar, TableSkeleton } from "@/components/feedback"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { FilterForm, SearchField, SearchSubmitButton, useSyncedDraft } from "@/components/SearchField"
import { DictSelect } from "@/components/ui/dict-select"
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
import { useState } from "react"

const empty = { name: "", code: "", path: "", method: "GET", kind: "button", description: "" }

export function PermissionsPage() {
  const { can } = useAuth()
  const { t } = useI18n()
  const kindDict = useDict(DICT.permKind)
  const [{ q, page, kind }, setParams] = usePermissionListParams()
  const [draftQ, setDraftQ] = useSyncedDraft(q)
  const [draftKind, setDraftKind] = useSyncedDraft(kind)
  const { data, isLoading, error } = usePermissions({
    page,
    pageSize: PAGE_SIZE,
    q: q || undefined,
    kind: kind || undefined,
  })
  const items = data?.items ?? []
  const createPerm = useCreatePermission()
  const deletePerm = useDeletePermission()
  const [open, setOpen] = useState(false)
  const [form, setForm] = useState(empty)
  const [pending, setPending] = useState<Permission | null>(null)
  const filtered = Boolean(q || kind)
  const draftFiltered = Boolean(draftQ || draftKind)
  const colSpan = can(P.permDelete) ? 7 : 6

  function searchPermissions() {
    void setParams({ q: draftQ.trim(), kind: draftKind, page: 1 })
  }

  function resetPermissions() {
    setDraftQ("")
    setDraftKind("")
    void setParams({ q: "", kind: "", page: 1 })
  }

  async function create() {
    try {
      await createPerm.mutateAsync(form)
      toast.success(t("app.saved"))
      setOpen(false)
      setForm(empty)
    } catch (e) {
      toast.error(translateApiError(e, t) || t("permissions.createFailed"))
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{t("permissions.title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("permissions.subtitle")}</p>
        </div>
        <Can perm={P.permCreate}>
          <Button onClick={() => setOpen(true)}>{t("permissions.create")}</Button>
        </Can>
      </div>
      <FilterForm onSubmit={searchPermissions}>
        <SearchField
          id="perm-q"
          label={t("app.search")}
          value={draftQ}
          placeholder={t("permissions.search")}
          inputClassName="w-64"
          onChange={setDraftQ}
        />
        <DictSelect
          id="perm-kind"
          className="w-36"
          label={t("permissions.type")}
          value={draftKind}
          items={kindDict.items}
          allowEmpty
          emptyLabel={t("app.all")}
          onChange={setDraftKind}
        />
        <SearchSubmitButton />
        {filtered || draftFiltered ? (
          <Button type="button" variant="outline" onClick={resetPermissions}>
            {t("app.resetFilters")}
          </Button>
        ) : null}
      </FilterForm>
      {error ? <p className="text-sm text-destructive">{translateApiError(error, t)}</p> : null}
      <div className="rounded-lg border bg-card">
        {isLoading ? (
          <TableSkeleton rows={8} cols={colSpan} />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>{t("app.name")}</TableHead>
                <TableHead>{t("app.code")}</TableHead>
                <TableHead>{t("permissions.type")}</TableHead>
                <TableHead>{t("permissions.method")}</TableHead>
                <TableHead>{t("permissions.path")}</TableHead>
                {can(P.permDelete) ? <TableHead className="text-right">{t("app.actions")}</TableHead> : null}
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.length === 0 ? (
                <EmptyTableRow colSpan={colSpan} />
              ) : (
                items.map((p) => (
                  <TableRow key={p.id}>
                    <TableCell className="tabular-nums text-muted-foreground">{p.id}</TableCell>
                    <TableCell className="font-medium">{permLabel(p.code, p.name, t)}</TableCell>
                    <TableCell className="font-mono text-xs">{p.code}</TableCell>
                    <TableCell>
                      <Badge variant={p.kind === "button" ? "default" : "muted"}>
                        {kindDict.items.length
                          ? kindDict.label(p.kind)
                          : t(`kinds.${p.kind || "api"}`)}
                      </Badge>
                    </TableCell>
                    <TableCell className="font-mono text-xs">{p.method}</TableCell>
                    <TableCell>
                      <span className="font-mono text-xs">{p.path}</span>
                    </TableCell>
                    {can(P.permDelete) ? (
                      <TableCell className="text-right">
                        <Button variant="ghost" size="sm" onClick={() => setPending(p)}>
                          {t("app.delete")}
                        </Button>
                      </TableCell>
                    ) : null}
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
        description={pending ? t("permissions.confirmDelete", { code: pending.code }) : ""}
        onConfirm={() => {
          if (!pending) return
          deletePerm.mutate(pending.id, {
            onSuccess: () => toast.success(t("app.saved")),
            onError: (e) => toast.error(translateApiError(e, t) || t("permissions.deleteFailed")),
          })
          setPending(null)
        }}
      />

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
            <DictSelect
              id="kind"
              label={t("permissions.type")}
              value={form.kind}
              items={kindDict.items}
              onChange={(value) => setForm((f) => ({ ...f, kind: value }))}
            />
          </div>
          <DialogFooter>
            <Button onClick={() => void create()}>{t("app.create")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
