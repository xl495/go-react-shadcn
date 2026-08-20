import { useState } from "react"
import { toast } from "sonner"
import { Can } from "@/components/auth/Can"
import { useAuth } from "@/providers/auth"
import { translateApiError, useI18n } from "@/providers/i18n"
import { P } from "@/constants/perms"
import { useConfigListParams } from "@/hooks/list-params"
import { PAGE_SIZE, useConfigs, useCreateConfig, useDeleteConfig, useTestMail, useUpdateConfig } from "@/hooks/queries"
import { FilterForm, SearchField, SearchSubmitButton, useSyncedDraft } from "@/components/SearchField"
import { ConfirmAlert, EmptyTableRow, PaginationBar, TableSkeleton } from "@/components/feedback"
import { Button } from "@/components/ui/button"
import { DictSelect } from "@/components/ui/dict-select"
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

type GroupTab = "all" | "app" | "mail"

const CONFIG_GROUPS = ["all", "app", "mail"] as const

function isSecretKey(key: string) {
  const k = key.toLowerCase()
  return k.includes("password") || k.includes("secret")
}

export function ConfigsPage() {
  const { can } = useAuth()
  const { t } = useI18n()
  const [{ page, q, group }, setParams] = useConfigListParams()
  const [draftQ, setDraftQ] = useSyncedDraft(q)
  const [draftGroup, setDraftGroup] = useSyncedDraft(group)
  const { data, isLoading, error } = useConfigs({
    page,
    pageSize: PAGE_SIZE,
    q: q || undefined,
    group: group === "all" ? undefined : group,
  })
  const rows = data?.items ?? []
  const createConfig = useCreateConfig()
  const updateConfig = useUpdateConfig()
  const deleteConfig = useDeleteConfig()
  const testMail = useTestMail()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<SysConfig | null>(null)
  const [form, setForm] = useState({ key: "", name: "", value: "", group: "app", remark: "" })
  const [pending, setPending] = useState<SysConfig | null>(null)
  const [testTo, setTestTo] = useState("")

  function searchConfigs() {
    void setParams({ q: draftQ.trim(), group: draftGroup, page: 1 })
  }

  function resetConfigs() {
    setDraftQ("")
    setDraftGroup("all")
    void setParams({ q: "", group: "all", page: 1 })
  }

  function openCreate() {
    setEditing(null)
    setForm({ key: "", name: "", value: "", group: group === "all" ? "app" : group, remark: "" })
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

  async function sendTest() {
    try {
      await testMail.mutateAsync(testTo)
      toast.success(t("config.testMailSent"))
    } catch (e) {
      toast.error(translateApiError(e, t))
    }
  }

  const secretEditing = !!editing && isSecretKey(editing.key)
  const filtered = Boolean(q || group !== "all")
  const draftFiltered = Boolean(draftQ || draftGroup !== "all")

  function groupLabel(value: string) {
    if (value === "all") return t("config.groupAll")
    if (value === "app") return t("config.groupApp")
    return t("config.groupMail")
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

      <FilterForm onSubmit={searchConfigs}>
        <SearchField
          id="config-q"
          label={t("app.search")}
          value={draftQ}
          placeholder={t("config.search")}
          inputClassName="w-64"
          onChange={setDraftQ}
        />
        <DictSelect
          id="config-group"
          className="w-36"
          label={t("config.group")}
          value={draftGroup}
          items={CONFIG_GROUPS.map((value) => ({ value, label: groupLabel(value) }))}
          onChange={(value) => setDraftGroup((value as GroupTab) || "all")}
        />
        <SearchSubmitButton />
        {filtered || draftFiltered ? (
          <Button type="button" variant="outline" onClick={resetConfigs}>
            {t("app.resetFilters")}
          </Button>
        ) : null}
      </FilterForm>

      {group === "mail" ? (
        <div className="flex flex-wrap items-end gap-3 rounded-lg border bg-card p-4">
          <div className="grid min-w-[220px] flex-1 gap-1.5">
            <Label htmlFor="test-to">{t("config.testMailTo")}</Label>
            <Input
              id="test-to"
              type="email"
              value={testTo}
              onChange={(e) => setTestTo(e.target.value)}
              placeholder="you@example.com"
            />
            <p className="text-xs text-muted-foreground">{t("config.testMailHint")}</p>
          </div>
          <Can perm={P.mailTest}>
            <Button onClick={() => void sendTest()} disabled={testMail.isPending || !testTo}>
              {t("config.testMail")}
            </Button>
          </Can>
        </div>
      ) : null}

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
                    <TableCell className="max-w-[200px] truncate font-mono text-xs">
                      {isSecretKey(row.key) && row.value ? "••••••••" : row.value || "—"}
                    </TableCell>
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
      <PaginationBar page={page} pageSize={PAGE_SIZE} total={data?.total ?? 0} onPageChange={(next) => void setParams({ page: next })} />

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
              <Input
                type={secretEditing ? "password" : "text"}
                value={form.value}
                onChange={(e) => setForm((f) => ({ ...f, value: e.target.value }))}
              />
              {secretEditing ? <p className="text-xs text-muted-foreground">{t("config.secretKeep")}</p> : null}
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
