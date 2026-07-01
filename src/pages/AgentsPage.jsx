import { useMemo, useState } from 'react'
import AgentCard from '../components/AgentCard'
import LoadingPanel from '../components/LoadingPanel'
import PageHeader from '../components/PageHeader'
import WizardModal from '../components/WizardModal'
import { getAgents } from '../api'
import { useT } from '../i18n'
import useAsyncData from '../hooks/useAsyncData'

export default function AgentsPage() {
  const [query, setQuery] = useState('')
  const [selectedTags, setSelectedTags] = useState([])
  const [createOpen, setCreateOpen] = useState(false)
  const { data: agents = [], loading } = useAsyncData(getAgents, [])
  const t = useT()

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
        title={t('agents.title')}
        description=""
        action={
          <button onClick={() => setCreateOpen(true)} className="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-sky-700">
            {t('agents.new_agent')}
          </button>
        }
      />
      <WizardModal
        open={createOpen}
        title={t('agents.new_agent')}
        onClose={() => setCreateOpen(false)}
        steps={[
          {
            label: t('common.name'),
            content: <div className="text-sm leading-6 text-slate-600">{t('agents.new_agent_hint')}</div>,
          },
          {
            label: t('progress.title', '能力'),
            content: <div className="text-sm leading-6 text-slate-600">{t('deploy.capabilities_desc')}</div>,
          },
          {
            label: t('common.confirm'),
            content: <div className="text-sm leading-6 text-slate-600">{t('deploy.review_and_deploy')}</div>,
          },
        ]}
      />

      <div className="mb-6 grid gap-4 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm lg:grid-cols-[1.2fr_1fr]">
        <label className="block">
          <div className="mb-2 text-sm text-slate-600">{t('agents.search_placeholder')}</div>
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={t('agents.search_placeholder')}
            className="w-full rounded-xl border border-slate-200 bg-white px-4 py-3 text-sm text-slate-900 outline-none ring-0 placeholder:text-slate-400 focus:border-sky-500"
          />
        </label>
        <label className="block">
          <div className="mb-2 text-sm text-slate-600">{t('agents.no_agents', '按 Tag 过滤')}</div>
          <div className="flex flex-wrap gap-2">
            <button
              onClick={() => setSelectedTags([])}
              className={`rounded-full px-3 py-2 text-sm transition ${
                selectedTags.length === 0 ? 'bg-sky-600 text-white' : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
              }`}
            >
              {t('common.all')}
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

      <div className="mb-4 text-sm text-slate-500">{filteredAgents.length} {t('agents.title')}</div>
      {loading ? (
        <LoadingPanel label={t('common.loading')} />
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
