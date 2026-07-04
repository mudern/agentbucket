import { useState } from 'react'
import { getUsers, patchUser } from '../api'
import LoadingPanel from '../components/LoadingPanel'
import PageHeader from '../components/PageHeader'
import { useT } from '../i18n'
import useAsyncData from '../hooks/useAsyncData'

export default function UsersPage() {
  const { data: users = [], loading } = useAsyncData(getUsers, [])
  const [localUsers, setLocalUsers] = useState([])
  const [updating, setUpdating] = useState(null)
  const t = useT()

  if (loading) {
    return <LoadingPanel label={t('common.loading')} />
  }

  const displayed = localUsers.length > 0 ? localUsers : users

  const handleRoleChange = async (userId, newRole) => {
    setUpdating(userId)
    try {
      await patchUser(userId, { role: newRole })
      if (localUsers.length === 0) setLocalUsers([...users])
      setLocalUsers((c) => c.map((u) => (u.id === userId ? { ...u, role: newRole } : u)))
    } catch (e) {
      alert(`${t('common.save_failed')}: ${e.message}`)
    } finally {
      setUpdating(null)
    }
  }

  const handleToggleActive = async (userId, active) => {
    setUpdating(userId)
    try {
      await patchUser(userId, { active })
      if (localUsers.length === 0) setLocalUsers([...users])
      setLocalUsers((c) => c.map((u) => (u.id === userId ? { ...u, active } : u)))
    } catch (e) {
      alert(`${t('common.save_failed')}: ${e.message}`)
    } finally {
      setUpdating(null)
    }
  }

  return (
    <div>
      <PageHeader title={t('users.title')} description="" />
      <div className="overflow-hidden rounded-2xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 shadow-sm">
        <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-700 text-sm">
          <thead className="bg-slate-50 dark:bg-slate-900 text-left text-slate-500 dark:text-slate-400">
            <tr>
              <th className="px-6 py-4">{t('users.username')}</th>
              <th className="px-6 py-4">{t('users.email')}</th>
              <th className="px-6 py-4">{t('users.role')}</th>
              <th className="px-6 py-4">{t('common.status')}</th>
              <th className="px-6 py-4">{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-200 dark:divide-slate-700 text-slate-700 dark:text-slate-300">
            {displayed.map((user) => (
              <tr key={user.id}>
                <td className="px-6 py-4 font-medium text-slate-950 dark:text-slate-50">{user.name}</td>
                <td className="px-6 py-4">{user.email}</td>
                <td className="px-6 py-4">
                  <span className={`rounded-full px-2.5 py-1 text-xs ${
                    user.role === 'super_admin' ? 'bg-indigo-50 text-indigo-700' :
                    user.role === 'admin' ? 'bg-sky-50 text-sky-700' :
                    'bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-400'
                  }`}>
                    {user.role === 'super_admin' ? t('users.role_super_admin', '超级管理员') :
                     user.role === 'admin' ? t('users.role_admin') : t('users.role_user')}
                  </span>
                </td>
                <td className="px-6 py-4">
                  <span className={`rounded-full px-2.5 py-1 text-xs ${user.active ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-100 dark:bg-slate-700 text-slate-500 dark:text-slate-400'}`}>
                    {user.active ? t('users.enabled_true') : t('users.enabled_false')}
                  </span>
                </td>
                <td className="px-6 py-4">
                  <div className="flex gap-2">
                    {user.role === 'super_admin' && (
                      <>
                        <button
                          onClick={() => handleRoleChange(user.id, 'admin')}
                          disabled={updating === user.id}
                          className="rounded-lg bg-sky-50 px-3 py-2 text-xs font-medium text-sky-700 hover:bg-sky-100 disabled:opacity-50"
                        >
                          {t('users.role_admin')}
                        </button>
                        <button
                          onClick={() => handleRoleChange(user.id, 'user')}
                          disabled={updating === user.id}
                          className="rounded-lg bg-slate-100 dark:bg-slate-700 px-3 py-2 text-xs font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-200 disabled:opacity-50"
                        >
                          {t('users.role_user')}
                        </button>
                      </>
                    )}
                    {user.role === 'admin' && (
                      <>
                        <button
                          onClick={() => handleRoleChange(user.id, 'super_admin')}
                          disabled={updating === user.id}
                          className="rounded-lg bg-indigo-50 px-3 py-2 text-xs font-medium text-indigo-700 hover:bg-indigo-100 disabled:opacity-50"
                        >
                          {t('users.role_super_admin', '超级管理员')}
                        </button>
                        <button
                          onClick={() => handleRoleChange(user.id, 'user')}
                          disabled={updating === user.id}
                          className="rounded-lg bg-slate-100 dark:bg-slate-700 px-3 py-2 text-xs font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-200 disabled:opacity-50"
                        >
                          {t('users.role_user')}
                        </button>
                      </>
                    )}
                    {user.role === 'user' && (
                      <>
                        <button
                          onClick={() => handleRoleChange(user.id, 'super_admin')}
                          disabled={updating === user.id}
                          className="rounded-lg bg-indigo-50 px-3 py-2 text-xs font-medium text-indigo-700 hover:bg-indigo-100 disabled:opacity-50"
                        >
                          {t('users.role_super_admin', '超级管理员')}
                        </button>
                        <button
                          onClick={() => handleRoleChange(user.id, 'admin')}
                          disabled={updating === user.id}
                          className="rounded-lg bg-sky-50 px-3 py-2 text-xs font-medium text-sky-700 hover:bg-sky-100 disabled:opacity-50"
                        >
                          {t('users.role_admin')}
                        </button>
                      </>
                    )}
                    {user.active ? (
                      <button
                        onClick={() => handleToggleActive(user.id, false)}
                        disabled={updating === user.id}
                        className="rounded-lg bg-red-50 px-3 py-2 text-xs font-medium text-red-700 hover:bg-red-100 disabled:opacity-50"
                      >
                        {t('common.disable')}
                      </button>
                    ) : (
                      <button
                        onClick={() => handleToggleActive(user.id, true)}
                        disabled={updating === user.id}
                        className="rounded-lg bg-emerald-50 px-3 py-2 text-xs font-medium text-emerald-700 hover:bg-emerald-100 disabled:opacity-50"
                      >
                        {t('common.enable')}
                      </button>
                    )}
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
