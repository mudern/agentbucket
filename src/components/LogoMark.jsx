const logoSvg = '/agentbucket-logo-mark.svg'

export default function LogoMark({ compact = false }) {
  if (compact) {
    return <img src={logoSvg} alt="AgentBucket" className="h-10 object-contain" />
  }

  return (
    <div className="flex min-w-0 items-center">
      <img src={logoSvg} alt="AgentBucket" className="h-10 shrink-0 object-contain" />
    </div>
  )
}
