import { useState, type FormEvent } from "react"
import { toast } from "sonner"
import { Can } from "@/components/auth/Can"
import { useI18n } from "@/providers/i18n"
import { P } from "@/constants/perms"
import { EmptyTableRow, ResourceTable } from "@/components/feedback"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
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
import { formatDateTime } from "@/utils/format"
import { useAnnounce, useNotifications, useReadAllNotifications, useReadNotification } from "@/hooks/queries"

export function NotificationsPage() {
  const { t } = useI18n()
  const list = useNotifications({ page: 1, pageSize: 50 })
  const readOne = useReadNotification()
  const readAll = useReadAllNotifications()
  const announce = useAnnounce()
  const [form, setForm] = useState({ kind: "admin", title: "", body: "" })
  const items = list.data?.items ?? []

  async function onAnnounce(e: FormEvent) {
    e.preventDefault()
    try {
      await announce.mutateAsync(form)
      toast.success(t("notify.sent"))
      setForm((p) => ({ ...p, title: "", body: "" }))
    } catch {
      // toasted
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{t("notify.title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("notify.subtitle")}</p>
        </div>
        <Button type="button" variant="outline" disabled={readAll.isPending} onClick={() => void readAll.mutateAsync()}>
          {t("notify.readAll")}
        </Button>
      </div>

      <Can perm={P.announceCreate}>
        <Card>
          <CardHeader>
            <CardTitle>{t("notify.announce")}</CardTitle>
            <CardDescription>{t("notify.announceHint")}</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={onAnnounce} className="grid gap-3 sm:grid-cols-2">
              <div className="grid gap-1.5">
                <Label htmlFor="kind">{t("notify.kind")}</Label>
                <select
                  id="kind"
                  className="h-9 rounded-md border bg-background px-2 text-sm"
                  value={form.kind}
                  onChange={(e) => setForm((p) => ({ ...p, kind: e.target.value }))}
                >
                  <option value="admin">{t("menus.admin")}</option>
                  <option value="web">{t("menus.web")}</option>
                </select>
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="title">{t("notify.headline")}</Label>
                <Input id="title" value={form.title} onChange={(e) => setForm((p) => ({ ...p, title: e.target.value }))} />
              </div>
              <div className="grid gap-1.5 sm:col-span-2">
                <Label htmlFor="body">{t("notify.body")}</Label>
                <Input id="body" value={form.body} onChange={(e) => setForm((p) => ({ ...p, body: e.target.value }))} />
              </div>
              <Button type="submit" disabled={announce.isPending || !form.title.trim()}>
                {t("notify.send")}
              </Button>
            </form>
          </CardContent>
        </Card>
      </Can>

      {list.error ? <p className="text-sm text-destructive">{String((list.error as Error).message)}</p> : null}
      <ResourceTable
        loading={list.isLoading}
        page={1}
        pageSize={50}
        total={list.data?.total ?? 0}
        onPageChange={() => undefined}
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("notify.headline")}</TableHead>
              <TableHead>{t("notify.body")}</TableHead>
              <TableHead>{t("notify.time")}</TableHead>
              <TableHead className="text-right">{t("app.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.length === 0 ? (
              <EmptyTableRow colSpan={4} />
            ) : (
              items.map((row) => (
                <TableRow key={row.id} className={row.readAt ? "text-muted-foreground" : ""}>
                  <TableCell>{row.title}</TableCell>
                  <TableCell className="max-w-sm truncate text-sm">{row.body}</TableCell>
                  <TableCell className="whitespace-nowrap text-xs">{formatDateTime(row.createdAt)}</TableCell>
                  <TableCell className="text-right">
                    {row.readAt ? null : (
                      <Button type="button" size="sm" variant="outline" onClick={() => void readOne.mutateAsync(row.id)}>
                        {t("notify.markRead")}
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
          <TableCaption className="sr-only">{t("notify.title")}</TableCaption>
        </Table>
      </ResourceTable>
    </div>
  )
}
