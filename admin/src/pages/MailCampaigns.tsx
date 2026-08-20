import { Link } from "react-router-dom"
import { toast } from "sonner"
import { Can } from "@/components/auth/Can"
import { translateApiError, useI18n } from "@/providers/i18n"
import { P } from "@/constants/perms"
import { useMailCampaignListParams } from "@/hooks/list-params"
import {
  PAGE_SIZE,
  useDeleteMailCampaign,
  useMailCampaigns,
  useScheduleMailCampaign,
  useUpdateMailCampaign,
} from "@/hooks/queries"
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
import type { MailCampaign } from "@/types"
import { useState } from "react"

function campaignLabel(t: (k: string) => string, prefix: string, value: string) {
  const key = `${prefix}.${value}`
  const got = t(key)
  return got === key ? value : got
}

const CAMPAIGN_STATUSES = ["draft", "scheduled", "running", "paused", "done"] as const

export function MailCampaignsPage() {
  const { t } = useI18n()
  const [{ page, status }, setParams] = useMailCampaignListParams()
  const [draftStatus, setDraftStatus] = useSyncedDraft(status)
  const { data, isLoading, error } = useMailCampaigns({
    page,
    pageSize: PAGE_SIZE,
    status: status || undefined,
  })
  const updateCampaign = useUpdateMailCampaign()
  const deleteCampaign = useDeleteMailCampaign()
  const scheduleCampaign = useScheduleMailCampaign()
  const rows = data?.items ?? []
  const [pending, setPending] = useState<MailCampaign | null>(null)
  const filtered = Boolean(status)
  const draftFiltered = Boolean(draftStatus)

  const editable = (row: MailCampaign) => row.status === "draft" || row.status === "paused"

  function searchCampaigns() {
    void setParams({ status: draftStatus, page: 1 })
  }

  function resetCampaigns() {
    setDraftStatus("")
    void setParams({ status: "", page: 1 })
  }

  return (
    <div className="space-y-4">
      <div className="flex items-end justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{t("mail.campaignsTitle")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("mail.campaignsSubtitle")}</p>
        </div>
        <Can perm={P.mailCampaignCreate}>
          <Button asChild>
            <Link to="/mail/campaigns/new">{t("mail.createCampaign")}</Link>
          </Button>
        </Can>
      </div>
      <FilterForm onSubmit={searchCampaigns}>
        <DictSelect
          id="campaign-status"
          className="w-36"
          label={t("app.status")}
          value={draftStatus}
          items={CAMPAIGN_STATUSES.map((value) => ({ value, label: campaignLabel(t, "mail.campaignStatus", value) }))}
          allowEmpty
          emptyLabel={t("app.all")}
          onChange={setDraftStatus}
        />
        <SearchSubmitButton />
        {filtered || draftFiltered ? (
          <Button type="button" variant="outline" onClick={resetCampaigns}>
            {t("app.resetFilters")}
          </Button>
        ) : null}
      </FilterForm>
      {error ? <p className="text-sm text-destructive">{translateApiError(error, t)}</p> : null}
      <div className="rounded-lg border bg-card">
        {isLoading ? (
          <TableSkeleton rows={8} cols={6} />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>{t("app.name")}</TableHead>
                <TableHead>{t("mail.subject")}</TableHead>
                <TableHead>{t("mail.audienceLabel")}</TableHead>
                <TableHead>{t("app.status")}</TableHead>
                <TableHead>{t("mail.jobCount")}</TableHead>
                <TableHead>{t("mail.scheduledAt")}</TableHead>
                <TableHead className="text-right">{t("app.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.length === 0 ? (
                <EmptyTableRow colSpan={8} />
              ) : (
                rows.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className="tabular-nums text-muted-foreground">{row.id}</TableCell>
                    <TableCell className="font-medium">
                      <Link to={`/mail/campaigns/${row.id}`} className="hover:underline">
                        {row.name}
                      </Link>
                    </TableCell>
                    <TableCell className="max-w-[16rem] truncate">{row.subject}</TableCell>
                    <TableCell>{campaignLabel(t, "mail.audience", row.audience)}</TableCell>
                    <TableCell>
                      <Badge variant={row.status === "done" ? "default" : "muted"}>
                        {campaignLabel(t, "mail.campaignStatus", row.status)}
                      </Badge>
                    </TableCell>
                    <TableCell>{row.jobCount ?? 0}</TableCell>
                    <TableCell className="whitespace-nowrap text-xs">{formatDateTime(row.scheduledAt)}</TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-1">
                        <Button variant="ghost" size="sm" asChild>
                          <Link to={`/mail/campaigns/${row.id}`}>{t("mail.openTemplate")}</Link>
                        </Button>
                        {(row.status === "draft" || row.status === "paused") && (
                          <Can perm={P.mailCampaignSchedule}>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() =>
                                scheduleCampaign.mutate(
                                  { id: row.id },
                                  {
                                    onSuccess: () => toast.success(t("mail.scheduled")),
                                    onError: (e) => toast.error(translateApiError(e, t)),
                                  },
                                )
                              }
                            >
                              {t("mail.schedule")}
                            </Button>
                          </Can>
                        )}
                        {(row.status === "scheduled" || row.status === "running") && (
                          <Can perm={P.mailCampaignUpdate}>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() =>
                                updateCampaign.mutate(
                                  { id: row.id, body: { status: "paused" } },
                                  {
                                    onSuccess: () => toast.success(t("mail.paused")),
                                    onError: (e) => toast.error(translateApiError(e, t)),
                                  },
                                )
                              }
                            >
                              {t("mail.pause")}
                            </Button>
                          </Can>
                        )}
                        {editable(row) ? (
                          <Can perm={P.mailCampaignDelete}>
                            <Button variant="ghost" size="sm" onClick={() => setPending(row)}>
                              {t("app.delete")}
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
        title={t("app.delete")}
        description={pending ? t("mail.confirmDeleteCampaign", { name: pending.name }) : ""}
        onConfirm={() => {
          if (!pending) return
          deleteCampaign.mutate(pending.id, {
            onSuccess: () => toast.success(t("app.saved")),
            onError: (e) => toast.error(translateApiError(e, t)),
          })
          setPending(null)
        }}
      />
    </div>
  )
}
