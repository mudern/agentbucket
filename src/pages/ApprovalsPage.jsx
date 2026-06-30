import { useMemo, useState } from 'react'
import { getApprovals } from '../api'
import LoadingPanel from '../components/LoadingPanel'
import PageHeader from '../components/PageHeader'
import useAsyncData from '../hooks/useAsyncData'

export default function ApprovalsPage() {
  const [tab, setTab] = useState('pending')
  const { data: approvals = [], loading } = useAsyncData(getApprovals, [])
  const filteredApprovals = useMemo(
    () =>
      approvals.filter((item) =>
        tab === 'pending' ? item.status === '待审批' || item.status === '处理中' : item.status !== '待审批' && item.status !== '处理中',
      ),
    [approvals, tab],
  )

  if (loading) {
    return <LoadingPanel label="正在加载审批列表..." />
  }

  return (
    <div>
      <PageHeader title="审批中心" description="管理员可审批新 Agent 部署请求、AI Token 使用请求与鉴权 Token 访问请求。" />
      <div className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
        <div className="border-b border-slate-200 px-5 pt-4">
          <div className="flex gap-2">
            {[
              ['pending', '待审批'],
              ['done', '已审批'],
            ].map(([value, label]) => (
              <button
                key={value}
                onClick={() => setTab(value)}
                className={`rounded-t-xl px-4 py-2 text-sm font-medium transition ${
                  tab === value ? 'bg-sky-50 text-sky-700' : 'text-slate-500 hover:bg-slate-50 hover:text-slate-900'
                }`}
              >
                {label}
              </button>
            ))}
          </div>
        </div>

        <table className="min-w-full divide-y divide-slate-200 text-sm">
          <thead className="bg-slate-50 text-left text-slate-500">
            <tr>
              <th className="px-6 py-4">类型</th>
              <th className="px-6 py-4">申请人</th>
              <th className="px-6 py-4">说明</th>
              <th className="px-6 py-4">优先级</th>
              <th className="px-6 py-4">状态</th>
              <th className="px-6 py-4">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-200 text-slate-700">
            {filteredApprovals.map((item) => (
              <tr key={item.id}>
                <td className="px-6 py-4 font-medium text-slate-950">{item.type}</td>
                <td className="px-6 py-4">{item.applicant}</td>
                <td className="max-w-xl px-6 py-4 leading-6">{item.summary}</td>
                <td className="px-6 py-4">
                  <span className="rounded-full bg-amber-50 px-2.5 py-1 text-xs text-amber-700">{item.priority}</span>
                </td>
                <td className="px-6 py-4">{item.status}</td>
                <td className="px-6 py-4">
                  {tab === 'pending' ? (
                    <div className="flex gap-2">
                      <button className="rounded-lg bg-emerald-600 px-3 py-2 text-xs font-medium text-white">批准</button>
                      <button className="rounded-lg bg-slate-100 px-3 py-2 text-xs font-medium text-slate-700">驳回</button>
                    </div>
                  ) : (
                    <span className="text-slate-500">审批人：{item.reviewer}</span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
