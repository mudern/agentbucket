import Sidebar from './Sidebar'

export default function Layout({ children, onLogout }) {
  return (
    <div className="min-h-screen bg-slate-50 text-slate-950">
      <div className="flex min-h-screen">
        <Sidebar onLogout={onLogout} />
        <main className="min-w-0 flex-1 p-8">{children}</main>
      </div>
    </div>
  )
}
