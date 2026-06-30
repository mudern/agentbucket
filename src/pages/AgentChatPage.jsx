import { useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import { getAgentById, getAgentSessionMessages, getAgentSessions, sendAgentMessage, createAgentSession } from '../api'
import LoadingPanel from '../components/LoadingPanel'
import useAsyncData from '../hooks/useAsyncData'

const QUESTION_RE = /\[QUESTION:\s*(.+?)\s*\|\s*(.+?)\s*\|\s*(.+?)\s*\]/

function parseQuestion(text) {
  const match = text.match(QUESTION_RE)
  if (!match) return null
  return { prompt: match[1], options: [match[2], match[3]], full: match[0] }
}

function formatTime(dateStr) {
  const d = new Date(dateStr)
  if (isNaN(d.getTime())) return ''
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

function MessageBubble({ message, onSelectOption }) {
  const question = parseQuestion(message.content)
  const displayContent = question ? message.content.replace(question.full, '').trim() : message.content

  return (
    <div className={`flex gap-3 ${message.role === 'user' ? 'flex-row-reverse' : ''}`}>
      <div className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xs font-bold ${
        message.role === 'user'
          ? 'bg-sky-600 text-white'
          : 'bg-gradient-to-br from-indigo-500 to-purple-600 text-white'
      }`}>
        {message.role === 'user' ? '你' : 'AI'}
      </div>

      <div className={`min-w-0 max-w-[75%] ${message.role === 'user' ? 'items-end' : 'items-start'}`}>
        {displayContent && (
          <div className={`overflow-hidden rounded-2xl px-4 py-3 text-sm leading-7 ${
            message.role === 'user'
              ? 'bg-sky-600 text-white'
              : 'border border-slate-200 bg-white text-slate-700 shadow-sm'
          }`}>
            <div className={`prose prose-sm max-w-none break-words prose-pre:max-w-[calc(75vw-6rem)] prose-pre:overflow-x-auto prose-img:max-w-full ${
              message.role === 'user'
                ? 'prose-invert'
                : 'prose-slate prose-headings:text-slate-900'
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
                  className="rounded-xl border-2 border-sky-300 bg-white px-5 py-2 text-sm font-medium text-sky-700 shadow-sm transition hover:bg-sky-600 hover:text-white hover:border-sky-600"
                >
                  {opt}
                </button>
              ))}
            </div>
          </div>
        )}

        <div className={`mt-1 text-[10px] text-slate-400 ${message.role === 'user' ? 'text-right' : 'text-left'}`}>
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
  const messagesEnd = useRef(null)

  const { data: agent, loading: agentLoading } = useAsyncData(() => getAgentById(agentId), [agentId])
  const { data: sessions = [], loading: sessionsLoading } = useAsyncData(
    () => getAgentSessions(agentId),
    [agentId, sessionsRefresh],
  )
  const currentSessionId = activeSession ?? sessions[0]?.id
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
    // Optimistic: show user message immediately
    setMessages((current) => [...current, {
      id: tempId,
      sessionId: currentSessionId,
      agentId,
      role: 'user',
      content: text,
      createdAt: new Date().toISOString(),
    }])
    try {
      const nextMessages = await sendAgentMessage(agentId, { sessionId: currentSessionId, content: text })
      setMessages((current) => [...current.filter((m) => m.id !== tempId), ...nextMessages])
      setSessionsRefresh((n) => n + 1)
    } catch (error) {
      setSendError(error.message)
      // Remove temp message on error
      setMessages((current) => current.filter((m) => m.id !== tempId))
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

  if (agentLoading || !agent) {
    return <LoadingPanel label="正在加载 Agent 对话..." />
  }

  return (
    <div className="flex h-full flex-col">
      {/* Compact top bar */}
      <div className="flex shrink-0 items-center gap-3 border-b border-slate-200 bg-white px-4 py-2.5">
        <Link to="/" className="rounded-md border border-slate-200 px-2.5 py-1 text-xs text-slate-500 transition hover:border-slate-300 hover:text-slate-700">
          ← 所有 Agent
        </Link>
        <div className="h-4 w-px bg-slate-200" />
        <div>
          <span className="text-sm font-semibold text-slate-900">{agent.name}</span>
          <span className="ml-2 text-xs text-slate-400">{agent.model}</span>
        </div>
        <div className="ml-auto flex items-center gap-2">
          <span className="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] text-slate-500">{agent.runtime}</span>
          <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
            agent.status === '已部署' ? 'bg-emerald-50 text-emerald-700'
            : agent.status === '离线' ? 'bg-slate-100 text-slate-500'
            : 'bg-emerald-50 text-emerald-700'
          }`}>
            {agent.status}
          </span>
        </div>
      </div>

      <div className="grid min-h-0 flex-1 gap-5 xl:grid-cols-[260px_1fr]">
        <aside className="flex min-h-0 flex-col rounded-2xl border border-slate-200 bg-white shadow-sm">
          <div className="border-b border-slate-200 p-3">
            <div className="flex gap-1.5">
              <input
                type="text"
                value={newTitle}
                onChange={(e) => setNewTitle(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') handleCreateSession() }}
                placeholder="新建会话..."
                className="min-w-0 flex-1 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-sky-400"
              />
              <button
                onClick={handleCreateSession}
                disabled={creating || !newTitle.trim()}
                className="shrink-0 rounded-lg bg-sky-600 px-3 py-2 text-sm font-medium text-white hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {creating ? '...' : '+'}
              </button>
            </div>
            {createError && <div className="mt-1.5 text-xs text-rose-600">{createError}</div>}
          </div>
          <div className="flex-1 space-y-0.5 overflow-y-auto p-2">
            {sessions.map((session) => (
              <button
                key={session.id}
                onClick={() => setActiveSession(session.id)}
                className={`w-full rounded-xl px-3 py-2.5 text-left transition ${
                  currentSessionId === session.id
                    ? 'bg-sky-50 ring-1 ring-sky-200'
                    : 'hover:bg-slate-50'
                }`}
              >
                <div className={`truncate text-sm font-medium ${
                  currentSessionId === session.id ? 'text-sky-900' : 'text-slate-700'
                }`}>
                  {session.title}
                </div>
                {session.preview && (
                  <div className="mt-0.5 truncate text-xs text-slate-400">{session.preview}</div>
                )}
              </button>
            ))}
            {sessionsLoading && <div className="px-3 py-2 text-sm text-slate-400">加载中...</div>}
            {!sessionsLoading && sessions.length === 0 && (
              <div className="px-3 py-4 text-center text-xs text-slate-400">暂无会话，输入标题创建</div>
            )}
          </div>
          <div className="border-t border-slate-200 p-3">
            <div className="text-[10px] leading-relaxed text-slate-400">
              {agent.skills?.length > 0 && (
                <div className="mb-1"><span className="font-medium text-slate-500">Skills:</span> {agent.skills.join(', ')}</div>
              )}
              {agent.mcps?.length > 0 && (
                <div><span className="font-medium text-slate-500">MCPs:</span> {agent.mcps.join(', ')}</div>
              )}
            </div>
          </div>
        </aside>

        <div className="flex min-h-0 flex-col rounded-2xl border border-slate-200 bg-white shadow-sm">
          <div className="shrink-0 border-b border-slate-200 px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-8 w-8 items-center justify-center rounded-full bg-gradient-to-br from-indigo-500 to-purple-600 text-xs font-bold text-white">
                AI
              </div>
              <div>
                <div className="text-sm font-semibold text-slate-900">
                  {sessions.find((s) => s.id === currentSessionId)?.title ?? agent.name}
                </div>
                <div className="text-xs text-slate-400">{agent.model}</div>
              </div>
            </div>
          </div>

          <div className="flex-1 overflow-y-auto bg-slate-50/50 p-5">
            <div className="mx-auto max-w-4xl space-y-6">
              {messagesLoading && (
                <div className="rounded-xl border border-dashed border-slate-200 p-8 text-center text-sm text-slate-400">
                  正在加载消息...
                </div>
              )}
              {!messagesLoading && messages.length === 0 && (
                <div className="rounded-xl border border-dashed border-slate-200 p-10 text-center">
                  <div className="mb-3 text-4xl">💬</div>
                  <div className="text-sm font-medium text-slate-600">开始与 {agent.name} 对话</div>
                  <div className="mt-1 text-xs text-slate-400">输入消息并按 Ctrl+Enter 发送</div>
                </div>
              )}
              {messages.map((msg) => (
                <MessageBubble
                  key={msg.id}
                  message={msg}
                  onSelectOption={(opt) => handleSend(opt)}
                />
              ))}
              {sending && (
                <div className="flex gap-3">
                  <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-indigo-500 to-purple-600 text-xs font-bold text-white">AI</div>
                  <div className="rounded-2xl border border-slate-200 bg-white px-4 py-3 shadow-sm">
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

          <div className="shrink-0 border-t border-slate-200 p-4">
            <div className="mx-auto max-w-4xl">
              {sendError && (
                <div className="mb-3 rounded-xl border border-rose-200 bg-rose-50 px-4 py-2 text-sm text-rose-600">
                  {sendError}
                </div>
              )}
              <div className="rounded-2xl border border-slate-200 bg-white shadow-sm transition focus-within:border-sky-300 focus-within:shadow-md">
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
                  placeholder={currentSessionId ? '输入消息...' : '请先创建或选择一个会话'}
                  className="w-full resize-none rounded-t-2xl bg-transparent px-4 pt-3 text-sm text-slate-900 outline-none placeholder:text-slate-400"
                />
                <div className="flex items-center justify-between rounded-b-2xl bg-slate-50 px-4 py-2">
                  <div className="text-xs text-slate-400">
                    {currentSessionId ? 'Ctrl+Enter 发送 · 支持 Markdown' : '先在左侧创建会话'}
                  </div>
                  <button
                    onClick={() => handleSend()}
                    disabled={sending || !draft.trim() || !currentSessionId}
                    className="rounded-xl bg-sky-600 px-5 py-2 text-sm font-medium text-white transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    {sending ? (
                      <span className="flex items-center gap-1.5">
                        <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-white border-t-transparent" />
                        思考中
                      </span>
                    ) : '发送'}
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
