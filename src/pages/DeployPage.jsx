import { useMemo, useState } from 'react'
import { useEffect } from 'react'
import { createDeployment, getDeployOptions, getDeployments, stopDeployment, startDeployment, deleteDeployment } from '../api'
import LoadingPanel from '../components/LoadingPanel'
import PageHeader from '../components/PageHeader'
import useAsyncData from '../hooks/useAsyncData'

const steps = ['选择仓库', '选择 Commit', '选择 Agent', '配置能力', '确认']

export default function DeployPage() {
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
  const { data, loading } = useAsyncData(getDeployOptions, [])

  useEffect(() => {
    getDeployments().then(setDeployments).catch(() => {})
  }, [deployResult])

  const repositories = data?.repositories ?? []
  const selectedRepository = repositories.find((repo) => repo.id === form.repositoryId) ?? repositories[0]
  const commits = selectedRepository?.commits ?? []
  const selectedCommit = commits.find((commit) => commit.hash === form.commitHash) ?? commits[0]
  const agents = selectedCommit?.agents ?? []
  const selectedAgent = agents.find((agent) => agent.id === form.agentId) ?? agents[0]
  const selectedSkills = form.skills.length > 0 ? form.skills : selectedAgent?.skills ?? []
  const selectedMcps = form.mcps.length > 0 ? form.mcps : selectedAgent?.mcps ?? []
  const selectedApiTokenId = form.apiTokenId || data?.aiTokens?.[0]?.id || ''
  const selectedModel = form.model || selectedAgent?.model || data?.models?.[0] || ''
  const selectedRuntime = form.runtime || data?.runtimes?.[0] || ''
  const selectedRuntimeVersion = form.runtimeVersion || data?.runtimeTags?.[0] || 'latest'
  const selectedApiToken = data?.aiTokens?.find((token) => token.id === selectedApiTokenId)

  const selectedTokenNames = useMemo(() => {
    if (!data) return []
    return data.authTokens.filter((token) => form.authTokens.includes(token.id)).map((token) => token.name)
  }, [data, form.authTokens])

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
  }

  const selectAgent = (agentId) => {
    setForm((current) => ({ ...current, agentId, model: '', skills: [], mcps: [] }))
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
    setForm((current) => ({
      ...current,
      [key]: values.includes(value) ? values.filter((item) => item !== value) : [...values, value],
    }))
  }

  const submitDeployment = async () => {
    setDeploying(true)
    setDeployError('')
    setDeployResult(null)
    try {
      const result = await createDeployment({
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
      })
      setDeployResult(result)
    } catch (error) {
      setDeployError(error.message)
    } finally {
      setDeploying(false)
    }
  }

  if (loading || !data) {
    return <LoadingPanel label="正在加载部署配置..." />
  }

  return (
    <div>
      <PageHeader title="部署 Agent" description="选择已绑定仓库、指定 commit hash，再部署这个 commit 下声明的 Agent。" />

      <div className="rounded-2xl border border-slate-200 bg-white shadow-sm">
        <div className="border-b border-slate-200 px-6 py-5">
          <div className="flex items-center gap-3">
            {steps.map((label, index) => (
              <div key={label} className="flex flex-1 items-center gap-3">
                <button
                  onClick={() => setStep(index)}
                  className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-sm font-semibold transition ${
                    index <= step ? 'bg-sky-600 text-white' : 'bg-slate-100 text-slate-400'
                  }`}
                >
                  {index + 1}
                </button>
                <div className="hidden text-sm font-medium text-slate-700 md:block">{label}</div>
                {index < steps.length - 1 && <div className="h-px flex-1 bg-slate-200" />}
              </div>
            ))}
          </div>
        </div>

        <div className="min-h-[460px] p-6">
          {step === 0 && (
            <div>
              <div className="mb-4 text-sm font-medium text-slate-950">从仓库管理中选择已绑定仓库</div>
              <div className="grid gap-4 xl:grid-cols-2">
                {repositories.map((repo) => (
                  <button
                    key={repo.id}
                    onClick={() => selectRepository(repo.id)}
                    className={`rounded-xl border p-5 text-left transition ${
                      selectedRepository?.id === repo.id ? 'border-sky-200 bg-sky-50' : 'border-slate-200 hover:bg-slate-50'
                    }`}
                  >
                    <div className="flex items-center justify-between gap-3">
                      <div className="font-medium text-slate-950">{repo.url.replace('https://', '')}</div>
                      <span className="rounded-full bg-white px-2.5 py-1 text-xs text-slate-500">{repo.provider}</span>
                    </div>
                    <div className="mt-3 grid gap-2 text-xs text-slate-500 sm:grid-cols-3">
                      <div>默认分支：{repo.branch}</div>
                      <div>Agent 目录：{repo.agentsPath}</div>
                      <div>Commits：{repo.commits.length}</div>
                    </div>
                  </button>
                ))}
              </div>
              {repositories.length === 0 && (
                <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50 p-6 text-sm text-slate-500">
                  还没有绑定仓库，请先到仓库管理中绑定 GitHub/GitLab 仓库。
                </div>
              )}
            </div>
          )}

          {step === 1 && (
            <div>
              <div className="mb-4 text-sm font-medium text-slate-950">选择 commit hash</div>
              <div className="space-y-3">
                {commits.map((commit) => (
                  <button
                    key={commit.hash}
                    onClick={() =>
                      setForm((current) => ({
                        ...current,
                        commitHash: commit.hash,
                        agentId: commit.agents[0]?.id ?? '',
                        model: '',
                        skills: [],
                        mcps: [],
                      }))
                    }
                    className={`w-full rounded-xl border p-4 text-left transition ${
                      selectedCommit?.hash === commit.hash ? 'border-sky-200 bg-sky-50' : 'border-slate-200 hover:bg-slate-50'
                    }`}
                  >
                    <div className="flex items-center justify-between gap-3">
                      <div className="font-mono text-sm font-semibold text-slate-950">{commit.hash}</div>
                      <div className="text-xs text-slate-400">{commit.committedAt}</div>
                    </div>
                    <div className="mt-2 text-sm text-slate-600">{commit.message}</div>
                    <div className="mt-2 text-xs text-slate-400">发现 {commit.agents.length} 个 Agent</div>
                  </button>
                ))}
              </div>
            </div>
          )}

          {step === 2 && (
            <div>
              <div className="mb-4 text-sm font-medium text-slate-950">仓库内发现的 Agent</div>
              <div className="grid gap-4 xl:grid-cols-2">
                {agents.map((agent) => (
                  <button
                    key={agent.id}
                    onClick={() => selectAgent(agent.id)}
                    className={`rounded-xl border p-5 text-left transition ${
                      selectedAgent?.id === agent.id ? 'border-sky-200 bg-sky-50' : 'border-slate-200 hover:bg-slate-50'
                    }`}
                  >
                    <div className="text-base font-semibold text-slate-950">{agent.name}</div>
                    <div className="mt-2 text-sm leading-6 text-slate-600">{agent.description}</div>
                    <div className="mt-4 text-xs text-slate-400">{agent.path}</div>
                  </button>
                ))}
              </div>
            </div>
          )}

          {step === 3 && selectedAgent && (
            <div className="grid gap-6 xl:grid-cols-2">
              <div>
                <div className="mb-3 text-sm font-medium text-slate-950">API Token</div>
                <div className="space-y-3">
                  {data.aiTokens.map((token) => (
                    <label key={token.id} className="flex items-center gap-3 rounded-xl border border-slate-200 p-4 text-sm text-slate-700">
                      <input
                        type="radio"
                        name="api-token"
                        checked={selectedApiTokenId === token.id}
                        onChange={() => updateForm('apiTokenId', token.id)}
                      />
                      <span>{token.name}</span>
                    </label>
                  ))}
                </div>
              </div>
              <div className="grid gap-5">
                <label className="block text-sm text-slate-700">
                  模型
                  <select
                    value={selectedModel}
                    onChange={(event) => updateForm('model', event.target.value)}
                    className="mt-2 w-full rounded-xl border border-slate-200 bg-white px-4 py-3 text-slate-900 outline-none focus:border-sky-500"
                  >
                    {data.models.map((model) => (
                      <option key={model}>{model}</option>
                    ))}
                  </select>
                </label>
                <label className="block text-sm text-slate-700">
                  Runtime
                  <select
                    value={selectedRuntime}
                    onChange={(event) => updateForm('runtime', event.target.value)}
                    className="mt-2 w-full rounded-xl border border-slate-200 bg-white px-4 py-3 text-slate-900 outline-none focus:border-sky-500"
                  >
                    {data.runtimes.map((runtime) => (
                      <option key={runtime}>{runtime}</option>
                    ))}
                  </select>
                </label>
                <label className="block text-sm text-slate-700">
                  Runtime 版本
                  <select
                    value={selectedRuntimeVersion}
                    onChange={(event) => updateForm('runtimeVersion', event.target.value)}
                    className="mt-2 w-full rounded-xl border border-slate-200 bg-white px-4 py-3 text-slate-900 outline-none focus:border-sky-500"
                  >
                    {(data.runtimeTags ?? ['latest']).map((version) => (
                      <option key={version}>{version}</option>
                    ))}
                  </select>
                </label>
              </div>
              <div>
                <div className="mb-3 text-sm font-medium text-slate-950">Agent 声明的 Skill</div>
                <div className="space-y-3">
                  {selectedAgent.skills.map((skill) => (
                    <label key={skill} className="flex items-center gap-3 rounded-xl border border-slate-200 p-4 text-sm text-slate-700">
                      <input type="checkbox" checked={selectedSkills.includes(skill)} onChange={() => toggleFromValues('skills', skill, selectedSkills)} />
                      <span>{skill}</span>
                    </label>
                  ))}
                </div>
              </div>
              <div>
                <div className="mb-3 text-sm font-medium text-slate-950">Agent 声明的 MCP</div>
                <div className="space-y-3">
                  {selectedAgent.mcps.map((mcp) => (
                    <label key={mcp} className="flex items-center gap-3 rounded-xl border border-slate-200 p-4 text-sm text-slate-700">
                      <input type="checkbox" checked={selectedMcps.includes(mcp)} onChange={() => toggleFromValues('mcps', mcp, selectedMcps)} />
                      <span>{data.mcpServers.find((server) => server.id === mcp)?.name ?? mcp}</span>
                    </label>
                  ))}
                </div>
              </div>
              <div>
                <div className="mb-3 text-sm font-medium text-slate-950">鉴权 Token</div>
                <div className="space-y-3">
                  {data.authTokens.map((token) => (
                    <label key={token.id} className="flex items-center gap-3 rounded-xl border border-slate-200 p-4 text-sm text-slate-700">
                      <input type="checkbox" checked={form.authTokens.includes(token.id)} onChange={() => toggleValue('authTokens', token.id)} />
                      <span>{token.name}</span>
                    </label>
                  ))}
                </div>
              </div>
            </div>
          )}

          {step === 4 && (
            <div className="max-w-4xl rounded-2xl border border-slate-200 bg-slate-50 p-5">
              <div className="mb-4 text-sm font-semibold text-slate-950">部署确认</div>
              <dl className="grid gap-4 text-sm md:grid-cols-2">
                <div>
                  <dt className="text-slate-400">仓库</dt>
                  <dd className="mt-1 text-slate-900">{selectedRepository?.url}</dd>
                </div>
                <div>
                  <dt className="text-slate-400">分支</dt>
                  <dd className="mt-1 text-slate-900">{selectedRepository?.branch}</dd>
                </div>
                <div>
                  <dt className="text-slate-400">Commit</dt>
                  <dd className="mt-1 font-mono text-slate-900">{selectedCommit?.hash}</dd>
                </div>
                <div>
                  <dt className="text-slate-400">Agent</dt>
                  <dd className="mt-1 text-slate-900">{selectedAgent?.name}</dd>
                </div>
                <div>
                  <dt className="text-slate-400">API Token</dt>
                  <dd className="mt-1 text-slate-900">{selectedApiToken?.name}</dd>
                </div>
                <div>
                  <dt className="text-slate-400">模型</dt>
                  <dd className="mt-1 text-slate-900">{selectedModel}</dd>
                </div>
                <div>
                  <dt className="text-slate-400">Runtime</dt>
                  <dd className="mt-1 text-slate-900">{selectedRuntime}:{selectedRuntimeVersion}</dd>
                </div>
                <div>
                  <dt className="text-slate-400">声明文件</dt>
                  <dd className="mt-1 text-slate-900">{selectedAgent?.path}</dd>
                </div>
                <div>
                  <dt className="text-slate-400">Skill</dt>
                  <dd className="mt-1 text-slate-900">{selectedSkills.join('、') || '未选择'}</dd>
                </div>
                <div>
                  <dt className="text-slate-400">MCP</dt>
                  <dd className="mt-1 text-slate-900">{selectedMcps.join('、') || '未选择'}</dd>
                </div>
                <div className="md:col-span-2">
                  <dt className="text-slate-400">鉴权 Token</dt>
                  <dd className="mt-1 text-slate-900">{selectedTokenNames.join('、') || '未选择'}</dd>
                </div>
              </dl>
              {deployResult && (
                <div className="mt-5 rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-800">
                  部署已提交：{deployResult.status}，镜像 {deployResult.imageTag}
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

        <div className="flex items-center justify-between border-t border-slate-200 px-6 py-4">
          <button
            onClick={() => setStep((current) => Math.max(current - 1, 0))}
            disabled={step === 0}
            className="rounded-xl bg-slate-100 px-4 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-200 disabled:cursor-not-allowed disabled:opacity-40"
          >
            上一步
          </button>
          {step < steps.length - 1 ? (
            <button
              onClick={() => setStep((current) => Math.min(current + 1, steps.length - 1))}
              className="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-sky-700"
            >
              下一步
            </button>
          ) : (
            <button
              onClick={submitDeployment}
              disabled={deploying}
              className="rounded-xl bg-emerald-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {deploying ? '部署中...' : '提交部署'}
            </button>
          )}
        </div>
      </div>

      {deployments.length > 0 && (
        <div className="mt-8 rounded-2xl border border-slate-200 bg-white shadow-sm">
          <div className="border-b border-slate-200 px-6 py-4">
            <h2 className="text-lg font-semibold text-slate-900">运行中的部署</h2>
          </div>
          <div className="divide-y divide-slate-100">
            {deployments.map((d) => (
              <div key={d.id} className="flex items-center justify-between px-6 py-4">
                <div>
                  <div className="text-sm font-medium text-slate-900">{d.agentId}</div>
                  <div className="mt-0.5 text-xs text-slate-400">{d.runtime} · {d.sidecarUrl}</div>
                </div>
                <div className="flex items-center gap-2">
                  <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${d.status === 'running' ? 'bg-emerald-50 text-emerald-700' : d.status === 'stopped' ? 'bg-amber-50 text-amber-700' : 'bg-slate-100 text-slate-500'}`}>{d.status}</span>
                  {d.status === 'running' && (
                    <button onClick={async () => { await stopDeployment(d.id); setDeployments((c) => c.map((x) => x.id === d.id ? { ...x, status: 'stopped' } : x)) }} className="rounded-lg border border-slate-200 px-3 py-1 text-xs text-slate-600 hover:bg-slate-50">停止</button>
                  )}
                  {d.status === 'stopped' && (
                    <button onClick={async () => { await startDeployment(d.id); setDeployments((c) => c.map((x) => x.id === d.id ? { ...x, status: 'running' } : x)) }} className="rounded-lg bg-emerald-600 px-3 py-1 text-xs font-medium text-white hover:bg-emerald-700">启动</button>
                  )}
                  <button onClick={async () => { await deleteDeployment(d.id); setDeployments((c) => c.filter((x) => x.id !== d.id)) }} className="rounded-lg px-3 py-1 text-xs text-rose-600 hover:bg-rose-50">删除</button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
