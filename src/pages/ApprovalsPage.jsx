import { useMemo, useState } from 'react'
import { getApprovals, approveApproval, rejectApproval } from '../api'
import LoadingPanel from '../components/LoadingPanel'
import PageHeader from '../components/PageHeader'
import { useT } from '../i18n'
import useAsyncData from '../hooks/useAsyncData'

export default function ApprovalsPage() {
  const [tab, setTab] = useState('pending')
  const { data: approvals = [], loading } = useAsyncData(getApprovals, [])
  const [local, setLocal] = useState([])
  const [acting, setActing] = useState(null)
  const t = useT()

  const displayed = local.length > 0 ? local : approvals

  const filteredApprovals = useMemo(
    () => displayed.filter((item) =>
      tab === 'pending' ? item.status === '待审批' || item.status === '处理中' : item.status !== '待审批' && item.status !== '处理中',
    ),
    [displayed, tab],
  )

  if (loading) {
    return <LoadingPanel label={t('common.loading')} />
  }

  const handleAction = async (id, action) => {
    setActing(id)
    try {
      if (action === 'approve') {
        await approveApproval(id)
      } else {
        await rejectApproval(id)
      }
      if (local.length === 0) setLocal([...approvals])
      setLocal((c) => c.map((a) => (a.id === id ? { ...a, status: action === 'approve' ? '已通过' : '已拒绝', reviewer: 'admin' } : a)))
    } catch (e) {
      alert(`${t('common.save_failed')}: ${e.message}`)
    } finally {
      setActing(null)
    }
  }

  return (
    <div>
      <PageHeader title={t('approvals.title')} description={t('approvals.pending_requests')} />
      <div className="overflow-hidden rounded-2xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 shadow-sm">
        <div className="border-b border-slate-200 dark:border-slate-700 px-5 pt-4">
          <div className="flex gap-2">
            {[
              ['pending', t('approvals.status_pending')],
              ['done', t('approvals.status_approved')],
            ].map(([value, label]) => (
              <button
                key={value}
                onClick={() => setTab(value)}
                className={`rounded-t-xl px-4 py-2 text-sm font-medium transition ${
                  tab === value ? 'bg-sky-50 text-sky-700' : 'text-slate-500 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 dark:bg-slate-900 hover:text-slate-900 dark:text-slate-100'
                }`}
              >
                {label}
              </button>
            ))}
          </div>
        </div>

        <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-700 text-sm">
          <thead className="bg-slate-50 dark:bg-slate-900 text-left text-slate-500 dark:text-slate-400">
            <tr>
              <th className="px-6 py-4">{t('common.type')}</th>
              <th className="px-6 py-4">{t('approvals.requester')}</th>
              <th className="px-6 py-4">{t('common.description')}</th>
              <th className="px-6 py-4">{t('common.status')}</th>
              <th className="px-6 py-4">{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-200 dark:divide-slate-700 text-slate-700 dark:text-slate-300">
            {filteredApprovals.map((item) => (
              <tr key={item.id}>
                <td className="px-6 py-4 font-medium text-slate-950 dark:text-slate-50">{item.type}</td>
                <td className="px-6 py-4">{item.applicant}</td>
                <td className="max-w-xl px-6 py-4 leading-6">{item.summary}</td>
                <td className="px-6 py-4">
                  <span className={`rounded-full px-2.5 py-1 text-xs ${
                    item.status === '已通过' ? 'bg-emerald-50 text-emerald-700' :
                    item.status === '已拒绝' ? 'bg-red-50 text-red-700' :
                    'bg-amber-50 text-amber-700'
                  }`}>{item.status}</span>
                </td>
                <td className="px-6 py-4">
                  {tab === 'pending' ? (
                    <div className="flex gap-2">
                      <button
                        onClick={() => handleAction(item.id, 'approve')}
                        disabled={acting === item.id}
                        className="rounded-lg bg-emerald-600 px-3 py-2 text-xs font-medium text-white hover:bg-emerald-700 disabled:opacity-50"
                      >
                        {t('approvals.approve')}
                      </button>
                      <button
                        onClick={() => handleAction(item.id, 'reject')}
                        disabled={acting === item.id}
                        className="rounded-lg bg-slate-100 dark:bg-slate-700 px-3 py-2 text-xs font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-200 disabled:opacity-50"
                      >
                        {t('approvals.reject')}
                      </button>
                    </div>
                  ) : (
                    <span className="text-slate-500 dark:text-slate-400">{item.reviewer || '-'}</span>
                  )}
                </td>
              </tr>
            ))}
            {filteredApprovals.length === 0 && (
              <tr>
                <td colSpan={5} className="px-6 py-12 text-center text-slate-400 dark:text-slate-500">{t('approvals.no_approvals')}</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
