import { useState } from "react"
import { toast } from "sonner"
import { Can } from "@/components/auth/Can"
import { translateApiError, useI18n } from "@/providers/i18n"
import { P } from "@/constants/perms"
import { PAGE_SIZE, useAPILogs, useClearLogs, useLoginLogs, useOpLogs, usePurgeLogs } from "@/hooks/queries"
import { ConfirmAlert, EmptyTableRow, PaginationBar, TableSkeleton } from "@/components/feedback"
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
import { formatDateTime } from "@/utils/format"
import type { APILog, LoginLog, OpLog } from "@/types"

type Tab = "op" | "login" | "api"

export function LogsPage() {
  const { t } = useI18n()
  const [tab, setTab] = useState<Tab>("op")
  const [page, setPage] = useState(1)
  const [username, setUsername] = useState("")
  const [module, setModule] = useState("")
  const [action, setAction] = useState("")
  const [traceId, setTraceId] = useState("")
  const [confirm, setConfirm] = useState<"clear" | "purge" | null>(null)

  const params = { page, pageSize: PAGE_SIZE }
  const opQuery = useOpLogs({
    ...params,
    username: username || undefined,
    module: module || undefined,
    action: action || undefined,
  })
  const loginQuery = useLoginLogs({ ...params, username: username || undefined })
  const apiQuery = useAPILogs({ ...params, traceId: traceId || undefined })
  const clearLogs = useClearLogs()
  const purgeLogs = usePurgeLogs()

  const active = tab === "login" ? loginQuery : tab === "api" ? apiQuery : opQuery
  const error = active.error ? translateApiError(active.error, t) : ""

  function switchTab(next: Tab) {
    setTab(next)
    setPage(1)
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
              <Input className="w-36" placeholder={t("login.username")} value={username} onChange={(e) => { setUsername(e.target.value); setPage(1) }} />
              <Input className="w-32" placeholder={t("log.module")} value={module} onChange={(e) => { setModule(e.target.value); setPage(1) }} />
              <Input className="w-32" placeholder={t("log.action")} value={action} onChange={(e) => { setAction(e.target.value); setPage(1) }} />
            </>
          ) : null}
          {tab === "login" ? (
            <Input className="w-36" placeholder={t("login.username")} value={username} onChange={(e) => { setUsername(e.target.value); setPage(1) }} />
          ) : null}
          {tab === "api" ? (
            <Input className="w-48" placeholder="Trace ID" value={traceId} onChange={(e) => { setTraceId(e.target.value); setPage(1) }} />
          ) : null}
          <Button variant="outline" onClick={() => void active.refetch()}>
            {t("log.filter")}
          </Button>
          <Can perm={P.logClear}>
            <Button variant="destructive" onClick={() => setConfirm("clear")}>
              {t("log.clear")}
            </Button>
            <Button variant="outline" onClick={() => setConfirm("purge")}>
              {t("log.purge")}
            </Button>
          </Can>
        </div>
      </div>

      <div className="flex gap-2">
        {(["op", "login", "api"] as Tab[]).map((k) => (
          <Button key={k} size="sm" variant={tab === k ? "default" : "outline"} onClick={() => switchTab(k)}>
            {t(`log.tab.${k}`)}
          </Button>
        ))}
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {active.isLoading ? <TableSkeleton rows={8} cols={6} /> : null}

      {!active.isLoading && tab === "op" ? <OpLogTable rows={opQuery.data?.items ?? []} t={t} /> : null}
      {!active.isLoading && tab === "login" ? <LoginLogTable rows={loginQuery.data?.items ?? []} t={t} /> : null}
      {!active.isLoading && tab === "api" ? <APILogTable rows={apiQuery.data?.items ?? []} t={t} /> : null}
      <PaginationBar page={page} pageSize={PAGE_SIZE} total={active.data?.total ?? 0} onPageChange={setPage} />

      <ConfirmAlert
        open={confirm === "clear"}
        onOpenChange={(next) => {
          if (!next) setConfirm(null)
        }}
        title={t("log.clear")}
        description={t("log.confirmClear")}
        onConfirm={() => {
          clearLogs.mutate(tab === "login" ? "login" : tab === "api" ? "api" : "op", {
            onSuccess: () => toast.success(t("app.saved")),
            onError: (e) => toast.error(translateApiError(e, t)),
          })
          setConfirm(null)
        }}
      />
      <ConfirmAlert
        open={confirm === "purge"}
        onOpenChange={(next) => {
          if (!next) setConfirm(null)
        }}
        title={t("log.purge")}
        description={t("log.confirmPurge")}
        onConfirm={() => {
          purgeLogs.mutate(30, {
            onSuccess: () => toast.success(t("app.saved")),
            onError: (e) => toast.error(translateApiError(e, t)),
          })
          setConfirm(null)
        }}
      />
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
            <EmptyTableRow colSpan={10} />
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
                  <Badge variant={(row.status ?? 0) >= 400 ? "muted" : "default"}>{row.status || "—"}</Badge>
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
          {rows.length === 0 ? (
            <EmptyTableRow colSpan={8} />
          ) : (
            rows.map((row) => (
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
            ))
          )}
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
          {rows.length === 0 ? (
            <EmptyTableRow colSpan={8} />
          ) : (
            rows.map((row) => (
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
            ))
          )}
        </TableBody>
      </Table>
    </div>
  )
}
