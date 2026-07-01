import { useState } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import Layout from './components/Layout'
import AgentChatPage from './pages/AgentChatPage'
import AgentsPage from './pages/AgentsPage'
import AiTokensPage from './pages/AiTokensPage'
import ApprovalsPage from './pages/ApprovalsPage'
import AuthTokensPage from './pages/AuthTokensPage'
import DeployPage from './pages/DeployPage'
import DeployProgressPage from './pages/DeployProgressPage'
import AuthPage from './pages/AuthPage'
import RepositoriesPage from './pages/RepositoriesPage'
import UsersPage from './pages/UsersPage'

export default function App() {
  const [authenticated, setAuthenticated] = useState(() => localStorage.getItem('agentbucket.auth') === 'true')

  const handleLogin = () => {
    localStorage.setItem('agentbucket.auth', 'true')
    setAuthenticated(true)
  }

  const handleLogout = () => {
    localStorage.removeItem('agentbucket.auth')
    localStorage.removeItem('agentbucket.token')
    setAuthenticated(false)
  }

  return (
    <Routes>
      <Route
        path="/login"
        element={authenticated ? <Navigate to="/" replace /> : <AuthPage mode="login" onAuthenticated={handleLogin} />}
      />
      <Route
        path="/register"
        element={authenticated ? <Navigate to="/" replace /> : <AuthPage mode="register" />}
      />
      <Route
        path="/*"
        element={
          authenticated ? (
          <Layout onLogout={handleLogout}>
            <Routes>
              <Route path="/" element={<AgentsPage />} />
              <Route path="/deploy" element={<DeployPage />} />
              <Route path="/deploy/progress" element={<DeployProgressPage />} />
              <Route path="/agents/:agentId" element={<AgentChatPage />} />
              <Route path="/users" element={<UsersPage />} />
              <Route path="/approvals" element={<ApprovalsPage />} />
              <Route path="/repositories" element={<RepositoriesPage />} />
              <Route path="/ai-tokens" element={<AiTokensPage />} />
              <Route path="/auth-tokens" element={<AuthTokensPage />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </Layout>
          ) : (
            <Navigate to="/login" replace />
          )
        }
      />
    </Routes>
  )
}
