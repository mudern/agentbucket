import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { createDeployment, getDeployOptions, getDeployments, getAgents, checkDeploymentHealth, stopDeployment, startDeployment, deleteDeployment } from '../api'
import LoadingPanel from '../components/LoadingPanel'
import PageHeader from '../components/PageHeader'
import useAsyncData from '../hooks/useAsyncData'
import { useT } from '../i18n'
import ConfirmDialog from '../components/ConfirmDialog'

function summarizeItems(items, t) {
  if (!items.length) return t('common.not_selected')
  if (items.length <= 2) return items.join('、')
  return `${items.slice(0, 2).join('、')} 等 ${items.length} 项`
}

function CapabilityCard({ title, description, value, count, onOpen }) {
  const t = useT()
  return (
    <button
      type="button"
      onClick={onOpen}
      className="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4 text-left transition hover:border-sky-200 hover:bg-sky-50/40"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="text-sm font-semibold text-slate-950 dark:text-slate-50">{title}</div>
          <div className="mt-1 text-xs leading-5 text-slate-500 dark:text-slate-400 dark:text-slate-400">{description}</div>
        </div>
        <span className="shrink-0 rounded-full bg-slate-100 dark:bg-slate-700 px-2 py-0.5 text-[10px] font-medium text-slate-500 dark:text-slate-400">
          {count}
        </span>
      </div>
      <div className="mt-4 truncate text-sm text-slate-700 dark:text-slate-300">{value}</div>
      <div className="mt-3 text-xs font-medium text-sky-700">{t('common.open_picker')}</div>
    </button>
  )
}

