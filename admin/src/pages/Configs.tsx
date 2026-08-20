import { useEffect, useMemo, useRef, useState, type FormEvent, type KeyboardEvent } from "react"
import { toast } from "sonner"
import { Can } from "@/components/auth/Can"
import { useAuth } from "@/providers/auth"
import { translateApiError, useI18n } from "@/providers/i18n"
import { P } from "@/constants/perms"
import { useConfigListParams } from "@/hooks/list-params"
import { useConfigs, useSaveConfigs, useTestMail } from "@/hooks/queries"
import { cn } from "@/utils/cn"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { DictSelect } from "@/components/ui/dict-select"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { TimezoneSelect } from "@/components/ui/timezone-select"
import type { SysConfig } from "@/types"

const SECRET_MASK = "********"
const PAGE_SIZE = 500

type TabId = "app" | "auth" | "mail" | "other"
type MailSection = "smtp" | "policy"
type FieldKind = "text" | "password" | "number" | "switch" | "select" | "timezone" | "time" | "url" | "email" | "heading"

type FieldSpec = {
  key: string
  kind: FieldKind
  titleKey?: string
  options?: { value: string; labelKey: string }[]
}

type Group = {
  id: MailSection | "app" | "auth" | "other"
  titleKey: string
  hintKey: string
  fields: FieldSpec[]
}

const APP_GROUP: Group = {
  id: "app",
  titleKey: "config.groupApp",
  hintKey: "config.appHint",
  fields: [
    { key: "app.name", kind: "text" },
    {
      key: "app.default_locale",
      kind: "select",
      options: [
        { value: "zh-CN", labelKey: "config.localeZh" },
        { value: "en", labelKey: "config.localeEn" },
      ],
    },
  ],
}

const AUTH_GROUP: Group = {
  id: "auth",
  titleKey: "config.groupAuth",
  hintKey: "config.authHint",
  fields: [
    { key: "google", kind: "heading", titleKey: "config.googleAuth" },
    { key: "auth.google_enabled", kind: "switch" },
    { key: "auth.google_register_enabled", kind: "switch" },
    { key: "auth.google_client_id", kind: "text" },
    { key: "auth.google_client_secret", kind: "password" },
    { key: "captcha", kind: "heading", titleKey: "config.captchaAuth" },
    {
      key: "auth.captcha_provider",
      kind: "select",
      options: [
        { value: "none", labelKey: "config.captchaNone" },
        { value: "image", labelKey: "config.captchaImage" },
        { value: "recaptcha", labelKey: "config.captchaRecaptcha" },
        { value: "turnstile", labelKey: "config.captchaTurnstile" },
      ],
    },
    { key: "auth.recaptcha_site_key_v3", kind: "text" },
    { key: "auth.recaptcha_secret_v3", kind: "password" },
    { key: "auth.recaptcha_site_key_v2", kind: "text" },
    { key: "auth.recaptcha_secret_v2", kind: "password" },
    { key: "auth.recaptcha_min_score", kind: "number" },
    { key: "auth.turnstile_site_key", kind: "text" },
    { key: "auth.turnstile_secret", kind: "password" },
  ],
}

const MAIL_GROUPS: Record<MailSection, Group> = {
  smtp: {
    id: "smtp",
    titleKey: "config.mailSmtp",
    hintKey: "config.mailSmtpHint",
    fields: [
      { key: "mail.enabled", kind: "switch" },
      { key: "mail.host", kind: "text" },
      { key: "mail.port", kind: "number" },
      { key: "mail.username", kind: "text" },
      { key: "mail.password", kind: "password" },
      { key: "mail.from", kind: "email" },
      { key: "mail.from_name", kind: "text" },
      {
        key: "mail.tls",
        kind: "select",
        options: [
          { value: "starttls", labelKey: "config.tlsStarttls" },
          { value: "ssl", labelKey: "config.tlsSsl" },
          { value: "none", labelKey: "config.tlsNone" },
        ],
      },
    ],
  },
  policy: {
    id: "policy",
    titleKey: "config.mailPolicy",
    hintKey: "config.mailPolicyHint",
    fields: [
      { key: "mail.reset_base_url", kind: "url" },
      { key: "mail.default_timezone", kind: "timezone" },
      { key: "quiet", kind: "heading", titleKey: "config.quietHours" },
      { key: "mail.quiet_start", kind: "time" },
      { key: "mail.quiet_end", kind: "time" },
      { key: "marketing", kind: "heading", titleKey: "config.marketingHours" },
      { key: "mail.marketing_start", kind: "time" },
      { key: "mail.marketing_end", kind: "time" },
      { key: "mail.rate_per_minute", kind: "number" },
      { key: "mail.max_attempts", kind: "number" },
      { key: "mail.worker_tick_ms", kind: "number" },
    ],
  },
}

