import {
  ArrowSquareOutIcon,
  MonitorIcon,
  MoonIcon,
  PlusIcon,
  SunIcon,
  TrashIcon,
} from '@phosphor-icons/react'
import { useTheme } from 'next-themes'
import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'

import {
  APP_SETTINGS_PAGES,
  appSettingsPageLabel,
  type AppSettingsPageId,
} from '@/components/chat/app-settings-nav'
import { useTrustedDomains } from '@/components/chat/trusted-domains'
import { UserAvatarMark } from '@/components/chat/user-avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ButtonGroup } from '@/components/ui/button-group'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { updateProfile } from '@/lib/api/auth'
import { formatUserError } from '@/lib/api/client'
import {
  UPDATE_CHANNELS,
  isUpdateChannel,
  useAppPreferences,
  type UpdateChannel,
} from '@/lib/app-preferences'
import { useAuth } from '@/lib/auth'
import { applyLocale, useLocale } from '@/lib/i18n'
import { useDesktopUpdate } from '@/lib/updates'
import {
  APP_LOCALES,
  LOCALE_LABELS,
  isAppLocale,
  type AppLocale,
} from '@/lib/i18n/locales'
import { normalizeDomain } from '@/lib/trusted-domains'
import { cn } from '@/lib/utils'

const THEME_OPTIONS = [
  { value: 'light', labelKey: 'settings.appearance.light', icon: SunIcon },
  { value: 'dark', labelKey: 'settings.appearance.dark', icon: MoonIcon },
  { value: 'system', labelKey: 'settings.appearance.system', icon: MonitorIcon },
] as const

const UPDATE_CHANNEL_LABELS: Record<
  UpdateChannel,
  'settings.updates.stable' | 'settings.updates.prerelease'
> = {
  stable: 'settings.updates.stable',
  prerelease: 'settings.updates.prerelease',
}

function AppearanceSection() {
  const { t } = useLocale()
  const { theme, setTheme } = useTheme()
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true)
  }, [])

  const activeTheme = mounted ? (theme ?? 'system') : 'system'

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h2 className="text-sm font-semibold">{t('settings.appearance.title')}</h2>
        <p className="text-sm text-muted-foreground">
          {t('settings.appearance.description')}
        </p>
      </div>

      <ButtonGroup
        aria-label={t('settings.appearance.theme')}
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
              {t(option.labelKey)}
            </Button>
          )
        })}
      </ButtonGroup>
    </section>
  )
}

