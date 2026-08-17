import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type MouseEvent,
  type ReactNode,
} from 'react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { listDomains } from '@/lib/api/domains'
import {
  addTrustedDomain,
  hostFromHref,
  isHostTrusted,
  normalizeDomain,
  readTrustedDomains,
  removeTrustedDomain,
  writeTrustedDomains,
} from '@/lib/trusted-domains'
import { cn } from '@/lib/utils'

type TrustedDomainSource = 'user' | 'workspace'

export type TrustedDomainEntry = {
  domain: string
  source: TrustedDomainSource
  workspaceNames: string[]
}

type PendingLink = {
  url: string
  host: string
}

type TrustedDomainsContextValue = {
  userDomains: string[]
  workspaceDomains: TrustedDomainEntry[]
  allTrusted: string[]
  isHrefTrusted: (href: string) => boolean
  requestOpen: (href: string) => void
  trustDomain: (domain: string) => void
  untrustDomain: (domain: string) => void
  setUserDomains: (domains: string[]) => void
}

const TrustedDomainsContext = createContext<TrustedDomainsContextValue | null>(
  null,
)

function openExternal(url: string) {
  window.open(url, '_blank', 'noopener,noreferrer')
}

export function TrustedDomainsProvider({
  serverUrl,
  token,
  workspaces,
  children,
}: {
  serverUrl?: string
  token?: string
  workspaces: readonly { id: string; name: string }[]
  children: ReactNode
}) {
  const [userDomains, setUserDomainsState] = useState<string[]>(() =>
    readTrustedDomains(),
  )
  const [workspaceDomainMap, setWorkspaceDomainMap] = useState<
    Map<string, string[]>
  >(() => new Map())
  const [pending, setPending] = useState<PendingLink | null>(null)

  useEffect(() => {
    setUserDomainsState(readTrustedDomains())
  }, [])

  useEffect(() => {
    if (!serverUrl || !token || workspaces.length === 0) {
      setWorkspaceDomainMap(new Map())
      return
    }
    const controller = new AbortController()
    void (async () => {
      const next = new Map<string, string[]>()
      await Promise.all(
        workspaces.map(async (workspace) => {
          try {
            const result = await listDomains(
              serverUrl,
              token,
              workspace.id,
              controller.signal,
            )
            if (controller.signal.aborted) return
            const verified = result.domains
              .filter((item) => item.verified)
              .map((item) => normalizeDomain(item.domain))
              .filter(Boolean)
            if (verified.length > 0) {
              next.set(workspace.id, verified)
            }
          } catch {
            // Workspace domain list is best-effort for link trust.
          }
        }),
      )
      if (!controller.signal.aborted) {
        setWorkspaceDomainMap(next)
      }
    })()
    return () => controller.abort()
  }, [serverUrl, token, workspaces])

  const workspaceDomains = useMemo(() => {
    const byDomain = new Map<string, string[]>()
    for (const workspace of workspaces) {
      const domains = workspaceDomainMap.get(workspace.id) ?? []
      for (const domain of domains) {
        const names = byDomain.get(domain) ?? []
        if (!names.includes(workspace.name)) {
          names.push(workspace.name)
        }
        byDomain.set(domain, names)
      }
    }
    return [...byDomain.entries()]
      .map(([domain, workspaceNames]) => ({
        domain,
        source: 'workspace' as const,
        workspaceNames: workspaceNames.sort(),
      }))
      .sort((a, b) => a.domain.localeCompare(b.domain))
  }, [workspaceDomainMap, workspaces])

  const allTrusted = useMemo(() => {
    const set = new Set<string>(userDomains)
    for (const entry of workspaceDomains) {
      set.add(entry.domain)
    }
    return [...set]
  }, [userDomains, workspaceDomains])

  const isHrefTrusted = (href: string) => {
    const host = hostFromHref(href)
    if (!host) return false
    return isHostTrusted(host, allTrusted)
  }

  const requestOpen = (href: string) => {
    const trimmed = href.trim()
    if (!trimmed) return
    const lower = trimmed.toLowerCase()
    if (lower.startsWith('mailto:')) {
      openExternal(trimmed)
      return
    }
    const host = hostFromHref(trimmed)
    if (!host) return
    if (isHostTrusted(host, allTrusted)) {
      openExternal(trimmed)
      return
    }
    setPending({ url: trimmed, host })
  }

  const trustDomain = (domain: string) => {
    setUserDomainsState(addTrustedDomain(domain))
  }

  const untrustDomain = (domain: string) => {
    setUserDomainsState(removeTrustedDomain(domain))
  }

  const setUserDomains = (domains: string[]) => {
    setUserDomainsState(writeTrustedDomains(domains))
  }

  const value: TrustedDomainsContextValue = {
    userDomains,
    workspaceDomains,
    allTrusted,
    isHrefTrusted,
    requestOpen,
    trustDomain,
    untrustDomain,
    setUserDomains,
  }

  return (
    <TrustedDomainsContext.Provider value={value}>
      {children}
      <Dialog
        open={pending !== null}
        onOpenChange={(open) => {
          if (!open) setPending(null)
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Open external link?</DialogTitle>
            <DialogDescription>
              This domain is not on your trusted list. Only continue if you trust
              the destination.
            </DialogDescription>
          </DialogHeader>
          {pending ? (
            <div className="flex flex-col gap-2 rounded-md border bg-muted/40 px-3 py-2">
              <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                {pending.host}
              </p>
              <p className="break-all text-sm text-foreground">{pending.url}</p>
            </div>
          ) : null}
          <DialogFooter className="gap-2 sm:gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => setPending(null)}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                if (!pending) return
                const { url, host } = pending
                trustDomain(host)
                setPending(null)
                openExternal(url)
              }}
            >
              Always trust
            </Button>
            <Button
              type="button"
              onClick={() => {
                if (!pending) return
                const { url } = pending
                setPending(null)
                openExternal(url)
              }}
            >
              Open once
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </TrustedDomainsContext.Provider>
  )
}

export function useTrustedDomains() {
  const value = useContext(TrustedDomainsContext)
  if (!value) {
    throw new Error('useTrustedDomains must be used within TrustedDomainsProvider')
  }
  return value
}

export function SafeExternalLink({
  href,
  className,
  children,
}: {
  href: string
  className?: string
  children: ReactNode
}) {
  const { requestOpen, isHrefTrusted } = useTrustedDomains()
  const trusted = isHrefTrusted(href)

  const onClick = (event: MouseEvent<HTMLAnchorElement>) => {
    event.preventDefault()
    requestOpen(href)
  }

  return (
    <a
      href={href}
      target="_blank"
      rel="noreferrer noopener"
      onClick={onClick}
      className={cn(className)}
      data-trusted-link={trusted ? 'true' : 'false'}
    >
      {children}
    </a>
  )
}
