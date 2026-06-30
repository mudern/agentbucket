import { getUsers } from '../api'
import LoadingPanel from '../components/LoadingPanel'
import PageHeader from '../components/PageHeader'
import { roles } from '../data'
import useAsyncData from '../hooks/useAsyncData'

export default function UsersPage() {
  const { data: users = [], loading } = useAsyncData(getUsers, [])

  if (loading) {
    return <LoadingPanel label="正在加载用户列表..." />
  }

  return (
    <div>
      <PageHeader title="用户权限" description="管理注册用户身份，决定其为管理员或普通用户。" />
      <div className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
        <table className="min-w-full divide-y divide-slate-200 text-sm">
          <thead className="bg-slate-50 text-left text-slate-500">
            <tr>
              <th className="px-6 py-4">用户</th>
              <th className="px-6 py-4">邮箱</th>
              <th className="px-6 py-4">当前角色</th>
              <th className="px-6 py-4">状态</th>
              <th className="px-6 py-4">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-200 text-slate-700">
            {users.map((user) => (
              <tr key={user.id}>
                <td className="px-6 py-4 font-medium text-slate-950">{user.name}</td>
                <td className="px-6 py-4">{user.email}</td>
                <td className="px-6 py-4">{roles[user.role]}</td>
                <td className="px-6 py-4">
                  <span className={`rounded-full px-2.5 py-1 text-xs ${user.active ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-100 text-slate-500'}`}>
                    {user.active ? '活跃' : '停用'}
                  </span>
                </td>
                <td className="px-6 py-4">
                  <div className="flex gap-2">
                    <button className="rounded-lg bg-sky-50 px-3 py-2 text-sky-700">设为管理员</button>
                    <button className="rounded-lg bg-slate-100 px-3 py-2 text-slate-700">设为普通用户</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
