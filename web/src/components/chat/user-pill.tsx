import {
  BellIcon,
  EnvelopeSimpleIcon,
  GearIcon,
  PlusIcon,
  SignOutIcon,
  SmileyIcon,
  UserCircleIcon,
  UserPlusIcon,
} from '@phosphor-icons/react'
import { useState } from 'react'
import { toast } from 'sonner'

import { SetStatusDialog } from '@/components/chat/set-status-dialog'
import { UserAvatarMark } from '@/components/chat/user-avatar'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  logout as apiLogout,
  resendVerificationEmail,
} from '@/lib/api/auth'
import type { PresenceState } from '@/lib/api/auth'
import { formatUserError } from '@/lib/api/client'
import { useAuth } from '@/lib/auth'
import { useLocale } from '@/lib/i18n'
import { formatCustomStatus } from '@/lib/presence'
import { cn } from '@/lib/utils'

function presenceLabel(
  presence: PresenceState | undefined,
  t: (key: string) => string,
) {
  switch (presence) {
    case 'online':
      return t('presence.online')
    case 'away':
      return t('presence.away')
    default:
      return t('presence.offline')
  }
}

export function UserPill({
  serverUrl,
  token,
  presence,
  onProfileSaved,
  onSwitchAccount,
  onEditProfile,
  activeView,
  onOpenNotifications,
  notificationUnreadCount,
  onOpenAppSettings,
}: {
  serverUrl?: string
  token?: string
  presence?: PresenceState
  onProfileSaved?: () => void
  onSwitchAccount?: (sessionId: string) => void
  onEditProfile: () => void
  activeView: 'chat' | 'notifications' | 'workspace-settings' | 'app-settings'
  onOpenNotifications: () => void
  notificationUnreadCount: number
  onOpenAppSettings: () => void
}) {
  const { t } = useLocale()
  const {
    session,
    sessions,
    signOut,
    removeSession,
    setAddingAccount,
  } = useAuth()
  const [setStatusOpen, setSetStatusOpen] = useState(false)
  const [resendingVerification, setResendingVerification] = useState(false)
  const displayName = session?.displayName ?? ''
  const email = session?.email ?? ''
  const emailVerified = session?.emailVerified !== false
  const customStatus = formatCustomStatus(
    session?.statusEmoji,
    session?.statusText,
    session?.statusExpiresAt,
  )
  const statusLine = customStatus || presenceLabel(presence, t)

  const handleResendVerification = () => {
    if (!serverUrl || !token || resendingVerification) return
    setResendingVerification(true)
    void (async () => {
      try {
        await resendVerificationEmail(serverUrl, token)
        toast.success(t('profile.verificationSent'))
      } catch (err) {
        const message = formatUserError(err, t('profile.verificationError'))
        toast.error(message)
      } finally {
        setResendingVerification(false)
      }
    })()
  }

  return (
    <>
      <div className="pointer-events-none absolute inset-x-1.5 bottom-1.5 z-20">
        <div className="pointer-events-auto flex flex-wrap items-center justify-center gap-1 rounded-xl bg-sidebar-accent px-1.5 py-1.5 text-sidebar-accent-foreground shadow-md shadow-black/10 md:flex-nowrap md:justify-start border border-sidebar-accent-foreground/10 dark:border-sidebar-accent-foreground/10 dark:shadow-sm dark:shadow-white/10">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                type="button"
                className="flex min-w-0 flex-1 items-center justify-center gap-2 rounded-lg px-1 py-0.5 text-left outline-none hover:bg-sidebar-accent-foreground/8 focus-visible:ring-2 focus-visible:ring-ring md:justify-start"
                aria-label={t('profile.openMenu')}
              >
                <UserAvatarMark
                  userId={session?.userId ?? ''}
                  name={displayName}
                  hasAvatar={session?.hasAvatar}
                  avatarUpdatedAt={session?.avatarUpdatedAt}
                  presence={presence}
                  showPresence
                  serverUrl={serverUrl ?? session?.serverUrl}
                  token={token ?? session?.token}
                  className="size-8 shrink-0 rounded-md after:rounded-md"
                  imageClassName="rounded-md"
                  fallbackClassName="rounded-md bg-primary text-xs font-semibold text-primary-foreground"
                />
                <span className="hidden min-w-0 flex-1 flex-col md:flex">
                  <span className="truncate text-sm font-semibold leading-tight">
                    {displayName}
                  </span>
                  <span className="truncate text-xs leading-tight text-muted-foreground">
                    {statusLine}
                  </span>
                </span>
                <span className="sr-only">{displayName}</span>
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              className="min-w-64 rounded-lg"
              side="top"
              align="start"
              sideOffset={8}
            >
              <DropdownMenuLabel className="p-0 font-normal">
                <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                  <UserAvatarMark
                    userId={session?.userId ?? ''}
                    name={displayName}
                    hasAvatar={session?.hasAvatar}
                    avatarUpdatedAt={session?.avatarUpdatedAt}
                    presence={presence}
                    showPresence
                    serverUrl={serverUrl ?? session?.serverUrl}
                    token={token ?? session?.token}
                    className="size-9 rounded-md after:rounded-md"
                    imageClassName="rounded-md"
                    fallbackClassName="rounded-md bg-primary font-medium text-primary-foreground"
                  />
                  <div className="grid min-w-0 flex-1 text-left text-sm leading-tight">
                    <span className="truncate font-medium">{displayName}</span>
                    {customStatus ? (
                      <span className="truncate text-xs text-muted-foreground">
                        {customStatus}
                      </span>
                    ) : null}
                    <span className="truncate text-xs text-muted-foreground">
                      {email}
                    </span>
                    {!emailVerified ? (
                      <span className="truncate text-xs text-amber-700 dark:text-amber-400">
                        {t('profile.emailUnverified')}
                      </span>
                    ) : null}
                    {session?.serverLabel ? (
                      <span className="truncate text-xs text-muted-foreground">
                        {session.serverLabel}
                      </span>
                    ) : null}
                  </div>
                </div>
              </DropdownMenuLabel>
              {!emailVerified ? (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    disabled={resendingVerification}
                    onSelect={() => handleResendVerification()}
                  >
                    <EnvelopeSimpleIcon />
                    {resendingVerification
                      ? t('profile.sendingVerification')
                      : t('profile.resendVerification')}
                  </DropdownMenuItem>
                </>
              ) : null}
              <DropdownMenuSeparator />
              <DropdownMenuItem onSelect={() => setSetStatusOpen(true)}>
                <SmileyIcon />
                {customStatus ? t('profile.updateStatus') : t('profile.setStatus')}
              </DropdownMenuItem>
              <DropdownMenuItem onSelect={() => onEditProfile()}>
                <UserCircleIcon />
                {t('profile.edit')}
              </DropdownMenuItem>
              <DropdownMenuSub>
                <DropdownMenuSubTrigger>
                  <UserPlusIcon />
                  {t('profile.accounts')}
                </DropdownMenuSubTrigger>
                <DropdownMenuSubContent className="min-w-56">
                  {sessions.map((item) => (
                    <DropdownMenuItem
                      key={item.id}
                      onSelect={() => {
                        if (item.id === session?.id) return
                        onSwitchAccount?.(item.id)
                      }}
                    >
                      <div className="flex min-w-0 flex-1 flex-col">
                        <span className="truncate text-sm font-medium">
                          {item.displayName}
                          {item.id === session?.id ? ` · ${t('profile.active')}` : ''}
                        </span>
                        <span className="truncate text-xs text-muted-foreground">
                          {item.serverLabel || item.serverUrl}
                        </span>
                      </div>
                    </DropdownMenuItem>
                  ))}
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onSelect={() => setAddingAccount(true)}>
                    <PlusIcon />
                    {t('profile.addAccount')}
                  </DropdownMenuItem>
                  {sessions.length > 1 && session ? (
                    <DropdownMenuItem
                      variant="destructive"
                      onSelect={() => {
                        const current = session
                        void (async () => {
                          try {
                            await apiLogout(current.serverUrl, current.token)
                          } catch {
                            // Always clear the local session.
                          }
                          removeSession(current.id)
                        })()
                      }}
                    >
                      <SignOutIcon />
                      {t('profile.removeAccount')}
                    </DropdownMenuItem>
                  ) : null}
                </DropdownMenuSubContent>
              </DropdownMenuSub>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                variant="destructive"
                onSelect={() => {
                  const current = session
                  void (async () => {
                    if (current?.token && current.serverUrl) {
                      try {
                        await apiLogout(current.serverUrl, current.token)
                      } catch {
                        // Always clear the local session.
                      }
                    }
                    signOut()
                  })()
                }}
              >
                <SignOutIcon />
                {t('profile.signOut')}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>

          <div className="flex shrink-0 items-center gap-0.5">
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              className={cn(
                'text-muted-foreground hover:text-foreground',
                activeView === 'notifications' && 'bg-accent text-accent-foreground',
              )}
              aria-label={
                notificationUnreadCount > 0
                  ? t('notifications.sidebarLabel', {
                      label: t('notifications.title'),
                      count: notificationUnreadCount,
                    })
                  : t('notifications.title')
              }
              aria-current={activeView === 'notifications' ? 'page' : undefined}
              onClick={onOpenNotifications}
            >
              <span className="relative">
                <BellIcon strokeWidth={2} />
                {notificationUnreadCount > 0 ? (
                  <span
                    aria-hidden
                    className="absolute -top-1.5 -right-1.5 z-10 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold leading-none text-primary-foreground"
                  >
                    {notificationUnreadCount > 99
                      ? t('common.unreadOverflow')
                      : notificationUnreadCount}
                  </span>
                ) : null}
              </span>
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              className={cn(
                'text-muted-foreground hover:text-foreground',
                activeView === 'app-settings' && 'bg-accent text-accent-foreground',
              )}
              aria-label={t('common.settings')}
              aria-current={activeView === 'app-settings' ? 'page' : undefined}
              onClick={onOpenAppSettings}
            >
              <GearIcon strokeWidth={2} />
            </Button>
          </div>
        </div>
      </div>
      <SetStatusDialog
        open={setStatusOpen}
        onOpenChange={setSetStatusOpen}
        serverUrl={serverUrl}
        token={token}
        onSaved={() => onProfileSaved?.()}
      />
    </>
  )
}
