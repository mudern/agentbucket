# AgentBucket Frontend Process

## 2026-06-28

### 已完成
- 初始化 React + Vite 前端骨架。
- 切换项目包管理约定为 `pnpm`。
- 配置 Tailwind CSS 与全局暗色风格。
- 搭建左侧贯通侧栏布局，并接入现有两个 logo 资源。
- 实现按角色显示的导航结构：部署 Agent、所有 Agent、用户权限、审批中心、AI Token、鉴权 Token。
- 抽离前端 `API` 层，统一通过 mock 异步接口返回假数据，便于后续替换真实后端。
- 实现所有 Agent 页面：
  - Agent 卡片
  - 状态点显示
  - Tag 展示
  - 搜索
  - 按 Tag 过滤
- 实现 Agent 对话详情页，界面风格参考 GPT / Gemini / DeepSeek。
- 实现部署 Agent 页面原型。
- 实现超级管理员用户权限页原型。
- 实现审批中心页面原型。
- 实现 AI Token 管理页面原型。
- 实现鉴权 Token 管理页面原型。
- 补充项目 README 使用说明。

### 当前状态
- 前端静态页面与交互原型已完成。
- 当前已使用仓库中的 `agentbucket-logo-full.png` 与 `agentbucket-logo-mark.png`。
- 目前页面统一走 `src/api/index.js` 的 mock 接口，尚未接入后端接口。
- 按要求统一使用 `pnpm` 作为包管理与启动方式。

### 下一步
- 修复 `pnpm` 本地 store 与依赖安装问题后运行页面进行视觉检查。
- 接入真实登录态、角色鉴权与后端 API。
- 继续补充移动端适配与表单提交逻辑。
