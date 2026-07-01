import { useEffect, useMemo, useState } from 'react'
import { getAuthTokens, deleteAuthToken, patchAuthToken, createAuthToken } from '../api'
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
import { useT } from '../i18n'
import useAsyncData from '../hooks/useAsyncData'

export default function AuthTokensPage() {
  const t = useT()
  const [importOpen, setImportOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [functionFilter, setFunctionFilter] = useState('all')
  const [status, setStatus] = useState('all')
  const { data, loading } = useAsyncData(getAuthTokens, [])
  const [authTokens, setAuthTokens] = useState([])
  const [form, setForm] = useState({ name: '', accessTarget: '', script: '', functionName: 'get_token', argument: '' })
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

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
    return <LoadingPanel label={t('common.loading')} />
  }

  const updateTokenStatus = async (tokenId, nextStatus) => {
    await patchAuthToken(tokenId, { status: nextStatus })
    setAuthTokens((current) => current.map((token) => (token.id === tokenId ? { ...token, status: nextStatus } : token)))
  }

  const handleDelete = async (id) => {
    try {
      await deleteAuthToken(id)
      setAuthTokens((c) => c.filter((t) => t.id !== id))
    } catch (e) { alert(`${t('common.save_failed')}: ${e.message}`) }
  }

  const handleCreate = async () => {
    if (!form.name || !form.script) {
      setFormError(t('repositories.no_repos'))
      return
    }
    setSaving(true)
    setFormError('')
    try {
      const status = '启用'
      const created = await createAuthToken({ ...form, status })
      setAuthTokens((c) => [...c, created])
      setImportOpen(false)
      setForm({ name: '', accessTarget: '', script: '', functionName: 'get_token', argument: '' })
    } catch (e) {
      setFormError(e.message)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div>
      <PageHeader
        title={t('authTokens.title')}
        description={t('authTokens.import_hint')}
        action={<button onClick={() => setImportOpen(true)} className="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white">{t('authTokens.import_token')}</button>}
      />
      {importOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/30 px-4 py-8" onClick={() => setImportOpen(false)}>
          <div className="w-full max-w-lg overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl" onClick={(e) => e.stopPropagation()}>
            <div className="border-b border-slate-200 px-5 py-4">
              <div className="text-base font-semibold text-slate-950">{t('authTokens.create_form_title')}</div>
            </div>
            <div className="grid gap-4 p-5">
              <label className="block text-sm text-slate-700">
                {t('common.name')}
                <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="e.g. GitHub Token" className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 text-sm outline-none focus:border-sky-500" />
              </label>
              <label className="block text-sm text-slate-700">
                {t('authTokens.access_target')}
                <input value={form.accessTarget} onChange={(e) => setForm({ ...form, accessTarget: e.target.value })} placeholder="api.github.com" className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 text-sm outline-none focus:border-sky-500" />
              </label>
              <label className="block text-sm text-slate-700">
                {t('authTokens.function_name')}
                <input value={form.functionName} onChange={(e) => setForm({ ...form, functionName: e.target.value })} placeholder="get_token" className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 text-sm outline-none focus:border-sky-500" />
              </label>
              <label className="block text-sm text-slate-700">
                {t('authTokens.description_label', 'Script path')}
                <input value={form.script} onChange={(e) => setForm({ ...form, script: e.target.value })} placeholder="tokens/my_script.py" className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 text-sm outline-none focus:border-sky-500" />
              </label>
              <label className="block text-sm text-slate-700">
                {t('authTokens.description_label', 'Argument')}
                <input value={form.argument} onChange={(e) => setForm({ ...form, argument: e.target.value })} placeholder="repo" className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 text-sm outline-none focus:border-sky-500" />
              </label>
              {formError && <div className="text-sm text-rose-600">{formError}</div>}
            </div>
            <div className="flex justify-end gap-3 border-t border-slate-200 px-5 py-4">
              <button onClick={() => setImportOpen(false)} className="rounded-lg px-4 py-2 text-sm text-slate-500 hover:bg-slate-100">{t('common.cancel')}</button>
              <button onClick={handleCreate} disabled={saving} className="rounded-lg bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50">{saving ? t('common.loading') : t('common.save')}</button>
            </div>
          </div>
        </div>
      )}

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
                <HeaderFilter label={t('common.name')} active={query.trim() !== ''}>
                  <FilterInput value={query} onChange={setQuery} placeholder={t('authTokens.search_placeholder')} />
                </HeaderFilter>
              </th>
              <th className={tableHeaderCellClass}>
                <HeaderFilter label={t('authTokens.access_target')} active={functionFilter !== 'all'}>
                  <FilterSelect value={functionFilter} onChange={setFunctionFilter} options={functions} />
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
