import { useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { Can } from "@/components/auth/Can"
import { api } from "@/lib/api"
import { translateApiError, useI18n } from "@/lib/i18n"
import { P } from "@/lib/perms"
import { useAPILogs, useLoginLogs, useOpLogs } from "@/lib/queries"
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
import { formatDateTime } from "@/lib/format"
import type { APILog, LoginLog, OpLog } from "@/lib/types"

type Tab = "op" | "login" | "api"

export function LogsPage() {
  const { t } = useI18n()
  const qc = useQueryClient()
  const [tab, setTab] = useState<Tab>("op")
  const [username, setUsername] = useState("")
  const [module, setModule] = useState("")
  const [action, setAction] = useState("")
  const [traceId, setTraceId] = useState("")

  const opQuery = useOpLogs({
    username: username || undefined,
    module: module || undefined,
    action: action || undefined,
    pageSize: 200,
  })
  const loginQuery = useLoginLogs({ username: username || undefined, pageSize: 200 })
  const apiQuery = useAPILogs({ traceId: traceId || undefined, pageSize: 200 })

  const active = tab === "login" ? loginQuery : tab === "api" ? apiQuery : opQuery
  const error = active.error ? translateApiError(active.error as Error, t) : ""

  async function clearAll() {
    if (!confirm(t("log.confirmClear"))) return
    await api.clearLogs(tab === "login" ? "login" : tab === "api" ? "api" : "op")
    await qc.invalidateQueries({ queryKey: ["logs"] })
  }

  async function purgeOld() {
    if (!confirm(t("log.confirmPurge"))) return
    await api.purgeLogs(30)
    await qc.invalidateQueries({ queryKey: ["logs"] })
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{t("log.title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("log.subtitle")}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {tab === "op" ? (
            <>
              <Input className="w-36" placeholder={t("login.username")} value={username} onChange={(e) => setUsername(e.target.value)} />
              <Input className="w-32" placeholder={t("log.module")} value={module} onChange={(e) => setModule(e.target.value)} />
              <Input className="w-32" placeholder={t("log.action")} value={action} onChange={(e) => setAction(e.target.value)} />
            </>
          ) : null}
          {tab === "login" ? (
            <Input className="w-36" placeholder={t("login.username")} value={username} onChange={(e) => setUsername(e.target.value)} />
          ) : null}
          {tab === "api" ? (
            <Input className="w-48" placeholder="Trace ID" value={traceId} onChange={(e) => setTraceId(e.target.value)} />
          ) : null}
          <Button variant="outline" onClick={() => void active.refetch()}>
            {t("log.filter")}
          </Button>
          <Can perm={P.logClear}>
            <Button variant="destructive" onClick={() => void clearAll()}>
              {t("log.clear")}
            </Button>
            <Button variant="outline" onClick={() => void purgeOld()}>
              {t("log.purge")}
            </Button>
          </Can>
        </div>
      </div>

      <div className="flex gap-2">
        {(["op", "login", "api"] as Tab[]).map((k) => (
          <Button key={k} size="sm" variant={tab === k ? "default" : "outline"} onClick={() => setTab(k)}>
            {t(`log.tab.${k}`)}
          </Button>
        ))}
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {active.isLoading ? <p className="text-sm text-muted-foreground">{t("app.loading")}</p> : null}

      {tab === "op" ? <OpLogTable rows={opQuery.data?.items ?? []} t={t} /> : null}
      {tab === "login" ? <LoginLogTable rows={loginQuery.data?.items ?? []} t={t} /> : null}
      {tab === "api" ? <APILogTable rows={apiQuery.data?.items ?? []} t={t} /> : null}
    </div>
  )
}

function OpLogTable({ rows, t }: { rows: OpLog[]; t: (k: string) => string }) {
  return (
    <div className="rounded-lg border bg-card">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>ID</TableHead>
            <TableHead>Trace</TableHead>
            <TableHead>{t("login.username")}</TableHead>
            <TableHead>{t("log.module")}</TableHead>
            <TableHead>{t("log.action")}</TableHead>
            <TableHead>{t("log.path")}</TableHead>
            <TableHead>{t("app.status")}</TableHead>
            <TableHead>IP</TableHead>
            <TableHead>{t("log.latency")}</TableHead>
            <TableHead>{t("log.time")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.length === 0 ? (
            <TableRow>
              <TableCell colSpan={10} className="py-10 text-center text-sm text-muted-foreground">
                {t("log.empty")}
              </TableCell>
            </TableRow>
          ) : (
            rows.map((row) => (
              <TableRow key={row.id}>
                <TableCell className="text-muted-foreground">{row.id}</TableCell>
                <TableCell className="max-w-[8rem] truncate font-mono text-xs">{row.traceId || "—"}</TableCell>
                <TableCell>{row.username || "—"}</TableCell>
                <TableCell>{row.module}</TableCell>
                <TableCell>{row.action}</TableCell>
                <TableCell className="font-mono text-xs">{row.path}</TableCell>
                <TableCell>
                  <Badge variant={row.status >= 400 ? "muted" : "default"}>{row.status || "—"}</Badge>
                </TableCell>
                <TableCell className="text-xs">{row.ip}</TableCell>
                <TableCell className="text-xs">{row.latencyMs ? `${row.latencyMs}ms` : "—"}</TableCell>
                <TableCell className="whitespace-nowrap text-xs">{formatDateTime(row.createdAt)}</TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
  )
}

function LoginLogTable({ rows, t }: { rows: LoginLog[]; t: (k: string) => string }) {
  return (
    <div className="rounded-lg border bg-card">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>ID</TableHead>
            <TableHead>{t("login.username")}</TableHead>
            <TableHead>{t("app.status")}</TableHead>
            <TableHead>IP</TableHead>
            <TableHead>{t("log.location")}</TableHead>
            <TableHead>UA</TableHead>
            <TableHead>{t("log.failReason")}</TableHead>
            <TableHead>{t("log.time")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row) => (
            <TableRow key={row.id}>
              <TableCell>{row.id}</TableCell>
              <TableCell>{row.username}</TableCell>
              <TableCell>
                <Badge variant={row.status === "success" ? "default" : "muted"}>{row.status}</Badge>
              </TableCell>
              <TableCell className="text-xs">{row.ip}</TableCell>
              <TableCell className="text-xs">{row.location || "—"}</TableCell>
              <TableCell className="max-w-[12rem] truncate text-xs" title={row.userAgent}>
                {row.userAgent || "—"}
              </TableCell>
              <TableCell className="text-xs">{row.failReason || "—"}</TableCell>
              <TableCell className="whitespace-nowrap text-xs">{formatDateTime(row.createdAt)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

function APILogTable({ rows, t }: { rows: APILog[]; t: (k: string) => string }) {
  return (
    <div className="rounded-lg border bg-card">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>ID</TableHead>
            <TableHead>Trace</TableHead>
            <TableHead>{t("login.username")}</TableHead>
            <TableHead>Method</TableHead>
            <TableHead>{t("log.path")}</TableHead>
            <TableHead>{t("app.status")}</TableHead>
            <TableHead>{t("log.latency")}</TableHead>
            <TableHead>{t("log.time")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row) => (
            <TableRow key={row.id}>
              <TableCell>{row.id}</TableCell>
              <TableCell className="font-mono text-xs">{row.traceId}</TableCell>
              <TableCell>{row.username || "—"}</TableCell>
              <TableCell>{row.method}</TableCell>
              <TableCell className="font-mono text-xs">{row.path}</TableCell>
              <TableCell>{row.status}</TableCell>
              <TableCell className="text-xs">{row.latencyMs ? `${row.latencyMs}ms` : "—"}</TableCell>
              <TableCell className="whitespace-nowrap text-xs">{formatDateTime(row.createdAt)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
