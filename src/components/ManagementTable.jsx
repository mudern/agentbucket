import { useState } from 'react'

export function ManagementTable({ children }) {
  return (
    <div className="max-w-full rounded-2xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 shadow-sm">
      <table className="w-full table-fixed divide-y divide-slate-200 dark:divide-slate-700 text-sm">{children}</table>
    </div>
  )
}

export function FilterInput({ value, onChange, placeholder }) {
  return (
    <div className="relative">
      <input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        className="h-9 w-full rounded-md border border-transparent bg-slate-100 dark:bg-slate-700 px-3 text-xs font-normal text-slate-700 dark:text-slate-300 outline-none transition placeholder:text-slate-400 dark:placeholder:text-slate-500 dark:text-slate-500 hover:border-slate-200 dark:border-slate-700 hover:bg-white focus:border-sky-500 focus:bg-white dark:bg-slate-800"
      />
    </div>
  )
}

export function FilterSelect({ value, onChange, options }) {
  return (
    <select
      value={value}
      onChange={(event) => onChange(event.target.value)}
      className="h-9 w-full rounded-md border border-transparent bg-slate-100 dark:bg-slate-700 px-3 text-xs font-normal text-slate-700 dark:text-slate-300 outline-none transition hover:border-slate-200 dark:border-slate-700 hover:bg-white focus:border-sky-500 focus:bg-white dark:bg-slate-800"
    >
      <option value="all">全部</option>
      {options.map((option) => (
        <option key={option} value={option}>
          {option}
        </option>
      ))}
    </select>
  )
}

function FilterIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" aria-hidden="true" className="h-4 w-4">
      <path d="M4 5h12M6.5 10h7M9 15h2" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
    </svg>
  )
}

export function HeaderFilter({ label, active = false, align = 'left', children }) {
  const [open, setOpen] = useState(false)

  return (
    <div className="relative flex min-w-0 items-center gap-2">
      <span className="min-w-0 truncate">{label}</span>
      {children && (
        <>
          <button
            type="button"
            onClick={() => setOpen((current) => !current)}
            className={`inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-lg transition ${
              active
                ? 'bg-sky-100 text-sky-700 hover:bg-sky-200'
                : 'text-slate-400 dark:text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-700 hover:text-slate-700 dark:text-slate-300'
            }`}
            aria-label={`${label}筛选`}
          >
            <FilterIcon />
          </button>
          {open && (
            <div
              className={`absolute top-9 z-30 w-56 rounded-xl border border-slate-200 dark:border-slate-700 bg-white p-3 shadow-lg ${
                align === 'right' ? 'right-0' : 'left-0'
              }`}
            >
              {children}
            </div>
          )}
        </>
      )}
    </div>
  )
}

export function HoverCard({ children, content, align = 'left' }) {
  return (
    <span className="group relative inline-flex max-w-full">
      {children}
      <span
        className={`pointer-events-none absolute top-7 z-30 hidden w-72 rounded-xl border border-slate-200 dark:border-slate-700 bg-white p-4 text-left text-xs font-normal text-slate-600 dark:text-slate-400 shadow-lg group-hover:block ${
          align === 'right' ? 'right-0' : 'left-0'
        }`}
      >
        {content}
      </span>
    </span>
  )
}

export function SoftTag({ children }) {
  return (
    <span className="inline-flex max-w-full items-center rounded-full bg-slate-100 dark:bg-slate-700 px-2.5 py-1 text-xs font-medium text-slate-600 dark:text-slate-400">
      <span className="truncate">{children}</span>
    </span>
  )
}

export function StatusBadge({ status }) {
  return (
    <span className={`inline-flex whitespace-nowrap rounded-full px-2.5 py-1 text-xs ${status === '启用' ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-100 dark:bg-slate-700 text-slate-500 dark:text-slate-400'}`}>
      {status}
    </span>
  )
}

export function RowActions({ status, onEnable, onDisable, onDelete }) {
  return (
    <div className="flex items-center justify-end gap-3 whitespace-nowrap text-sm font-medium">
      <button
        type="button"
        onClick={onEnable}
        disabled={status === '启用'}
        className="text-emerald-700 transition hover:text-emerald-800 disabled:cursor-not-allowed disabled:text-slate-300 dark:text-slate-600"
      >
        启用
      </button>
      <button
        type="button"
        onClick={onDisable}
        disabled={status === '停用'}
        className="text-amber-700 transition hover:text-amber-800 disabled:cursor-not-allowed disabled:text-slate-300 dark:text-slate-600"
      >
        停用
      </button>
      <button type="button" onClick={onDelete} className="text-rose-700 transition hover:text-rose-800">
        删除
      </button>
    </div>
  )
}

export const tableHeadClass = 'bg-slate-50 dark:bg-slate-900 text-left text-slate-500 dark:text-slate-400'
export const tableBodyClass = 'divide-y divide-slate-200 dark:divide-slate-700 text-slate-700 dark:text-slate-300'
export const tableHeaderCellClass = 'whitespace-nowrap px-6 py-4 align-top'
export const tableCellClass = 'px-6 py-4 align-top dark:text-slate-300'