const FIELD_DEFAULTS: Record<string, string> = {
  "app.default_locale": "zh-CN",
  "auth.google_enabled": "0",
  "auth.google_register_enabled": "0",
  "auth.captcha_provider": "image",
  "auth.recaptcha_min_score": "0.5",
  "mail.enabled": "0",
  "mail.port": "587",
  "mail.tls": "starttls",
  "mail.from_name": "Latch",
  "mail.reset_base_url": "http://127.0.0.1:5173",
  "mail.default_timezone": "Asia/Shanghai",
  "mail.quiet_start": "22:00",
  "mail.quiet_end": "08:00",
  "mail.marketing_start": "09:00",
  "mail.marketing_end": "21:00",
  "mail.rate_per_minute": "30",
  "mail.max_attempts": "5",
  "mail.worker_tick_ms": "2000",
}

function configGroupOf(key: string) {
  return key.split(".")[0] || "app"
}

function placeholderRow(key: string): SysConfig {
  return {
    id: 0,
    key,
    value: FIELD_DEFAULTS[key] ?? "",
    name: "",
    group: configGroupOf(key),
    remark: "",
  }
}

function isSecretKey(key: string) {
  const k = key.toLowerCase()
  return k.includes("password") || k.includes("secret")
}

function isOn(value: string) {
  return value === "1" || value.toLowerCase() === "true"
}

function fieldLabelKey(key: string) {
  return `config.field.${key}`
}

function fieldHintKey(key: string) {
  return `config.field.${key}Hint`
}

function valueKeys(fields: FieldSpec[]) {
  return fields.filter((field) => field.kind !== "heading").map((field) => field.key)
}

