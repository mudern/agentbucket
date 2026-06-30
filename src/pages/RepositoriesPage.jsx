import { useEffect, useMemo, useState } from 'react'
import { getDeployOptions } from '../api'
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
import WizardModal from '../components/WizardModal'
import useAsyncData from '../hooks/useAsyncData'

export default function RepositoriesPage() {
  const { data, loading } = useAsyncData(getDeployOptions, [])
  const [repositories, setRepositories] = useState([])
  const [bindOpen, setBindOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [platform, setPlatform] = useState('all')
  const [branch, setBranch] = useState('all')
  const [agentsPath, setAgentsPath] = useState('all')
  const [status, setStatus] = useState('all')

  useEffect(() => {
    if (data?.repositories) {
      setRepositories(data.repositories)
    }
  }, [data])

  const platforms = useMemo(() => [...new Set(repositories.map((repo) => repo.provider))], [repositories])
  const branches = useMemo(() => [...new Set(repositories.map((repo) => repo.branch))], [repositories])
  const agentPaths = useMemo(() => [...new Set(repositories.map((repo) => repo.agentsPath))], [repositories])
  const statuses = useMemo(() => [...new Set(repositories.map((repo) => repo.status))], [repositories])
  const filteredRepositories = useMemo(() => {
    return repositories.filter((repo) => {
      const latestCommit = repo.commits[0]
      const count = String(latestCommit?.agents.length ?? 0)
      const text = [repo.url, repo.provider, repo.branch, repo.agentsPath, repo.status, latestCommit?.hash, latestCommit?.message, count].join(' ').toLowerCase()
      return (
        (query.trim() === '' || text.includes(query.toLowerCase())) &&
        (platform === 'all' || repo.provider === platform) &&
        (branch === 'all' || repo.branch === branch) &&
        (agentsPath === 'all' || repo.agentsPath === agentsPath) &&
        (status === 'all' || repo.status === status)
      )
    })
  }, [agentsPath, branch, platform, query, repositories, status])

  if (loading) {
    return <LoadingPanel label="正在加载仓库列表..." />
  }

  const updateRepositoryStatus = (repoId, status) => {
    setRepositories((current) => current.map((repo) => (repo.id === repoId ? { ...repo, status } : repo)))
  }

  return (
    <div>
      <PageHeader
        title="仓库管理"
        description="绑定 GitHub/GitLab 仓库，部署时可选择某个仓库、某个 commit hash 下声明的 Agent。"
        action={<button onClick={() => setBindOpen(true)} className="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white">绑定仓库</button>}
      />
      <WizardModal
        open={bindOpen}
        title="绑定仓库"
        onClose={() => setBindOpen(false)}
        steps={[
          {
            label: '平台',
            content: <label className="block text-sm text-slate-700">代码平台<select className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 outline-none focus:border-sky-500"><option>GitHub</option><option>GitLab</option></select></label>,
          },
          {
            label: '仓库',
            content: (
              <div className="grid gap-4 text-sm">
                <label className="text-slate-700">仓库地址<input placeholder="https://github.com/org/repo" className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 outline-none placeholder:text-slate-400 focus:border-sky-500" /></label>
                <label className="text-slate-700">默认分支<input placeholder="main" className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 outline-none placeholder:text-slate-400 focus:border-sky-500" /></label>
              </div>
            ),
          },
          {
            label: 'Agent 目录',
            content: <label className="block text-sm text-slate-700">Agent 声明目录<input placeholder="agents/" className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 outline-none placeholder:text-slate-400 focus:border-sky-500" /></label>,
          },
        ]}
      />

      <ManagementTable>
          <colgroup>
            <col className="w-[36%]" />
            <col className="w-[20%]" />
            <col className="w-[16%]" />
            <col className="w-[12%]" />
            <col className="w-[16%]" />
          </colgroup>
          <thead className={tableHeadClass}>
            <tr>
              <th className={tableHeaderCellClass}>
                <HeaderFilter label="仓库" active={query.trim() !== ''}>
                  <FilterInput value={query} onChange={setQuery} placeholder="搜索仓库" />
                </HeaderFilter>
              </th>
              <th className={tableHeaderCellClass}>
                <HeaderFilter label="信息" active={platform !== 'all' || branch !== 'all' || agentsPath !== 'all'}>
                  <div className="space-y-3">
                    <label className="block">
                      <span className="mb-1 block text-xs font-medium text-slate-500">平台</span>
                      <FilterSelect value={platform} onChange={setPlatform} options={platforms} />
                    </label>
                    <label className="block">
                      <span className="mb-1 block text-xs font-medium text-slate-500">分支</span>
                      <FilterSelect value={branch} onChange={setBranch} options={branches} />
                    </label>
                    <label className="block">
                      <span className="mb-1 block text-xs font-medium text-slate-500">Agent 目录</span>
                      <FilterSelect value={agentsPath} onChange={setAgentsPath} options={agentPaths} />
                    </label>
                  </div>
                </HeaderFilter>
              </th>
              <th className={tableHeaderCellClass}>最近 Commit</th>
              <th className={tableHeaderCellClass}>
                <HeaderFilter label="状态" active={status !== 'all'}>
                  <FilterSelect value={status} onChange={setStatus} options={statuses} />
                </HeaderFilter>
              </th>
              <th className={`${tableHeaderCellClass} text-right`}>操作</th>
            </tr>
          </thead>
          <tbody className={tableBodyClass}>
            {filteredRepositories.map((repo) => {
              const latestCommit = repo.commits[0]
              return (
                <tr key={repo.id}>
                  <td className={`${tableCellClass} font-medium text-slate-950`}>
                    <HoverCard
                      content={
                        <div className="space-y-3">
                          <div>
                            <div className="font-semibold text-slate-950">{repo.id}</div>
                            <div className="mt-1 break-all font-mono text-[11px] text-slate-500">{repo.url}</div>
                          </div>
                          <div className="grid grid-cols-[72px_1fr] gap-y-2">
                            <span className="text-slate-400">平台</span>
                            <span>{repo.provider}</span>
                            <span className="text-slate-400">默认分支</span>
                            <span className="font-mono">{repo.branch}</span>
                            <span className="text-slate-400">Agent 目录</span>
                            <span className="font-mono">{repo.agentsPath}</span>
                            <span className="text-slate-400">最近提交</span>
                            <span className="font-mono">{latestCommit.hash}</span>
                          </div>
                        </div>
                      }
                    >
                      <span className="cursor-default underline decoration-slate-300 decoration-dotted underline-offset-4">
                        {repo.url.replace(/^https?:\/\//, '')}
                      </span>
                    </HoverCard>
                  </td>
                  <td className={tableCellClass}>
                    <div className="flex max-w-full flex-wrap gap-1.5">
                      <SoftTag>{repo.provider}</SoftTag>
                      <SoftTag>{repo.branch}</SoftTag>
                      <SoftTag>{repo.agentsPath}</SoftTag>
                    </div>
                  </td>
                  <td className={tableCellClass}>
                    <div className="font-mono text-xs text-slate-900">{latestCommit.hash}</div>
                    <div className="mt-1 text-xs text-slate-400">{latestCommit.message}</div>
                  </td>
                  <td className={tableCellClass}>
                    <StatusBadge status={repo.status} />
                  </td>
                  <td className={tableCellClass}>
                    <RowActions
                      status={repo.status}
                      onEnable={() => updateRepositoryStatus(repo.id, '启用')}
                      onDisable={() => updateRepositoryStatus(repo.id, '停用')}
                      onDelete={() => setRepositories((current) => current.filter((item) => item.id !== repo.id))}
                    />
                  </td>
                </tr>
              )
            })}
          </tbody>
      </ManagementTable>
    </div>
  )
}
