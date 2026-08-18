import { useEffect, useState } from "react"
import { Can } from "@/components/auth/Can"
import { api } from "@/lib/api"
import { formatDateTime } from "@/lib/format"
import { translateApiError, useI18n } from "@/lib/i18n"
import { P } from "@/lib/perms"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import type { OpLog } from "@/lib/types"

export function LogsPage() {
  const { t } = useI18n()
  const [rows, setRows] = useState<OpLog[]>([])
  const [error, setError] = useState("")
  const [username, setUsername] = useState("")
  const [module, setModule] = useState("")

  async function reload() {
    setRows(await api.logs({ username: username || undefined, module: module || undefined, limit: 200 }))
  }

  useEffect(() => {
    reload().catch((e: Error) => setError(translateApiError(e, t)))
  }, [])

  async function clearAll() {
    if (!confirm(t("log.confirmClear"))) return
    try {
      await api.clearLogs()
      await reload()
    } catch (e) {
      setError(translateApiError(e, t))
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{t("log.title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("log.subtitle")}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Input
            className="w-36"
            placeholder={t("login.username")}
            value={username}
            onChange={(e) => setUsername(e.target.value)}
          />
          <Input
            className="w-32"
            placeholder={t("log.module")}
            value={module}
            onChange={(e) => setModule(e.target.value)}
          />
          <Button variant="outline" onClick={() => void reload()}>
            {t("log.filter")}
          </Button>
          <Can perm={P.logClear}>
            <Button variant="destructive" onClick={() => void clearAll()}>
              {t("log.clear")}
            </Button>
          </Can>
        </div>
      </div>
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      <div className="rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>{t("login.username")}</TableHead>
              <TableHead>{t("log.module")}</TableHead>
              <TableHead>{t("log.action")}</TableHead>
              <TableHead>{t("log.path")}</TableHead>
              <TableHead>{t("app.status")}</TableHead>
              <TableHead>IP</TableHead>
              <TableHead>{t("log.time")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row) => (
              <TableRow key={row.id}>
                <TableCell className="text-muted-foreground">{row.id}</TableCell>
                <TableCell>{row.username || "—"}</TableCell>
                <TableCell>{row.module}</TableCell>
                <TableCell>{row.action}</TableCell>
                <TableCell className="font-mono text-xs">{row.path}</TableCell>
                <TableCell>
                  <Badge variant={row.status >= 400 ? "muted" : "default"}>{row.status || "—"}</Badge>
                </TableCell>
                <TableCell className="text-xs">{row.ip}</TableCell>
                <TableCell className="whitespace-nowrap text-xs">{formatDateTime(row.createdAt)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
