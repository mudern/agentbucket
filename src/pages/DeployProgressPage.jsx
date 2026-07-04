import { useEffect, useMemo, useState } from 'react'
import { getDeployments, getAgents } from '../api'
import { Link } from 'react-router-dom'
import LoadingPanel from '../components/LoadingPanel'
import PageHeader from '../components/PageHeader'
import { useT } from '../i18n'
import useAsyncData from '../hooks/useAsyncData'

const STATUS_TABS = [
  { key: 'all', label: 'common.all' },
  { key: 'running', label: 'common.running' },
  { key: 'building_context', label: 'progress.building_context' },
  { key: 'building_image', label: 'progress.building_image' },
  { key: 'starting_container', label: 'progress.starting_container' },
  { key: 'stopped', label: 'common.stopped' },
  { key: 'build_failed', label: 'common.build_failed' },
  { key: 'run_failed', label: 'common.run_failed' },
  { key: 'crashed', label: 'common.crashed' },
]

const COLORS = {
  running: 'border-emerald-200 bg-emerald-50 text-emerald-700',
  stopped: 'border-slate-200 bg-slate-50 text-slate-600',
  crashed: 'border-red-200 bg-red-50 text-red-700',
  build_failed: 'border-red-200 bg-red-50 text-red-700',
  run_failed: 'border-red-200 bg-red-50 text-red-700',
  building_context: 'border-sky-200 bg-sky-50 text-sky-700',
  building_image: 'border-indigo-200 bg-indigo-50 text-indigo-700',
  starting_container: 'border-amber-200 bg-amber-50 text-amber-700',
}

const DOT = {
  running: 'bg-emerald-400',
  stopped: 'bg-slate-300',
  crashed: 'bg-red-400',
  build_failed: 'bg-red-400',
  run_failed: 'bg-red-400',
  building_context: 'bg-sky-400 animate-pulse',
  building_image: 'bg-indigo-400 animate-pulse',
  starting_container: 'bg-amber-400 animate-pulse',
}

export default function DeployProgressPage() {
  const [deployments, setDeployments] = useState([])
  const [loading, setLoading] = useState(true)
  const [tab, setTab] = useState('all')
  const [drillAgent, setDrillAgent] = useState(null)
  const { data: agents = [] } = useAsyncData(getAgents, [])
  const t = useT()

  const fetchDeployments = async () => {
    try {
      const data = await getDeployments()
      setDeployments(data)
    } catch (e) { /* ignore */ }
    finally { setLoading(false) }
  }

  useEffect(() => {
    fetchDeployments()
    const interval = setInterval(fetchDeployments, 5000)
    return () => clearInterval(interval)
  }, [])

  const agentMap = useMemo(() => {
    const map = {}
    for (const a of agents) map[a.id] = a
    return map
  }, [agents])

  const filtered = useMemo(() => {
    let list = drillAgent ? deployments.filter((d) => d.agentId === drillAgent).slice(0, 20) : deployments
    if (tab !== 'all') list = list.filter((d) => d.status === tab)
    return list
  }, [deployments, tab, drillAgent])

  const sorted = useMemo(() =>
    [...filtered].sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt)),
  [filtered])

  const hasActive = useMemo(() =>
    deployments.some((d) =>
      d.status === 'building_context' || d.status === 'building_image' || d.status === 'starting_container'
    ),
  [deployments])

  if (loading) {
    return <LoadingPanel label={t('common.loading')} />
  }

  return (
    <div>
      <PageHeader
        title={t('progress.title')}
        description={hasActive ? t('progress.step_building') : t('progress.no_active')}
      />

      {/* Status tab bar */}
      <div className="mb-4 flex flex-wrap items-center gap-1.5">
        {STATUS_TABS.map(({ key, label }) => {
          const count = key === 'all'
            ? deployments.length
            : deployments.filter((d) => d.status === key).length
          if (key !== 'all' && count === 0) return null
          return (
            <button
              key={key}
              onClick={() => { setTab(key); setDrillAgent(null) }}
              className={`rounded-lg px-3 py-1.5 text-xs font-medium transition ${
                tab === key && !drillAgent
                  ? 'bg-sky-100 text-sky-700 shadow-sm'
                  : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
              }`}
            >
              {t(label)} ({count})
            </button>
          )
        })}
        {drillAgent && (
          <button
            onClick={() => setDrillAgent(null)}
            className="rounded-lg bg-sky-600 px-3 py-1.5 text-xs font-medium text-white"
          >
            ← {agentMap[drillAgent]?.name ?? drillAgent}
          </button>
        )}
      </div>

      {sorted.length === 0 && (
        <div className="rounded-2xl border border-dashed border-slate-300 bg-white p-16 text-center">
          <div className="text-sm font-medium text-slate-400">{t('deploy.no_deployments')}</div>
          <Link to="/deploy" className="mt-4 inline-block rounded-xl bg-sky-600 px-5 py-2 text-sm font-medium text-white hover:bg-sky-700">
            {t('deploy.deploy_button')}
          </Link>
        </div>
      )}

      <div className="space-y-3">
        {sorted.map((dep) => {
          const agent = agentMap[dep.agentId]
          return (
            <div
              key={dep.id}
              className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm"
            >
              <div className="flex items-center gap-3 px-5 py-3.5">
                <span className={`h-2.5 w-2.5 rounded-full ${DOT[dep.status] || 'bg-slate-300'}`} />
                <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ring-1 ring-inset ${COLORS[dep.status] || 'border-slate-200 bg-slate-50 text-slate-600'}`}>
                  {dep.status}
                </span>

                <button
                  onClick={() => setDrillAgent(drillAgent === dep.agentId ? null : dep.agentId)}
                  className="min-w-0 flex-1 text-left"
                >
                  <span className="text-sm font-semibold text-slate-950 hover:text-sky-700 transition">
                    {agent?.name ?? dep.agentId}
                  </span>
                </button>

                <div className="flex shrink-0 items-center gap-2 text-xs text-slate-400">
                  {dep.model && <span className="rounded-full bg-slate-100 px-2 py-0.5">{dep.model}</span>}
                  {dep.runtime && <span className="rounded-full bg-slate-100 px-2 py-0.5">{dep.runtime}</span>}
                  {dep.sidecarUrl && <span className="text-sky-600">{dep.sidecarUrl}</span>}
                  <span>{new Date(dep.createdAt).toLocaleString()}</span>
                </div>
              </div>
              {dep.message && (
                <div className="border-t border-slate-100 bg-slate-50 px-5 py-3">
                  <pre className="max-h-32 overflow-auto rounded-lg bg-slate-900 p-3 text-[11px] text-slate-300 leading-5 whitespace-pre-wrap font-mono">{dep.message}</pre>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
