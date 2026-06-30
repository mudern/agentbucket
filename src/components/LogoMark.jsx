const markLogo = '/assets/agentbucket-mark-cutout.png'

export default function LogoMark({ compact = false }) {
  if (compact) {
    return <img src={markLogo} alt="AgentBucket mark" className="h-12 w-12 object-contain" />
  }

  return (
    <div className="flex min-w-0 items-center gap-3">
      <img src={markLogo} alt="AgentBucket mark" className="h-12 w-12 shrink-0 object-contain" />
      <div className="min-w-0 overflow-visible">
        <div className="whitespace-nowrap text-lg font-semibold leading-7 text-slate-950">AgentBucket</div>
      </div>
    </div>
  )
}
