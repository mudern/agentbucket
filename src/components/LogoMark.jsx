const mascotPng = '/agentbucket-logo-mark-transparent.png'
const logoSvg = '/agentbucket-logo-mark.svg'

export default function LogoMark({ compact = false }) {
  if (compact) {
    return <img src={mascotPng} alt="AgentBucket" className="h-10 w-10 object-contain" />
  }

  return (
    <div className="flex min-w-0 items-center gap-2.5">
      <img src={mascotPng} alt="AgentBucket" className="h-10 w-10 shrink-0 object-contain" />
      <img src={logoSvg} alt="AgentBucket" className="h-8 shrink-0 object-contain" />
    </div>
  )
}
