import { NavLink } from 'react-router-dom'
import LogoMark from './LogoMark'
import { navGroups } from '../data'
import { getCurrentUser } from '../api'
import useAsyncData from '../hooks/useAsyncData'

export default function Sidebar({ collapsed, onToggle, onLogout }) {
  const { data: currentUser } = useAsyncData(getCurrentUser, [])
  const allowed = (item) => item.roles.includes(currentUser?.role)

  return (
    <aside className={`relative flex min-h-screen shrink-0 flex-col border-r border-slate-200 bg-white shadow-sm transition-all duration-200 ${collapsed ? 'w-16 px-3 py-4' : 'w-64 px-5 py-5'}`}>
      <button
        onClick={onToggle}
        className="absolute -right-3 top-16 z-10 flex h-7 w-7 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-400 shadow-sm transition hover:border-slate-300 hover:bg-slate-50 hover:text-slate-700"
        title={collapsed ? '展开侧栏' : '收起侧栏'}
      >
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
          {collapsed ? <path d="M9 6l6 6-6 6" /> : <path d="M15 6l-6 6 6 6" />}
        </svg>
      </button>

      {/* Logo */}
      <div className={`mb-7 flex items-center ${collapsed ? 'justify-center' : 'justify-start'}`}>
        <LogoMark compact={collapsed} />
      </div>

      {/* Navigation */}
      {!collapsed && currentUser ? (
        <nav className="flex-1 space-y-6">
          {navGroups.map((group) => (
            <div key={group.title}>
              <div className="mb-2 px-3 text-xs font-semibold uppercase tracking-[0.14em] text-slate-400">
                {group.title}
              </div>
              <div className="space-y-1">
                {group.items.filter(allowed).map((item) => (
                  <NavLink
                    key={item.path}
                    to={item.path}
                    className={({ isActive }) =>
                      `block rounded-xl px-3 py-2.5 text-sm transition ${
                        isActive
                          ? 'bg-sky-50 text-sky-700 shadow-sm'
                          : 'text-slate-600 hover:bg-slate-100 hover:text-slate-950'
                      }`
                    }
                  >
                    {item.label}
                  </NavLink>
                ))}
              </div>
            </div>
          ))}
        </nav>
      ) : null}

      {/* Logout */}
      {!collapsed && currentUser && (
        <button
          type="button"
          onClick={onLogout}
          className="mt-6 w-full rounded-lg px-3 py-2.5 text-left text-sm font-medium text-slate-500 transition hover:bg-slate-50 hover:text-slate-950"
        >
          退出登录
        </button>
      )}
    </aside>
  )
}
