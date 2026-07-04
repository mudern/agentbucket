import { useState } from 'react'
import Sidebar from './Sidebar'

export default function Layout({ children, onLogout }) {
  const [collapsed, setCollapsed] = useState(false)
  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 text-slate-950 dark:text-slate-50 dark:bg-slate-900 dark:text-slate-100">
      <div className="flex min-h-screen">
        <Sidebar collapsed={collapsed} onToggle={() => setCollapsed((c) => !c)} onLogout={onLogout} />
        <main className="min-w-0 flex-1 p-8">{children}</main>
      </div>
    </div>
  )
}
