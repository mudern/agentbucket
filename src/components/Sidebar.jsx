import { NavLink } from 'react-router-dom'
import LogoMark from './LogoMark'
import { navGroups } from '../data'
import { getCurrentUser } from '../api'
import useAsyncData from '../hooks/useAsyncData'

export default function Sidebar({ onLogout }) {
  const { data: currentUser } = useAsyncData(getCurrentUser, [])
  if (!currentUser) {
    return (
      <aside className="flex min-h-screen w-64 shrink-0 flex-col border-r border-slate-200 bg-white px-5 py-5">
        <LogoMark />
      </aside>
    )
  }

  const allowed = (item) => item.roles.includes(currentUser.role)

  return (
    <aside className="flex min-h-screen w-64 shrink-0 flex-col border-r border-slate-200 bg-white px-5 py-5 shadow-sm">
      <div className="mb-7">
        <LogoMark />
      </div>

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

      <button
        type="button"
        onClick={onLogout}
        className="mt-6 w-full rounded-lg px-3 py-2.5 text-left text-sm font-medium text-slate-500 transition hover:bg-slate-50 hover:text-slate-950"
      >
        退出登录
      </button>
    </aside>
  )
}
