import { Component, type ErrorInfo, type ReactNode } from "react"

type Props = {
  children: ReactNode
  fallback?: ReactNode
}

type State = {
  error: Error | null
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null }

  static getDerivedStateFromError(error: Error): State {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("ui error", error, info.componentStack)
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
        <button
          type="button"
          className="rounded-md border px-3 py-1.5 text-sm"
          onClick={() => this.setState({ error: null })}
        >
          Retry
        </button>
      </div>
    )
  }
}
