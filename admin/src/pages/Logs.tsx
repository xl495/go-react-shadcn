import { useState } from "react"
import { toast } from "sonner"
import { Can } from "@/components/auth/Can"
import { api } from "@/api/client"
import { translateApiError, useI18n } from "@/providers/i18n"
import { P } from "@/constants/perms"
import { useLogListParams } from "@/hooks/list-params"
import { PAGE_SIZE, useAPILogs, useClearLogs, useLoginLogs, useOpLogs, usePurgeLogs } from "@/hooks/queries"
import { FilterForm, SearchField, SearchSubmitButton, useSyncedDraft } from "@/components/SearchField"
import { ConfirmAlert, DesktopOnly, EmptyState, EmptyTableRow, ResourceTable, StackedCards } from "@/components/feedback"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { DictSelect } from "@/components/ui/dict-select"
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
import type { APILog, LoginLog, OpLog } from "@/types"

type Tab = "op" | "login" | "api"

const LOGIN_STATUSES = ["success", "failed", "warning"] as const
const OP_MODULES = ["user", "role", "perm", "dict", "mail", "config", "dept", "auth", "system"] as const
const OP_ACTIONS = ["create", "update", "delete"] as const

export function LogsPage() {
  const { t } = useI18n()
  const [{ tab, page, username, module, action, traceId, status, path }, setParams] = useLogListParams()
  const [draftUsername, setDraftUsername] = useSyncedDraft(username)
  const [draftModule, setDraftModule] = useSyncedDraft(module)
  const [draftAction, setDraftAction] = useSyncedDraft(action)
  const [draftTraceId, setDraftTraceId] = useSyncedDraft(traceId)
  const [draftStatus, setDraftStatus] = useSyncedDraft(status)
  const [draftPath, setDraftPath] = useSyncedDraft(path)
  const [confirm, setConfirm] = useState<"clear" | "purge" | null>(null)

  function searchLogs() {
    void setParams({
      username: draftUsername.trim(),
      module: draftModule,
      action: draftAction,
      traceId: draftTraceId.trim(),
      status: draftStatus,
      path: draftPath.trim(),
      page: 1,
    })
  }

  function resetLogs() {
    setDraftUsername("")
    setDraftModule("")
    setDraftAction("")
    setDraftTraceId("")
    setDraftStatus("")
    setDraftPath("")
    void setParams({
      username: "",
      module: "",
      action: "",
      traceId: "",
      status: "",
      path: "",
      page: 1,
    })
  }

  const filtered = Boolean(username || module || action || traceId || status || path)
  const draftFiltered = Boolean(draftUsername || draftModule || draftAction || draftTraceId || draftStatus || draftPath)

  const params = { page, pageSize: PAGE_SIZE }
  const opQuery = useOpLogs(
    {
      ...params,
      username: username || undefined,
      module: module || undefined,
      action: action || undefined,
    },
    tab === "op",
  )
  const loginQuery = useLoginLogs(
    {
      ...params,
      username: username || undefined,
      status: status || undefined,
    },
    tab === "login",
  )
  const apiQuery = useAPILogs(
    {
      ...params,
      traceId: traceId || undefined,
      path: path || undefined,
    },
    tab === "api",
  )
  const clearLogs = useClearLogs()
  const purgeLogs = usePurgeLogs()

  const active = tab === "login" ? loginQuery : tab === "api" ? apiQuery : opQuery
  const error = active.error ? translateApiError(active.error, t) : ""

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{t("log.title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("log.subtitle")}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Can perm={P.logExport}>
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                void api
                  .exportLogs({
                    username: username || undefined,
                    module: module || undefined,
                    action: action || undefined,
                  })
                  .catch((err) => {
                    toast.error(translateApiError(err, t))
                  })
              }}
            >
              {t("app.export")}
            </Button>
          </Can>
          <Can perm={P.logClear}>
            <Button type="button" variant="destructive" onClick={() => setConfirm("clear")}>
              {t("log.clear")}
            </Button>
            <Button type="button" variant="outline" onClick={() => setConfirm("purge")}>
              {t("log.purge")}
            </Button>
          </Can>
        </div>
      </div>
      <div className="flex gap-2" role="tablist" aria-label={t("nav.logs")}>
        {(["op", "login", "api"] as Tab[]).map((k) => (
          <Button
            key={k}
            size="sm"
            role="tab"
            aria-selected={tab === k}
            variant={tab === k ? "default" : "outline"}
            onClick={() => void setParams({ tab: k, page: 1 })}
            onKeyDown={(e) => {
              if (e.key !== "ArrowRight" && e.key !== "ArrowLeft") return
              const tabs: Tab[] = ["op", "login", "api"]
              const i = tabs.indexOf(tab)
              const next = e.key === "ArrowRight" ? (i + 1) % tabs.length : (i + tabs.length - 1) % tabs.length
              void setParams({ tab: tabs[next], page: 1 })
            }}
          >
            {t(`log.tab.${k}`)}
          </Button>
        ))}
      </div>
      <FilterForm onSubmit={searchLogs}>
        {tab === "op" ? (
          <>
            <SearchField id="log-username" label={t("login.username")} value={draftUsername} placeholder={t("login.username")} inputClassName="w-40" onChange={setDraftUsername} />
            <DictSelect
              id="log-module"
              className="w-36"
              label={t("log.module")}
              value={draftModule}
              items={OP_MODULES.map((value) => ({ value, label: t(`log.modules.${value}`) }))}
              allowEmpty
              emptyLabel={t("app.all")}
              onChange={setDraftModule}
            />
            <DictSelect
              id="log-action"
              className="w-36"
              label={t("log.action")}
              value={draftAction}
              items={OP_ACTIONS.map((value) => ({ value, label: t(`log.actions.${value}`) }))}
              allowEmpty
              emptyLabel={t("app.all")}
              onChange={setDraftAction}
            />
          </>
        ) : null}
        {tab === "login" ? (
          <>
            <SearchField id="login-username" label={t("login.username")} value={draftUsername} placeholder={t("login.username")} inputClassName="w-40" onChange={setDraftUsername} />
            <DictSelect
              id="login-status"
              className="w-36"
              label={t("app.status")}
              value={draftStatus}
              items={LOGIN_STATUSES.map((value) => ({ value, label: t(`log.loginStatus.${value}`) }))}
              allowEmpty
              emptyLabel={t("app.all")}
              onChange={setDraftStatus}
            />
          </>
        ) : null}
        {tab === "api" ? (
          <>
            <SearchField id="api-trace" label={t("log.traceId")} value={draftTraceId} placeholder={t("log.traceId")} inputClassName="w-56" onChange={setDraftTraceId} />
            <SearchField id="api-path" label={t("log.path")} value={draftPath} placeholder={t("log.path")} inputClassName="w-56" onChange={setDraftPath} />
          </>
        ) : null}
        <SearchSubmitButton />
        {filtered || draftFiltered ? (
          <Button type="button" variant="outline" onClick={resetLogs}>
            {t("app.resetFilters")}
          </Button>
        ) : null}
      </FilterForm>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      <ResourceTable
        loading={active.isLoading}
        page={page}
        pageSize={PAGE_SIZE}
        total={active.data?.total ?? 0}
        onPageChange={(next) => void setParams({ page: next })}
      >
        {tab === "op" ? <OpLogTable rows={opQuery.data?.items ?? []} t={t} /> : null}
        {tab === "login" ? <LoginLogTable rows={loginQuery.data?.items ?? []} t={t} /> : null}
        {tab === "api" ? <APILogTable rows={apiQuery.data?.items ?? []} t={t} /> : null}
      </ResourceTable>

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
          })
          setConfirm(null)
        }}
      />
    </div>
  )
}

