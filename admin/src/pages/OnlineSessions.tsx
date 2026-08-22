import { toast } from "sonner"
import { Can } from "@/components/auth/Can"
import { P } from "@/constants/perms"
import { formatDateTime } from "@/utils/format"
import { translateApiError, useI18n } from "@/providers/i18n"
import { useOnlineSessions, useRevokeUserSession } from "@/hooks/queries"
import { EmptyState } from "@/components/feedback"
import { PageFallback } from "@/components/PageFallback"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"

export function OnlineSessionsPage() {
  const { t } = useI18n()
  const { data, isLoading, error, refetch } = useOnlineSessions()
  const kick = useRevokeUserSession()
  const rows = data ?? []

  if (isLoading) return <PageFallback />
  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-xl font-semibold tracking-tight">{t("online.title")}</h2>
        <p className="mt-1 text-sm text-muted-foreground">{t("online.subtitle")}</p>
      </div>
      {error ? <p className="text-sm text-destructive">{translateApiError(error as Error, t)}</p> : null}
      <Card>
        <CardHeader>
          <CardTitle>{t("online.title")}</CardTitle>
          <CardDescription>{t("online.hint")}</CardDescription>
        </CardHeader>
        <CardContent>
          {rows.length === 0 ? (
            <EmptyState />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("login.username")}</TableHead>
                  <TableHead>{t("users.account")}</TableHead>
                  <TableHead>IP</TableHead>
                  <TableHead>UA</TableHead>
                  <TableHead>{t("users.createdAt")}</TableHead>
                  <TableHead className="text-right">{t("app.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell>{row.username || "—"}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{row.userKind}</TableCell>
                    <TableCell className="font-mono text-xs">{row.ip || "—"}</TableCell>
                    <TableCell className="max-w-[18rem] truncate text-xs text-muted-foreground">
                      {row.userAgent || "—"}
                    </TableCell>
                    <TableCell className="whitespace-nowrap text-xs">{formatDateTime(row.createdAt)}</TableCell>
                    <TableCell className="text-right">
                      {row.current ? (
                        <span className="text-xs text-muted-foreground">{t("settings.thisDevice")}</span>
                      ) : (
                        <Can perm={P.userSessionRevoke}>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() =>
                              kick.mutate(
                                { id: row.userId, sid: row.id, kind: row.userKind },
                                { onSuccess: () => { toast.success(t("app.saved")); void refetch() } },
                              )
                            }
                          >
                            {t("users.kickSession")}
                          </Button>
                        </Can>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
