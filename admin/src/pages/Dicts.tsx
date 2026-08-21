import { useState } from "react"
import { toast } from "sonner"
import { Can } from "@/components/auth/Can"
import { useAuth } from "@/providers/auth"
import { translateApiError, useI18n } from "@/providers/i18n"
import { P } from "@/constants/perms"
import { useDictListParams } from "@/hooks/list-params"
import {
  PAGE_SIZE,
  useCreateDict,
  useCreateDictItem,
  useDeleteDict,
  useDeleteDictItem,
  useDictItems,
  useDicts,
} from "@/hooks/queries"
import { ConfirmAlert, EmptyTableRow, ResourceTable } from "@/components/feedback"
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
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import type { DictItem, DictType } from "@/types"

export function DictsPage() {
  const { can } = useAuth()
  const { t } = useI18n()
  const [{ page, q, itemPage, typeId }, setParams] = useDictListParams()
  const [draftQ, setDraftQ] = useSyncedDraft(q)
  const { data, isLoading, error } = useDicts({ page, pageSize: PAGE_SIZE, q: q || undefined })
  const types = data?.items ?? []
  const active = types.find((row) => row.id === typeId) ?? types[0] ?? null
  const currentId = active?.id ?? 0
  const itemsQuery = useDictItems(currentId, { page: itemPage, pageSize: PAGE_SIZE })
  const items = itemsQuery.data?.items ?? []
  const createDict = useCreateDict()
  const deleteDict = useDeleteDict()
  const createItem = useCreateDictItem()
  const deleteItem = useDeleteDictItem()
  const [typeOpen, setTypeOpen] = useState(false)
  const [itemOpen, setItemOpen] = useState(false)
  const [typeForm, setTypeForm] = useState({ code: "", name: "", remark: "" })
  const [itemForm, setItemForm] = useState({ label: "", value: "", sort: "0", remark: "" })
  const [pendingType, setPendingType] = useState<DictType | null>(null)
  const [pendingItem, setPendingItem] = useState<DictItem | null>(null)

  return (
    <div className="space-y-4">
      <div className="flex items-end justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{t("dict.title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("dict.subtitle")}</p>
        </div>
        <Can perm={P.dictCreate}>
          <Button onClick={() => setTypeOpen(true)}>{t("dict.createType")}</Button>
        </Can>
      </div>
      <FilterForm onSubmit={() => void setParams({ q: draftQ.trim(), page: 1, typeId: null, itemPage: 1 })}>
        <SearchField
          id="dict-q"
          label={t("app.search")}
          value={draftQ}
          placeholder={t("dict.search")}
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
              void setParams({ q: "", page: 1, typeId: null, itemPage: 1 })
            }}
          >
            {t("app.resetFilters")}
          </Button>
        ) : null}
      </FilterForm>
      {error ? <p className="text-sm text-destructive">{translateApiError(error, t)}</p> : null}
      <div className="grid gap-4 lg:grid-cols-[280px_1fr]">
        <div className="space-y-3">
          <ResourceTable
            loading={isLoading}
            skeletonCols={2}
            page={page}
            pageSize={PAGE_SIZE}
            total={data?.total ?? 0}
            onPageChange={(next) => void setParams({ page: next })}
          >
            <Table>
                <TableCaption className="sr-only">{t("dict.title")}</TableCaption>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("dict.type")}</TableHead>
                    {can(P.dictDelete) ? <TableHead className="text-right">{t("app.actions")}</TableHead> : null}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {types.length === 0 ? (
                    <EmptyTableRow colSpan={can(P.dictDelete) ? 2 : 1} />
                  ) : (
                    types.map((row) => (
                      <TableRow
                        key={row.id}
                        className={active?.id === row.id ? "bg-accent" : "cursor-pointer"}
                        onClick={() => {
                          void setParams({ typeId: row.id, itemPage: 1 })
                        }}
                      >
                        <TableCell>
                          <div className="font-medium">{row.name}</div>
                          <div className="font-mono text-xs text-muted-foreground">{row.code}</div>
                        </TableCell>
                        {can(P.dictDelete) ? (
                          <TableCell className="text-right">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={(e) => {
                                e.stopPropagation()
                                setPendingType(row)
                              }}
                            >
                              {t("app.delete")}
                            </Button>
                          </TableCell>
                        ) : null}
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
          </ResourceTable>
        </div>
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium">{active ? active.name : t("dict.pickType")}</h3>
            {active ? (
              <Can perm={P.dictItemCreate}>
                <Button size="sm" onClick={() => setItemOpen(true)}>
                  {t("dict.createItem")}
                </Button>
              </Can>
            ) : null}
          </div>
          <ResourceTable
            loading={itemsQuery.isLoading}
            page={itemPage}
            pageSize={PAGE_SIZE}
            total={itemsQuery.data?.total ?? 0}
            onPageChange={(next) => void setParams({ itemPage: next })}
          >
            <Table>
                <TableCaption className="sr-only">{active?.name ?? t("dict.pickType")}</TableCaption>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("dict.label")}</TableHead>
                    <TableHead>{t("dict.value")}</TableHead>
                    <TableHead>{t("dict.sort")}</TableHead>
                    <TableHead>{t("app.status")}</TableHead>
                    {can(P.dictItemDelete) ? (
                      <TableHead className="text-right">{t("app.actions")}</TableHead>
                    ) : null}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.length === 0 ? (
                    <EmptyTableRow colSpan={can(P.dictItemDelete) ? 5 : 4} />
                  ) : (
                    items.map((row) => (
                      <TableRow key={row.id}>
                        <TableCell>{row.label}</TableCell>
                        <TableCell className="font-mono text-xs">{row.value}</TableCell>
                        <TableCell>{row.sort}</TableCell>
                        <TableCell>
                          <Badge variant={row.status === "active" ? "default" : "muted"}>
                            {row.status === "active" ? t("app.active") : t("app.disabled")}
                          </Badge>
                        </TableCell>
                        {can(P.dictItemDelete) ? (
                          <TableCell className="text-right">
                            <Button variant="ghost" size="sm" onClick={() => setPendingItem(row)}>
                              {t("app.delete")}
                            </Button>
                          </TableCell>
                        ) : null}
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
          </ResourceTable>
        </div>
      </div>

      <ConfirmAlert
        open={!!pendingType}
        onOpenChange={(next) => {
          if (!next) setPendingType(null)
        }}
        title={t("app.delete")}
        description={pendingType ? t("dict.confirmDeleteType", { name: pendingType.name }) : ""}
        onConfirm={() => {
          if (!pendingType) return
          deleteDict.mutate(pendingType.id, {
            onSuccess: () => {
              void setParams({ typeId: null, itemPage: 1 })
              toast.success(t("app.saved"))
            },
          })
          setPendingType(null)
        }}
      />
      <ConfirmAlert
        open={!!pendingItem}
        onOpenChange={(next) => {
          if (!next) setPendingItem(null)
        }}
        title={t("app.delete")}
        description={pendingItem ? t("dict.confirmDeleteItem", { name: pendingItem.label }) : ""}
        onConfirm={() => {
          if (!pendingItem) return
          deleteItem.mutate(pendingItem.id, {
            onSuccess: () => toast.success(t("app.saved")),
          })
          setPendingItem(null)
        }}
      />

      <Dialog open={typeOpen} onOpenChange={setTypeOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("dict.createType")}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-3">
            <div className="grid gap-1.5">
              <Label>{t("app.code")}</Label>
              <Input value={typeForm.code} onChange={(e) => setTypeForm((f) => ({ ...f, code: e.target.value }))} />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("app.name")}</Label>
              <Input value={typeForm.name} onChange={(e) => setTypeForm((f) => ({ ...f, name: e.target.value }))} />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("app.description")}</Label>
              <Input value={typeForm.remark} onChange={(e) => setTypeForm((f) => ({ ...f, remark: e.target.value }))} />
            </div>
          </div>
          <DialogFooter>
            <Button
              onClick={() =>
                void createDict.mutateAsync(typeForm).then(() => {
                  toast.success(t("app.saved"))
                  setTypeOpen(false)
                  setTypeForm({ code: "", name: "", remark: "" })
                }).catch(() => undefined)
              }
            >
              {t("app.create")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={itemOpen} onOpenChange={setItemOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("dict.createItem")}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-3">
            <div className="grid gap-1.5">
              <Label>{t("dict.label")}</Label>
              <Input value={itemForm.label} onChange={(e) => setItemForm((f) => ({ ...f, label: e.target.value }))} />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("dict.value")}</Label>
              <Input value={itemForm.value} onChange={(e) => setItemForm((f) => ({ ...f, value: e.target.value }))} />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("dict.sort")}</Label>
              <Input value={itemForm.sort} onChange={(e) => setItemForm((f) => ({ ...f, sort: e.target.value }))} />
            </div>
          </div>
          <DialogFooter>
            <Button
              onClick={() => {
                if (!active) return
                void createItem
                  .mutateAsync({
                    id: active.id,
                    body: {
                      label: itemForm.label,
                      value: itemForm.value,
                      sort: Number(itemForm.sort) || 0,
                      remark: itemForm.remark,
                    },
                  })
                  .then(() => {
                    toast.success(t("app.saved"))
                    setItemOpen(false)
                    setItemForm({ label: "", value: "", sort: "0", remark: "" })
                  })
                  .catch(() => undefined)
              }}
            >
              {t("app.create")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
