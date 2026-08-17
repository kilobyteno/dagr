import {
  ArrowLeftIcon,
  MonitorIcon,
  MoonIcon,
  PlusIcon,
  SunIcon,
  TrashIcon,
} from '@phosphor-icons/react'
import { useTheme } from 'next-themes'
import { useEffect, useMemo, useState } from 'react'

import { useTrustedDomains } from '@/components/chat/trusted-domains'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ButtonGroup } from '@/components/ui/button-group'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Switch } from '@/components/ui/switch'
import { useAppPreferences } from '@/lib/app-preferences'
import { normalizeDomain } from '@/lib/trusted-domains'
import { cn } from '@/lib/utils'

const THEME_OPTIONS = [
  { value: 'light', label: 'Light', icon: SunIcon },
  { value: 'dark', label: 'Dark', icon: MoonIcon },
  { value: 'system', label: 'System', icon: MonitorIcon },
] as const

export function AppSettingsPage({ onBack }: { onBack: () => void }) {
  const { theme, setTheme } = useTheme()
  const { preferences, setPreference } = useAppPreferences()
  const {
    userDomains,
    workspaceDomains,
    trustDomain,
    untrustDomain,
  } = useTrustedDomains()
  const [mounted, setMounted] = useState(false)
  const [newDomain, setNewDomain] = useState('')
  const [domainError, setDomainError] = useState<string | null>(null)

  useEffect(() => {
    setMounted(true)
  }, [])

  const activeTheme = mounted ? (theme ?? 'system') : 'system'
  const workspaceDomainSet = useMemo(
    () => new Set(workspaceDomains.map((item) => item.domain)),
    [workspaceDomains],
  )
  const manualDomains = userDomains.filter(
    (domain) => !workspaceDomainSet.has(domain),
  )

  const addDomain = () => {
    const normalized = normalizeDomain(newDomain)
    if (!normalized || !normalized.includes('.')) {
      setDomainError('Enter a valid domain such as example.com')
      return
    }
    trustDomain(normalized)
    setNewDomain('')
    setDomainError(null)
  }

  return (
    <div className="flex h-full min-h-0 min-w-0 flex-1 flex-col bg-background">
      <header className="flex h-14 shrink-0 items-center gap-2 border-b px-3">
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label="Back to chat"
          onClick={onBack}
        >
          <ArrowLeftIcon strokeWidth={2} data-icon />
        </Button>
        <div className="flex min-w-0 flex-1 flex-col gap-0.5">
          <h1 className="truncate text-sm font-semibold">Settings</h1>
          <p className="truncate text-xs text-muted-foreground">
            Preferences for this app
          </p>
        </div>
      </header>

      <ScrollArea className="min-h-0 flex-1">
        <div className="mx-auto flex w-full max-w-xl flex-col gap-8 p-6">
          <section className="flex flex-col gap-4">
            <div className="flex flex-col gap-1">
              <h2 className="text-sm font-semibold">Appearance</h2>
              <p className="text-sm text-muted-foreground">
                Choose how Dagr looks on this device.
              </p>
            </div>

            <ButtonGroup
              aria-label="Theme"
              className="w-full [&>button]:flex-1"
            >
              {THEME_OPTIONS.map((option) => {
                const Icon = option.icon
                const selected = activeTheme === option.value
                return (
                  <Button
                    key={option.value}
                    type="button"
                    variant={selected ? 'default' : 'outline'}
                    aria-pressed={selected}
                    onClick={() => setTheme(option.value)}
                    className={cn(!selected && 'bg-background')}
                  >
                    <Icon strokeWidth={2} data-icon="inline-start" />
                    {option.label}
                  </Button>
                )
              })}
            </ButtonGroup>
          </section>

          <section className="flex flex-col gap-4">
            <div className="flex flex-col gap-1">
              <h2 className="text-sm font-semibold">Media</h2>
              <p className="text-sm text-muted-foreground">
                Control how animated images play in chat.
              </p>
            </div>

            <div className="flex items-center justify-between gap-4 rounded-md border px-3 py-3">
              <div className="flex min-w-0 flex-col gap-0.5">
                <Label htmlFor="gifs-on-hover-only" className="text-sm font-medium">
                  GIFs on hover only
                </Label>
                <p className="text-xs text-muted-foreground">
                  Keep GIFs frozen until you hover them.
                </p>
              </div>
              <Switch
                id="gifs-on-hover-only"
                checked={preferences.gifsOnHoverOnly}
                onCheckedChange={(checked) =>
                  setPreference('gifsOnHoverOnly', checked)
                }
              />
            </div>
          </section>

          <section className="flex flex-col gap-4">
            <div className="flex flex-col gap-1">
              <h2 className="text-sm font-semibold">Trusted domains</h2>
              <p className="text-sm text-muted-foreground">
                Links from these domains open without asking. Verified workspace
                domains are trusted automatically.
              </p>
            </div>

            <div className="flex flex-col gap-2">
              <Label htmlFor="trusted-domain-input">Add domain</Label>
              <form
                className="flex gap-2"
                onSubmit={(event) => {
                  event.preventDefault()
                  addDomain()
                }}
              >
                <Input
                  id="trusted-domain-input"
                  value={newDomain}
                  onChange={(event) => {
                    setNewDomain(event.target.value)
                    if (domainError) setDomainError(null)
                  }}
                  placeholder="example.com"
                  autoCapitalize="none"
                  autoCorrect="off"
                  spellCheck={false}
                />
                <Button type="submit" variant="outline">
                  <PlusIcon strokeWidth={2} data-icon="inline-start" />
                  Add
                </Button>
              </form>
              {domainError ? (
                <p className="text-xs text-destructive">{domainError}</p>
              ) : null}
            </div>

            {workspaceDomains.length === 0 && manualDomains.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No trusted domains yet. Verified workspace domains appear here
                automatically, or add one above.
              </p>
            ) : (
              <ul className="flex flex-col gap-2">
                {workspaceDomains.map((entry) => (
                  <li
                    key={`workspace:${entry.domain}`}
                    className="flex items-center gap-3 rounded-md border px-3 py-2"
                  >
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">
                        {entry.domain}
                      </p>
                      <p className="truncate text-xs text-muted-foreground">
                        From {entry.workspaceNames.join(', ')}
                      </p>
                    </div>
                    <Badge variant="secondary">Workspace</Badge>
                  </li>
                ))}
                {manualDomains.map((domain) => (
                  <li
                    key={`user:${domain}`}
                    className="flex items-center gap-3 rounded-md border px-3 py-2"
                  >
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">{domain}</p>
                      <p className="truncate text-xs text-muted-foreground">
                        Added on this device
                      </p>
                    </div>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-sm"
                      aria-label={`Remove ${domain}`}
                      onClick={() => untrustDomain(domain)}
                    >
                      <TrashIcon strokeWidth={2} data-icon />
                    </Button>
                  </li>
                ))}
              </ul>
            )}
          </section>
        </div>
      </ScrollArea>
    </div>
  )
}