function LanguageSection() {
  const { t, locale } = useLocale()
  const { session, signIn } = useAuth()

  const changeLocale = (next: AppLocale) => {
    if (next === locale) return
    applyLocale(next)
    if (!session) return
    void updateProfile(session.serverUrl, session.token, {
      displayName: session.displayName,
      notificationLevel: session.notificationLevel,
      locale: next,
    })
      .then((result) => {
        signIn({
          ...session,
          displayName: result.user.displayName,
          notificationLevel: result.user.notificationLevel,
          locale: next,
        })
      })
      .catch((err) => {
        toast.error(formatUserError(err, t('settings.language.error')))
      })
  }

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h2 className="text-sm font-semibold">{t('settings.language.title')}</h2>
        <p className="text-sm text-muted-foreground">
          {t('settings.language.description')}
        </p>
      </div>

      <Select
        value={locale}
        onValueChange={(value) => {
          if (isAppLocale(value)) changeLocale(value)
        }}
      >
        <SelectTrigger
          className="w-full"
          aria-label={t('settings.language.label')}
        >
          <SelectValue />
        </SelectTrigger>
        <SelectContent position="popper" align="start">
          <SelectGroup>
            {APP_LOCALES.map((option) => (
              <SelectItem key={option} value={option}>
                {LOCALE_LABELS[option]}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    </section>
  )
}

function MediaSection() {
  const { t } = useLocale()
  const { preferences, setPreference } = useAppPreferences()

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h2 className="text-sm font-semibold">{t('settings.media.title')}</h2>
        <p className="text-sm text-muted-foreground">
          {t('settings.media.description')}
        </p>
      </div>

      <div className="flex items-center justify-between gap-4 rounded-md border px-3 py-3">
        <div className="flex min-w-0 flex-col gap-0.5">
          <Label htmlFor="gifs-on-hover-only" className="text-sm font-medium">
            {t('settings.media.gifsOnHover')}
          </Label>
          <p className="text-xs text-muted-foreground">
            {t('settings.media.gifsOnHoverHint')}
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
  )
}

function UpdatesSection() {
  const { t } = useLocale()
  const { preferences, setPreference } = useAppPreferences()
  const {
    status: updateStatus,
    checking: updateChecking,
    check: checkForUpdates,
    openUpdate,
  } = useDesktopUpdate()

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h2 className="text-sm font-semibold">{t('settings.updates.title')}</h2>
        <p className="text-sm text-muted-foreground">
          {t('settings.updates.description')}
        </p>
      </div>

      <div className="flex flex-col gap-3 rounded-md border px-3 py-3">
        <div className="flex flex-col gap-2">
          <Label className="text-sm font-medium">
            {t('settings.updates.channel')}
          </Label>
          <p className="text-xs text-muted-foreground">
            {t('settings.updates.channelHint')}
          </p>
          <ButtonGroup
            aria-label={t('settings.updates.channel')}
            className="w-full [&>button]:flex-1"
          >
            {UPDATE_CHANNELS.map((channel) => {
              const selected = preferences.updateChannel === channel
              return (
                <Button
                  key={channel}
                  type="button"
                  variant={selected ? 'default' : 'outline'}
                  aria-pressed={selected}
                  onClick={() => {
                    if (!isUpdateChannel(channel)) return
                    setPreference('updateChannel', channel)
                  }}
                  className={cn(!selected && 'bg-background')}
                >
                  {t(UPDATE_CHANNEL_LABELS[channel])}
                </Button>
              )
            })}
          </ButtonGroup>
        </div>
        <p className="text-sm text-muted-foreground">
          {updateStatus?.currentVersion
            ? t('settings.updates.current', {
                version: updateStatus.currentVersion,
              })
            : t('settings.updates.currentUnknown')}
        </p>
        {updateStatus?.skipped ? (
          <p className="text-sm text-muted-foreground">
            {t('settings.updates.skipped')}
          </p>
        ) : updateStatus?.error ? (
          <p className="text-sm text-destructive">
            {t('settings.updates.error')}
          </p>
        ) : updateStatus?.available && updateStatus.latestVersion ? (
          <p className="text-sm">
            {t('settings.updates.available', {
              version: updateStatus.latestVersion,
            })}
          </p>
        ) : updateStatus?.latestVersion && !updateChecking ? (
          <p className="text-sm text-muted-foreground">
            {t('settings.updates.upToDate')}
          </p>
        ) : !updateChecking ? (
          <p className="text-sm text-muted-foreground">
            {preferences.updateChannel === 'prerelease'
              ? t('settings.updates.nonePrerelease')
              : t('settings.updates.noneStable')}
          </p>
        ) : null}
        <div className="flex flex-wrap gap-2">
          <Button
            type="button"
            variant="outline"
            disabled={updateChecking}
            onClick={() => void checkForUpdates(true)}
          >
            {updateChecking
              ? t('settings.updates.checking')
              : t('settings.updates.check')}
          </Button>
          {updateStatus?.available ? (
            <Button type="button" onClick={() => void openUpdate()}>
              <ArrowSquareOutIcon strokeWidth={2} data-icon="inline-start" />
              {t('settings.updates.download')}
            </Button>
          ) : null}
        </div>
      </div>
    </section>
  )
}

