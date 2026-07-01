import { useEffect, useMemo, useState } from 'react'
import { getAiTokens, deleteAIToken, patchAIToken, createAiToken } from '../api'
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
import { useT } from '../i18n'

export default function AiTokensPage() {
  const [tokens, setTokens] = useState([])
  const [loading, setLoading] = useState(true)
  const [query, setQuery] = useState('')
  const [provider, setProvider] = useState('all')
  const [status, setStatus] = useState('all')
  const [formOpen, setFormOpen] = useState(false)
  const [form, setForm] = useState({ name: '', provider: 'DeepSeek', apiKey: '', baseUrl: '', model: '', scope: '' })

  const t = useT()

  useEffect(() => {
    getAiTokens()
      .then(setTokens)
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const providers = useMemo(() => [...new Set(tokens.map((t) => t.provider))], [tokens])
  const statuses = useMemo(() => [...new Set(tokens.map((t) => t.status))], [tokens])
  const filteredTokens = useMemo(() => {
    return tokens.filter((token) => {
      const text = [token.name, token.provider, token.scope, token.usage, token.status].join(' ').toLowerCase()
      return (
        (query.trim() === '' || text.includes(query.toLowerCase())) &&
        (provider === 'all' || token.provider === provider) &&
        (status === 'all' || token.status === status)
      )
    })
  }, [tokens, provider, query, status])

  if (loading) {
    return <LoadingPanel label={t('common.loading')} />
  }

  const updateTokenStatus = async (tokenId, nextStatus) => {
    await patchAIToken(tokenId, { status: nextStatus })
    setTokens((current) => current.map((t) => (t.id === tokenId ? { ...t, status: nextStatus } : t)))
  }

  const handleDelete = async (id) => {
    try {
      await deleteAIToken(id)
      setTokens((c) => c.filter((t) => t.id !== id))
    } catch (e) { alert(`${t('common.save_failed')}: ${e.message}`) }
  }

  const handleSubmit = async () => {
    if (!form.name || !form.apiKey) return
    try {
      const created = await createAiToken(form)
      setTokens((c) => [...c, created])
      setFormOpen(false)
      setForm({ name: '', provider: 'DeepSeek', apiKey: '', baseUrl: '', model: '', scope: '' })
    } catch (e) {
      alert(`${t('common.save_failed')}: ${e.message}`)
    }
  }

  return (
    <div>
      <PageHeader
        title={t('aITokens.title')}
        description={t('aITokens.import_hint')}
        action={<button onClick={() => setFormOpen(true)} className="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white">{t('aITokens.create_token')}</button>}
      />

      {formOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/30 px-4 py-8">
          <div className="w-full max-w-lg overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl">
            <div className="border-b border-slate-200 px-5 py-4">
              <div className="text-base font-semibold text-slate-950">{t('aITokens.create_form_title')}</div>
            </div>
            <div className="grid gap-4 p-5">
              <label className="block text-sm text-slate-700">
                {t('aITokens.token_name')}
                <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 outline-none focus:border-sky-500" placeholder="e.g. DeepSeek V4" />
              </label>
              <label className="block text-sm text-slate-700">
                {t('aITokens.provider_label')}
                <select value={form.provider} onChange={(e) => setForm({ ...form, provider: e.target.value })} className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 outline-none focus:border-sky-500">
                  <option>DeepSeek</option><option>OpenAI</option><option>Anthropic</option><option>GLM</option><option>Kimi</option><option>MiniMax</option>
                </select>
              </label>
              <label className="block text-sm text-slate-700">
                {t('aITokens.api_key')}
                <input type="password" value={form.apiKey} onChange={(e) => setForm({ ...form, apiKey: e.target.value })} className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 outline-none focus:border-sky-500" />
              </label>
              <label className="block text-sm text-slate-700">
                {t('aITokens.base_url')}
                <input value={form.baseUrl} onChange={(e) => setForm({ ...form, baseUrl: e.target.value })} className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 outline-none focus:border-sky-500" placeholder="https://api.deepseek.com" />
              </label>
              <label className="block text-sm text-slate-700">
                {t('aITokens.model')}
                <input value={form.model} onChange={(e) => setForm({ ...form, model: e.target.value })} className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 outline-none focus:border-sky-500" placeholder="deepseek-v4-pro" />
              </label>
              <label className="block text-sm text-slate-700">
                {t('aITokens.scope')}
                <input value={form.scope} onChange={(e) => setForm({ ...form, scope: e.target.value })} className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 outline-none focus:border-sky-500" placeholder={t('aITokens.scope')} />
              </label>
            </div>
            <div className="flex justify-end gap-3 border-t border-slate-200 px-5 py-4">
              <button onClick={() => setFormOpen(false)} className="rounded-lg px-4 py-2 text-sm text-slate-500 hover:bg-slate-100">{t('common.cancel')}</button>
              <button onClick={handleSubmit} className="rounded-lg bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700">{t('common.save')}</button>
            </div>
          </div>
        </div>
      )}

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
                <HeaderFilter label={t('common.name')} active={query.trim() !== ''}>
                  <FilterInput value={query} onChange={setQuery} placeholder={t('aITokens.search_placeholder')} />
                </HeaderFilter>
              </th>
              <th className={tableHeaderCellClass}>
                <HeaderFilter label={t('aITokens.provider_label')} active={provider !== 'all'}>
                  <FilterSelect value={provider} onChange={setProvider} options={providers} />
                </HeaderFilter>
              </th>
              <th className={tableHeaderCellClass}>
                <HeaderFilter label={t('common.status')} active={status !== 'all'}>
                  <FilterSelect value={status} onChange={setStatus} options={statuses} />
                </HeaderFilter>
              </th>
              <th className={`${tableHeaderCellClass} text-right`}>{t('common.actions')}</th>
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
                          <div className="mt-1 text-slate-500">{t('aITokens.import_hint')}</div>
                        </div>
                        <div className="grid grid-cols-[72px_1fr] gap-y-2">
                          <span className="text-slate-400">{t('aITokens.provider_label')}</span>
                          <span>{token.provider}</span>
                          <span className="text-slate-400">{t('aITokens.scope')}</span>
                          <span>{token.scope}</span>
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
