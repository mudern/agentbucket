import { useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import { API_BASE, getAgentById, getAgentSessionMessages, getAgentSessions, createAgentSession, deleteSession, renameSession } from '../api'
import { useT, useLanguage } from '../i18n'
import LoadingPanel from '../components/LoadingPanel'
import useAsyncData from '../hooks/useAsyncData'

const QUESTION_RE = /\[QUESTION:\s*(.+?)\s*\|\s*(.+?)\s*\|\s*(.+?)\s*\]/

function parseQuestion(text) {
  const match = text.match(QUESTION_RE)
  if (!match) return null
  return { prompt: match[1], options: [match[2], match[3]], full: match[0] }
}


function MessageBubble({ message, onSelectOption, formatTime, t }) {
  const question = parseQuestion(message.content)
  const displayContent = question ? message.content.replace(question.full, '').trim() : message.content

  return (
    <div className={`flex gap-3 ${message.role === 'user' ? 'flex-row-reverse' : ''}`}>
      <div className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xs font-bold ${
        message.role === 'user'
          ? 'bg-sky-600 text-white'
          : 'bg-gradient-to-br from-indigo-500 to-purple-600 text-white'
      }`}>
        {message.role === 'user' ? t('chat.you', '你') : 'AI'}
      </div>

      <div className={`min-w-0 max-w-[75%] ${message.role === 'user' ? 'items-end' : 'items-start'}`}>
        {displayContent && (
          <div className={`overflow-hidden rounded-2xl px-4 py-3 text-sm leading-7 ${
            message.role === 'user'
              ? 'bg-sky-600 text-white'
              : 'border border-slate-200 dark:border-slate-700 bg-white text-slate-700 dark:text-slate-300 shadow-sm'
          }`}>
            <div className={`prose prose-sm max-w-none break-words prose-pre:max-w-[calc(75vw-6rem)] prose-pre:overflow-x-auto prose-img:max-w-full ${
              message.role === 'user'
                ? 'prose-invert'
                : 'prose-slate prose-headings:text-slate-900 dark:text-slate-100'
            }`}>
              <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                rehypePlugins={[rehypeHighlight]}
              >
                {displayContent}
              </ReactMarkdown>
            </div>
          </div>
        )}

        {question && (
          <div className="mt-2 rounded-2xl border-2 border-sky-200 bg-sky-50 p-4">
            <div className="mb-3 text-sm font-medium text-sky-900">{question.prompt}</div>
            <div className="flex flex-wrap gap-2">
              {question.options.map((opt, idx) => (
                <button
                  key={idx}
                  onClick={() => onSelectOption?.(opt)}
                  className="rounded-xl border-2 border-sky-300 bg-white dark:bg-slate-800 px-5 py-2 text-sm font-medium text-sky-700 shadow-sm transition hover:bg-sky-600 hover:text-white hover:border-sky-600"
                >
                  {opt}
                </button>
              ))}
            </div>
          </div>
        )}

        <div className={`mt-1 text-[10px] text-slate-400 dark:text-slate-500 ${message.role === 'user' ? 'text-right' : 'text-left'}`}>
          {formatTime(message.createdAt)}
        </div>
      </div>
    </div>
  )
}

export default function AgentChatPage() {
  const { agentId } = useParams()
  const [activeSession, setActiveSession] = useState(null)
  const [messages, setMessages] = useState([])
  const [draft, setDraft] = useState('')
  const [sending, setSending] = useState(false)
  const [sendError, setSendError] = useState('')
  const [newTitle, setNewTitle] = useState('')
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState('')
  const [sessionsRefresh, setSessionsRefresh] = useState(0)
  const [sessionsOpen, setSessionsOpen] = useState(true)
  const [editingSession, setEditingSession] = useState(null)
  const [editTitle, setEditTitle] = useState('')
  const messagesEnd = useRef(null)
  const { lang } = useLanguage()
  const locale = { zh: 'zh-CN', en: 'en-US' }[lang] || 'zh-CN'
  const fmtTime = (dateStr) => {
    const d = new Date(dateStr)
    if (isNaN(d.getTime())) return ''
    return d.toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' })
  }

  const { data: agent, loading: agentLoading } = useAsyncData(() => getAgentById(agentId), [agentId])
  const { data: sessions = [], loading: sessionsLoading } = useAsyncData(
    () => getAgentSessions(agentId),
    [agentId, sessionsRefresh],
  )
  const currentSessionId = activeSession ?? sessions[0]?.id
  const currentSession = sessions.find((session) => session.id === currentSessionId)
  const { data: loadedMessages = [], loading: messagesLoading } = useAsyncData(
    () => currentSessionId ? getAgentSessionMessages(agentId, currentSessionId) : Promise.resolve([]),
    [agentId, currentSessionId],
  )

  useEffect(() => {
    setMessages(loadedMessages)
  }, [loadedMessages])

  useEffect(() => {
    messagesEnd.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const handleSend = async (content) => {
    const text = (content || draft).trim()
    if (!text || sending || !currentSessionId) return
    setSending(true)
    setSendError('')
    setDraft('')
    const tempId = `temp-${Date.now()}`
    const assistantId = `temp-${Date.now() + 1}`
    setMessages((current) => [...current, {
      id: tempId, sessionId: currentSessionId, agentId, role: 'user', content: text, createdAt: new Date().toISOString(),
    }, {
      id: assistantId, sessionId: currentSessionId, agentId, role: 'assistant', content: '', createdAt: new Date().toISOString(),
    }])

    const token = localStorage.getItem('agentbucket.token') || sessionStorage.getItem('agentbucket.token') || ''
    try {
      const response = await fetch(`${API_BASE}/api/agents/${agentId}/messages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
        body: JSON.stringify({ sessionId: currentSessionId, content: text, stream: true }),
      })
      if (!response.ok) {
        const err = await response.json().catch(() => ({}))
        throw new Error(err.error || `HTTP ${response.status}`)
      }
      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''
        for (const line of lines) {
          if (line.startsWith('data: ')) {
            const data = line.slice(6)
            if (data === '[DONE]') {
              buffer = ''
              break
            }
            try {
              const parsed = JSON.parse(data)
              if (parsed.error) throw new Error(parsed.error)
            } catch (e) {
              if (e.message && !data.startsWith('{')) {
                // It's a text delta
                const decoded = data.replace(/\\n/g, '\n')
                setMessages((current) => current.map((m) =>
                  m.id === assistantId ? { ...m, content: m.content + decoded } : m
                ))
              } else if (e.message) {
                throw e
              }
            }
          }
        }
      }
      setMessages((current) => current.map((m) =>
        m.id === assistantId && !m.content ? { ...m, content: t('agents.title', 'AI 返回了空响应。') } : m
      ))
      setSessionsRefresh((n) => n + 1)
    } catch (error) {
      setSendError(error.message)
      setMessages((current) => current.filter((m) => m.id !== tempId && m.id !== assistantId))
    } finally {
      setSending(false)
    }
  }

  const handleCreateSession = async () => {
    const title = newTitle.trim()
    if (!title || creating) return
    setCreating(true)
    setCreateError('')
    try {
      const session = await createAgentSession(agentId, { title })
      setActiveSession(session.id)
      setNewTitle('')
      setSessionsRefresh((n) => n + 1)
    } catch (error) {
      setCreateError(error.message)
    } finally {
      setCreating(false)
    }
  }

  const t = useT()

  const handleRenameSession = async (sessionId, title) => {
    if (!title.trim()) return
    try { await renameSession(agentId, sessionId, title); setSessionsRefresh((n) => n + 1); setEditingSession(null) }
    catch (e) { console.error('rename failed:', e) }
  }

  const handleDeleteSession = async (e, sessionId, sessionTitle) => {
    e.stopPropagation()
    if (!window.confirm(t('session.deleteConfirm', `确定要删除会话「${sessionTitle}」吗？`))) return
    try {
      await deleteSession(agentId, sessionId)
      if (currentSessionId === sessionId) {
        setActiveSession(null)
      }
      setSessionsRefresh((n) => n + 1)
    } catch (error) {
      console.error('delete session failed:', error)
    }
  }

  if (agentLoading || !agent) {
    return <LoadingPanel label={t('common.loading')} />
  }

  return (
    <div className="flex h-[calc(100vh-4rem)] min-h-0 flex-col overflow-hidden rounded-2xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 shadow-sm">
      <div className="flex h-14 shrink-0 items-center gap-3 border-b border-slate-200 dark:border-slate-700 px-4 sm:px-5">
        <Link
          to="/agents"
          className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-400 dark:text-slate-500 shadow-sm transition hover:border-slate-300 dark:border-slate-600 hover:bg-slate-50 dark:hover:bg-slate-700 dark:bg-slate-900 hover:text-slate-700 dark:text-slate-300"
          aria-label={t('chat.back_to_agents')}
          title={t('chat.back_to_agents')}
        >
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <path d="M15 6l-6 6 6 6" />
          </svg>
        </Link>
        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 items-center gap-2">
            <h1 className="truncate text-sm font-semibold text-slate-950 dark:text-slate-50 sm:text-base">{agent.name}</h1>
            <span className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium ${
              agent.status === '已部署' ? 'bg-emerald-50 text-emerald-700'
              : agent.status === t('chat.status_offline') ? 'bg-slate-100 dark:bg-slate-700 text-slate-500 dark:text-slate-400'
              : 'bg-emerald-50 text-emerald-700'
            }`}>
              {agent.status}
            </span>
          </div>
          <div className="mt-0.5 truncate text-xs text-slate-400 dark:text-slate-500 sm:hidden">
            {currentSession?.title ?? t('chat.no_session', '未选择会话')} · {agent.model}
          </div>
          <div className="mt-0.5 hidden truncate text-xs text-slate-400 dark:text-slate-500 sm:block">
            {currentSession?.title ?? t('chat.no_session', '未选择会话')}
          </div>
        </div>
        <div className="hidden shrink-0 items-center gap-2 md:flex">
          <span className="max-w-44 truncate rounded-full border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900 px-2.5 py-1 text-xs text-slate-500 dark:text-slate-400">
            {agent.model}
          </span>
          <span className="rounded-full border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900 px-2.5 py-1 text-xs text-slate-500 dark:text-slate-400">
            {agent.runtime}
          </span>
        </div>
      </div>

      <div className="relative flex min-h-0 flex-1">
        <button
          type="button"
          onClick={() => setSessionsOpen((open) => !open)}
          className={`absolute z-20 hidden h-7 w-7 items-center justify-center rounded-full border border-slate-200 dark:border-slate-700 bg-white text-slate-400 dark:text-slate-500 shadow-sm transition-all duration-200 hover:border-slate-300 dark:border-slate-600 hover:bg-slate-50 dark:hover:bg-slate-700 dark:bg-slate-900 hover:text-slate-700 dark:text-slate-300 xl:flex ${
            sessionsOpen ? 'left-[17.25rem] top-16' : 'left-4 top-4'
          }`}
          aria-label={sessionsOpen ? t('chat.collapse_sessions', '收起会话栏') : t('chat.expand_sessions', '展开会话栏')}
          title={sessionsOpen ? t('chat.collapse_sessions', '收起会话栏') : t('chat.expand_sessions', '展开会话栏')}
        >
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            {sessionsOpen ? <path d="M15 6l-6 6 6 6" /> : <path d="M9 6l6 6-6 6" />}
          </svg>
        </button>
        {sessionsOpen && (
          <aside className="hidden w-72 shrink-0 flex-col border-r border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/80 xl:flex">
            <div className="shrink-0 border-b border-slate-200 dark:border-slate-700 px-3 pb-3 pt-3">
              <div className="mb-2 flex items-center justify-between px-1">
                <div className="text-xs font-semibold text-slate-500 dark:text-slate-400">{t('chat.sessions', '会话')}</div>
                <div className="text-[10px] text-slate-400 dark:text-slate-500">{sessions.length} 个</div>
              </div>
              <div className="flex gap-1.5">
                <input
                  type="text"
                  value={newTitle}
                  onChange={(e) => setNewTitle(e.target.value)}
                  onKeyDown={(e) => { if (e.key === 'Enter') handleCreateSession() }}
                  placeholder={t('chat.new_session')}
                  className="min-w-0 flex-1 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 px-3 py-2 text-sm text-slate-900 dark:text-slate-100 outline-none placeholder:text-slate-400 dark:placeholder:text-slate-500 dark:text-slate-500 focus:border-sky-400"
                />
                <button
                  onClick={handleCreateSession}
                  disabled={creating || !newTitle.trim()}
                  className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-sky-600 text-sm font-semibold text-white transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-50"
                  aria-label={t('chat.new_session')}
                  title={t('chat.new_session')}
                >
                  {creating ? '...' : '+'}
                </button>
              </div>
              {createError && <div className="mt-1.5 text-xs text-rose-600">{createError}</div>}
            </div>
            <div className="flex-1 space-y-1 overflow-y-auto p-2">
              {sessions.map((session) => (
                <button
                  key={session.id}
                  onClick={() => setActiveSession(session.id)}
                  className={`group relative w-full rounded-xl px-3 py-2.5 pr-8 text-left transition ${
                    currentSessionId === session.id
                      ? 'bg-white shadow-sm ring-1 ring-sky-200 dark:bg-slate-700 dark:ring-sky-800'
                      : 'hover:bg-white dark:hover:bg-slate-700'
                  }`}
                >
                  {editingSession === session.id ? (
                    <input
                      className="w-full rounded border border-sky-300 bg-white dark:bg-slate-800 px-2 py-1 text-sm text-slate-900 dark:text-slate-100 outline-none"
                      value={editTitle}
                      onChange={(e) => setEditTitle(e.target.value)}
                      onKeyDown={(e) => { if (e.key === 'Enter') handleRenameSession(session.id, editTitle); if (e.key === 'Escape') setEditingSession(null) }}
                      onBlur={() => setEditingSession(null)}
                      autoFocus
                      onClick={(e) => e.stopPropagation()}
                    />
                  ) : (
                    <div
                      className={`truncate text-sm font-medium cursor-pointer ${
                        currentSessionId === session.id ? 'text-sky-900' : 'text-slate-700 dark:text-slate-300'
                      }`}
                      onDoubleClick={(e) => { e.stopPropagation(); setEditingSession(session.id); setEditTitle(session.title) }}
                    >
                      {session.title}
                    </div>
                  )}
                  <div className="mt-1 flex min-w-0 items-center gap-2 text-xs text-slate-400 dark:text-slate-500">
                    {session.preview && <span className="truncate">{session.preview}</span>}
                    {session.updatedAt && <span className="ml-auto shrink-0">{fmtTime(session.updatedAt)}</span>}
                  </div>
                  <button
                    onClick={(e) => handleDeleteSession(e, session.id, session.title)}
                    className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-slate-300 dark:text-slate-600 opacity-0 transition hover:text-red-500 group-hover:opacity-100"
                    title={t('chat.delete_session')}
                    aria-label={t('chat.delete_session')}
                  >
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <path d="M18 6L6 18M6 6l12 12" />
                    </svg>
                  </button>
                </button>
              ))}
              {sessionsLoading && <div className="px-3 py-2 text-sm text-slate-400 dark:text-slate-500">{t('common.loading')}</div>}
              {!sessionsLoading && sessions.length === 0 && (
                <div className="px-3 py-4 text-center text-xs text-slate-400 dark:text-slate-500">{t('chat.no_session', '暂无会话，输入标题创建')}</div>
              )}
            </div>
            {(agent.skills?.length > 0 || agent.mcps?.length > 0) && (
              <div className="shrink-0 border-t border-slate-200 dark:border-slate-700 p-3">
                <div className="space-y-1 text-[10px] leading-relaxed text-slate-400 dark:text-slate-500">
                  {agent.skills?.length > 0 && (
                    <div><span className="font-medium text-slate-500 dark:text-slate-400">Skills:</span> {agent.skills.join(', ')}</div>
                  )}
                  {agent.mcps?.length > 0 && (
                    <div><span className="font-medium text-slate-500 dark:text-slate-400">MCPs:</span> {agent.mcps.join(', ')}</div>
                  )}
                </div>
              </div>
            )}
          </aside>
        )}

        <div className="flex min-w-0 flex-1 flex-col">
          <div className="flex-1 overflow-y-auto bg-slate-50 dark:bg-slate-900/50 p-4 sm:p-5" style={{ scrollbarWidth: 'thin', scrollbarColor: '#cbd5e1 #f1f5f9' }}>
            <div className="mx-auto max-w-4xl space-y-6">
              {messagesLoading && (
                <div className="rounded-xl border border-dashed border-slate-200 dark:border-slate-700 p-8 text-center text-sm text-slate-400 dark:text-slate-500">
                  {t('common.loading')}
                </div>
              )}
              {!messagesLoading && messages.length === 0 && (
                <div className="rounded-xl border border-dashed border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-10 text-center">
                  <div className="text-sm font-medium text-slate-600 dark:text-slate-400">{t('chat.start_chat_with', '开始与 {name} 对话').replace('{name}', agent.name)}</div>
                  <div className="mt-1 text-xs text-slate-400 dark:text-slate-500">{t('chat.empty_state', '输入消息并按 Ctrl+Enter 发送')}</div>
                </div>
              )}
              {messages.map((msg) => (
                <MessageBubble
                  key={msg.id}
                  message={msg}
                  onSelectOption={(opt) => handleSend(opt)}
                  formatTime={fmtTime}
                  t={t}
                />
              ))}
              {sending && (
                <div className="flex gap-3">
                  <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-indigo-500 to-purple-600 text-xs font-bold text-white">AI</div>
                  <div className="rounded-2xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 px-4 py-3 shadow-sm">
                    <div className="flex gap-1.5">
                      <div className="h-2 w-2 animate-bounce rounded-full bg-sky-400" style={{ animationDelay: '0ms' }} />
                      <div className="h-2 w-2 animate-bounce rounded-full bg-sky-400" style={{ animationDelay: '150ms' }} />
                      <div className="h-2 w-2 animate-bounce rounded-full bg-sky-400" style={{ animationDelay: '300ms' }} />
                    </div>
                  </div>
                </div>
              )}
              <div ref={messagesEnd} />
            </div>
          </div>

          <div className="shrink-0 border-t border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4">
            <div className="mx-auto max-w-4xl">
              {sendError && (
                <div className="mb-3 rounded-xl border border-rose-200 bg-rose-50 px-4 py-2 text-sm text-rose-600">
                  {sendError}
                </div>
              )}
              <div className="rounded-2xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 shadow-sm transition focus-within:border-sky-300 focus-within:shadow-md">
                <textarea
                  rows="3"
                  value={draft}
                  onChange={(e) => setDraft(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
                      e.preventDefault()
                      handleSend()
                    }
                  }}
                  disabled={sending || !currentSessionId}
                  placeholder={currentSessionId ? t('chat.input_placeholder') : t('chat.empty_state')}
                  className="w-full resize-none rounded-t-2xl bg-transparent px-4 pt-3 text-sm text-slate-900 dark:text-slate-100 dark:text-slate-100 outline-none placeholder:text-slate-400 dark:placeholder:text-slate-500 dark:text-slate-500"
                />
                <div className="flex items-center justify-between rounded-b-2xl bg-slate-50 dark:bg-slate-900 px-4 py-2">
                  <div className="text-xs text-slate-400 dark:text-slate-500">
                    {currentSessionId ? t('chat.send_hint', 'Ctrl+Enter 发送 · 支持 Markdown') : t('chat.create_session_hint', '先在左侧创建会话')}
                  </div>
                  <button
                    onClick={() => handleSend()}
                    disabled={sending || !draft.trim() || !currentSessionId}
                    className="rounded-xl bg-sky-600 px-5 py-2 text-sm font-medium text-white transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    {sending ? (
                      <span className="flex items-center gap-1.5">
                        <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-white border-t-transparent" />
                        {t('chat.typing')}
                      </span>
                    ) : t('chat.send')}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
