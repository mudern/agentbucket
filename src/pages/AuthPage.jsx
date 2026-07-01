import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import LogoMark from '../components/LogoMark'
import { useT } from '../i18n'

const API_BASE = import.meta.env.VITE_API_BASE ?? 'http://127.0.0.1:8080'

export default function AuthPage({ mode = 'login', onAuthenticated }) {
  const navigate = useNavigate()
  const isRegister = mode === 'register'
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const t = useT()

  const handleSubmit = async (event) => {
    event.preventDefault()
    setError('')

    if (isRegister) {
      navigate('/login', { replace: true, state: { registered: true } })
      return
    }

    setLoading(true)
    try {
      const resp = await fetch(`${API_BASE}/api/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      if (!resp.ok) {
        const body = await resp.json().catch(() => ({}))
        throw new Error(body.error || 'Login failed')
      }
      const data = await resp.json()
      localStorage.setItem('agentbucket.token', data.token)
      localStorage.setItem('agentbucket.auth', 'true')
      onAuthenticated?.()
      navigate('/', { replace: true })
    } catch (e) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-100 px-6 py-10">
      <main className="w-full max-w-[440px]">
        <div className="mb-8 flex justify-center">
          <LogoMark />
        </div>

        <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <div className="mb-6 grid grid-cols-2 rounded-xl bg-slate-100 p-1 text-sm font-medium">
            <Link
              to="/login"
              className={`rounded-lg px-3 py-2.5 text-center transition ${
                !isRegister ? 'bg-white text-slate-950 shadow-sm' : 'text-slate-500 hover:text-slate-900'
              }`}
            >
              {t('auth.login_button')}
            </Link>
            <Link
              to="/register"
              className={`rounded-lg px-3 py-2.5 text-center transition ${
                isRegister ? 'bg-white text-slate-950 shadow-sm' : 'text-slate-500 hover:text-slate-900'
              }`}
            >
              {t('auth.register_button')}
            </Link>
          </div>

          <div className="mb-6">
            <h1 className="text-xl font-semibold text-slate-950">{isRegister ? t('auth.register_title') : t('auth.login_title')}</h1>
            <p className="mt-2 text-sm leading-6 text-slate-500">
              {isRegister ? '注册后进入用户权限审批，通过后可登录控制台。' : '使用组织账号进入 Agent 管理工作台。'}
            </p>
          </div>

          {error && (
            <div className="mb-4 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700">
              {error}
            </div>
          )}

          <form className="space-y-4" onSubmit={handleSubmit}>
            <label className="block text-sm font-medium text-slate-700">
              {t('auth.username_placeholder')}
              <input
                required
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder={t('auth.username_placeholder')}
                className="mt-2 h-11 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-sky-500 focus:ring-4 focus:ring-sky-100"
              />
            </label>

            <label className="block text-sm font-medium text-slate-700">
              {t('auth.password_placeholder')}
              <input
                required
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder={t('auth.password_placeholder')}
                className="mt-2 h-11 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-sky-500 focus:ring-4 focus:ring-sky-100"
              />
            </label>

            <button
              type="submit"
              disabled={loading}
              className="h-11 w-full rounded-xl bg-sky-600 px-4 text-sm font-semibold text-white transition hover:bg-sky-700 disabled:opacity-50"
            >
              {loading ? t('common.loading') : isRegister ? t('auth.register_button') : t('auth.login_button')}
            </button>
          </form>
        </section>
      </main>
    </div>
  )
}