function TrustedDomainsSection() {
  const { t } = useLocale()
  const {
    userDomains,
    workspaceDomains,
    trustDomain,
    untrustDomain,
  } = useTrustedDomains()
  const [newDomain, setNewDomain] = useState('')
  const [domainError, setDomainError] = useState<string | null>(null)
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
      setDomainError(t('settings.trustedDomains.invalid'))
      return
    }
    trustDomain(normalized)
    setNewDomain('')
    setDomainError(null)
  }

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h2 className="text-sm font-semibold">
          {t('settings.trustedDomains.title')}
        </h2>
        <p className="text-sm text-muted-foreground">
          {t('settings.trustedDomains.description')}
        </p>
      </div>

      <div className="flex flex-col gap-2">
        <Label htmlFor="trusted-domain-input">
          {t('settings.trustedDomains.addLabel')}
        </Label>
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
            {t('common.add')}
          </Button>
        </form>
        {domainError ? (
          <p className="text-xs text-destructive">{domainError}</p>
        ) : null}
      </div>

      {workspaceDomains.length === 0 && manualDomains.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          {t('settings.trustedDomains.empty')}
        </p>
      ) : (
        <ul className="flex flex-col gap-2">
          {workspaceDomains.map((entry) => (
            <li
              key={`workspace:${entry.domain}`}
              className="flex items-center gap-3 rounded-md border px-3 py-2"
            >
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">{entry.domain}</p>
                <p className="truncate text-xs text-muted-foreground">
                  {t('settings.trustedDomains.fromWorkspaces', {
                    names: entry.workspaceNames.join(', '),
                  })}
                </p>
              </div>
              <Badge variant="secondary">
                {t('settings.trustedDomains.workspaceBadge')}
              </Badge>
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
                  {t('settings.trustedDomains.addedOnDevice')}
                </p>
              </div>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                aria-label={t('settings.trustedDomains.remove', { domain })}
                onClick={() => untrustDomain(domain)}
              >
                <TrashIcon strokeWidth={2} data-icon />
              </Button>
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}

export function AppSettingsPage({
  page,
  onSelectPage,
}: {
  page: AppSettingsPageId
  onSelectPage: (page: AppSettingsPageId) => void
}) {
  const { t } = useLocale()
  const { session } = useAuth()

  return (
    <div className="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-background">
      <header className="flex h-14 shrink-0 items-center gap-2 border-b px-3">
        <div className="hidden min-w-0 flex-1 flex-col gap-0.5 md:flex">
          <h1 className="truncate text-sm font-semibold">
            {appSettingsPageLabel(page, t)}
          </h1>
          <p className="truncate text-xs text-muted-foreground">
            {t('settings.subtitle')}
          </p>
        </div>
        <ButtonGroup
          aria-label={t('settings.title')}
          className="min-w-0 flex-1 md:hidden [&>button]:flex-1"
        >
          {APP_SETTINGS_PAGES.map((item) => {
            const selected = page === item.id
            return (
              <Button
                key={item.id}
                type="button"
                size="sm"
                variant={selected ? 'default' : 'outline'}
                aria-pressed={selected}
                onClick={() => onSelectPage(item.id)}
                className={cn(!selected && 'bg-background')}
              >
                {t(item.labelKey)}
              </Button>
            )
          })}
        </ButtonGroup>
        <UserAvatarMark
          userId={session?.userId ?? ''}
          name={session?.displayName ?? ''}
          hasAvatar={session?.hasAvatar}
          avatarUpdatedAt={session?.avatarUpdatedAt}
          serverUrl={session?.serverUrl}
          token={session?.token}
          className="size-8 shrink-0 rounded-md after:rounded-md"
          imageClassName="rounded-md"
          fallbackClassName="rounded-md bg-primary text-xs font-semibold text-primary-foreground"
        />
      </header>

      <ScrollArea className="min-h-0 flex-1">
        <div className="mx-auto flex w-full max-w-xl flex-col gap-8 p-6">
          {page === 'appearance' ? <AppearanceSection /> : null}
          {page === 'language' ? <LanguageSection /> : null}
          {page === 'media' ? <MediaSection /> : null}
          {page === 'updates' ? <UpdatesSection /> : null}
          {page === 'trusted-domains' ? <TrustedDomainsSection /> : null}
        </div>
      </ScrollArea>
    </div>
  )
}
