import { NavLink } from 'react-router-dom'
import LogoMark from './LogoMark'
import { navGroups } from '../data'
import { getCurrentUser } from '../api'
import useAsyncData from '../hooks/useAsyncData'
import { useLanguage, useT, langNames, supportedLangs } from '../i18n'

const navKeyMap = {
  '部署 Agent': 'nav.deploy_agent',
  '部署进度': 'progress.title',
  '所有 Agent': 'nav.all_agents',
  '用户权限': 'nav.users',
  '审批中心': 'nav.approvals',
  '仓库管理': 'nav.repositories',
  'AI Token': 'nav.ai_tokens',
  '鉴权 Token': 'nav.auth_tokens',
}

const groupKeyMap = {
  'Agent': 'nav.agent',
  '管理': 'nav.admin',
}

export default function Sidebar({ collapsed, onToggle, onLogout }) {
  const { data: currentUser } = useAsyncData(getCurrentUser, [])
  const { lang, setLang } = useLanguage()
  const t = useT()
  const allowed = (item) => item.roles.includes(currentUser?.role)

  return (
    <aside className={`relative flex min-h-screen shrink-0 flex-col border-r border-slate-200 bg-white shadow-sm transition-all duration-200 ${collapsed ? 'w-16 px-3 py-4' : 'w-64 px-5 py-5'}`}>
      <button
        onClick={onToggle}
        className="absolute -right-3 top-16 z-10 flex h-7 w-7 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-400 shadow-sm transition hover:border-slate-300 hover:bg-slate-50 hover:text-slate-700"
        title={collapsed ? 'Expand Sidebar' : 'Collapse Sidebar'}
      >
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
          {collapsed ? <path d="M9 6l6 6-6 6" /> : <path d="M15 6l-6 6 6 6" />}
        </svg>
      </button>

      {/* Logo */}
      <div className={`mb-7 flex items-center ${collapsed ? 'justify-center' : 'justify-start'}`}>
        <LogoMark compact={collapsed} />
      </div>

      {/* Navigation — always render the flex-1 container for consistent layout */}
      {!collapsed && (
        <nav className="flex-1 space-y-6">
          {currentUser && navGroups.map((group) => (
            <div key={group.title}>
              <div className="mb-2 px-3 text-xs font-semibold uppercase tracking-[0.14em] text-slate-400">
                {t(groupKeyMap[group.title], group.title)}
              </div>
              <div className="space-y-1">
                {group.items.filter(allowed).map((item) => (
                  <NavLink
                    key={item.path}
                    to={item.path}
                    end
                    className={({ isActive }) =>
                      `block rounded-xl px-3 py-2.5 text-sm transition ${
                        isActive
                          ? 'bg-sky-50 text-sky-700 shadow-sm'
                          : 'text-slate-600 hover:bg-slate-100 hover:text-slate-950'
                      }`
                    }
                  >
                    {t(navKeyMap[item.label], item.label)}
                  </NavLink>
                ))}
              </div>
            </div>
          ))}
        </nav>
      )}

      {/* Bottom section: always at bottom */}
      {!collapsed && (
        <div className="mt-auto border-t border-slate-100 pt-3">
          <div className="grid grid-cols-5 gap-1 px-1">
            {supportedLangs.map((l) => (
              <button
                key={l}
                onClick={() => setLang(l)}
                className={`rounded py-1 text-[10px] font-medium transition ${
                  lang === l ? 'bg-sky-50 text-sky-700 shadow-sm' : 'text-slate-400 hover:bg-slate-50 hover:text-slate-600'
                }`}
              >
                {langNames[l]}
              </button>
            ))}
          </div>
          {currentUser && (
            <button
              type="button"
              onClick={onLogout}
              className="mt-2 w-full rounded-lg px-3 py-2.5 text-center text-sm font-medium text-slate-500 transition hover:bg-slate-50 hover:text-slate-950"
            >
              {t('common.logout')}
            </button>
          )}
        </div>
      )}
    </aside>
  )
}
