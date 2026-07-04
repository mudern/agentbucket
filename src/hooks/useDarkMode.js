import { useEffect, useState } from 'react'

export default function useDarkMode() {
  const [dark, setDark] = useState(() => {
    const s = localStorage.getItem('agentbucket.dark')
    if (s !== null) return s === 'true'
    return window.matchMedia('(prefers-color-scheme: dark)').matches
  })

  useEffect(() => {
    document.documentElement.classList.toggle('dark', dark)
    localStorage.setItem('agentbucket.dark', dark.toString())
  }, [dark])

  return [dark, () => setDark((d) => !d)]
}
