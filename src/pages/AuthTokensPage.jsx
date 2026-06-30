import { useEffect, useMemo, useState } from 'react'
import { getAuthTokens } from '../api'
import LoadingPanel from '../components/LoadingPanel'
import {
  FilterInput,
  FilterSelect,
  HeaderFilter,
  HoverCard,
  ManagementTable,
  RowActions,
  StatusBadge,
  tableBodyClass,
  tableCellClass,
  tableHeadClass,
  tableHeaderCellClass,
} from '../components/ManagementTable'
import PageHeader from '../components/PageHeader'
import WizardModal from '../components/WizardModal'
import useAsyncData from '../hooks/useAsyncData'

export default function AuthTokensPage() {
  const [importOpen, setImportOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [functionFilter, setFunctionFilter] = useState('all')
  const [status, setStatus] = useState('all')
  const { data, loading } = useAsyncData(getAuthTokens, [])
  const [authTokens, setAuthTokens] = useState([])

  useEffect(() => {
    if (data) {
      setAuthTokens(data)
    }
  }, [data])

  const functions = useMemo(() => [...new Set(authTokens.map((token) => token.functionName))], [authTokens])
  const statuses = useMemo(() => [...new Set(authTokens.map((token) => token.status))], [authTokens])
  const filteredTokens = useMemo(() => {
    return authTokens.filter((token) => {
      const text = [token.name, token.accessTarget, token.script, token.functionName, token.argument, token.status, token.updatedAt].join(' ').toLowerCase()
      const hitQuery = query.trim() === '' || text.includes(query.toLowerCase())
      const hitFunction = functionFilter === 'all' || token.functionName === functionFilter
      const hitStatus = status === 'all' || token.status === status
      return hitQuery && hitFunction && hitStatus
    })
  }, [authTokens, functionFilter, query, status])

  if (loading) {
    return <LoadingPanel label="正在加载鉴权 Token..." />
  }

  const updateTokenStatus = (tokenId, nextStatus) => {
    setAuthTokens((current) => current.map((token) => (token.id === tokenId ? { ...token, status: nextStatus } : token)))
  }

  return (
    <div>
      <PageHeader
        title="鉴权 Token"
        description="通过 Python 脚本向已授权 Agent 暴露 get_token(param)，让 skill 按需获取外部系统访问 Token。"
        action={<button onClick={() => setImportOpen(true)} className="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white">导入 Python 脚本</button>}
      />
      <WizardModal
        open={importOpen}
        title="导入 Python 脚本"
        onClose={() => setImportOpen(false)}
        steps={[
          {
            label: '上传脚本',
            content: (
              <div className="grid gap-4 text-sm">
                <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50 px-4 py-10 text-center text-slate-500">选择包含 get_token(param) 的 Python 脚本</div>
                <label className="text-slate-700">函数名<input defaultValue="get_token" className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 outline-none focus:border-sky-500" /></label>
              </div>
            ),
          },
          {
            label: '访问资源',
            content: <label className="block text-sm text-slate-700">这个 Token 可以访问什么<input placeholder="例如：Salesforce CRM 客户数据" className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 outline-none placeholder:text-slate-400 focus:border-sky-500" /></label>,
          },
          {
            label: '参数',
            content: <label className="block text-sm text-slate-700">函数参数<input placeholder="例如：project_key" className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 outline-none placeholder:text-slate-400 focus:border-sky-500" /></label>,
          },
        ]}
      />

      <ManagementTable>
          <colgroup>
            <col className="w-[30%]" />
            <col className="w-[34%]" />
            <col className="w-[14%]" />
            <col className="w-[22%]" />
          </colgroup>
          <thead className={tableHeadClass}>
            <tr>
              <th className={tableHeaderCellClass}>
                <HeaderFilter label="Token" active={query.trim() !== ''}>
                  <FilterInput value={query} onChange={setQuery} placeholder="搜索 Token" />
                </HeaderFilter>
              </th>
              <th className={tableHeaderCellClass}>
                <HeaderFilter label="访问与脚本" active={functionFilter !== 'all'}>
                  <FilterSelect value={functionFilter} onChange={setFunctionFilter} options={functions} />
                </HeaderFilter>
              </th>
              <th className={tableHeaderCellClass}>
                <HeaderFilter label="状态" active={status !== 'all'}>
                  <FilterSelect value={status} onChange={setStatus} options={statuses} />
                </HeaderFilter>
              </th>
              <th className={`${tableHeaderCellClass} text-right`}>操作</th>
            </tr>
          </thead>
          <tbody className={tableBodyClass}>
            {filteredTokens.map((token) => (
              <tr key={token.id}>
                <td className={`${tableCellClass} font-medium text-slate-950`}>
                  <HoverCard
                    content={
                      <div className="space-y-3">
                        <div>
                          <div className="font-semibold text-slate-950">{token.name}</div>
                          <div className="mt-1 leading-5 text-slate-500">{token.accessTarget}</div>
                        </div>
                        <div className="grid grid-cols-[72px_1fr] gap-y-2">
                          <span className="text-slate-400">脚本</span>
                          <span className="font-mono">{token.script}</span>
                          <span className="text-slate-400">函数</span>
                          <span className="font-mono">{token.functionName}({token.argument})</span>
                        </div>
                      </div>
                    }
                  >
                    <span className="cursor-default underline decoration-slate-300 decoration-dotted underline-offset-4">
                      {token.name}
                    </span>
                  </HoverCard>
                </td>
                <td className={tableCellClass}>
                  <div className="max-w-sm truncate text-slate-700">{token.accessTarget}</div>
                  <div className="font-mono text-xs text-slate-900">{token.script}</div>
                  <div className="mt-1 font-mono text-xs text-slate-500">
                    {token.functionName}({token.argument})
                  </div>
                </td>
                <td className={tableCellClass}>
                  <StatusBadge status={token.status} />
                </td>
                <td className={tableCellClass}>
                  <RowActions
                    status={token.status}
                    onEnable={() => updateTokenStatus(token.id, '启用')}
                    onDisable={() => updateTokenStatus(token.id, '停用')}
                    onDelete={() => setAuthTokens((current) => current.filter((item) => item.id !== token.id))}
                  />
                </td>
              </tr>
            ))}
          </tbody>
      </ManagementTable>
    </div>
  )
}
