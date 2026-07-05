import { useEffect, useMemo, useRef, useState } from 'react'
import { getDeployments, getAgents } from '../api'
import { Link } from 'react-router-dom'
import LoadingPanel from '../components/LoadingPanel'
import PageHeader from '../components/PageHeader'
import { useT } from '../i18n'
import { useToast } from '../components/Toast'
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
  stopped: 'border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900 text-slate-600 dark:text-slate-400',
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
  const [logSearch, setLogSearch] = useState('')
  const [expandedLogs, setExpandedLogs] = useState({})
  const { data: agents = [] } = useAsyncData(getAgents, [])
  const prevStatus = useRef({})
  const toast = useToast()
  const t = useT()

  useEffect(() => {
    const API_BASE = import.meta.env.VITE_API_BASE ?? 'http://127.0.0.1:8080'
    let abort = false

    // Load initial data via fast API call
    getDeployments().then((data) => { setDeployments(data); setLoading(false) }).catch(() => setLoading(false))

    // Then connect SSE for live updates
    const connectSSE = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/deployments/stream`)
        const reader = resp.body.getReader()
        const decoder = new TextDecoder()
        let buffer = ''
        while (!abort) {
          const { done, value } = await reader.read()
          if (done) break
          buffer += decoder.decode(value, { stream: true })
          const lines = buffer.split('\n')
          buffer = lines.pop() || ''
          for (const line of lines) {
            if (line.startsWith('data: ')) {
              try {
                    const data = JSON.parse(line.slice(6))
                    // Toast on new deployments or status changes
                    for (const dep of data) {
                      const prev = prevStatus.current[dep.id]
                      if (!prev) { prevStatus.current[dep.id] = dep.status; continue }
                      if (prev !== dep.status) {
                        prevStatus.current[dep.id] = dep.status
                        if (dep.status === 'running') toast(dep.agentId + ' deployed successfully', 'success')
                        else if (dep.status === 'build_failed' || dep.status === 'run_failed') toast(dep.agentId + ' failed: ' + dep.status, 'error')
                        else if (dep.status === 'crashed') toast(dep.agentId + ' crashed and is restarting', 'error')
                      }
                    }
                    setDeployments(data)
                  } catch (_) {}
            }
          }
        }
      } catch (_) {}
    }
    connectSSE()

    return () => { abort = true }
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
                  : 'bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-200'
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
        <div className="rounded-2xl border border-dashed border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 p-16 text-center">
          <div className="text-sm font-medium text-slate-400 dark:text-slate-500">{t('deploy.no_deployments')}</div>
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
              className="overflow-hidden rounded-2xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 shadow-sm"
            >
              <div className="flex items-center gap-3 px-5 py-3.5">
                <span className={`h-2.5 w-2.5 rounded-full ${DOT[dep.status] || 'bg-slate-300'}`} />
                <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ring-1 ring-inset ${COLORS[dep.status] || 'border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900 text-slate-600 dark:text-slate-400'}`}>
                  {dep.status}
                </span>

                <button
                  onClick={() => setDrillAgent(drillAgent === dep.agentId ? null : dep.agentId)}
                  className="min-w-0 flex-1 text-left"
                >
                  <span className="text-sm font-semibold text-slate-950 dark:text-slate-50 hover:text-sky-700 transition">
                    {agent?.name ?? dep.agentId}
                  </span>
                </button>

                <div className="flex shrink-0 items-center gap-2 text-xs text-slate-400 dark:text-slate-500">
                  {dep.model && <span className="rounded-full bg-slate-100 dark:bg-slate-700 px-2 py-0.5">{dep.model}</span>}
                  {dep.runtime && <span className="rounded-full bg-slate-100 dark:bg-slate-700 px-2 py-0.5">{dep.runtime}</span>}
                  
                  <span>{new Date(dep.createdAt).toLocaleString()}</span>
                </div>
              </div>
              {dep.message && (
                <div className="border-t border-slate-100 dark:border-slate-700/50 bg-slate-50 dark:bg-slate-900 px-5 py-3">
                  <div className="mb-2 flex items-center gap-2">
                    <button
                      onClick={() => setExpandedLogs((e) => ({ ...e, [dep.id]: !e[dep.id] }))}
                      className="rounded px-2 py-0.5 text-xs text-slate-500 dark:text-slate-400 hover:bg-slate-200"
                    >
                      {expandedLogs[dep.id] ? '收起' : '展开'}日志
                    </button>
                    {expandedLogs[dep.id] && (
                      <input
                        className="min-w-[120px] rounded border border-slate-200 dark:border-slate-700 px-2 py-0.5 text-xs outline-none focus:border-sky-400"
                        placeholder="搜索日志..."
                        value={logSearch}
                        onChange={(e) => setLogSearch(e.target.value)}
                      />
                    )}
                  </div>
                  {expandedLogs[dep.id] && (
                    <pre className="max-h-64 overflow-auto rounded-lg bg-slate-900 p-3 text-[11px] text-slate-300 dark:text-slate-600 leading-5 whitespace-pre-wrap font-mono">
                      {logSearch ? dep.message.split('\n').filter((l) => l.toLowerCase().includes(logSearch.toLowerCase())).join('\n') : dep.message}
                    </pre>
                  )}
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
