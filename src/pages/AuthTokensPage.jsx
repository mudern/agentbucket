import { useEffect, useMemo, useState } from 'react'
import { getAuthTokens, deleteAuthToken, patchAuthToken, createAuthToken } from '../api'
import LoadingPanel from '../components/LoadingPanel'
import {
  FilterInput,
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
  const { data, loading } = useAsyncData(getAuthTokens, [])
  const [authTokens, setAuthTokens] = useState([])
  const [form, setForm] = useState({ name: '', description: '', secret: '' })
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')
  const [showSecret, setShowSecret] = useState(false)

  useEffect(() => {
    if (data) setAuthTokens(data)
  }, [data])

  const filteredTokens = useMemo(() => {
    return authTokens.filter((token) => {
      const text = [token.name, token.description, token.status, token.updatedAt].join(' ').toLowerCase()
      return query.trim() === '' || text.includes(query.toLowerCase())
    })
  }, [authTokens, query])

  if (loading) return <LoadingPanel label={t('common.loading')} />

  const updateTokenStatus = async (tokenId, nextStatus) => {
    await patchAuthToken(tokenId, { status: nextStatus })
    setAuthTokens((current) => current.map((t) => t.id === tokenId ? { ...t, status: nextStatus } : t))
  }

  const handleDelete = async (id) => {
    try {
      await deleteAuthToken(id)
      setAuthTokens((c) => c.filter((t) => t.id !== id))
    } catch (e) { alert(`${t('common.save_failed')}: ${e.message}`) }
  }

  const handleCreate = async () => {
    if (!form.name || !form.secret) {
      setFormError(t('authTokens.fill_required'))
      return
    }
    setSaving(true)
    setFormError('')
    try {
      const created = await createAuthToken({ ...form, status: '启用' })
      setAuthTokens((c) => [...c, created])
      setImportOpen(false)
      setForm({ name: '', description: '', secret: '' })
      setShowSecret(false)
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
        description={t('authTokens.desc', 'Agent 通过 Sidecar 自动获取已授权的 Token，无需编写脚本。')}
        action={<button onClick={() => setImportOpen(true)} className="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white">{t('authTokens.create_token')}</button>}
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
                <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="GitHub Token" className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 text-sm outline-none focus:border-sky-500" />
              </label>
              <label className="block text-sm text-slate-700">
                {t('common.description')}
                <input value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} placeholder="访问 GitHub 仓库和 Issues" className="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 text-sm outline-none focus:border-sky-500" />
              </label>
              <label className="block text-sm text-slate-700">
                {t('authTokens.secret')}
                <div className="mt-2 flex overflow-hidden rounded-xl border border-slate-200 bg-white focus-within:border-sky-500">
                  <input
                    type={showSecret ? 'text' : 'password'}
                    value={form.secret}
                    onChange={(e) => setForm({ ...form, secret: e.target.value })}
                    placeholder="ghp_xxxx"
                    className="min-w-0 flex-1 bg-transparent px-4 py-3 text-sm outline-none"
                  />
                  <button type="button" onClick={() => setShowSecret((s) => !s)} className="shrink-0 border-l border-slate-200 px-3 text-xs text-slate-500 hover:bg-slate-50">
                    {showSecret ? t('auth.hide_password') : t('auth.show_password')}
                  </button>
                </div>
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
            <col className="w-[32%]" />
            <col className="w-[38%]" />
            <col className="w-[14%]" />
            <col className="w-[16%]" />
          </colgroup>
          <thead className={tableHeadClass}>
            <tr>
              <th className={tableHeaderCellClass}>
                <HeaderFilter label={t('common.name')} active={query.trim() !== ''}>
                  <FilterInput value={query} onChange={setQuery} placeholder={t('authTokens.search_placeholder')} />
                </HeaderFilter>
              </th>
              <th className={tableHeaderCellClass}>{t('common.description')}</th>
              <th className={tableHeaderCellClass}>{t('common.status')}</th>
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
                          <div className="mt-1 leading-5 text-slate-500">{token.description}</div>
                        </div>
                        <div className="text-xs text-slate-400">Secret 不会在前端展示，Agent 通过 Sidecar 获取。</div>
                      </div>
                    }
                  >
                    <span className="cursor-default underline decoration-slate-300 decoration-dotted underline-offset-4">{token.name}</span>
                  </HoverCard>
                </td>
                <td className={tableCellClass}>
                  <div className="max-w-sm truncate text-slate-700">{token.description || '-'}</div>
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
