import { Component, type ErrorInfo, type ReactNode } from "react"
import { Button } from "@/components/ui/button"

type Props = {
  children: ReactNode
  fallback?: ReactNode
  resetKey?: string
}

type State = {
  error: Error | null
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
    this.disposeHot = hot.on("vite:beforeUpdate", () => {
      if (this.state.error) this.setState({ error: null })
    })
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
      <div className="flex min-h-[40vh] flex-col items-center justify-center gap-3 p-8 text-center">
        <p className="text-sm font-medium">Something went wrong.</p>
        <p className="max-w-md text-xs text-muted-foreground">{this.state.error.message}</p>
        <Button type="button" variant="outline" size="sm" onClick={() => this.setState({ error: null })}>
          Retry
        </Button>
      </div>
    )
  }
}
