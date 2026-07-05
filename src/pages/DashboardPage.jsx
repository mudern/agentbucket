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

const DOTS = 'running:bg-emerald-400 stopped:bg-slate-300 build_failed:bg-red-400 run_failed:bg-red-400 crashed:bg-red-400 building_context:bg-sky-400 animate-pulse building_image:bg-indigo-400 animate-pulse starting_container:bg-amber-400 animate-pulse'
function dot(status) {
  const m = DOTS.match(new RegExp(status + ':(\\S+)'))
  return m ? m[1] : 'bg-slate-300'
}

export default function DashboardPage() {
  const [stats, setStats] = useState(null)
  const [loading, setLoading] = useState(true)
  const t = useT()

  useEffect(() => {
    getStats().then(setStats).catch(() => {}).finally(() => setLoading(false))
  }, [])

  if (loading || !stats) return <LoadingPanel label={t('common.loading')} />

  const { tokens, users, deployments, chat, repositories, bus, system, agentActivity, tokenUsage, timeline, hourly, repoHealth } = stats
  const running = deployments?.running || 0
  const failed = deployments?.failed || 0
  const total = deployments?.total || 0
  const hasActivity = total > 0

  return (
    <div>
      <PageHeader title={t('common.dashboard', '首页')} description={t('common.dashboard_desc', '控制平面概览')} />

      {/* Welcome / Getting Started */}
      {!hasActivity && (
        <div className="mb-6 rounded-2xl border border-sky-200 bg-sky-50 p-6 dark:border-sky-800 dark:bg-sky-900/30">
          <h2 className="mb-2 text-lg font-semibold text-sky-900 dark:text-sky-100">{t('common.welcome', '欢迎使用 AgentBucket')}</h2>
          <p className="mb-4 text-sm text-sky-700 dark:text-sky-300">{t('common.welcome_desc', '开始使用 AgentBucket 管理你的 AI Agent：绑定仓库、创建 Token、部署 Agent。')}</p>
          <div className="flex flex-wrap gap-3">
            <Link to="/repositories" className="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700">{t('repositories.bind_repo')}</Link>
            <Link to="/ai-tokens" className="rounded-xl border border-sky-300 px-4 py-2 text-sm font-medium text-sky-700 hover:bg-sky-100 dark:border-sky-600 dark:text-sky-300 dark:hover:bg-sky-800">{t('aITokens.create_token')}</Link>
            <Link to="/deploy" className="rounded-xl border border-sky-300 px-4 py-2 text-sm font-medium text-sky-700 hover:bg-sky-100 dark:border-sky-600 dark:text-sky-300 dark:hover:bg-sky-800">{t('deploy.deploy_button')}</Link>
          </div>
        </div>
      )}

      {/* Row 1: Agents + Deployments */}
      <div className="mb-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-5 items-stretch">
        <StatCard label={t('agents.title')} value={running} sub={total > 0 ? `${total} total` : ''} icon="🤖" color="emerald" to="/agents" />
        <StatCard label={t('common.running')} value={running} icon="🚀" color="sky" to="/deploy/progress" />
        <StatCard label={t('common.failed', '失败')} value={failed} icon="⚠️" color={failed > 0 ? 'rose' : 'slate'} to="/deploy/progress" />
        <StatCard label={t('repositories.title')} value={repositories?.length || 0} icon="📦" color="slate" to="/repositories" />
        <StatCard label={t('users.title')} value={users?.active || 0} sub={`${users?.superAdmin || 0} admin`} icon="👥" color="slate" to="/users" />
      </div>

      {/* Row 2: Tokens + Chat */}
      <div className="mb-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4 items-stretch">
        <StatCard label={t('aITokens.title')} value={tokens?.ai?.enabled || 0} sub={`${tokens?.ai?.disabled || 0} disabled`} icon="🔑" color="sky" to="/ai-tokens" />
        <StatCard label={t('authTokens.title')} value={tokens?.auth?.enabled || 0} icon="🔐" color="slate" to="/auth-tokens" />
        <StatCard label={t('chat.sessions')} value={chat?.totalSessions || 0} sub={`${chat?.todayMessages || 0} today`} icon="💬" color="sky" to="/agents" />
        <StatCard label={t('progress.title')} value={deployments?.today || 0} sub={`${total > 0 ? Math.round(running / Math.max(total, 1) * 100) : 0}% running`} icon="📊" color="slate" to="/deploy/progress" />
      </div>

      {/* Alerts */}
      {failed > 0 && (
        <div className="mb-4 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-800 dark:bg-rose-900/30 dark:text-rose-300">
          {failed} {t('progress.step_failed')} — <Link to="/deploy/progress" className="font-medium underline">{t('common.view_all', '查看')}</Link>
        </div>
      )}

      <div className="grid gap-4 lg:grid-cols-3">
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
              {repositories.slice(0, 6).map((r) => (
                <div key={r.id} className="flex items-center justify-between rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-900/50">
                  <div className="min-w-0 flex-1">
                    <span className="text-sm font-medium text-slate-900 dark:text-slate-100 truncate block max-w-[180px]">{r.id}</span>
                    <span className="text-[11px] text-slate-400">{r.provider} · {r.agents || 0} agents · {r.commits || 0} commits</span>
                  </div>
                  <span className={`h-1.5 w-1.5 rounded-full ${r.status === '启用' ? 'bg-emerald-400' : 'bg-slate-300'}`} />
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Recent Deployments */}
        <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-600 dark:bg-slate-800">
          <div className="mb-3 flex items-center justify-between">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">{t('common.recent_deployments', '最近部署')}</h3>
            <Link to="/deploy/progress" className="text-xs text-sky-600 hover:underline dark:text-sky-400">{t('common.view_all', '查看')}</Link>
          </div>
          {total === 0 ? (
            <p className="py-6 text-center text-sm text-slate-400"><Link to="/deploy" className="text-sky-600 underline">{t('deploy.deploy_button')}</Link></p>
          ) : (
            <div className="space-y-2">
              {(deployments?.recent || []).slice(0, 6).map((d, i) => (
                <div key={i} className="flex items-center gap-2 rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-900/50">
                  <span className={`h-2 w-2 shrink-0 rounded-full ${dot(d.status)}`} />
                  <span className="min-w-0 flex-1 truncate text-sm text-slate-900 dark:text-slate-100">{d.agentId}</span>
                  <span className="text-[11px] text-slate-400">{d.status}</span>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* System */}
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
              <div className="font-mono text-slate-900 dark:text-slate-100">{system?.dockerAvailable ? 'Ready' : 'Not found'}</div>
            </div>
            <div className="rounded-lg bg-slate-50 p-3 dark:bg-slate-900/50">
              <div className="text-xs text-slate-400">Agent Bus</div>
              <div className="font-mono text-slate-900 dark:text-slate-100">{bus?.online || 0}/{bus?.total || 0} online</div>
            </div>
          </div>
          <div className="mt-4 space-y-2">
            <Link to="/deploy" className="block rounded-xl bg-sky-600 px-4 py-2.5 text-center text-sm font-medium text-white hover:bg-sky-700">{t('deploy.deploy_button')}</Link>
            <Link to="/repositories" className="block rounded-xl border border-slate-200 px-4 py-2.5 text-center text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700">{t('repositories.bind_repo')}</Link>
          </div>
        </div>
      </div>

      {/* Agent Activity Ranking */}
      {Object.keys(agentActivity || {}).length > 0 && (
        <div className="mt-4 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-600 dark:bg-slate-800">
          <h3 className="mb-3 text-sm font-semibold text-slate-900 dark:text-slate-100">{t('common.agent_activity', 'Agent 调用热度')}</h3>
          <div className="flex flex-wrap gap-2">
            {Object.entries(agentActivity || {}).sort((a,b) => b[1]-a[1]).slice(0, 8).map(([id, count]) => (
              <span key={id} className="inline-flex items-center gap-1.5 rounded-full bg-slate-100 px-3 py-1 text-xs dark:bg-slate-700">
                <span className="max-w-[120px] truncate font-medium text-slate-700 dark:text-slate-300">{id}</span>
                <span className="text-slate-400">{count}</span>
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Token Usage + Hourly Trends */}
      <div className="mt-4 grid gap-4 lg:grid-cols-2">
        {Object.keys(tokenUsage || {}).length > 0 && (
          <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-600 dark:bg-slate-800">
            <h3 className="mb-3 text-sm font-semibold text-slate-900 dark:text-slate-100">{t('common.token_usage', 'Token 使用分布')}</h3>
            <div className="space-y-2">
              {Object.entries(tokenUsage || {}).sort((a,b) => b[1]-a[1]).map(([name, count]) => {
                const max = Math.max(...Object.values(tokenUsage), 1)
                const pct = Math.round(count / max * 100)
                return (
                  <div key={name} className="flex items-center gap-2 text-xs">
                    <span className="w-16 shrink-0 text-slate-600 dark:text-slate-400">{name}</span>
                    <div className="h-2 flex-1 rounded-full bg-slate-100 dark:bg-slate-700">
                      <div className="h-2 rounded-full bg-sky-500" style={{width: pct + '%'}} />
                    </div>
                    <span className="w-8 text-right text-slate-400">{count}</span>
                  </div>
                )
              })}
            </div>
          </div>
        )}

        {hourly?.labels?.length > 0 && (
          <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-600 dark:bg-slate-800">
            <h3 className="mb-3 text-sm font-semibold text-slate-900 dark:text-slate-100">{t('common.hourly_trends', '24h 趋势')}</h3>
            <div className="flex items-end gap-[2px] h-20">
              {(hourly?.msgs || []).map((v, i) => {
                const max = Math.max(...(hourly.msgs || [1]), 1)
                return <div key={i} title={hourly.labels[i] + ': ' + v + ' msgs'} className="flex-1 rounded-t bg-sky-200 dark:bg-sky-800" style={{height: Math.max(v / max * 100, 2) + '%'}} />
              })}
            </div>
            <div className="mt-1 flex justify-between text-[9px] text-slate-400">
              <span>{hourly?.labels?.[0]}</span>
              <span>{hourly?.labels?.[12]}</span>
              <span>{hourly?.labels?.[23]}</span>
            </div>
          </div>
        )}
      </div>

      {/* Repo Sync Health */}
      {repoHealth?.length > 0 && (
        <div className="mt-4 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-600 dark:bg-slate-800">
          <h3 className="mb-3 text-sm font-semibold text-slate-900 dark:text-slate-100">{t('common.repo_health', '仓库同步状态')}</h3>
          <div className="flex flex-wrap gap-2">
            {repoHealth.map((r) => {
              const stale = r.lastSync && r.lastSync.includes('failed')
              return (
                <span key={r.id} className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs ${stale ? 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-400' : 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'}`}>
                  <span className={`h-1.5 w-1.5 rounded-full ${stale ? 'bg-red-400' : 'bg-emerald-400'}`} />
                  <span className="max-w-[150px] truncate">{r.id}</span>
                  <span className="opacity-60">{r.lastSync || '-'}</span>
                </span>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}
