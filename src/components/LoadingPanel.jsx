export default function LoadingPanel({ label = 'Loading...' }) {
  return (
    <div className="rounded-2xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-6 text-sm text-slate-500 dark:text-slate-400 shadow-sm">
      {label}
    </div>
  )
}
