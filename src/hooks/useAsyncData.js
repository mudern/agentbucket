import { useEffect, useState } from 'react'

export default function useAsyncData(loader, deps = []) {
  const [data, setData] = useState(undefined)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    let alive = true

    async function run() {
      setLoading(true)
      setError(null)
      try {
        const result = await loader()
        if (alive) {
          setData(result)
        }
      } catch (err) {
        if (alive) {
          setError(err)
        }
      } finally {
        if (alive) {
          setLoading(false)
        }
      }
    }

    run()

    return () => {
      alive = false
    }
  }, deps)

  return { data, loading, error }
}
