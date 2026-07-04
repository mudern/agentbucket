import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getStats } from '../api'
import PageHeader from '../components/PageHeader'
import LoadingPanel from '../components/LoadingPanel'
import { useT } from '../i18n'

function StatCard({ label, value, sub, icon, color, to }) {
  const c = {
    emerald: 'border-emerald-200 bg-emerald-50 dark:border-emerald-800 dark:bg-emerald-900/30',
    sky: 'border-sky-200 bg-sky-50 dark:border-sky-800 dark:bg-sky-900/30',
    amber: 'border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-900/30',
    rose: 'border-rose-200 bg-rose-50 dark:border-rose-800 dark:bg-rose-900/30',
    slate: 'border-slate-200 bg-slate-50 dark:border-slate-600 dark:bg-slate-800',
  }[color] || ''
  const inner = (
    <div className={`rounded-2xl border p-4 h-full ${c} ${to ? 'cursor-pointer transition hover:-translate-y-0.5 hover:shadow-md' : ''}`}>
      <div className="flex items-center justify-between">
        <div className="text-2xl font-bold text-slate-900 dark:text-slate-100">{value}</div>
        {icon && <span className="text-xl opacity-60">{icon}</span>}
      </div>
      <div className="text-xs text-slate-500 dark:text-slate-400">{label}</div>
      {sub && <div className="mt-1 text-[11px] text-slate-400 dark:text-slate-500">{sub}</div>}
    </div>
  )
  return to ? <Link to={to}>{inner}</Link> : inner
}

export default function DashboardPage() {
  const [stats, setStats] = useState(null)
  const [loading, setLoading] = useState(true)
  const t = useT()

  useEffect(() => {
    getStats().then(setStats).catch(() => {}).finally(() => setLoading(false))
  }, [])

  if (loading || !stats) return <LoadingPanel label={t('common.loading')} />

  const { tokens, users, deployments, chat, repositories, bus, system } = stats

  return (
    <div>
      <PageHeader title={t('common.dashboard', '首页')} description={t('common.dashboard_desc', '控制平面概览')} />

      {/* Row 1: Core stats */}
      <div className="mb-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-5 items-stretch">
        <StatCard label={t('agents.title')} value={(deployments?.running || 0) || 0} sub={`${(deployments?.total || 0) || 0} total`} icon="🟢" color="emerald" to="/agents" />
        <StatCard label={t('common.running')} value={(deployments?.running || 0) || 0} icon="🚀" color="sky" to="/deploy/progress" />
        <StatCard label={t('common.failed', '失败')} value={(deployments?.failed || 0) || 0} icon="⚠️" color={((deployments?.failed || 0) || 0) > 0 ? 'rose' : 'slate'} to="/deploy/progress" />
        <StatCard label={t('repositories.title')} value={repositories?.length || 0} icon="📦" color="slate" to="/repositories" />
        <StatCard label={t('users.title')} value={(users?.active || 0) || 0} sub={`${(users?.superAdmin || 0) || 0} admin`} icon="👥" color="slate" to="/users" />
      </div>

      {/* Row 2: Tokens + Chat + Bus */}
      <div className="mb-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4 items-stretch">
        <StatCard label={t('aITokens.title')} value={(tokens?.ai?.enabled || 0) || 0} sub={`${(tokens?.ai?.disabled || 0) || 0} disabled`} icon="🔑" color="sky" to="/ai-tokens" />
        <StatCard label={t('authTokens.title')} value={(tokens?.auth?.enabled || 0) || 0} icon="🔐" color="slate" to="/auth-tokens" />
        <StatCard label={t('chat.sessions')} value={(chat?.totalSessions || 0) || 0} sub={`${(chat?.todayMessages || 0) || 0} today`} icon="💬" color="sky" to="/agents" />
        <StatCard label={t('progress.title')} value={(deployments?.today || 0) || 0} sub={`${Math.round(((deployments?.recentSuccessRate || 0) || 0) / Math.max(((deployments?.total || 0) || 1), 1) * 100)}% success`} icon="📊" color="slate" to="/deploy/progress" />
      </div>

      {/* Row 3: Alerts + Details */}
      {(deployments?.failed || 0) > 0 && (
        <div className="mb-4 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-800 dark:bg-rose-900/30 dark:text-rose-300">
          {deployments.failed} {t('progress.step_failed')} — <Link to="/deploy/progress" className="font-medium underline">{t('common.view_all', '查看')}</Link>
        </div>
      )}

      <div className="grid gap-4 lg:grid-cols-2">
        {/* Repositories */}
        <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-600 dark:bg-slate-800">
          <div className="mb-3 flex items-center justify-between">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">{t('repositories.title')}</h3>
            <Link to="/repositories" className="text-xs text-sky-600 hover:underline dark:text-sky-400">{t('common.view_all', '查看')}</Link>
          </div>
          {(!repositories || repositories.length === 0) ? (
            <p className="py-6 text-center text-sm text-slate-400">{t('repositories.no_repos')}</p>
          ) : (
            <div className="space-y-2">
              {repositories.slice(0, 5).map((r) => (
                <div key={r.id} className="flex items-center justify-between rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-900/50">
                  <div className="min-w-0 flex-1">
                    <span className="text-sm font-medium text-slate-900 dark:text-slate-100">{r.id}</span>
                    <span className="ml-2 text-[11px] text-slate-400">{r.provider} · {r.agents || 0} agents</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className={`h-1.5 w-1.5 rounded-full ${r.status === '启用' ? 'bg-emerald-400' : 'bg-slate-300'}`} />
                    <span className="text-[11px] text-slate-400">{r.lastSync || '-'}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* System Info */}
        <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-600 dark:bg-slate-800">
          <h3 className="mb-3 text-sm font-semibold text-slate-900 dark:text-slate-100">{t('common.system', '系统')}</h3>
          <div className="grid grid-cols-2 gap-3 text-sm">
            <div className="rounded-lg bg-slate-50 p-3 dark:bg-slate-900/50">
              <div className="text-xs text-slate-400">AgentBucket</div>
              <div className="font-mono text-slate-900 dark:text-slate-100">v{system?.version || '1.0.0'}</div>
            </div>
            <div className="rounded-lg bg-slate-50 p-3 dark:bg-slate-900/50">
              <div className="text-xs text-slate-400">Go</div>
              <div className="font-mono text-slate-900 dark:text-slate-100">{system?.goVersion || '1.22+'}</div>
            </div>
            <div className="rounded-lg bg-slate-50 p-3 dark:bg-slate-900/50">
              <div className="text-xs text-slate-400">Docker</div>
              <div className="font-mono text-slate-900 dark:text-slate-100">{system?.dockerAvailable ? 'Available' : 'Not found'}</div>
            </div>
            <div className="rounded-lg bg-slate-50 p-3 dark:bg-slate-900/50">
              <div className="text-xs text-slate-400">Agent Bus</div>
              <div className="font-mono text-slate-900 dark:text-slate-100">{bus?.online || 0}/{bus?.total || 0} online</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