function OpLogTable({ rows, t }: { rows: OpLog[]; t: (k: string) => string }) {
  return (
    <>
      <StackedCards>
        {rows.length === 0 ? (
          <EmptyState />
        ) : (
          rows.map((row) => (
            <div key={row.id} className="rounded-lg border p-3 space-y-1">
              <div className="flex items-center justify-between gap-2">
                <p className="truncate font-medium">{row.username || "—"}</p>
                <Badge variant={(row.status ?? 0) >= 400 ? "muted" : "default"}>{row.status || "—"}</Badge>
              </div>
              <p className="text-xs text-muted-foreground">
                {row.module} · {row.action}
              </p>
              <p className="truncate font-mono text-xs">{row.path}</p>
              <p className="text-xs text-muted-foreground">{formatDateTime(row.createdAt)}</p>
            </div>
          ))
        )}
      </StackedCards>
      <DesktopOnly>
      <Table>
        <TableCaption className="sr-only">{t("log.tab.op")}</TableCaption>
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
      </DesktopOnly>
    </>
  )
}

function LoginLogTable({ rows, t }: { rows: LoginLog[]; t: (k: string) => string }) {
  function statusLabel(value: string) {
    const key = `log.loginStatus.${value}`
    const got = t(key)
    return got === key ? value : got
  }
  return (
    <>
      <StackedCards>
        {rows.length === 0 ? (
          <EmptyState />
        ) : (
          rows.map((row) => (
            <div key={row.id} className="rounded-lg border p-3 space-y-1">
              <div className="flex items-center justify-between gap-2">
                <p className="truncate font-medium">{row.username}</p>
                <Badge variant={row.status === "success" ? "default" : "muted"}>{statusLabel(row.status)}</Badge>
              </div>
              <p className="text-xs text-muted-foreground">{row.ip || "—"}</p>
              {row.failReason ? <p className="text-xs text-destructive">{row.failReason}</p> : null}
              <p className="text-xs text-muted-foreground">{formatDateTime(row.createdAt)}</p>
            </div>
          ))
        )}
      </StackedCards>
      <DesktopOnly>
      <Table>
        <TableCaption className="sr-only">{t("log.tab.login")}</TableCaption>
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
                  <Badge variant={row.status === "success" ? "default" : "muted"}>{statusLabel(row.status)}</Badge>
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
      </DesktopOnly>
    </>
  )
}

function APILogTable({ rows, t }: { rows: APILog[]; t: (k: string) => string }) {
  return (
    <>
      <StackedCards>
        {rows.length === 0 ? (
          <EmptyState />
        ) : (
          rows.map((row) => (
            <div key={row.id} className="rounded-lg border p-3 space-y-1">
              <div className="flex items-center justify-between gap-2">
                <p className="truncate font-medium">
                  {row.method} {row.path}
                </p>
                <Badge variant={row.status >= 400 ? "muted" : "default"}>{row.status}</Badge>
              </div>
              <p className="truncate font-mono text-xs text-muted-foreground">{row.traceId}</p>
              <p className="text-xs text-muted-foreground">
                {row.username || "—"} · {row.latencyMs ? `${row.latencyMs}ms` : "—"} · {formatDateTime(row.createdAt)}
              </p>
            </div>
          ))
        )}
      </StackedCards>
      <DesktopOnly>
      <Table>
        <TableCaption className="sr-only">{t("log.tab.api")}</TableCaption>
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
      </DesktopOnly>
    </>
  )
}
