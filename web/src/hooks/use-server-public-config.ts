import { useEffect, useState } from 'react'

import { AUTH_DEFAULTS } from '@/lib/auth-defaults'
import { isServerUnavailable } from '@/lib/api/client'
import {
  fetchServerPublicConfig,
  type ServerPublicConfig,
} from '@/lib/server-config'

const FALLBACK: ServerPublicConfig = {
  passwordPolicy: { ...AUTH_DEFAULTS.passwordPolicy },
}

export function useServerPublicConfig(serverUrl: string) {
  const [config, setConfig] = useState<ServerPublicConfig>(FALLBACK)
  const [loading, setLoading] = useState(false)
  const [unreachable, setUnreachable] = useState(false)

  useEffect(() => {
    const controller = new AbortController()
    const trimmed = serverUrl.trim()
    if (!trimmed) {
      setConfig(FALLBACK)
      setUnreachable(false)
      setLoading(false)
      return
    }

    const timer = window.setTimeout(() => {
      setLoading(true)
      void fetchServerPublicConfig(trimmed, controller.signal)
        .then((next) => {
          if (controller.signal.aborted) return
          setConfig(next)
          setUnreachable(false)
        })
        .catch((error: unknown) => {
          if (controller.signal.aborted) return
          setConfig(FALLBACK)
          setUnreachable(isServerUnavailable(error))
        })
        .finally(() => {
          if (!controller.signal.aborted) setLoading(false)
        })
    }, 300)

    return () => {
      controller.abort()
      window.clearTimeout(timer)
    }
  }, [serverUrl])

  return { config, loading, unreachable }
}
