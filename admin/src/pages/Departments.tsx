import { useState } from "react"
import { toast } from "sonner"
import { Can } from "@/components/auth/Can"
import { useAuth } from "@/providers/auth"
import { translateApiError, useI18n } from "@/providers/i18n"
import { P } from "@/constants/perms"
import {
  PAGE_SIZE,
  useCreateDepartment,
  useDeleteDepartment,
  useDepartments,
  useUpdateDepartment,
} from "@/hooks/queries"
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
import type { Department } from "@/types"

type FlatDept = { row: Department; depth: number }

function flatten(nodes: Department[], depth = 0): FlatDept[] {
  const out: FlatDept[] = []
  for (const n of nodes) {
    out.push({ row: n, depth })
    if (n.children?.length) out.push(...flatten(n.children, depth + 1))
  }
  return out
}

const emptyForm = { name: "", code: "", parentId: "", sort: "0", leader: "", status: "active" }

export function DepartmentsPage() {
  const { can } = useAuth()
  const { t } = useI18n()
  const [page, setPage] = useState(1)
  const { data, isLoading, error } = useDepartments({ page, pageSize: PAGE_SIZE })
  const roots = data?.items ?? []
  const rows = flatten(roots)
  const createDept = useCreateDepartment()
  const updateDept = useUpdateDepartment()
  const deleteDept = useDeleteDepartment()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<Department | null>(null)
  const [form, setForm] = useState(emptyForm)
  const [pending, setPending] = useState<Department | null>(null)

  function openCreate() {
    setEditing(null)
    setForm(emptyForm)
    setOpen(true)
  }

  function openEdit(row: Department) {
    setEditing(row)
    setForm({
      name: row.name,
      code: row.code,
      parentId: row.parentId ? String(row.parentId) : "",
      sort: String(row.sort ?? 0),
      leader: row.leader ?? "",
      status: row.status || "active",
    })
    setOpen(true)
  }

  async function submit() {
    const body = {
      name: form.name,
      code: form.code,
      parentId: form.parentId ? Number(form.parentId) : null,
      sort: Number(form.sort) || 0,
      leader: form.leader,
      status: form.status,
    }
    try {
      if (editing) {
        await updateDept.mutateAsync({ id: editing.id, body })
      } else {
        await createDept.mutateAsync(body)
      }
      toast.success(t("app.saved"))
      setOpen(false)
    } catch (e) {
      toast.error(translateApiError(e, t))
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-end justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{t("dept.title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("dept.subtitle")}</p>
        </div>
        <Can perm={P.deptCreate}>
          <Button onClick={openCreate}>{t("dept.create")}</Button>
        </Can>
      </div>
      {error ? <p className="text-sm text-destructive">{translateApiError(error, t)}</p> : null}
      <div className="rounded-lg border bg-card">
        {isLoading ? (
          <TableSkeleton />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("app.name")}</TableHead>
                <TableHead>{t("app.code")}</TableHead>
                <TableHead>{t("dept.leader")}</TableHead>
                <TableHead>{t("dept.sort")}</TableHead>
                <TableHead>{t("app.status")}</TableHead>
                {can(P.deptUpdate) || can(P.deptDelete) ? (
                  <TableHead className="text-right">{t("app.actions")}</TableHead>
                ) : null}
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.length === 0 ? (
                <EmptyTableRow colSpan={6} />
              ) : (
                rows.map(({ row, depth }) => (
                  <TableRow key={row.id}>
                    <TableCell style={{ paddingLeft: `${12 + depth * 16}px` }} className="font-medium">
                      {row.name}
                    </TableCell>
                    <TableCell className="font-mono text-xs">{row.code}</TableCell>
                    <TableCell>{row.leader || "—"}</TableCell>
                    <TableCell>{row.sort}</TableCell>
                    <TableCell>
                      <Badge variant={row.status === "active" ? "default" : "muted"}>
                        {row.status === "active" ? t("app.active") : t("app.disabled")}
                      </Badge>
                    </TableCell>
                    {can(P.deptUpdate) || can(P.deptDelete) ? (
                      <TableCell className="text-right">
                        <Can perm={P.deptUpdate}>
                          <Button variant="ghost" size="sm" onClick={() => openEdit(row)}>
                            {t("app.edit")}
                          </Button>
                        </Can>
                        <Can perm={P.deptDelete}>
                          <Button variant="ghost" size="sm" onClick={() => setPending(row)}>
                            {t("app.delete")}
                          </Button>
                        </Can>
                      </TableCell>
                    ) : null}
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        )}
      </div>
      <PaginationBar page={page} pageSize={PAGE_SIZE} total={data?.total ?? 0} onPageChange={setPage} />

      <ConfirmAlert
        open={!!pending}
        onOpenChange={(next) => {
          if (!next) setPending(null)
        }}
        title={t("app.delete")}
        description={pending ? t("dept.confirmDelete", { name: pending.name }) : ""}
        onConfirm={() => {
          if (!pending) return
          deleteDept.mutate(pending.id, {
            onSuccess: () => toast.success(t("app.saved")),
            onError: (e) => toast.error(translateApiError(e, t)),
          })
          setPending(null)
        }}
      />

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editing ? t("dept.edit") : t("dept.create")}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-3">
            <div className="grid gap-1.5">
              <Label>{t("app.name")}</Label>
              <Input value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("app.code")}</Label>
              <Input value={form.code} onChange={(e) => setForm((f) => ({ ...f, code: e.target.value }))} />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("dept.parent")}</Label>
              <Input
                value={form.parentId}
                placeholder={t("app.optional")}
                onChange={(e) => setForm((f) => ({ ...f, parentId: e.target.value }))}
              />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("dept.leader")}</Label>
              <Input value={form.leader} onChange={(e) => setForm((f) => ({ ...f, leader: e.target.value }))} />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("dept.sort")}</Label>
              <Input value={form.sort} onChange={(e) => setForm((f) => ({ ...f, sort: e.target.value }))} />
            </div>
          </div>
          <DialogFooter>
            <Button onClick={() => void submit()}>{editing ? t("app.save") : t("app.create")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
