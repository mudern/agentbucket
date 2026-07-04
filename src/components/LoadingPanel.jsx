export default function LoadingPanel({ label = 'Loading...' }) {
  return (
    <div className="rounded-2xl border border-slate-200 bg-white p-6 text-sm text-slate-500 shadow-sm">
      {label}
    </div>
  )
}
