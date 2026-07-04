import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getAgents, getDeployments, getRepositories } from '../api'
import useAsyncData from '../hooks/useAsyncData'
import PageHeader from '../components/PageHeader'
import LoadingPanel from '../components/LoadingPanel'
import { useT } from '../i18n'

const STATUS_COLORS = {
  running: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400',
  stopped: 'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300',
  build_failed: 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  run_failed: 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  crashed: 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  building_context: 'bg-sky-50 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400',
  building_image: 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400',
  starting_container: 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
}

function StatCard({ label, value, icon, color, to }) {
  const colors = {
    emerald: 'border-emerald-200 bg-emerald-50 dark:border-emerald-800 dark:bg-emerald-900/30',
    sky: 'border-sky-200 bg-sky-50 dark:border-sky-800 dark:bg-sky-900/30',
    amber: 'border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-900/30',
    rose: 'border-rose-200 bg-rose-50 dark:border-rose-800 dark:bg-rose-900/30',
    slate: 'border-slate-200 bg-slate-50 dark:border-slate-600 dark:bg-slate-800',
  }
  const inner = (
    <div className={`rounded-2xl border p-5 ${colors[color]} ${to ? 'cursor-pointer transition hover:-translate-y-0.5 hover:shadow-md' : ''}`}>
      <div className="flex items-center justify-between">
        <div className="text-3xl font-bold text-slate-900 dark:text-slate-100">{value}</div>
        {icon && <div className="text-2xl opacity-60">{icon}</div>}
      </div>
      <div className="mt-1 text-sm text-slate-500 dark:text-slate-400">{label}</div>
    </div>
  )
  return to ? <Link to={to}>{inner}</Link> : inner
}

export default function DashboardPage() {
  const { data: agents = [], loading: aLoad } = useAsyncData(getAgents, [])
  const { data: repos = [], loading: rLoad } = useAsyncData(getRepositories, [])
  const [deployments, setDeployments] = useState([])
  const [dLoad, setDLoad] = useState(true)
  const t = useT()

  useEffect(() => {
    getDeployments().then((d) => { setDeployments(d); setDLoad(false) }).catch(() => setDLoad(false))
  }, [])

  if (aLoad || rLoad || dLoad) return <LoadingPanel label={t('common.loading')} />

  const running = deployments.filter((d) => d.status === 'running')
  const failed = deployments.filter((d) => ['build_failed','run_failed','crashed'].includes(d.status))
  const inProgress = deployments.filter((d) => ['building_context','building_image','starting_container'].includes(d.status))
  const recent = [...deployments].sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt)).slice(0, 5)

  return (
    <div>
      <PageHeader
        title={t('common.dashboard', 'Dashboard')}
        description={t('common.dashboard_desc', 'AgentBucket 控制平面概览')}
      />

      {/* Stats */}
      <div className="mb-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label={t('agents.title')} value={agents.length} icon="🤖" color="sky" to="/agents" />
        <StatCard label={t('common.running')} value={running.length} icon="🟢" color="emerald" to="/deploy/progress" />
        <StatCard label={t('common.failed', 'Failed')} value={failed.length} icon="⚠️" color={failed.length > 0 ? 'rose' : 'slate'} to="/deploy/progress" />
        <StatCard label={t('repositories.title')} value={repos.length} icon="📦" color="slate" to="/repositories" />
      </div>

      {/* Alerts */}
      {inProgress.length > 0 && (
        <div className="mb-4 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-900/30 dark:text-amber-300">
          {inProgress.length} {t('deploy.deploying')} — <Link to="/deploy/progress" className="font-medium underline">{t('common.view_all', '查看')}</Link>
        </div>
      )}
      {failed.length > 0 && (
        <div className="mb-4 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-800 dark:bg-rose-900/30 dark:text-rose-300">
          {failed.length} {t('progress.step_failed')} — <Link to="/deploy/progress" className="font-medium underline">{t('common.view_all', '查看')}</Link>
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Quick Actions */}
        <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-600 dark:bg-slate-800">
          <h3 className="mb-4 text-sm font-semibold text-slate-900 dark:text-slate-100">{t('common.quick_actions', '快捷操作')}</h3>
          <div className="space-y-2">
            <Link to="/deploy" className="block rounded-xl bg-sky-600 px-4 py-3 text-center text-sm font-medium text-white transition hover:bg-sky-700">
              {t('deploy.deploy_button')}
            </Link>
            <Link to="/repositories" className="block rounded-xl border border-slate-200 px-4 py-3 text-center text-sm font-medium text-slate-700 transition hover:bg-slate-50 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700">
              {t('repositories.bind_repo')}
            </Link>
            <Link to="/ai-tokens" className="block rounded-xl border border-slate-200 px-4 py-3 text-center text-sm font-medium text-slate-700 transition hover:bg-slate-50 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700">
              {t('aITokens.create_token')}
            </Link>
          </div>
        </div>

        {/* Recent Deployments */}
        <div className="lg:col-span-2 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-600 dark:bg-slate-800">
          <div className="mb-4 flex items-center justify-between">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">{t('common.recent_deployments', '最近部署')}</h3>
            <Link to="/deploy/progress" className="text-xs text-sky-600 hover:underline dark:text-sky-400">{t('common.view_all', '查看全部')}</Link>
          </div>
          {recent.length === 0 ? (
            <p className="py-8 text-center text-sm text-slate-400 dark:text-slate-500">{t('common.no_deployments_yet', '暂无部署')} <Link to="/deploy" className="text-sky-600 underline">{t('deploy.deploy_button')}</Link></p>
          ) : (
            <div className="space-y-2">
              {recent.map((d) => (
                <div key={d.id} className="flex items-center justify-between rounded-lg bg-slate-50 px-4 py-2.5 dark:bg-slate-900/50">
                  <div>
                    <span className="text-sm font-medium text-slate-900 dark:text-slate-100">{d.agentId}</span>
                    <span className="ml-2 text-xs text-slate-400">{d.model} · {d.runtime}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${STATUS_COLORS[d.status] || 'bg-slate-100 text-slate-500'}`}>{d.status}</span>
                    <span className="text-xs text-slate-400">{new Date(d.createdAt).toLocaleDateString()}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