export function ConfigsPage() {
  const { can } = useAuth()
  const { t } = useI18n()
  const writable = can(P.configUpdate)
  const [{ tab, section }, setParams] = useConfigListParams()
  const { data, isLoading, error } = useConfigs({ page: 1, pageSize: PAGE_SIZE })
  const rows = data?.items ?? []
  const saveConfigs = useSaveConfigs()
  const testMail = useTestMail()
  const [values, setValues] = useState<Record<string, string>>({})
  const [testTo, setTestTo] = useState("")

  const byKey = useMemo(() => Object.fromEntries(rows.map((row) => [row.key, row])), [rows])
  const stamp = rows.map((row) => `${row.id}:${row.key}:${row.value}`).join("|")

  const specKeys = useMemo(
    () => [
      ...valueKeys(APP_GROUP.fields),
      ...valueKeys(AUTH_GROUP.fields),
      ...valueKeys(MAIL_GROUPS.smtp.fields),
      ...valueKeys(MAIL_GROUPS.policy.fields),
    ],
    [],
  )

  useEffect(() => {
    const next = Object.fromEntries(rows.map((row) => [row.key, row.value]))
    for (const key of specKeys) {
      if (next[key] === undefined) next[key] = FIELD_DEFAULTS[key] ?? ""
    }
    setValues(next)
  }, [stamp])

  const knownKeys = useMemo(
    () =>
      new Set([
        ...valueKeys(APP_GROUP.fields),
        ...valueKeys(AUTH_GROUP.fields),
        ...valueKeys(MAIL_GROUPS.smtp.fields),
        ...valueKeys(MAIL_GROUPS.policy.fields),
        "app.captcha_enabled",
      ]),
    [],
  )
  const extraRows = rows.filter((row) => !knownKeys.has(row.key))
  const tabs: { id: TabId; label: string }[] = [
    { id: "app", label: t("config.tabApp") },
    { id: "auth", label: t("config.tabAuth") },
    { id: "mail", label: t("config.tabMail") },
    ...(extraRows.length > 0 ? [{ id: "other" as const, label: t("config.tabOther") }] : []),
  ]
  const active: TabId = tab === "other" && extraRows.length === 0 ? "app" : tab
  const mailSection: MailSection = section

  function setValue(key: string, value: string) {
    setValues((prev) => ({ ...prev, [key]: value }))
  }

  function changedKeys(keys: string[]) {
    return keys.filter((key) => {
      const row = byKey[key]
      const next = values[key] ?? ""
      const current = row?.value ?? FIELD_DEFAULTS[key] ?? ""
      if (isSecretKey(key) && (next === "" || next === SECRET_MASK || next === current)) return false
      return next !== current
    })
  }

  function tabKeys(id: TabId, mail = mailSection) {
    if (id === "other") return extraRows.map((row) => row.key)
    if (id === "app") return valueKeys(APP_GROUP.fields)
    if (id === "auth") return valueKeys(AUTH_GROUP.fields)
    return valueKeys(MAIL_GROUPS[mail].fields)
  }

  function tabDirtyCount(id: TabId) {
    if (id !== "mail") return changedKeys(tabKeys(id)).length
    return changedKeys(tabKeys("mail", "smtp")).length + changedKeys(tabKeys("mail", "policy")).length
  }

  const visibleKeys = tabKeys(active)
  const dirtyCount = changedKeys(visibleKeys).length
  const dirty = dirtyCount > 0
  const leavingDirty = tabs.some((item) => tabDirtyCount(item.id) > 0)
  const mailOff = active === "mail" && mailSection === "smtp" && !isOn(values["mail.enabled"] ?? byKey["mail.enabled"]?.value ?? "")

  async function saveVisible() {
    if (!writable) return
    const dirtyKeys = changedKeys(visibleKeys)
    if (dirtyKeys.length === 0) {
      toast.message(t("config.unchanged"))
      return
    }
    try {
      await saveConfigs.mutateAsync(
        dirtyKeys.map((key) => {
          const row = byKey[key]
          const value = values[key] ?? ""
          if (!row) {
            const labelRaw = t(fieldLabelKey(key))
            const hintRaw = t(fieldHintKey(key))
            return {
              create: {
                key,
                value,
                name: labelRaw === fieldLabelKey(key) ? key : labelRaw,
                group: configGroupOf(key),
                remark: hintRaw === fieldHintKey(key) ? "" : hintRaw,
              },
            }
          }
          return {
            id: row.id,
            body: {
              value,
              name: row.name,
              group: row.group,
              remark: row.remark,
            },
          }
        }),
      )
      toast.success(t("app.saved"))
    } catch (e) {
      toast.error(translateApiError(e, t))
    }
  }

  const saveVisibleRef = useRef(saveVisible)
  saveVisibleRef.current = saveVisible

  useEffect(() => {
    if (!leavingDirty) return
    function onLeave(e: BeforeUnloadEvent) {
      e.preventDefault()
      e.returnValue = ""
    }
    window.addEventListener("beforeunload", onLeave)
    return () => window.removeEventListener("beforeunload", onLeave)
  }, [leavingDirty])

  useEffect(() => {
    function onKey(e: globalThis.KeyboardEvent) {
      if (!(e.metaKey || e.ctrlKey) || e.key.toLowerCase() !== "s") return
      e.preventDefault()
      void saveVisibleRef.current()
    }
    window.addEventListener("keydown", onKey)
    return () => window.removeEventListener("keydown", onKey)
  }, [])

  function discardVisible() {
    setValues((prev) => {
      const next = { ...prev }
      for (const key of visibleKeys) {
        next[key] = byKey[key]?.value ?? FIELD_DEFAULTS[key] ?? ""
      }
      return next
    })
  }

  async function sendTest() {
    try {
      await testMail.mutateAsync(testTo)
      toast.success(t("config.testMailSent"))
    } catch (e) {
      toast.error(translateApiError(e, t))
    }
  }

  function onTabKey(e: KeyboardEvent<HTMLDivElement>) {
    const ids = tabs.map((item) => item.id)
    const i = ids.indexOf(active)
    if (e.key === "ArrowRight") {
      e.preventDefault()
      void setParams({ tab: ids[(i + 1) % ids.length] })
    }
    if (e.key === "ArrowLeft") {
      e.preventDefault()
      void setParams({ tab: ids[(i - 1 + ids.length) % ids.length] })
    }
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    void saveVisible()
  }

  if (isLoading && rows.length === 0) {
    return (
      <div className="mx-auto max-w-3xl space-y-4">
        <div className="h-8 w-40 animate-pulse rounded-md bg-muted" />
        <div className="h-10 animate-pulse rounded-md bg-muted" />
        <div className="h-64 animate-pulse rounded-xl bg-muted" />
      </div>
    )
  }

  const group =
    active === "app" ? APP_GROUP : active === "auth" ? AUTH_GROUP : active === "mail" ? MAIL_GROUPS[mailSection] : null

  return (
    <form className="mx-auto max-w-3xl space-y-6 pb-20" onSubmit={onSubmit}>
      <div>
        <h2 className="text-xl font-semibold tracking-tight">{t("config.title")}</h2>
        <p className="mt-1 text-sm text-muted-foreground">{t("config.subtitle")}</p>
      </div>

      <div
        className="flex gap-1 border-b"
        role="tablist"
        aria-label={t("config.title")}
        onKeyDown={onTabKey}
      >
        {tabs.map((item) => {
          const selected = active === item.id
          const marked = tabDirtyCount(item.id) > 0
          return (
            <button
              key={item.id}
              type="button"
              role="tab"
              aria-selected={selected}
              tabIndex={selected ? 0 : -1}
              className={cn(
                "-mb-px border-b-2 px-3 py-2 text-sm transition-colors",
                selected
                  ? "border-foreground font-medium text-foreground"
                  : "border-transparent text-muted-foreground hover:text-foreground",
              )}
              onClick={() => void setParams({ tab: item.id })}
            >
              {item.label}
              {marked ? (
                <span className="ml-1.5 inline-block size-1.5 rounded-full bg-foreground align-middle" />
              ) : null}
            </button>
          )
        })}
      </div>

      {active === "mail" ? (
        <div className="flex gap-2" role="tablist" aria-label={t("config.tabMail")}>
          {(["smtp", "policy"] as const).map((id) => {
            const selected = mailSection === id
            const marked = changedKeys(tabKeys("mail", id)).length > 0
            return (
              <button
                key={id}
                type="button"
                role="tab"
                aria-selected={selected}
                className={cn(
                  "rounded-full px-3 py-1 text-xs font-medium transition-colors",
                  selected ? "bg-foreground text-background" : "bg-muted text-muted-foreground hover:text-foreground",
                )}
                onClick={() => void setParams({ section: id })}
              >
                {id === "smtp" ? t("config.subSmtp") : t("config.subPolicy")}
                {marked ? (
                  <span className="ml-1.5 inline-block size-1.5 rounded-full bg-current align-middle" />
                ) : null}
              </button>
            )
          })}
        </div>
      ) : null}

      {error ? <p className="text-sm text-destructive">{translateApiError(error, t)}</p> : null}

      <div role="tabpanel">
        {active === "other" ? (
          <Card>
            <CardHeader>
              <CardTitle>{t("config.other")}</CardTitle>
              <CardDescription>{t("config.otherHint")}</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-4 sm:grid-cols-2">
              {extraRows.map((row) => (
                <FieldControl
                  key={row.key}
                  spec={{ key: row.key, kind: isSecretKey(row.key) ? "password" : "text" }}
                  row={row}
                  value={values[row.key] ?? row.value}
                  disabled={!writable}
                  t={t}
                  onChange={(value) => setValue(row.key, value)}
                />
              ))}
            </CardContent>
          </Card>
        ) : group ? (
          <Card>
            <CardHeader>
              <CardTitle>{t(group.titleKey)}</CardTitle>
              <CardDescription>{t(group.hintKey)}</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-4 sm:grid-cols-2">
              {mailOff ? (
                <p className="rounded-md border bg-muted/50 px-3 py-2 text-xs text-muted-foreground sm:col-span-2">
                  {t("config.mailOff")}
                </p>
              ) : null}
              {group.fields.map((field) => {
                if (field.kind === "heading") {
                  return (
                    <p
                      key={field.key}
                      className="pt-2 text-xs font-medium text-muted-foreground sm:col-span-2"
                    >
                      {t(field.titleKey ?? "")}
                    </p>
                  )
                }
                const row = byKey[field.key] ?? placeholderRow(field.key)
                return (
                  <FieldControl
                    key={field.key}
                    spec={field}
                    row={row}
                    value={values[field.key] ?? row.value}
                    disabled={!writable}
                    t={t}
                    onChange={(value) => setValue(field.key, value)}
                  />
                )
              })}
              {group.id === "smtp" ? (
                <div className="grid gap-3 rounded-lg border bg-muted/40 p-4 sm:col-span-2 sm:grid-cols-[1fr_auto] sm:items-end">
                  <div className="grid gap-1.5">
                    <Label htmlFor="test-to">{t("config.testMailTo")}</Label>
                    <Input
                      id="test-to"
                      type="email"
                      value={testTo}
                      onChange={(e) => setTestTo(e.target.value)}
                      placeholder="you@example.com"
                    />
                    <p className="text-xs text-muted-foreground">{t("config.testMailHint")}</p>
                  </div>
                  <Can perm={P.mailTest}>
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => void sendTest()}
                      disabled={testMail.isPending || !testTo}
                    >
                      {t("config.testMail")}
                    </Button>
                  </Can>
                </div>
              ) : null}
            </CardContent>
          </Card>
        ) : null}
      </div>

      {writable ? (
        <div className="sticky bottom-0 z-10 -mx-6 flex items-center justify-between gap-3 border-t bg-background/95 px-6 py-3 backdrop-blur">
          <p className="text-xs text-muted-foreground">
            {dirty ? t("config.changedCount", { count: dirtyCount }) : t("config.unchanged")}
          </p>
          <div className="flex gap-2">
            <Button type="button" variant="ghost" disabled={!dirty || saveConfigs.isPending} onClick={discardVisible}>
              {t("config.discard")}
            </Button>
            <Button type="submit" disabled={!dirty || saveConfigs.isPending}>
              {saveConfigs.isPending ? t("app.saving") : t("config.saveSection")}
            </Button>
          </div>
        </div>
      ) : null}
    </form>
  )
}

