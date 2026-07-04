import { Link } from 'react-router-dom'

const statusConfig = {
  '已部署': { dot: 'bg-emerald-400', label: '已部署' },
  '离线': { dot: 'bg-slate-400', label: '未部署' },
}

export default function AgentCard({ agent }) {
  const status = statusConfig[agent.status] ?? { dot: 'bg-slate-400', label: agent.status || '未知' }
  const capabilityRows = [
    ['API Token', agent.apiToken],
    ['模型', agent.model],
    ['Runtime', agent.runtime],
    ['Skill', agent.skills?.join('、') || '未开启'],
    ['MCP', agent.mcps?.join('、') || '未开启'],
  ]

  return (
    <Link
      to={`/agents/${agent.id}`}
      className="group rounded-2xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-5 shadow-sm transition hover:-translate-y-0.5 hover:border-sky-200 dark:hover:border-sky-700 hover:shadow-md"
    >
      <div className="mb-4 flex items-start justify-between gap-4">
        <div>
          <h3 className="text-lg font-semibold text-slate-950 dark:text-slate-50 group-hover:text-sky-700">{agent.name}</h3>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{agent.model} · {agent.runtime}</p>
        </div>
        <span className="inline-flex items-center gap-2 rounded-full border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900 px-2.5 py-1 text-xs text-slate-600 dark:text-slate-400">
          <span className={`h-2.5 w-2.5 rounded-full ${status.dot}`} />
          {status.label}
        </span>
      </div>
      <p className="mb-5 text-sm leading-6 text-slate-600 dark:text-slate-400">{agent.description}</p>
      <div className="mb-5 grid gap-2 rounded-xl bg-slate-50 dark:bg-slate-900 p-3 text-xs">
        {capabilityRows.map(([label, value]) => (
          <div key={label} className="grid grid-cols-[78px_1fr] gap-3">
            <div className="text-slate-400 dark:text-slate-500">{label}</div>
            <div className="truncate text-slate-700 dark:text-slate-300">{value}</div>
          </div>
        ))}
      </div>
      <div className="mb-4 flex flex-wrap gap-2">
        {agent.tags.map((tag) => (
          <span key={tag} className="rounded-full bg-sky-50 px-2.5 py-1 text-xs text-sky-700 dark:bg-sky-900/50 dark:text-sky-300">
            #{tag}
          </span>
        ))}
      </div>
      <div className="text-xs text-slate-400 dark:text-slate-500">最近更新：{agent.updatedAt}</div>
    </Link>
  )
}