function CapabilityPickerModal({ open, title, mode, items, selected, onClose, onSelectOne, onToggle, onSelectAll, onClear }) {
  const [query, setQuery] = useState('')
  const t = useT()

  if (!open) return null

  const filteredItems = items
    .filter((item) => [item.label, item.description, item.meta].join(' ').toLowerCase().includes(query.trim().toLowerCase()))
    .sort((a, b) => Number(selected.includes(b.id)) - Number(selected.includes(a.id)))

  const selectedCount = selected.length
  const multi = mode === 'multi'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/30 px-4 py-8">
      <div className="flex max-h-[82vh] w-full max-w-3xl flex-col overflow-hidden rounded-2xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 shadow-xl">
        <div className="flex shrink-0 items-center justify-between border-b border-slate-200 dark:border-slate-700 px-5 py-4">
          <div>
            <div className="text-base font-semibold text-slate-950 dark:text-slate-50">{title}</div>
            <div className="mt-1 text-xs text-slate-400 dark:text-slate-500">
              {multi ? t('common.selected_count').replace(/\{count\}/g, selectedCount) : t('deploy.single_mode')}
            </div>
          </div>
          <button onClick={onClose} className="rounded-lg px-2 py-1 text-sm text-slate-400 dark:text-slate-500 dark:text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-700 hover:text-slate-700 dark:text-slate-300">
            {t('common.close')}
          </button>
        </div>

        <div className="shrink-0 border-b border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/70 p-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder={t('deploy.search_placeholder')}
              className="min-w-0 flex-1 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 px-4 py-2.5 text-sm text-slate-900 dark:text-slate-100 outline-none placeholder:text-slate-400 dark:placeholder:text-slate-500 dark:text-slate-500 focus:border-sky-400"
            />
            {multi && (
              <div className="flex shrink-0 gap-2">
                <button type="button" onClick={onSelectAll} className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 px-3 py-2 text-xs font-medium text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 dark:hover:bg-slate-700 dark:bg-slate-900">
                  {t('common.select_all')}
                </button>
                <button type="button" onClick={onClear} className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 px-3 py-2 text-xs font-medium text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 dark:hover:bg-slate-700 dark:bg-slate-900">
                  {t('common.clear')}
                </button>
              </div>
            )}
          </div>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto p-3">
          <div className="space-y-2">
            {filteredItems.map((item) => {
              const checked = selected.includes(item.id)
              return (
                <button
                  key={item.id}
                  type="button"
                  onClick={() => (multi ? onToggle(item.id) : onSelectOne(item.id))}
                  className={`flex w-full items-start gap-3 rounded-xl border p-3 text-left transition ${
                    checked ? 'border-sky-200 bg-sky-50 dark:border-sky-800 dark:bg-sky-900/50' : 'border-slate-200 dark:border-slate-700 bg-white hover:bg-slate-50 dark:hover:bg-slate-700 dark:hover:bg-slate-700 dark:bg-slate-900'
                  }`}
                >
                  <span className={`mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full border text-[10px] font-bold ${
                    checked ? 'border-sky-600 bg-sky-600 text-white dark:border-sky-400 dark:bg-sky-500' : 'border-slate-300 dark:border-slate-600 bg-white text-transparent'
                  }`}>
                    &#10003;
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-sm font-medium text-slate-900 dark:text-slate-100">{item.label}</span>
                    {item.description && <span className="mt-1 block text-xs leading-5 text-slate-500 dark:text-slate-400 dark:text-slate-400">{item.description}</span>}
                    {item.meta && <span className="mt-2 inline-flex rounded-full bg-white dark:bg-slate-800 px-2 py-0.5 text-[10px] text-slate-400 dark:text-slate-500 dark:text-slate-500 ring-1 ring-slate-200 dark:ring-slate-600">{item.meta}</span>}
                  </span>
                </button>
              )
            })}
          </div>
          {filteredItems.length === 0 && (
            <div className="rounded-xl border border-dashed border-slate-200 dark:border-slate-700 p-8 text-center text-sm text-slate-400 dark:text-slate-500 dark:text-slate-500">
              {t('common.no_match')}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default function DeployPage() {
  const navigate = useNavigate()
  const t = useT()
  const stepLabels = [t('deploy.step_repo'), t('deploy.step_commit'), t('deploy.step_agent'), t('deploy.step_config'), t('deploy.step_confirm')]

  const [step, setStep] = useState(0)
  const [form, setForm] = useState({
    repositoryId: '',
    commitHash: '',
    agentId: '',
    apiTokenId: '',
    model: '',
    runtime: '',
    runtimeVersion: 'latest',
    authTokens: [],
    skills: [],
    mcps: [],
  })
  const [deploying, setDeploying] = useState(false)
  const [deployResult, setDeployResult] = useState(null)
  const [deployError, setDeployError] = useState('')
  const [deployments, setDeployments] = useState([])
  const [picker, setPicker] = useState(null)
  const [capabilityTouched, setCapabilityTouched] = useState({ skills: false, mcps: false })
  const [confirm, setConfirm] = useState(null) // { id, title, message, action }
  const [healthStatus, setHealthStatus] = useState({}) // deploymentId -> { ok, error }
  const { data, loading } = useAsyncData(getDeployOptions, [])
  const { data: agentList = [] } = useAsyncData(getAgents, [])

  const agentNameMap = useMemo(() => {
    const map = {}
    for (const a of agentList) map[a.id] = a.name
    return map
  }, [agentList])

  const tokenNameMap = useMemo(() => {
    const map = {}
    if (data?.aiTokens) for (const t of data.aiTokens) map[t.id] = t.name
    return map
  }, [data])

  const agentDescMap = useMemo(() => {
    const map = {}
    for (const a of agentList) map[a.id] = a.description
    return map
  }, [agentList])

  // Periodic health check for running deployments
  useEffect(() => {
    const check = async () => {
      const deps = await getDeployments().catch(() => [])
      setDeployments(deps)
      const running = deps.filter((d) => d.status === 'running' && d.sidecarUrl)
      const status = {}
      await Promise.all(running.map(async (d) => {
        try {
          const result = await checkDeploymentHealth(d.id)
          status[d.id] = { ok: result?.ok ?? false, error: result?.error }
        } catch (e) {
          status[d.id] = { ok: false, error: 'unreachable' }
        }
      }))
      setHealthStatus(status)
    }
    check()
    const interval = setInterval(check, 15000)
    return () => clearInterval(interval)
  }, [deployResult])

  // Initialize form with first available values when data loads
  useEffect(() => {
    if (!data) return
    const firstRepo = data.repositories?.[0]
    const firstCommit = firstRepo?.commits?.[0]
    const firstAgent = firstCommit?.agents?.[0]
    const firstToken = data.aiTokens?.[0]
    if (firstRepo && firstCommit && firstAgent) {
      setForm((current) => ({
        ...current,
        repositoryId: current.repositoryId || firstRepo.id,
        commitHash: current.commitHash || firstCommit.hash,
        agentId: current.agentId || firstAgent.id,
        apiTokenId: current.apiTokenId || firstToken?.id || '',
        runtime: current.runtime || data.runtimes?.[0] || '',
        model: current.model || firstToken?.model || firstAgent.model || '',
        skills: capabilityTouched.skills ? current.skills : ['agentbucket-api'],
        mcps: capabilityTouched.mcps ? current.mcps : [],
      }))
    }
  }, [data])

  const repositories = data?.repositories ?? []
  const selectedRepository = repositories.find((repo) => repo.id === form.repositoryId) ?? repositories[0]
  const commits = selectedRepository?.commits ?? []
  const selectedCommit = commits.find((commit) => commit.hash === form.commitHash) ?? commits[0]
  const agents = selectedCommit?.agents ?? []
  const selectedAgent = agents.find((agent) => agent.id === form.agentId) ?? agents[0]
  const selectedSkills = capabilityTouched.skills ? form.skills : ['agentbucket-api']
  const selectedMcps = capabilityTouched.mcps ? form.mcps : []
  const selectedApiTokenId = form.apiTokenId || data?.aiTokens?.[0]?.id || ''
  const selectedApiToken = data?.aiTokens?.find((token) => token.id === selectedApiTokenId)
  // Model is determined by the selected API token, falling back to agent definition
  const selectedModel = selectedApiToken?.model || selectedAgent?.model || ''
  const selectedRuntime = form.runtime || data?.runtimes?.[0] || ''
  const selectedRuntimeVersion = form.runtimeVersion || data?.runtimeTags?.[0] || 'latest'
  const runtimeDescriptions = {
    codex: t('deploy.runtime_hint_codex'),
    claudecode: t('deploy.runtime_hint_claudecode'),
    opencode: t('deploy.runtime_hint_opencode'),
    gemini: t('deploy.runtime_hint_gemini'),
    reasonix: t('deploy.runtime_hint_reasonix'),
  }
  const selectedMcpNames = useMemo(() => {
    if (!data) return []
    return selectedMcps.map((mcp) => data.mcpServers.find((server) => server.id === mcp)?.name ?? mcp)
  }, [data, selectedMcps])

  const selectedTokenNames = useMemo(() => {
    if (!data) return []
    return data.authTokens.filter((token) => form.authTokens.includes(token.id)).map((token) => token.name)
  }, [data, form.authTokens])

  const pickerItems = useMemo(() => {
    if (!data) return { api: [], skills: [], mcps: [], auth: [] }
    return {
      api: data.aiTokens.map((token) => ({
        id: token.id,
        label: token.name,
        description: token.scope || token.provider,
        meta: [token.provider, token.model, token.status].filter(Boolean).join(' ·'),
      })),
      skills: (selectedAgent?.skills ?? []).map((skill) => ({
        id: skill,
        label: skill,
        description: 'Agent \u58f0\u660e\u7684\u6807\u51c6 Skill',
        meta: selectedSkills.includes(skill) ? '\u5df2\u542f\u7528' : '\u53ef\u9009',
      })),
      mcps: (selectedAgent?.mcps ?? []).map((mcp) => {
        const server = data.mcpServers.find((item) => item.id === mcp)
        return {
          id: mcp,
          label: server?.name ?? mcp,
          description: server?.scope ?? 'Agent \u58f0\u660e\u7684 MCP Server',
          meta: mcp,
        }
      }),
      auth: data.authTokens.map((token) => ({
        id: token.id,
        label: token.name,
        description: token.description || '',
        meta: token.status,
      })),
    }
  }, [data, selectedAgent, selectedMcps, selectedSkills])

  const updateForm = (key, value) => {
    setForm((current) => ({ ...current, [key]: value }))
  }

  const selectRepository = (repositoryId) => {
    const repo = repositories.find((item) => item.id === repositoryId)
    setForm((current) => ({
      ...current,
      repositoryId,
      commitHash: repo?.commits?.[0]?.hash ?? '',
      agentId: repo?.commits?.[0]?.agents?.[0]?.id ?? '',
      model: '',
      skills: [],
      mcps: [],
    }))
    setCapabilityTouched({ skills: false, mcps: false })
  }

  const selectAgent = (agentId) => {
    setForm((current) => ({ ...current, agentId, model: '', skills: [], mcps: [] }))
    setCapabilityTouched({ skills: false, mcps: false })
  }

  const toggleValue = (key, value) => {
    setForm((current) => {
      const values = current[key].includes(value)
        ? current[key].filter((item) => item !== value)
        : [...current[key], value]
      return { ...current, [key]: values }
    })
  }

  const toggleFromValues = (key, value, values) => {
    setCapabilityTouched((current) => ({ ...current, [key]: true }))
    setForm((current) => ({
      ...current,
      [key]: values.includes(value) ? values.filter((item) => item !== value) : [...values, value],
    }))
  }

  const setCapabilityValues = (key, values) => {
    setCapabilityTouched((current) => ({ ...current, [key]: true }))
    setForm((current) => ({ ...current, [key]: values }))
  }

  const submitDeployment = async () => {
    setDeploying(true)
    setDeployError('')
    setDeployResult(null)
    const payload = {
      repositoryId: selectedRepository?.id,
      commitHash: selectedCommit?.hash,
      agentId: selectedAgent?.id,
      apiTokenId: Number(selectedApiTokenId),
      model: selectedModel,
      runtime: selectedRuntime,
      runtimeVersion: selectedRuntimeVersion,
      skills: selectedSkills,
      mcps: selectedMcps,
      authTokens: form.authTokens,
    }
    console.log('[DEPLOY] SENDING:', JSON.stringify(payload, null, 2))
    try {
      const result = await createDeployment(payload)
      setDeployResult(result)
      // Navigate to progress page after successful deployment
      setTimeout(() => navigate('/deploy/progress'), 800)
    } catch (error) {
      setDeployError(error.message)
    } finally {
      setDeploying(false)
    }
  }

  if (loading || !data) {
    return <LoadingPanel label={t('deploy.title', '\u6b63\u5728\u52a0\u8f7d\u90e8\u7f72\u914d\u7f6e...')} />
  }

  return (
    <div>
      <PageHeader title={t('deploy.title')} description={t('deploy.select_repo')} />

      <div className="rounded-2xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 shadow-sm">
        <div className="border-b border-slate-200 dark:border-slate-700 px-6 py-5">
          <div className="flex items-center gap-3">
            {stepLabels.map((label, index) => (
              <div key={label} className="flex flex-1 items-center gap-3">
                <button
                  onClick={() => setStep(index)}
                  className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-sm font-semibold transition ${
                    index <= step ? 'bg-sky-600 text-white dark:bg-sky-500' : 'bg-slate-100 dark:bg-slate-700 text-slate-400 dark:text-slate-500'
                  }`}
                >
                  {index + 1}
                </button>
                <div className="hidden text-sm font-medium text-slate-700 dark:text-slate-300 dark:text-slate-300 md:block">{label}</div>
                {index < stepLabels.length - 1 && <div className="h-px flex-1 bg-slate-200" />}
              </div>
            ))}
          </div>
        </div>

        <div className="min-h-[460px] p-6">
          {step === 0 && (
            <div>
              <div className="mb-4 text-sm font-medium text-slate-950 dark:text-slate-50">{t('deploy.no_repos', '\u4ece\u4ed3\u5e93\u7ba1\u7406\u4e2d\u9009\u62e9\u5df2\u7ed1\u5b9a\u4ed3\u5e93')}</div>
              <div className="grid gap-4 xl:grid-cols-2">
                {repositories.map((repo) => (
                  <button
                    key={repo.id}
                    onClick={() => selectRepository(repo.id)}
                    className={`rounded-xl border p-5 text-left transition ${
                      selectedRepository?.id === repo.id ? 'border-sky-200 bg-sky-50 dark:border-sky-800 dark:bg-sky-900/50' : 'border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700 dark:hover:bg-slate-700 dark:bg-slate-900'
                    }`}
                  >
                    <div className="flex items-center justify-between gap-3">
                      <div className="font-medium text-slate-950 dark:text-slate-50">{(() => { const u = repo.url || ''; if (repo.provider === 'Local') return u.replace(/^file:\/\/\/?/, ''); const sm = u.match(/^git@([^:]+):(.+?)(?:\.git)?$/); if (sm) return sm[2]; const m = u.match(/\/([^\/]+\/[^\/]+?)(?:\.git)?$/); return m ? m[1] : u.replace(/^https?:\/\//, '') })()}</div>
                      <span className="rounded-full bg-white dark:bg-slate-800 px-2.5 py-1 text-xs text-slate-500 dark:text-slate-400">{repo.provider}</span>
                    </div>
                    <div className="mt-3 grid gap-2 text-xs text-slate-500 dark:text-slate-400 sm:grid-cols-3">
                      <div>{'\u9ed8\u8ba4\u5206\u652f\uff1a'}{repo.branch}</div>
                      <div>{'Agent \u76ee\u5f55\uff1a'}{repo.agentsPath}</div>
                      <div>{'Commits\uff1a'}{(repo.commits || []).length}</div>
                    </div>
                  </button>
                ))}
              </div>
              {repositories.length === 0 && (
                <div className="rounded-xl border border-dashed border-slate-300 dark:border-slate-600 bg-slate-50 dark:bg-slate-900 p-6 text-sm text-slate-500 dark:text-slate-400">
                  {t('deploy.no_repos')}
                </div>
              )}
            </div>
          )}

          {step === 1 && (
            <div>
              <div className="mb-4 text-sm font-medium text-slate-950 dark:text-slate-50">{t('deploy.select_commit')}</div>
              <div className="space-y-3">
                {commits.map((commit) => (
                  <button
                    key={commit.hash}
                    onClick={() => {
                      setForm((current) => ({
                        ...current,
                        commitHash: commit.hash,
                        agentId: (commit.agents || [])[0]?.id ?? '',
                        model: '',
                        skills: [],
                        mcps: [],
                      }))
                      setCapabilityTouched({ skills: false, mcps: false })
                    }}
                    className={`w-full rounded-xl border p-4 text-left transition ${
                      selectedCommit?.hash === commit.hash ? 'border-sky-200 bg-sky-50 dark:border-sky-800 dark:bg-sky-900/50' : 'border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700 dark:hover:bg-slate-700 dark:bg-slate-900'
                    }`}
                  >
                    <div className="flex items-center justify-between gap-3">
                      <div className="font-mono text-sm font-semibold text-slate-950 dark:text-slate-50">{commit.hash.slice(0, 8)}</div>
                      <div className="text-xs text-slate-400 dark:text-slate-500">{commit.committedAt}</div>
                    </div>
                    <div className="mt-2 text-sm text-slate-600 dark:text-slate-400">{commit.message}</div>
                    <div className="mt-2 text-xs text-slate-400 dark:text-slate-500">{'\u53d1\u73b0 '}{(commit.agents || []).length}{' \u4e2a Agent'}</div>
                  </button>
                ))}
              </div>
            </div>
          )}

          {step === 2 && (
            <div>
              <div className="mb-4 text-sm font-medium text-slate-950 dark:text-slate-50">{t('deploy.select_agent', '\u4ed3\u5e93\u5185\u53d1\u73b0\u7684 Agent')}</div>
              {agents.length === 0 ? (
                <div className="rounded-xl border border-dashed border-slate-300 dark:border-slate-600 bg-slate-50 dark:bg-slate-900 p-10 text-center text-sm text-slate-500 dark:text-slate-400">
                  {t('deploy.no_agents_in_commit', '\u6b64 commit \u4e2d\u672a\u627e\u5230 Agent \u5b9a\u4e49\u3002\u8bf7\u786e\u8ba4\u4ed3\u5e93\u4e2d\u5305\u542b agents/ \u76ee\u5f55\u548c agent.toml \u6587\u4ef6\u3002')}
                </div>
              ) : (
                <div className="grid gap-4 xl:grid-cols-2">
                  {agents.map((agent) => (
                    <button
                      key={agent.id}
                      onClick={() => selectAgent(agent.id)}
                      className={`rounded-xl border p-5 text-left transition ${
                        selectedAgent?.id === agent.id ? 'border-sky-200 bg-sky-50 dark:border-sky-800 dark:bg-sky-900/50' : 'border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700 dark:hover:bg-slate-700 dark:bg-slate-900'
                      }`}
                    >
                      <div className="text-base font-semibold text-slate-950 dark:text-slate-50">{agent.name}</div>
                      <div className="mt-2 text-sm leading-6 text-slate-600 dark:text-slate-400">{agent.description}</div>
                      <div className="mt-4 text-xs text-slate-400 dark:text-slate-500">{agent.path}</div>
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}

          {step === 3 && selectedAgent && (
            <div className="grid gap-6 xl:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
              <div className="rounded-2xl border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/70 p-5">
                <div className="mb-4">
                  <div className="text-sm font-semibold text-slate-950 dark:text-slate-50">{t('deploy.runtime_config')}</div>
                  <div className="mt-1 text-xs leading-5 text-slate-500 dark:text-slate-400 dark:text-slate-400">{t('deploy.runtime_config_desc')}</div>
                </div>
                <div className="grid gap-4">
                  <div className="block text-sm text-slate-700 dark:text-slate-300">
                    <div className="mb-1">{t('deploy.model')}</div>
                    <div className="flex items-center gap-2 rounded-xl border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900 px-4 py-3">
                      <span className="text-sm font-medium text-slate-900 dark:text-slate-100">{selectedModel || t('common.not_selected')}</span>
                      <span className="text-xs text-slate-400 dark:text-slate-500">— {t('deploy.model_from_token', '由 API Token 决定')}</span>
                    </div>
                  </div>
                  <label className="block text-sm text-slate-700 dark:text-slate-300">
                    {t('deploy.runtime')}
                    <select
                      value={selectedRuntime}
                      onChange={(event) => updateForm('runtime', event.target.value)}
                      className="mt-2 w-full rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 px-4 py-3 text-slate-900 dark:text-slate-100 outline-none focus:border-sky-500"
                    >
                      {data.runtimes.map((runtime) => (
                        <option key={runtime}>{runtime}</option>
                      ))}
                    </select>
                    <div className="mt-2 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 px-3 py-2 text-xs leading-5 text-slate-500 dark:text-slate-400 dark:text-slate-400">
                      {runtimeDescriptions[selectedRuntime] ?? t('deploy.runtime_hint_default')}
                    </div>
                  </label>
                  <label className="block text-sm text-slate-700 dark:text-slate-300">
                    {t('deploy.runtime_version')}
                    <select
                      value={selectedRuntimeVersion}
                      onChange={(event) => updateForm('runtimeVersion', event.target.value)}
                      className="mt-2 w-full rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 px-4 py-3 text-slate-900 dark:text-slate-100 outline-none focus:border-sky-500"
                    >
                      {(data.runtimeTags ?? ['latest']).map((version) => (
                        <option key={version}>{version}</option>
                      ))}
                    </select>
                  </label>
                </div>
              </div>

              <div>
                <div className="mb-4">
                  <div className="text-sm font-semibold text-slate-950 dark:text-slate-50">{t('deploy.capabilities')}</div>
                  <div className="mt-1 text-xs leading-5 text-slate-500 dark:text-slate-400 dark:text-slate-400">{t('deploy.capabilities_desc')}</div>
                </div>
                <div className="grid gap-3 md:grid-cols-2">
                  <CapabilityCard
                    title={t('deploy.api_token')}
                    description={t('deploy.api_token_desc')}
                    count={t('common.all', '\u5355\u9009')}
                    value={selectedApiToken?.name ?? t('common.not_selected')}
                    onOpen={() => setPicker('api')}
                  />
                  <CapabilityCard
                    title={t('deploy.skill')}
                    description={t('deploy.skill_desc')}
                    count={`${selectedSkills.length}/${selectedAgent.skills.length}`}
                    value={summarizeItems(selectedSkills, t)}
                    onOpen={() => setPicker('skills')}
                  />
                  <CapabilityCard
                    title={t('deploy.mcp')}
                    description={t('deploy.mcp_desc')}
                    count={`${selectedMcps.length}/${selectedAgent.mcps.length}`}
                    value={summarizeItems(selectedMcpNames, t)}
                    onOpen={() => setPicker('mcps')}
                  />
                  <CapabilityCard
                    title={t('deploy.auth_token')}
                    description={t('deploy.auth_token_desc')}
                    count={`${form.authTokens.length}/${data.authTokens.length}`}
                    value={summarizeItems(selectedTokenNames, t)}
                    onOpen={() => setPicker('auth')}
                  />
                </div>
              </div>
            </div>
          )}

          {step === 4 && (
            <div className="max-w-4xl rounded-2xl border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900 p-5">
              <div className="mb-4 text-sm font-semibold text-slate-950 dark:text-slate-50">{t('deploy.review_and_deploy')}</div>
              <dl className="grid gap-4 text-sm md:grid-cols-2">
                <div>
                  <dt className="text-slate-400 dark:text-slate-500">{t('deploy.repo_label', '\u4ed3\u5e93')}</dt>
                  <dd className="mt-1 text-slate-900 dark:text-slate-100">{selectedRepository?.url}</dd>
                </div>
                <div>
                  <dt className="text-slate-400 dark:text-slate-500">{'\u5206\u652f'}</dt>
                  <dd className="mt-1 text-slate-900 dark:text-slate-100">{selectedRepository?.branch}</dd>
                </div>
                <div>
                  <dt className="text-slate-400 dark:text-slate-500">{t('deploy.commit_label', 'Commit')}</dt>
                  <dd className="mt-1 font-mono text-slate-900 dark:text-slate-100">{(selectedCommit?.hash || '').slice(0, 8)}</dd>
                </div>
                <div>
                  <dt className="text-slate-400 dark:text-slate-500">{t('deploy.agent', 'Agent')}</dt>
                  <dd className="mt-1 text-slate-900 dark:text-slate-100">{selectedAgent?.name}</dd>
                </div>
                <div>
                  <dt className="text-slate-400 dark:text-slate-500">{t('deploy.api_token')}</dt>
                  <dd className="mt-1 text-slate-900 dark:text-slate-100">{selectedApiToken?.name}</dd>
                </div>
                <div>
                  <dt className="text-slate-400 dark:text-slate-500">{t('deploy.model')}</dt>
                  <dd className="mt-1 text-slate-900 dark:text-slate-100">{selectedModel}</dd>
                </div>
                <div>
                  <dt className="text-slate-400 dark:text-slate-500">{t('deploy.runtime')}</dt>
                  <dd className="mt-1 text-slate-900 dark:text-slate-100">{selectedRuntime}:{selectedRuntimeVersion}</dd>
                </div>
                <div>
                  <dt className="text-slate-400 dark:text-slate-500">{'\u58f0\u660e\u6587\u4ef6'}</dt>
                  <dd className="mt-1 text-slate-900 dark:text-slate-100">{selectedAgent?.path}</dd>
                </div>
                <div>
                  <dt className="text-slate-400 dark:text-slate-500">{t('deploy.skill')}</dt>
                  <dd className="mt-1 text-slate-900 dark:text-slate-100">{selectedSkills.join('、') || t('common.not_selected')}</dd>
                </div>
                <div>
                  <dt className="text-slate-400 dark:text-slate-500">{t('deploy.mcp')}</dt>
                  <dd className="mt-1 text-slate-900 dark:text-slate-100">{selectedMcps.join('、') || t('common.not_selected')}</dd>
                </div>
                <div className="md:col-span-2">
                  <dt className="text-slate-400 dark:text-slate-500">{t('deploy.auth_token')}</dt>
                  <dd className="mt-1 text-slate-900 dark:text-slate-100">{selectedTokenNames.join('、') || t('common.not_selected')}</dd>
                </div>
              </dl>
              {deployResult && (
                <div className="mt-5 rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-800">
                  {'\u90e8\u7f72\u5df2\u63d0\u4ea4\uff1a'}{deployResult.status}{'\uff0c\u955c\u50cf '}{deployResult.imageTag}
                </div>
              )}
              {deployError && (
                <div className="mt-5 rounded-xl border border-rose-200 bg-rose-50 p-4 text-sm text-rose-700">
                  {deployError}
                </div>
              )}
            </div>
          )}
        </div>

        <div className="flex items-center justify-between border-t border-slate-200 dark:border-slate-700 px-6 py-4">
          <button
            onClick={() => setStep((current) => Math.max(current - 1, 0))}
            disabled={step === 0}
            className="rounded-xl bg-slate-100 dark:bg-slate-700 px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300 dark:text-slate-300 transition hover:bg-slate-200 disabled:cursor-not-allowed disabled:opacity-40"
          >
            {t('deploy.prev_step', '\u4e0a\u4e00\u6b65')}
          </button>
          {step < stepLabels.length - 1 ? (
            <button
              onClick={() => setStep((current) => Math.min(current + 1, stepLabels.length - 1))}
              disabled={step === 2 && agents.length === 0}
              className="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-40"
            >
              {t('deploy.next_step', '\u4e0b\u4e00\u6b65')}
            </button>
          ) : (
            <button
              onClick={submitDeployment}
              disabled={deploying}
              className="rounded-xl bg-emerald-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {deploying ? t('deploy.deploying') : t('deploy.deploy_button')}
            </button>
          )}
        </div>
      </div>

      <CapabilityPickerModal
        open={picker === 'api'}
        title={t('deploy.picker_api_title')}
        mode="single"
        items={pickerItems.api}
        selected={selectedApiTokenId ? [selectedApiTokenId] : []}
        onClose={() => setPicker(null)}
        onSelectOne={(id) => {
          updateForm('apiTokenId', id)
          setPicker(null)
        }}
      />
      <CapabilityPickerModal
        open={picker === 'skills'}
        title={t('deploy.picker_skill_title')}
        mode="multi"
        items={pickerItems.skills}
        selected={selectedSkills}
        onClose={() => setPicker(null)}
        onToggle={(id) => toggleFromValues('skills', id, selectedSkills)}
        onSelectAll={() => setCapabilityValues('skills', pickerItems.skills.map((item) => item.id))}
        onClear={() => setCapabilityValues('skills', [])}
      />
      <CapabilityPickerModal
        open={picker === 'mcps'}
        title={t('deploy.picker_mcp_title')}
        mode="multi"
        items={pickerItems.mcps}
        selected={selectedMcps}
        onClose={() => setPicker(null)}
        onToggle={(id) => toggleFromValues('mcps', id, selectedMcps)}
        onSelectAll={() => setCapabilityValues('mcps', pickerItems.mcps.map((item) => item.id))}
        onClear={() => setCapabilityValues('mcps', [])}
      />
      <CapabilityPickerModal
        open={picker === 'auth'}
        title={t('deploy.picker_auth_title')}
        mode="multi"
        items={pickerItems.auth}
        selected={form.authTokens}
        onClose={() => setPicker(null)}
        onToggle={(id) => toggleValue('authTokens', id)}
        onSelectAll={() => updateForm('authTokens', pickerItems.auth.map((item) => item.id))}
        onClear={() => updateForm('authTokens', [])}
      />

      {deployments.filter((d) => d.status === 'running').length > 0 && (
        <div className="mt-8 rounded-2xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 shadow-sm">
          <div className="border-b border-slate-200 dark:border-slate-700 px-6 py-4">
            <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">{t('deploy.running_deployments')}</h2>
          </div>
          <div className="divide-y divide-slate-100 dark:divide-slate-700">
            {deployments.filter((d) => d.status === 'running').map((d) => (
              <div key={d.id} className="flex items-center justify-between px-6 py-4">
                <div>
                  <div className="flex items-center gap-2">
                    {(() => { const h = healthStatus[d.id]; const healthy = !h || h.ok; return <span className={`h-2.5 w-2.5 rounded-full ${healthy ? 'bg-emerald-400' : 'bg-red-400 animate-pulse'}`} title={healthy ? 'Healthy' : (h?.error || 'Unreachable')} /> })()}
                    <span className="text-sm font-medium text-slate-900 dark:text-slate-100">{agentNameMap[d.agentId] || d.agentId}</span>
                  </div>
                  <div className="mt-0.5 text-xs text-slate-400 dark:text-slate-500">
                    {[d.model, tokenNameMap[d.apiTokenId], d.runtime, ...(d.skills || [])].filter(Boolean).join(' · ')}
                  </div>
                  {agentDescMap[d.agentId] && (
                    <div className="mt-0.5 max-w-xs truncate text-xs text-slate-400 dark:text-slate-500">
                      {agentDescMap[d.agentId]}
                    </div>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  <button onClick={async () => { await stopDeployment(d.id); setDeployments((c) => c.map((x) => x.id === d.id ? { ...x, status: 'stopped' } : x)) }} className="rounded-lg border border-slate-200 dark:border-slate-700 px-3 py-1 text-xs text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 dark:hover:bg-slate-700 dark:bg-slate-900">{t('deploy.stop')}</button>
                  <button onClick={() => setConfirm({ id: d.id, title: t('common.confirm_delete_title'), message: t('common.confirm_delete_msg'), action: async () => { await deleteDeployment(d.id); setDeployments((c) => c.filter((x) => x.id !== d.id)); setConfirm(null) } })} className="rounded-lg px-3 py-1 text-xs text-rose-600 hover:bg-rose-50">{t('deploy.delete_deployment')}</button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
      <ConfirmDialog open={!!confirm} title={confirm?.title || ''} message={confirm?.message || ''} onConfirm={confirm?.action} onCancel={() => setConfirm(null)} />
    </div>
  )
}
