import { useEffect, useMemo, useState } from 'react'
import { deleteRepository } from '../api'
import LoadingPanel from '../components/LoadingPanel'
import {
  FilterInput,
  FilterSelect,
  HeaderFilter,
  HoverCard,
  ManagementTable,
  RowActions,
  SoftTag,
  StatusBadge,
  tableBodyClass,
  tableCellClass,
  tableHeadClass,
  tableHeaderCellClass,
} from '../components/ManagementTable'
import PageHeader from '../components/PageHeader'
import useAsyncData from '../hooks/useAsyncData'

const REPOS_API = `${import.meta.env.VITE_API_BASE ?? 'http://127.0.0.1:8080'}/api/repositories`

export default function RepositoriesPage() {
  const [repositories, setRepositories] = useState([])
  const [loading, setLoading] = useState(true)
  const [bindOpen, setBindOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [platform, setPlatform] = useState('all')
  const [branch, setBranch] = useState('all')
  const [agentsPath, setAgentsPath] = useState('all')
  const [status, setStatus] = useState('all')
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')
  // Bind form
  const [bindUrl, setBindUrl] = useState('')
  const [bindBranch, setBindBranch] = useState('main')
  const [bindPath, setBindPath] = useState('agents')
  const [bindLocalPath, setBindLocalPath] = useState('')
  const [bindProvider, setBindProvider] = useState('GitHub')

  const fetchRepos = async () => {
    try {
      const resp = await fetch(REPOS_API, { headers: { 'Content-Type': 'application/json' } })
      if (!resp.ok) throw new Error('API error')
      const data = await resp.json()
      setRepositories(data)
    } catch (e) { /* ignore */ }
    finally { setLoading(false) }
  }

  useEffect(() => { fetchRepos() }, [])

  const platforms = useMemo(() => [...new Set(repositories.map((r) => r.provider))], [repositories])
  const branches = useMemo(() => [...new Set(repositories.map((r) => r.branch))], [repositories])
  const agentPaths = useMemo(() => [...new Set(repositories.map((r) => r.agentsPath))], [repositories])
  const statuses = useMemo(() => [...new Set(repositories.map((r) => r.status))], [repositories])

  const filtered = useMemo(() => repositories.filter((repo) => {
    const c = repo.commits[0]
    const text = [repo.url, repo.provider, repo.branch, repo.agentsPath, repo.status, c?.hash, c?.message, String(c?.agents?.length ?? 0)].join(' ').toLowerCase()
    return (!query || text.includes(query.toLowerCase())) && (platform === 'all' || repo.provider === platform) && (branch === 'all' || repo.branch === branch) && (agentsPath === 'all' || repo.agentsPath === agentsPath) && (status === 'all' || repo.status === status)
  }), [query, platform, branch, agentsPath, status, repositories])

  const handleDelete = async (id) => {
    try {
      await deleteRepository(id)
      setRepositories((c) => c.filter((r) => r.id !== id))
    } catch (e) {
      alert(`删除失败: ${e.message}`)
    }
  }

  const handleBind = async () => {
    if (!bindUrl || !bindLocalPath) {
      setFormError('仓库地址和本地路径为必填')
      return
    }
    setSaving(true)
    setFormError('')
    try {
      const id = bindUrl.replace(/^https?:\/\//, '').replace(/[\/.]/g, '-').slice(0, 40)
      const resp = await fetch(REPOS_API, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id, provider: bindProvider, url: bindUrl, branch: bindBranch, agentsPath: bindPath, localPath: bindLocalPath, status: '启用' }),
      })
      if (!resp.ok) throw new Error((await resp.json().catch(() => ({}))).error || 'HTTP error')
      setBindOpen(false)
      setBindUrl('')
      setBindLocalPath('')
      fetchRepos()
    } catch (e) {
      setFormError(e.message)
    } finally {
      setSaving(false)
    }
  }

  if (loading) { return <LoadingPanel label="正在加载仓库列表..." /> }

  return (
    <div>
      <PageHeader title="仓库管理" description="绑定 GitHub/GitLab 仓库，作为 Agent 部署源。" action={<button onClick={() => setBindOpen(true)} className="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white">绑定仓库</button>} />

      {bindOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={() => setBindOpen(false)}>
          <div className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl" onClick={(e) => e.stopPropagation()}>
            <h2 className="mb-4 text-lg font-semibold text-slate-900">绑定仓库</h2>
            <div className="space-y-4">
              <label className="block text-sm text-slate-700">
                平台
                <select value={bindProvider} onChange={(e) => setBindProvider(e.target.value)} className="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5 text-sm outline-none focus:border-sky-500">
                  <option>GitHub</option><option>GitLab</option><option>Local</option>
                </select>
              </label>
              <label className="block text-sm text-slate-700">
                仓库地址
                <input value={bindUrl} onChange={(e) => setBindUrl(e.target.value)} placeholder="https://github.com/org/repo" className="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5 text-sm outline-none placeholder:text-slate-400 focus:border-sky-500" />
              </label>
              <label className="block text-sm text-slate-700">
                本地路径
                <input value={bindLocalPath} onChange={(e) => setBindLocalPath(e.target.value)} placeholder="/path/to/cloned/repo" className="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5 text-sm outline-none placeholder:text-slate-400 focus:border-sky-500" />
              </label>
              <div className="grid grid-cols-2 gap-4">
                <label className="block text-sm text-slate-700">
                  分支
                  <input value={bindBranch} onChange={(e) => setBindBranch(e.target.value)} className="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5 text-sm outline-none focus:border-sky-500" />
                </label>
                <label className="block text-sm text-slate-700">
                  Agent 目录
                  <input value={bindPath} onChange={(e) => setBindPath(e.target.value)} className="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5 text-sm outline-none focus:border-sky-500" />
                </label>
              </div>
              {formError && <div className="text-sm text-rose-600">{formError}</div>}
              <div className="flex justify-end gap-3 pt-2">
                <button onClick={() => setBindOpen(false)} className="rounded-xl px-4 py-2 text-sm text-slate-600 hover:bg-slate-100">取消</button>
                <button onClick={handleBind} disabled={saving} className="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50">{saving ? '保存中...' : '绑定'}</button>
              </div>
            </div>
          </div>
        </div>
      )}

      <ManagementTable>
        <colgroup>
          <col className="w-[36%]" /><col className="w-[20%]" /><col className="w-[16%]" /><col className="w-[12%]" /><col className="w-[16%]" />
        </colgroup>
        <thead className={tableHeadClass}>
          <tr>
            <th className={tableHeaderCellClass}><HeaderFilter label="仓库" active={!!query}><FilterInput value={query} onChange={setQuery} placeholder="搜索仓库" /></HeaderFilter></th>
            <th className={tableHeaderCellClass}><HeaderFilter label="信息" active={platform !== 'all' || branch !== 'all' || agentsPath !== 'all'}><div className="space-y-3"><label className="block"><span className="mb-1 block text-xs font-medium text-slate-500">平台</span><FilterSelect value={platform} onChange={setPlatform} options={platforms} /></label><label className="block"><span className="mb-1 block text-xs font-medium text-slate-500">分支</span><FilterSelect value={branch} onChange={setBranch} options={branches} /></label><label className="block"><span className="mb-1 block text-xs font-medium text-slate-500">Agent 目录</span><FilterSelect value={agentsPath} onChange={setAgentsPath} options={agentPaths} /></label></div></HeaderFilter></th>
            <th className={tableHeaderCellClass}>最近 Commit</th>
            <th className={tableHeaderCellClass}><HeaderFilter label="状态" active={status !== 'all'}><FilterSelect value={status} onChange={setStatus} options={statuses} /></HeaderFilter></th>
            <th className={`${tableHeaderCellClass} text-right`}>操作</th>
          </tr>
        </thead>
        <tbody className={tableBodyClass}>
          {filtered.map((repo) => {
            const c = repo.commits[0]
            return (
              <tr key={repo.id}>
                <td className={`${tableCellClass} font-medium text-slate-950`}>
                  <HoverCard content={<div className="space-y-3"><div><div className="font-semibold text-slate-950">{repo.id}</div><div className="mt-1 break-all font-mono text-[11px] text-slate-500">{repo.url}</div></div><div className="grid grid-cols-[72px_1fr] gap-y-2"><span className="text-slate-400">平台</span><span>{repo.provider}</span><span className="text-slate-400">分支</span><span className="font-mono">{repo.branch}</span><span className="text-slate-400">Agent 目录</span><span className="font-mono">{repo.agentsPath}</span><span className="text-slate-400">最近提交</span><span className="font-mono">{c?.hash}</span></div></div>}>
                    <span className="cursor-default underline decoration-slate-300 decoration-dotted underline-offset-4">{repo.url.replace(/^https?:\/\//, '')}</span>
                  </HoverCard>
                </td>
                <td className={tableCellClass}><div className="flex max-w-full flex-wrap gap-1.5"><SoftTag>{repo.provider}</SoftTag><SoftTag>{repo.branch}</SoftTag><SoftTag>{repo.agentsPath}</SoftTag></div></td>
                <td className={tableCellClass}><div className="font-mono text-xs text-slate-900">{c?.hash}</div><div className="mt-1 text-xs text-slate-400">{c?.message}</div></td>
                <td className={tableCellClass}><StatusBadge status={repo.status} /></td>
                <td className={tableCellClass}><RowActions status={repo.status} onEnable={() => setRepositories((c) => c.map((r) => r.id === repo.id ? { ...r, status: '启用' } : r))} onDisable={() => setRepositories((c) => c.map((r) => r.id === repo.id ? { ...r, status: '停用' } : r))} onDelete={() => handleDelete(repo.id)} /></td>
              </tr>
            )
          })}
        </tbody>
      </ManagementTable>
    </div>
  )
}
