import { useEffect, useState } from 'react'
import { getDeployments, getDeployOptions } from '../api'
import { Link } from 'react-router-dom'
import LoadingPanel from '../components/LoadingPanel'
import PageHeader from '../components/PageHeader'
import { useT } from '../i18n'
import useAsyncData from '../hooks/useAsyncData'

const statusColors = {
  running: 'bg-emerald-50 text-emerald-700 ring-emerald-200',
  stopped: 'bg-slate-50 text-slate-600 ring-slate-200',
  crashed: 'bg-red-50 text-red-700 ring-red-200',
  build_failed: 'bg-amber-50 text-amber-700 ring-amber-200',
  run_failed: 'bg-amber-50 text-amber-700 ring-amber-200',
  packaged: 'bg-sky-50 text-sky-700 ring-sky-200',
  building: 'bg-indigo-50 text-indigo-700 ring-indigo-200 animate-pulse',
}

const statusDot = {
  running: 'bg-emerald-400',
  stopped: 'bg-slate-300',
  crashed: 'bg-red-400',
  build_failed: 'bg-red-400',
  run_failed: 'bg-red-400',
  packaged: 'bg-sky-400',
  building: 'bg-indigo-400 animate-pulse',
}

export default function DeployProgressPage() {
  const [deployments, setDeployments] = useState([])
  const [loading, setLoading] = useState(true)
  const { data: options } = useAsyncData(getDeployOptions, [])
  const t = useT()

  const fetchDeployments = async () => {
    try {
      const data = await getDeployments()
      setDeployments(data)
    } catch (e) {
      // ignore
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchDeployments()
    const interval = setInterval(fetchDeployments, 5000)
    return () => clearInterval(interval)
  }, [])

  if (loading) {
    return <LoadingPanel label={t('common.loading')} />
  }

  const hasActive = deployments.some((d) => d.status === 'building' || d.status === 'packaged')
  const sorted = [...deployments].sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt))

  return (
    <div>
      <PageHeader
        title={t('progress.title')}
        description={hasActive ? t('progress.step_building') : t('progress.no_active')}
      />

      <div className="space-y-4">
        {sorted.map((dep) => {
          const agent = options?.repositories
            ?.flatMap((r) => r.commits?.flatMap((c) => c.agents) || [])
            ?.find((a) => a?.id === dep.agentId)

          return (
            <div
              key={dep.id}
              className={`overflow-hidden rounded-2xl border shadow-sm transition ${
                dep.status === 'running' ? 'border-emerald-200 bg-white' :
                dep.status === 'build_failed' || dep.status === 'run_failed' ? 'border-red-200 bg-white' :
                'border-slate-200 bg-white'
              }`}
            >
              <div className="flex items-center gap-4 px-6 py-4">
                {/* Status dot */}
                <div className="flex items-center gap-3">
                  <span className={`h-3 w-3 rounded-full ${statusDot[dep.status] || 'bg-slate-300'}`} />
                  <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 ring-inset ${statusColors[dep.status] || 'bg-slate-50 text-slate-600 ring-slate-200'}`}>
                    {dep.status === 'running' ? t('common.running') :
                     dep.status === 'stopped' ? t('common.stopped') :
                     dep.status === 'crashed' ? t('common.crashed') :
                     dep.status === 'build_failed' ? t('common.build_failed') :
                     dep.status === 'run_failed' ? t('common.run_failed') :
                     dep.status === 'packaged' ? t('common.packaged') :
                     dep.status === 'building' ? '构建中' :
                     dep.status}
                  </span>
                </div>

                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-3">
                    <span className="text-sm font-semibold text-slate-950">
                      {agent?.name ?? dep.agentID}
                    </span>
                    <span className="text-xs text-slate-400">{dep.imageTag}</span>
                  </div>
                  <div className="mt-1 flex items-center gap-3 text-xs text-slate-400">
                    <span>{dep.id}</span>
                    {dep.sidecarUrl && (
                      <span className="text-sky-600">{dep.sidecarUrl}</span>
                    )}
                  </div>
                </div>

                <div className="flex shrink-0 items-center gap-3">
                  {dep.model && (
                    <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">{dep.model}</span>
                  )}
                  {dep.runtime && (
                    <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">{dep.runtime}</span>
                  )}
                  <Link
                    to={`/agents/${dep.agentID}`}
                    className="rounded-lg bg-sky-50 px-3 py-1.5 text-xs font-medium text-sky-700 hover:bg-sky-100"
                  >
                    {t('agents.chat')}
                  </Link>
                </div>
              </div>

              {/* Build log */}
              {dep.message && (
                <div className="border-t border-slate-100 bg-slate-50 px-6 py-4">
                  <div className="mb-2 text-xs font-medium text-slate-500">{t('progress.build_log')}</div>
                  <pre className="max-h-48 overflow-auto rounded-lg bg-slate-900 p-4 text-xs text-slate-300 leading-5 whitespace-pre-wrap font-mono">
                    {dep.message}
                  </pre>
                </div>
              )}

              {/* Timestamp */}
              <div className="border-t border-slate-100 bg-slate-50/50 px-6 py-2">
                <span className="text-xs text-slate-400">
                  {new Date(dep.createdAt).toLocaleString()}
                </span>
              </div>
            </div>
          )
        })}

        {deployments.length === 0 && (
          <div className="rounded-2xl border border-dashed border-slate-300 bg-white p-16 text-center">
            <div className="text-sm font-medium text-slate-400">{t('deploy.no_deployments')}</div>
            <Link to="/deploy" className="mt-4 inline-block rounded-xl bg-sky-600 px-5 py-2 text-sm font-medium text-white hover:bg-sky-700">
              {t('deploy.deploy_button')}
            </Link>
          </div>
        )}
      </div>
    </div>
  )
}