function FieldControl({
  spec,
  row,
  value,
  disabled,
  t,
  onChange,
}: {
  spec: FieldSpec
  row: SysConfig
  value: string
  disabled: boolean
  t: (key: string) => string
  onChange: (value: string) => void
}) {
  const labelRaw = t(fieldLabelKey(spec.key))
  const label = labelRaw === fieldLabelKey(spec.key) ? row.name : labelRaw
  const hintRaw = t(fieldHintKey(spec.key))
  const hint = hintRaw === fieldHintKey(spec.key) ? row.remark : hintRaw
  const id = spec.key.replaceAll(".", "-")
  const wide = spec.kind === "switch" || spec.kind === "url"

  if (spec.kind === "switch") {
    return (
      <div className="flex items-start justify-between gap-4 rounded-lg border px-3 py-3 sm:col-span-2">
        <div className="min-w-0">
          <Label htmlFor={id}>{label}</Label>
          {hint ? <p className="mt-1 text-xs text-muted-foreground">{hint}</p> : null}
        </div>
        <Switch
          id={id}
          checked={isOn(value)}
          disabled={disabled}
          onCheckedChange={(checked) => onChange(checked ? "1" : "0")}
        />
      </div>
    )
  }

  if (spec.kind === "select" && spec.options) {
    const resolved = spec.options.some((opt) => opt.value === value)
      ? value
      : (FIELD_DEFAULTS[spec.key] ?? spec.options[0]?.value ?? value)
    return (
      <div className="grid gap-1.5">
        <DictSelect
          id={id}
          label={label}
          value={resolved}
          items={spec.options.map((opt) => ({ value: opt.value, label: t(opt.labelKey) }))}
          onChange={onChange}
          className={disabled ? "pointer-events-none opacity-50" : undefined}
        />
        {hint ? <p className="text-xs text-muted-foreground">{hint}</p> : null}
      </div>
    )
  }

  if (spec.kind === "timezone") {
    return (
      <div className="grid gap-1.5">
        <Label htmlFor={id}>{label}</Label>
        <TimezoneSelect
          id={id}
          value={value || FIELD_DEFAULTS[spec.key] || "Asia/Shanghai"}
          onChange={onChange}
          className={disabled ? "pointer-events-none opacity-50" : undefined}
        />
        {hint ? <p className="text-xs text-muted-foreground">{hint}</p> : null}
      </div>
    )
  }

  return (
    <div className={cn("grid gap-1.5", wide && "sm:col-span-2")}>
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type={spec.kind === "password" ? "password" : spec.kind}
        value={value}
        disabled={disabled}
        autoComplete={spec.kind === "password" ? "new-password" : undefined}
        onChange={(e) => onChange(e.target.value)}
      />
      {hint ? <p className="text-xs text-muted-foreground">{hint}</p> : null}
      {spec.kind === "password" ? <p className="text-xs text-muted-foreground">{t("config.secretKeep")}</p> : null}
    </div>
  )
}
