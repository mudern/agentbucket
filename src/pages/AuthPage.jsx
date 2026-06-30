import { Link, useNavigate } from 'react-router-dom'
import LogoMark from '../components/LogoMark'

export default function AuthPage({ mode = 'login', onAuthenticated }) {
  const navigate = useNavigate()
  const isRegister = mode === 'register'

  const handleSubmit = (event) => {
    event.preventDefault()

    if (isRegister) {
      navigate('/login', { replace: true, state: { registered: true } })
      return
    }

    onAuthenticated?.()
    navigate('/', { replace: true })
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
              登录
            </Link>
            <Link
              to="/register"
              className={`rounded-lg px-3 py-2.5 text-center transition ${
                isRegister ? 'bg-white text-slate-950 shadow-sm' : 'text-slate-500 hover:text-slate-900'
              }`}
            >
              注册
            </Link>
          </div>

          <div className="mb-6">
            <h1 className="text-xl font-semibold text-slate-950">{isRegister ? '申请账号' : '登录 AgentBucket'}</h1>
            <p className="mt-2 text-sm leading-6 text-slate-500">
              {isRegister ? '注册后进入用户权限审批，通过后可登录控制台。' : '使用组织账号进入 Agent 管理工作台。'}
            </p>
          </div>

          <form className="space-y-4" onSubmit={handleSubmit}>
            {isRegister && (
              <label className="block text-sm font-medium text-slate-700">
                姓名
                <input
                  required
                  placeholder="你的姓名"
                  className="mt-2 h-11 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-sky-500 focus:ring-4 focus:ring-sky-100"
                />
              </label>
            )}

            <label className="block text-sm font-medium text-slate-700">
              邮箱
              <input
                required
                type="email"
                placeholder="name@company.com"
                className="mt-2 h-11 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-sky-500 focus:ring-4 focus:ring-sky-100"
              />
            </label>

            <label className="block text-sm font-medium text-slate-700">
              <span>密码</span>
              <input
                required
                type="password"
                placeholder={isRegister ? '设置密码' : '输入密码'}
                className="mt-2 h-11 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-sky-500 focus:ring-4 focus:ring-sky-100"
              />
            </label>

            {!isRegister && (
              <div className="flex items-center justify-between text-sm">
                <label className="flex items-center gap-2 text-slate-600">
                  <input type="checkbox" className="rounded border-slate-300 text-sky-600 focus:ring-sky-500" />
                  保持登录
                </label>
                <button type="button" className="font-medium text-sky-700 hover:text-sky-800">
                  忘记密码
                </button>
              </div>
            )}

            <button className="h-11 w-full rounded-xl bg-sky-600 px-4 text-sm font-semibold text-white transition hover:bg-sky-700">
              {isRegister ? '提交注册申请' : '登录'}
            </button>
          </form>
        </section>
      </main>
    </div>
  )
}
