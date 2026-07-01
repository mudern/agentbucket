const API_BASE = import.meta.env.VITE_API_BASE ?? 'http://127.0.0.1:8080'

async function request(path, options = {}) {
  const response = await fetch(`${API_BASE}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers ?? {}),
    },
    ...options,
  })

  if (!response.ok) {
    const body = await response.json().catch(() => ({}))
    throw new Error(body.error ?? `Request failed: ${response.status}`)
  }

  return response.json()
}

export async function getCurrentUser() {
  return request('/api/current-user')
}

export async function getAgents() {
  return request('/api/agents')
}

export async function getAgentById(agentId) {
  const agents = await getAgents()
  return agents.find((item) => item.id === agentId) ?? agents[0]
}

export async function getAgentMessages() {
  return []
}

export async function getAgentSessions(agentId) {
  return request(`/api/agents/${agentId}/sessions`)
}

export async function getAgentSessionMessages(agentId, sessionId) {
  const query = sessionId ? `?sessionId=${encodeURIComponent(sessionId)}` : ''
  return request(`/api/agents/${agentId}/messages${query}`)
}

export async function sendAgentMessage(agentId, payload) {
  return request(`/api/agents/${agentId}/messages`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function createAgentSession(agentId, payload) {
  return request(`/api/agents/${agentId}/sessions`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function getUsers() {
  return request('/api/users')
}

export async function getApprovals() {
  return request('/api/approvals')
}

export async function getAiTokens() {
  return request('/api/ai-tokens')
}

export async function getAuthTokens() {
  return request('/api/auth-tokens')
}

export async function getDeployOptions() {
  return request('/api/deploy-options')
}

export async function createDeployment(payload) {
  return request('/api/deployments', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function deleteRepository(id) {
  return request(`/api/repositories/${id}`, { method: 'DELETE' })
}

export async function deleteAIToken(id) {
  return request(`/api/ai-tokens/${id}`, { method: 'DELETE' })
}

export async function deleteAuthToken(id) {
  return request(`/api/auth-tokens/${id}`, { method: 'DELETE' })
}

export async function deleteSession(agentId, sessionId) {
  return request(`/api/agents/${agentId}/sessions/${sessionId}`, { method: 'DELETE' })
}
