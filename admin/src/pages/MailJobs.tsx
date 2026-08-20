import { useState } from "react"
import { toast } from "sonner"
import { Can } from "@/components/auth/Can"
import { translateApiError, useI18n } from "@/providers/i18n"
import { P } from "@/constants/perms"
import { useMailJobListParams } from "@/hooks/list-params"
import { PAGE_SIZE, useCancelMailJob, useMailJobs, useRetryMailJob } from "@/hooks/queries"
import { FilterForm, SearchSubmitButton, useSyncedDraft } from "@/components/SearchField"
import { ConfirmAlert, EmptyTableRow, PaginationBar, TableSkeleton } from "@/components/feedback"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { DictSelect } from "@/components/ui/dict-select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { formatDateTime } from "@/utils/format"
import type { MailJob } from "@/types"

function jobLabel(t: (k: string) => string, prefix: string, value: string) {
  const key = `${prefix}.${value}`
  const got = t(key)
  return got === key ? value : got
}

const JOB_STATUSES = ["queued", "sending", "sent", "failed", "dead", "canceled"] as const
const JOB_CLASSES = ["transactional", "operational", "marketing"] as const

export function MailJobsPage() {
  const { t } = useI18n()
  const [{ page, status, class: klass }, setParams] = useMailJobListParams()
  const [draftStatus, setDraftStatus] = useSyncedDraft(status)
  const [draftClass, setDraftClass] = useSyncedDraft(klass)
  const { data, isLoading, error } = useMailJobs({
    page,
    pageSize: PAGE_SIZE,
    status: status || undefined,
    class: klass || undefined,
  })
  const retryJob = useRetryMailJob()
  const cancelJob = useCancelMailJob()
  const rows = data?.items ?? []
  const [pending, setPending] = useState<{ job: MailJob; action: "retry" | "cancel" } | null>(null)

  function searchJobs() {
    void setParams({ status: draftStatus, class: draftClass, page: 1 })
  }

  function resetJobs() {
    setDraftStatus("")
    setDraftClass("")
    void setParams({ status: "", class: "", page: 1 })
  }

  const filtered = Boolean(status || klass)
  const draftFiltered = Boolean(draftStatus || draftClass)

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{t("mail.jobsTitle")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("mail.jobsSubtitle")}</p>
        </div>
      </div>
      <FilterForm onSubmit={searchJobs}>
          <DictSelect
            id="job-status"
            className="w-36"
            label={t("app.status")}
            value={draftStatus}
            items={JOB_STATUSES.map((value) => ({ value, label: jobLabel(t, "mail.jobStatus", value) }))}
            allowEmpty
            emptyLabel={t("app.all")}
            onChange={setDraftStatus}
          />
          <DictSelect
            id="job-class"
            className="w-36"
            label={t("mail.classFilter")}
            value={draftClass}
            items={JOB_CLASSES.map((value) => ({ value, label: jobLabel(t, "mail.class", value) }))}
            allowEmpty
            emptyLabel={t("app.all")}
            onChange={setDraftClass}
          />
          <SearchSubmitButton />
          {filtered || draftFiltered ? (
            <Button type="button" variant="outline" onClick={resetJobs}>
              {t("app.resetFilters")}
            </Button>
          ) : null}
        </FilterForm>
      {error ? <p className="text-sm text-destructive">{translateApiError(error, t)}</p> : null}
      <div className="rounded-lg border bg-card">
        {isLoading ? (
          <TableSkeleton rows={8} cols={7} />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>{t("mail.classFilter")}</TableHead>
                <TableHead>{t("mail.to")}</TableHead>
                <TableHead>{t("mail.subject")}</TableHead>
                <TableHead>{t("app.status")}</TableHead>
                <TableHead>{t("mail.sendAfter")}</TableHead>
                <TableHead>{t("mail.lastError")}</TableHead>
                <TableHead className="text-right">{t("app.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.length === 0 ? (
                <EmptyTableRow colSpan={8} />
              ) : (
                rows.map((job) => (
                  <TableRow key={job.id}>
                    <TableCell className="tabular-nums text-muted-foreground">{job.id}</TableCell>
                    <TableCell>{jobLabel(t, "mail.class", job.class)}</TableCell>
                    <TableCell>{job.toEmail}</TableCell>
                    <TableCell className="max-w-[16rem] truncate">{job.subject}</TableCell>
                    <TableCell>
                      <Badge variant={job.status === "sent" ? "default" : "muted"}>
                        {jobLabel(t, "mail.jobStatus", job.status)}
                      </Badge>
                    </TableCell>
                    <TableCell className="whitespace-nowrap text-xs">{formatDateTime(job.sendAfter)}</TableCell>
                    <TableCell className="max-w-56 truncate text-xs text-muted-foreground">
                      {job.lastError || "—"}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-1">
                        {(job.status === "dead" || job.status === "failed" || job.status === "canceled") && (
                          <Can perm={P.mailJobsRetry}>
                            <Button variant="ghost" size="sm" onClick={() => setPending({ job, action: "retry" })}>
                              {t("mail.retry")}
                            </Button>
                          </Can>
                        )}
                        {job.status === "queued" ? (
                          <Can perm={P.mailJobsCancel}>
                            <Button variant="ghost" size="sm" onClick={() => setPending({ job, action: "cancel" })}>
                              {t("app.cancel")}
                            </Button>
                          </Can>
                        ) : null}
                      </div>
                    </TableCell>
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
        title={pending?.action === "retry" ? t("mail.retry") : t("app.cancel")}
        description={
          pending?.action === "retry" ? t("mail.confirmRetry") : t("mail.confirmCancel")
        }
        onConfirm={() => {
          if (!pending) return
          const { job, action } = pending
          const run = action === "retry" ? retryJob.mutate : cancelJob.mutate
          run(job.id, {
            onSuccess: () => toast.success(t("app.saved")),
            onError: (e) => toast.error(translateApiError(e, t)),
          })
          setPending(null)
        }}
      />
    </div>
  )
}
