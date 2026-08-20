import { useState } from "react"
import { toast } from "sonner"
import { Can } from "@/components/auth/Can"
import { useAuth } from "@/providers/auth"
import { translateApiError, useI18n } from "@/providers/i18n"
import { P } from "@/constants/perms"
import { PAGE_SIZE, useConfigs, useCreateConfig, useDeleteConfig, useUpdateConfig } from "@/hooks/queries"
import { ConfirmAlert, EmptyTableRow, PaginationBar, TableSkeleton } from "@/components/feedback"
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
import type { SysConfig } from "@/types"

export function ConfigsPage() {
  const { can } = useAuth()
  const { t } = useI18n()
  const [page, setPage] = useState(1)
  const { data, isLoading, error } = useConfigs({ page, pageSize: PAGE_SIZE })
  const rows = data?.items ?? []
  const createConfig = useCreateConfig()
  const updateConfig = useUpdateConfig()
  const deleteConfig = useDeleteConfig()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<SysConfig | null>(null)
  const [form, setForm] = useState({ key: "", name: "", value: "", group: "app", remark: "" })
  const [pending, setPending] = useState<SysConfig | null>(null)

  function openCreate() {
    setEditing(null)
    setForm({ key: "", name: "", value: "", group: "app", remark: "" })
    setOpen(true)
  }

  function openEdit(row: SysConfig) {
    setEditing(row)
    setForm({ key: row.key, name: row.name, value: row.value, group: row.group, remark: row.remark })
    setOpen(true)
  }

  async function submit() {
    try {
      if (editing) {
        await updateConfig.mutateAsync({ id: editing.id, body: form })
      } else {
        await createConfig.mutateAsync(form)
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
          <h2 className="text-xl font-semibold tracking-tight">{t("config.title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("config.subtitle")}</p>
        </div>
        <Can perm={P.configCreate}>
          <Button onClick={openCreate}>{t("config.create")}</Button>
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
                <TableHead>{t("config.key")}</TableHead>
                <TableHead>{t("app.name")}</TableHead>
                <TableHead>{t("config.value")}</TableHead>
                <TableHead>{t("config.group")}</TableHead>
                <TableHead>{t("app.description")}</TableHead>
                {can(P.configUpdate) || can(P.configDelete) ? (
                  <TableHead className="text-right">{t("app.actions")}</TableHead>
                ) : null}
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.length === 0 ? (
                <EmptyTableRow colSpan={6} />
              ) : (
                rows.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className="font-mono text-xs">{row.key}</TableCell>
                    <TableCell>{row.name}</TableCell>
                    <TableCell className="max-w-[200px] truncate">{row.value}</TableCell>
                    <TableCell>{row.group}</TableCell>
                    <TableCell className="text-muted-foreground">{row.remark}</TableCell>
                    {can(P.configUpdate) || can(P.configDelete) ? (
                      <TableCell className="text-right">
                        <Can perm={P.configUpdate}>
                          <Button variant="ghost" size="sm" onClick={() => openEdit(row)}>
                            {t("app.edit")}
                          </Button>
                        </Can>
                        <Can perm={P.configDelete}>
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
        description={pending ? t("config.confirmDelete", { name: pending.name }) : ""}
        onConfirm={() => {
          if (!pending) return
          deleteConfig.mutate(pending.id, {
            onSuccess: () => toast.success(t("app.saved")),
            onError: (e) => toast.error(translateApiError(e, t)),
          })
          setPending(null)
        }}
      />

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editing ? t("config.edit") : t("config.create")}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-3">
            <div className="grid gap-1.5">
              <Label>{t("config.key")}</Label>
              <Input
                value={form.key}
                disabled={!!editing}
                onChange={(e) => setForm((f) => ({ ...f, key: e.target.value }))}
              />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("app.name")}</Label>
              <Input value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("config.value")}</Label>
              <Input value={form.value} onChange={(e) => setForm((f) => ({ ...f, value: e.target.value }))} />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("config.group")}</Label>
              <Input value={form.group} onChange={(e) => setForm((f) => ({ ...f, group: e.target.value }))} />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("app.description")}</Label>
              <Input value={form.remark} onChange={(e) => setForm((f) => ({ ...f, remark: e.target.value }))} />
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
