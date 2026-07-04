export const API_BASE = import.meta.env.VITE_API_BASE ?? 'http://127.0.0.1:8080'

function getToken() {
  return localStorage.getItem('agentbucket.token') || sessionStorage.getItem('agentbucket.token') || ''
}

async function request(path, options = {}) {
  const token = getToken()
  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers ?? {}),
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  const response = await fetch(`${API_BASE}${path}`, { headers, ...options })

  if (response.status === 401 || response.status === 403) {
    localStorage.removeItem('agentbucket.auth')
    localStorage.removeItem('agentbucket.token')
    // Redirect to login (avoid redirect loop if already on login)
    if (!window.location.pathname.includes('/login')) {
      window.location.href = '/login'
    }
    throw new Error('Session expired, please login again')
  }
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

export async function getRepositories() {
  return request('/api/repositories')
}

export async function createRepository(data) {
  return request('/api/repositories', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export async function listBranches(url) {
  return request('/api/repositories/branches', {
    method: 'POST',
    body: JSON.stringify({ url }),
  })
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

export async function renameSession(agentId, sessionId, title) {
  return request(`/api/agents/${agentId}/sessions/${sessionId}`, {
    method: 'PATCH',
    body: JSON.stringify({ title }),
  })
}

export async function patchRepository(id, updates) {
  return request(`/api/repositories/${id}`, { method: 'PATCH', body: JSON.stringify(updates) })
}

export async function patchAIToken(id, updates) {
  return request(`/api/ai-tokens/${id}`, { method: 'PATCH', body: JSON.stringify(updates) })
}

export async function patchAuthToken(id, updates) {
  return request(`/api/auth-tokens/${id}`, { method: 'PATCH', body: JSON.stringify(updates) })
}

export async function getDeployments() {
  return request('/api/deployments')
}

export async function stopDeployment(id) {
  return request(`/api/deployments/${id}/stop`, { method: 'POST' })
}

export async function startDeployment(id) {
  return request(`/api/deployments/${id}/start`, { method: 'POST' })
}

export async function deleteDeployment(id) {
  return request(`/api/deployments/${id}`, { method: 'DELETE' })
}

export async function createAiToken(data) {
  return request('/api/ai-tokens', { method: 'POST', body: JSON.stringify(data) })
}

export async function createAuthToken(data) {
  return request('/api/auth-tokens', { method: 'POST', body: JSON.stringify(data) })
}

export async function patchUser(id, updates) {
  return request(`/api/users/${id}`, { method: 'PATCH', body: JSON.stringify(updates) })
}

export async function approveApproval(id) {
  return request(`/api/approvals/${id}/approve`, { method: 'POST', body: JSON.stringify({ action: 'approve' }) })
}

export async function rejectApproval(id) {
  return request(`/api/approvals/${id}/reject`, { method: 'POST', body: JSON.stringify({ action: 'reject' }) })
}

export async function createApproval(data) {
  return request('/api/approvals', { method: 'POST', body: JSON.stringify(data) })
}
