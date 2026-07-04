import { Component } from 'react'

export default class ErrorBoundary extends Component {
  constructor(props) {
    super(props)
    this.state = { error: null }
  }

  static getDerivedStateFromError(error) {
    return { error }
  }

  render() {
    if (this.state.error) {
      return (
        <div className="flex min-h-[60vh] items-center justify-center p-8">
          <div className="max-w-md rounded-2xl border border-rose-200 bg-white p-8 text-center shadow-sm dark:border-rose-800 dark:bg-slate-800">
            <div className="mb-2 text-4xl">⚠️</div>
            <h2 className="mb-2 text-lg font-semibold text-slate-900 dark:text-slate-100">Something went wrong</h2>
            <p className="mb-6 text-sm text-slate-500 dark:text-slate-400">{this.state.error?.message || 'An unexpected error occurred'}</p>
            <button
              onClick={() => { this.setState({ error: null }); window.location.reload() }}
              className="rounded-xl bg-sky-600 px-5 py-2 text-sm font-medium text-white hover:bg-sky-700"
            >
              Retry
            </button>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}
