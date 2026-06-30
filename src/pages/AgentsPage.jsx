import { useMemo, useState } from 'react'
import AgentCard from '../components/AgentCard'
import LoadingPanel from '../components/LoadingPanel'
import PageHeader from '../components/PageHeader'
import WizardModal from '../components/WizardModal'
import { getAgents } from '../api'
import useAsyncData from '../hooks/useAsyncData'

export default function AgentsPage() {
  const [query, setQuery] = useState('')
  const [selectedTags, setSelectedTags] = useState([])
  const [createOpen, setCreateOpen] = useState(false)
  const { data: agents = [], loading } = useAsyncData(getAgents, [])

  const tags = useMemo(() => [...new Set(agents.flatMap((agent) => agent.tags))], [agents])

  const toggleTag = (tag) => {
    setSelectedTags((current) => (current.includes(tag) ? current.filter((item) => item !== tag) : [...current, tag]))
  }

  const filteredAgents = useMemo(() => {
    return agents.filter((agent) => {
      const hitQuery =
        query.trim() === '' ||
        [agent.name, agent.model, agent.description, ...agent.tags]
          .join(' ')
          .toLowerCase()
          .includes(query.toLowerCase())
      const hitTag = selectedTags.length === 0 || selectedTags.every((tag) => agent.tags.includes(tag))
      return hitQuery && hitTag
    })
  }, [agents, query, selectedTags])

  return (
    <div>
      <PageHeader
        title="所有 Agent"
        description="浏览、搜索并按标签过滤所有已部署 Agent，点击卡片可进入对话界面。"
        action={
          <button onClick={() => setCreateOpen(true)} className="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-sky-700">
            新建 Agent
          </button>
        }
      />
      <WizardModal
        open={createOpen}
        title="新建 Agent"
        onClose={() => setCreateOpen(false)}
        steps={[
          {
            label: '来源',
            content: <div className="text-sm leading-6 text-slate-600">Agent 需要从仓库管理中已绑定的仓库部署。请选择部署页面中的仓库和 commit。</div>,
          },
          {
            label: '能力',
            content: <div className="text-sm leading-6 text-slate-600">部署时会配置 API Token、模型、runtime、Skill、MCP 和鉴权 Token。</div>,
          },
          {
            label: '确认',
            content: <div className="text-sm leading-6 text-slate-600">完成后会进入审批或直接部署，取决于当前权限设置。</div>,
          },
        ]}
      />

      <div className="mb-6 grid gap-4 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm lg:grid-cols-[1.2fr_1fr]">
        <label className="block">
          <div className="mb-2 text-sm text-slate-600">搜索 Agent</div>
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="按名称、模型、标签搜索"
            className="w-full rounded-xl border border-slate-200 bg-white px-4 py-3 text-sm text-slate-900 outline-none ring-0 placeholder:text-slate-400 focus:border-sky-500"
          />
        </label>
        <label className="block">
          <div className="mb-2 text-sm text-slate-600">按 Tag 过滤</div>
          <div className="flex flex-wrap gap-2">
            <button
              onClick={() => setSelectedTags([])}
              className={`rounded-full px-3 py-2 text-sm transition ${
                selectedTags.length === 0 ? 'bg-sky-600 text-white' : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
              }`}
            >
              全部
            </button>
            {tags.map((tag) => (
              <button
                key={tag}
                onClick={() => toggleTag(tag)}
                className={`rounded-full px-3 py-2 text-sm transition ${
                  selectedTags.includes(tag)
                    ? 'bg-sky-600 text-white'
                    : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
                }`}
              >
                #{tag}
              </button>
            ))}
          </div>
        </label>
      </div>

      <div className="mb-4 text-sm text-slate-500">共找到 {filteredAgents.length} 个 Agent</div>
      {loading ? (
        <LoadingPanel label="正在加载 Agent 列表..." />
      ) : (
        <div className="grid gap-5 xl:grid-cols-2 2xl:grid-cols-3">
          {filteredAgents.map((agent) => (
            <AgentCard key={agent.id} agent={agent} />
          ))}
        </div>
      )}
    </div>
  )
}
