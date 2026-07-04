export const roles = {
  super_admin: '超级管理员',
  admin: '管理员',
  user: '普通用户',
}

export const navGroups = [
  {
    title: 'Agent',
    items: [
      { label: '部署 Agent', path: '/deploy', roles: ['admin', 'super_admin'] },
      { label: '所有 Agent', path: '/agents', roles: ['user', 'admin', 'super_admin'] },
      { label: '部署进度', path: '/deploy/progress', roles: ['admin', 'super_admin', 'user'] },
    ],
  },
  {
    title: '管理',
    items: [
      { label: '用户权限', path: '/users', roles: ['super_admin'] },
      { label: '审批中心', path: '/approvals', roles: ['admin', 'super_admin'] },
      { label: '仓库管理', path: '/repositories', roles: ['admin', 'super_admin'] },
      { label: 'AI Token', path: '/ai-tokens', roles: ['admin', 'super_admin'] },
      { label: '鉴权 Token', path: '/auth-tokens', roles: ['admin', 'super_admin'] },
    ],
  },
]
