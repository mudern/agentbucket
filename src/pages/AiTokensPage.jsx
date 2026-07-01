import { useEffect, useMemo, useState } from 'react'
import { getAiTokens, deleteAIToken } from '../api'
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

export default function AiTokensPage() {
  const [createOpen, setCreateOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [provider, setProvider] = useState('all')
  const [status, setStatus] = useState('all')
  const { data, loading } = useAsyncData(getAiTokens, [])
  const [aiTokens, setAiTokens] = useState([])

  useEffect(() => {
    if (data) {
      setAiTokens(data)
    }
  }, [data])

  const providers = useMemo(() => [...new Set(aiTokens.map((token) => token.provider))], [aiTokens])
  const statuses = useMemo(() => [...new Set(aiTokens.map((token) => token.status))], [aiTokens])
  const filteredTokens = useMemo(() => {
    return aiTokens.filter((token) => {
      const text = [token.name, token.provider, token.scope, token.usage, token.status].join(' ').toLowerCase()
      return (
        (query.trim() === '' || text.includes(query.toLowerCase())) &&
        (provider === 'all' || token.provider === provider) &&
        (status === 'all' || token.status === status)
      )
    })
  }, [aiTokens, provider, query, status])

  if (loading) {
    return <LoadingPanel label="正在加载 AI Token..." />
  }

  const updateTokenStatus = (tokenId, nextStatus) => {
    setAiTokens((current) => current.map((token) => (token.id === tokenId ? { ...token, status: nextStatus } : token)))
  }

  const handleDelete = async (id) => {
    try {
      await deleteAIToken(id)
      setAiTokens((c) => c.filter((t) => t.id !== id))
    } catch (e) { alert(`删除失败: ${e.message}`) }
  }

  return (
    <div>
      <PageHeader
        title="AI Token"
        description="统一管理可复用的模型 Token，例如 DeepSeek、GLM 等，供 Agent 创建时直接绑定。"
        action={<button onClick={() => setCreateOpen(true)} className="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white">新建 AI Token</button>}
      />
      <WizardModal
        open={createOpen}
        title="新建 AI Token"
        onClose={() => setCreateOpen(false)}
        steps={[
          {
            label: '供应商',
            content: (
              <div className="grid gap-4 text-sm">
                <label className="text-slate-700">名称<input className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 outline-none focus:border-sky-500" /></label>
                <label className="text-slate-700">供应商<select className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 outline-none focus:border-sky-500"><option>OpenAI</option><option>DeepSeek</option><option>Google</option><option>GLM</option></select></label>
              </div>
            ),
          },
          {
            label: 'Token',
            content: <label className="block text-sm text-slate-700">Token 值<input type="password" className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 outline-none focus:border-sky-500" /></label>,
          },
          {
            label: '范围',
            content: <label className="block text-sm text-slate-700">适用范围<input placeholder="例如：研发团队" className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 outline-none placeholder:text-slate-400 focus:border-sky-500" /></label>,
          },
        ]}
      />

      <ManagementTable>
          <colgroup>
            <col className="w-[42%]" />
            <col className="w-[20%]" />
            <col className="w-[14%]" />
            <col className="w-[24%]" />
          </colgroup>
          <thead className={tableHeadClass}>
            <tr>
              <th className={tableHeaderCellClass}>
                <HeaderFilter label="Token" active={query.trim() !== ''}>
                  <FilterInput value={query} onChange={setQuery} placeholder="搜索 Token" />
                </HeaderFilter>
              </th>
              <th className={tableHeaderCellClass}>
                <HeaderFilter label="供应商" active={provider !== 'all'}>
                  <FilterSelect value={provider} onChange={setProvider} options={providers} />
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
                          <div className="mt-1 text-slate-500">供 Agent 选择模型服务时绑定使用。</div>
                        </div>
                        <div className="grid grid-cols-[72px_1fr] gap-y-2">
                          <span className="text-slate-400">供应商</span>
                          <span>{token.provider}</span>
                          <span className="text-slate-400">适用范围</span>
                          <span>{token.scope}</span>
                          <span className="text-slate-400">使用量</span>
                          <span>{token.usage}</span>
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
                  <SoftTag>{token.provider}</SoftTag>
                </td>
                <td className={tableCellClass}>
                  <StatusBadge status={token.status} />
                </td>
                <td className={tableCellClass}>
                  <RowActions
                    status={token.status}
                    onEnable={() => updateTokenStatus(token.id, '启用')}
                    onDisable={() => updateTokenStatus(token.id, '停用')}
                    onDelete={() => handleDelete(token.id)}
                  />
                </td>
              </tr>
            ))}
          </tbody>
      </ManagementTable>
    </div>
  )
}
