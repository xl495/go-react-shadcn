import {
  parseAsInteger,
  parseAsString,
  parseAsStringLiteral,
  useQueryStates,
} from "nuqs"

const replace = { history: "replace" as const }

const pageParser = parseAsInteger.withDefault(1)
const textParser = parseAsString.withDefault("")

export function useUserListParams() {
  return useQueryStates(
    {
      q: textParser,
      page: pageParser,
      gender: textParser,
      status: textParser,
      department: textParser,
      roleId: parseAsInteger,
    },
    replace,
  )
}

export function usePermissionListParams() {
  return useQueryStates(
    {
      q: textParser,
      page: pageParser,
      kind: textParser,
    },
    replace,
  )
}

export function usePageParam() {
  return useQueryStates({ page: pageParser }, replace)
}

export function useSearchPageParams() {
  return useQueryStates({ q: textParser, page: pageParser }, replace)
}

export function useLogListParams() {
  return useQueryStates(
    {
      tab: parseAsStringLiteral(["op", "login", "api"] as const).withDefault("op"),
      page: pageParser,
      username: textParser,
      module: textParser,
      action: textParser,
      traceId: textParser,
      status: textParser,
      path: textParser,
    },
    replace,
  )
}

export function useMailJobListParams() {
  return useQueryStates(
    {
      page: pageParser,
      status: textParser,
      class: textParser,
    },
    replace,
  )
}

export function useMailCampaignListParams() {
  return useQueryStates(
    {
      page: pageParser,
      status: textParser,
    },
    replace,
  )
}

export function useConfigListParams() {
  return useQueryStates(
    {
      tab: parseAsStringLiteral(["app", "auth", "mail", "other"] as const).withDefault("app"),
      section: parseAsStringLiteral(["smtp", "policy"] as const).withDefault("smtp"),
    },
    replace,
  )
}

export function useDictListParams() {
  return useQueryStates(
    {
      page: pageParser,
      q: textParser,
      itemPage: pageParser,
      typeId: parseAsInteger,
    },
    replace,
  )
}
