import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getAgents, getDeployments, getRepositories } from '../api'
import useAsyncData from '../hooks/useAsyncData'
import PageHeader from '../components/PageHeader'
import LoadingPanel from '../components/LoadingPanel'
import { useT } from '../i18n'

function StatCard({ label, value, color, to }) {
  const c = { emerald: 'border-emerald-200 bg-emerald-50 dark:border-emerald-800 dark:bg-emerald-900/30',
    sky: 'border-sky-200 bg-sky-50 dark:border-sky-800 dark:bg-sky-900/30',
    amber: 'border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-900/30',
    slate: 'border-slate-200 bg-slate-50 dark:border-slate-600 dark:bg-slate-800',
  }[color] || ''
  const inner = (
    <div className={`rounded-2xl border p-5 ${c} ${to ? 'cursor-pointer transition hover:-translate-y-0.5 hover:shadow-md' : ''}`}>
      <div className="text-3xl font-bold text-slate-900 dark:text-slate-100">{value}</div>
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

  const running = deployments.filter((d) => d.status === 'running').length
  const failed = deployments.filter((d) => d.status === 'build_failed' || d.status === 'run_failed' || d.status === 'crashed').length
  const inProgress = deployments.filter((d) => d.status === 'building_context' || d.status === 'building_image' || d.status === 'starting_container').length

  return (
    <div>
      <PageHeader title={t('common.dashboard', 'Dashboard')} description={t('common.dashboard_desc', 'Overview of your AgentBucket instance')} />
      <div className="mb-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label={t('agents.title')} value={agents.length} color="sky" to="/" />
        <StatCard label={t('common.running')} value={running} color="emerald" to="/deploy/progress" />
        <StatCard label={t('deploy.no_deployments')} value={deployments.length} color="slate" to="/deploy/progress" />
        <StatCard label={t('repositories.title')} value={repos.length} color="slate" to="/repositories" />
      </div>
      {inProgress > 0 && (
        <div className="mb-6 rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-900/30 dark:text-amber-300">
          {inProgress} {t('deploy.deploying')} — <Link to="/deploy/progress" className="font-medium underline">View progress</Link>
        </div>
      )}
      {failed > 0 && (
        <div className="mb-6 rounded-2xl border border-rose-200 bg-rose-50 p-4 text-sm text-rose-800 dark:border-rose-800 dark:bg-rose-900/30 dark:text-rose-300">
          {failed} {t('progress.step_failed')} — <Link to="/deploy/progress" className="font-medium underline">View details</Link>
        </div>
      )}
      <div className="grid gap-4 lg:grid-cols-2">
        <Link to="/deploy" className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md dark:border-slate-600 dark:bg-slate-800">
          <div className="text-lg font-semibold text-slate-900 dark:text-slate-100">{t('deploy.deploy_button')}</div>
          <div className="mt-1 text-sm text-slate-500 dark:text-slate-400">{t('deploy.select_repo')}</div>
        </Link>
        <Link to="/repositories" className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md dark:border-slate-600 dark:bg-slate-800">
          <div className="text-lg font-semibold text-slate-900 dark:text-slate-100">{t('repositories.bind_repo')}</div>
          <div className="mt-1 text-sm text-slate-500 dark:text-slate-400">{t('repositories.no_repos')}</div>
        </Link>
      </div>
    </div>
  )
}
