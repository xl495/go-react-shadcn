import { useEffect, useRef, useState, type FormEvent } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
import { toast } from "sonner"
import { Can } from "@/components/auth/Can"
import { PageFallback } from "@/components/PageFallback"
import { useAuth } from "@/providers/auth"
import { translateApiError, useI18n } from "@/providers/i18n"
import { P } from "@/constants/perms"
import {
  useCreateMailCampaign,
  useMailCampaign,
  useScheduleMailCampaign,
  useUpdateMailCampaign,
} from "@/hooks/queries"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { Badge } from "@/components/ui/badge"

const VARS = ["{{nickname}}", "{{username}}", "{{email}}", "{{unsubscribe}}"] as const

const emptyForm = {
  name: "",
  subject: "你好 {{nickname}}",
  body: "您好 {{nickname}}，\n\n在这里写邮件内容。\n\n不想再收到此类邮件：{{unsubscribe}}\n",
  audience: "opted_in",
}

function looksHTML(s: string) {
  return /<\s*(html|p|div|br|h1|h2|table|span|a)\b/i.test(s)
}

function previewText(template: string, vars: Record<string, string>) {
  return template
    .replaceAll("{{nickname}}", vars.nickname)
    .replaceAll("{{name}}", vars.nickname)
    .replaceAll("{{username}}", vars.username)
    .replaceAll("{{email}}", vars.email)
    .replaceAll("{{unsubscribe}}", vars.unsubscribe)
}

export function MailTemplateEditorPage() {
  const { id } = useParams()
  const isNew = !id || id === "new"
  const campaignId = isNew ? 0 : Number(id)
  const { t } = useI18n()
  const { user } = useAuth()
  const navigate = useNavigate()
  const { data, isLoading, error } = useMailCampaign(campaignId)
  const createCampaign = useCreateMailCampaign()
  const updateCampaign = useUpdateMailCampaign()
  const scheduleCampaign = useScheduleMailCampaign()
  const [form, setForm] = useState(emptyForm)
  const bodyRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    if (!data) return
    setForm({
      name: data.name,
      subject: data.subject,
      body: data.body ?? "",
      audience: data.audience || "opted_in",
    })
  }, [data])

  const status = data?.status ?? "draft"
  const locked = !isNew && status !== "draft" && status !== "paused"
  const saving = createCampaign.isPending || updateCampaign.isPending

  const sample = {
    nickname: user?.nickname || user?.username || "张三",
    username: user?.username || "zhangsan",
    email: user?.email || "user@latch.local",
    unsubscribe: "https://example.com/unsubscribe?token=preview",
  }
  const subjectPreview = previewText(form.subject, sample)
  const bodyPreview = previewText(form.body, sample)

  function insertVar(token: string) {
    const el = bodyRef.current
    if (!el) {
      setForm((f) => ({ ...f, body: f.body + token }))
      return
    }
    const start = el.selectionStart
    const end = el.selectionEnd
    const next = form.body.slice(0, start) + token + form.body.slice(end)
    setForm((f) => ({ ...f, body: next }))
    requestAnimationFrame(() => {
      el.focus()
      const pos = start + token.length
      el.setSelectionRange(pos, pos)
    })
  }

  async function onSave(e: FormEvent) {
    e.preventDefault()
    try {
      if (isNew) {
        const row = await createCampaign.mutateAsync(form)
        toast.success(t("app.saved"))
        navigate(`/mail/campaigns/${row.id}`, { replace: true })
        return
      }
      await updateCampaign.mutateAsync({ id: campaignId, body: form })
      toast.success(t("app.saved"))
    } catch (err) {
      toast.error(translateApiError(err, t))
    }
  }

  if (!isNew && !campaignId) {
    return <p className="text-sm text-destructive">{t("errors.40410")}</p>
  }
  if (!isNew && isLoading) return <PageFallback />
  if (!isNew && error) {
    return <p className="text-sm text-destructive">{translateApiError(error as Error, t)}</p>
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">
            {isNew ? t("mail.createCampaign") : t("mail.editCampaign")}
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("mail.templateHint")}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {!isNew ? (
            <Badge variant={status === "done" ? "default" : "muted"}>
              {t(`mail.campaignStatus.${status}`)}
            </Badge>
          ) : null}
          <Button asChild variant="outline">
            <Link to="/mail/campaigns">{t("mail.backToList")}</Link>
          </Button>
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{t("mail.compose")}</CardTitle>
            <CardDescription>{t("mail.bodyHint")}</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={(e) => void onSave(e)} className="grid gap-4">
              <div className="grid gap-1.5">
                <Label htmlFor="tn">{t("app.name")}</Label>
                <Input
                  id="tn"
                  value={form.name}
                  disabled={locked}
                  onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="ts">{t("mail.subject")}</Label>
                <Input
                  id="ts"
                  value={form.subject}
                  disabled={locked}
                  onChange={(e) => setForm((f) => ({ ...f, subject: e.target.value }))}
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="tb">{t("mail.body")}</Label>
                <div className="flex flex-wrap gap-1 pb-1">
                  {VARS.map((token) => (
                    <Button
                      key={token}
                      type="button"
                      variant="outline"
                      size="sm"
                      className="h-7 font-mono text-xs"
                      disabled={locked}
                      onClick={() => insertVar(token)}
                    >
                      {token}
                    </Button>
                  ))}
                </div>
                <Textarea
                  ref={bodyRef}
                  id="tb"
                  value={form.body}
                  disabled={locked}
                  onChange={(e) => setForm((f) => ({ ...f, body: e.target.value }))}
                  className="min-h-56 font-mono text-sm"
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="ta">{t("mail.audienceLabel")}</Label>
                <Select
                  value={form.audience}
                  onValueChange={(audience) => setForm((f) => ({ ...f, audience }))}
                  disabled={locked}
                >
                  <SelectTrigger id="ta">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="opted_in">{t("mail.audience.opted_in")}</SelectItem>
                    <SelectItem value="all_active">{t("mail.audience.all_active")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="flex flex-wrap gap-2">
                {!locked ? (
                  <Can perm={isNew ? P.mailCampaignCreate : P.mailCampaignUpdate}>
                    <Button type="submit" disabled={saving}>
                      {saving ? t("app.saving") : t("app.save")}
                    </Button>
                  </Can>
                ) : null}
                {!isNew && (status === "draft" || status === "paused") ? (
                  <Can perm={P.mailCampaignSchedule}>
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() =>
                        scheduleCampaign.mutate(
                          { id: campaignId },
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
                ) : null}
              </div>
            </form>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t("mail.preview")}</CardTitle>
            <CardDescription>{t("mail.previewHint")}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <p className="text-sm font-medium">{subjectPreview || t("mail.subject")}</p>
            {looksHTML(form.body) ? (
              <iframe
                title={t("mail.preview")}
                sandbox=""
                className="min-h-80 w-full rounded-md border bg-background"
                srcDoc={bodyPreview}
              />
            ) : (
              <pre className="min-h-80 whitespace-pre-wrap rounded-md border bg-muted/40 p-3 text-sm">
                {bodyPreview}
              </pre>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
