import { Component, type ErrorInfo, type ReactNode } from "react"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/providers/i18n"

type Props = {
  children: ReactNode
  fallback?: ReactNode
  resetKey?: string
}

type State = {
  error: Error | null
}

function ErrorFallback({ error, onRetry }: { error: Error; onRetry: () => void }) {
  const { t } = useI18n()
  return (
    <div className="flex min-h-[40vh] flex-col items-center justify-center gap-3 p-8 text-center">
      <p className="text-sm font-medium">{t("errors.crashTitle")}</p>
      <p className="max-w-md text-xs text-muted-foreground">{error.message}</p>
      <Button type="button" variant="outline" size="sm" onClick={onRetry}>
        {t("errors.crashRetry")}
      </Button>
    </div>
  )
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null }
  private disposeHot: (() => void) | undefined

  static getDerivedStateFromError(error: Error): State {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("ui error", error, info.componentStack)
  }

  componentDidMount() {
    const hot = import.meta.hot
    if (!hot) return
    const reset = () => {
      if (this.state.error) this.setState({ error: null })
    }
    hot.on("vite:beforeUpdate", reset)
    this.disposeHot = undefined
  }

  componentWillUnmount() {
    this.disposeHot?.()
  }

  componentDidUpdate(prevProps: Props) {
    if (this.state.error && prevProps.resetKey !== this.props.resetKey) {
      this.setState({ error: null })
    }
  }

  render() {
    if (!this.state.error) {
      return this.props.children
    }
    if (this.props.fallback) {
      return this.props.fallback
    }
    return (
      <ErrorFallback error={this.state.error} onRetry={() => this.setState({ error: null })} />
    )
  }
}
