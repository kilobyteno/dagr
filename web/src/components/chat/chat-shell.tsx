import {
  BellIcon,
  CalendarBlankIcon,
  CaretDownIcon,
  CheckCircleIcon,
  ClockCountdownIcon,
  ClockIcon,
  CopyIcon,
  DotsThreeOutlineIcon,
  EnvelopeSimpleIcon,
  GearIcon,
  HashIcon,
  ImageIcon,
  LockKeyIcon,
  PaperPlaneRightIcon,
  PencilSimpleIcon,
  PlusIcon,
  SidebarSimpleIcon,
  SignOutIcon,
  SmileyIcon,
  TrashIcon,
  UserCircleIcon,
  UserPlusIcon,
  WarningCircleIcon,
} from '@phosphor-icons/react'
import { toast } from 'sonner'
import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
} from 'react'

import { AppLoadingScreen } from '@/components/app-loading-screen'
import { AppSettingsPage } from '@/components/chat/app-settings-page'
import { ChannelDetailsSidebar } from '@/components/chat/channel-details-sidebar'
import {
  ComposerMarkdownToolbar,
  type ComposerSelectionEdit,
} from '@/components/chat/composer-markdown-toolbar'
import { EmojiPickerButton } from '@/components/chat/emoji-picker'
import { MessageActionBar } from '@/components/chat/message-action-bar'
import { TrustedDomainsProvider } from '@/components/chat/trusted-domains'
import { WorkspaceIconMark } from '@/components/chat/workspace-icon'
import {
  MessageLinkPreviews,
  messageBodyWithoutGifLinks,
} from '@/components/chat/link-preview-card'
import { MessageReactions } from '@/components/chat/message-reactions'
import { NotificationsPage } from '@/components/chat/notifications-page'
import { MessageMarkdown } from '@/components/chat/message-markdown'
import { SetStatusDialog } from '@/components/chat/set-status-dialog'
import { UserAvatarMark } from '@/components/chat/user-avatar'
import {
  DirectMessageProvider,
  UserHandle,
  type ChatUserRef,
} from '@/components/chat/user-handle'
import { WorkspaceSettingsPage } from '@/components/chat/workspace-settings-page'
import { TitleBar } from '@/components/desktop/title-bar'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { ButtonGroup } from '@/components/ui/button-group'
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuShortcut,
  ContextMenuTrigger,
} from '@/components/ui/context-menu'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import {
  InputGroup,
  InputGroupAddon,
  InputGroupTextarea,
} from '@/components/ui/input-group'
import { Label } from '@/components/ui/label'
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from '@/components/ui/resizable'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  useSidebar,
} from '@/components/ui/sidebar'
import { Switch } from '@/components/ui/switch'
import {
  logout as apiLogout,
  me,
  removeAvatar,
  resendVerificationEmail,
  updateProfile,
  uploadAvatar,
  verifyEmail,
  type ApiUser,
  type NotificationLevel,
} from '@/lib/api/auth'
import {
  formatCustomStatus,
  hasCustomStatus,
  useOwnPresence,
} from '@/lib/presence'
import type { PresenceState } from '@/lib/api/auth'
import {
  addChannelMember,
  createChannel,
  deleteChannel,
  updateChannel,
} from '@/lib/api/channels'
import { ApiError, formatUserError, isServerUnavailable } from '@/lib/api/client'
import {
  formatPlanAmount,
  getWorkspaceBilling,
  type ApiWorkspaceBilling,
} from '@/lib/api/billing'
import { acceptInvite, inviteToWorkspace } from '@/lib/api/invites'
import {
  deleteMessage,
  listMessages,
  markChannelRead,
  markChannelUnread,
  postMessage,
  scheduleMessage,
  toggleMessageReaction,
  updateMessage,
  type ApiMessage,
  type ChannelUnreadResponse,
} from '@/lib/api/messages'
import { Textarea } from '@/components/ui/textarea'
import {
  listNotifications,
  type ApiNotification,
} from '@/lib/api/notifications'
import {
  createWorkspace,
  getWorkspaceMe,
  listChannels,
  listWorkspaceMembers,
  listWorkspaces,
  openDirectMessage,
  updateWorkspaceMe,
  workspaceInitials,
  type ApiChannel,
  type ApiWorkspace,
} from '@/lib/api/workspaces'
import { useAuth } from '@/lib/auth'
import { i18n, useLocale } from '@/lib/i18n'
import { parseAppLocale } from '@/lib/i18n/locales'
import {
  consumeDeepLink,
  parseDagrDeepLink,
  subscribeDeepLink,
} from '@/lib/deep-link'
import { setDesktopBadgeCount, showDesktopNotification } from '@/lib/desktop'
import {
  EmailVerificationBanner,
  ServerConnectionBanner,
  useServerConnection,
} from '@/lib/server-connection'
import {
  readStoredServerHost,
  resolveServerUrl,
} from '@/lib/server-host'
import { cn } from '@/lib/utils'

type ShellLocation = {
  sessionId?: string
  workspaceId: string
  conversationId: string
  view: 'chat' | 'notifications' | 'workspace-settings' | 'app-settings'
}

function sameLocation(a: ShellLocation, b: ShellLocation) {
  return (
    (a.sessionId || '') === (b.sessionId || '') &&
    a.workspaceId === b.workspaceId &&
    a.conversationId === b.conversationId &&
    a.view === b.view
  )
}

const DETAILS_MIN_BREAKPOINT = 1024

function useShowDetailsPanel() {
  const [show, setShow] = useState(false)

  useEffect(() => {
    const mql = window.matchMedia(`(min-width: ${DETAILS_MIN_BREAKPOINT}px)`)
    const onChange = () => setShow(mql.matches)
    onChange()
    mql.addEventListener('change', onChange)
    return () => mql.removeEventListener('change', onChange)
  }, [])

  return show
}

const CURRENT_USER = {
  name: 'Avery Chen',
  email: 'avery@kilobyte.no',
  initials: 'AC',
}

type ChannelConversation = {
  kind: 'channel'
  id: string
  name: string
  topic?: string
  isPrivate: boolean
  isDm: boolean
  createdBy?: string
  unreadCount: number
  firstUnreadMessageId?: string
  peerUserId?: string
  peerDisplayName?: string
  peerHandle?: string
  peerHasAvatar?: boolean
  peerAvatarUpdatedAt?: string | null
}

type RailWorkspace = {
  railKey: string
  id: string
  sessionId: string
  serverUrl: string
  serverLabel: string
  token: string
  name: string
  slug: string
  role: string
  initials: string
  hasIcon?: boolean
  iconUpdatedAt?: string
}

function toRailWorkspace(
  ws: ApiWorkspace,
  auth: { id: string; serverUrl: string; serverLabel?: string; token: string },
): RailWorkspace {
  const serverLabel =
    auth.serverLabel ||
    (() => {
      try {
        return new URL(auth.serverUrl).host
      } catch {
        return auth.serverUrl
      }
    })()
  return {
    railKey: `${auth.id}:${ws.id}`,
    id: ws.id,
    sessionId: auth.id,
    serverUrl: auth.serverUrl,
    serverLabel,
    token: auth.token,
    name: ws.name,
    slug: ws.slug,
    role: ws.role,
    initials: workspaceInitials(ws.name),
    hasIcon: Boolean(ws.hasIcon),
    iconUpdatedAt: ws.iconUpdatedAt,
  }
}

function toChannelConversation(ch: ApiChannel): ChannelConversation {
  return {
    kind: 'channel',
    id: ch.id,
    name: ch.name,
    topic: ch.topic,
    isPrivate: ch.isPrivate,
    isDm: Boolean(ch.isDm),
    createdBy: ch.createdBy,
    unreadCount: ch.unreadCount ?? 0,
    firstUnreadMessageId: ch.firstUnreadMessageId,
    peerUserId: ch.peerUserId,
    peerDisplayName: ch.peerDisplayName,
    peerHandle: ch.peerHandle,
    peerHasAvatar: ch.peerHasAvatar,
    peerAvatarUpdatedAt: ch.peerAvatarUpdatedAt,
  }
}

function conversationDisplayName(conversation: ChannelConversation) {
  if (conversation.isDm) {
    return (
      conversation.peerDisplayName?.trim() ||
      (conversation.peerHandle ? `@${conversation.peerHandle}` : conversation.name)
    )
  }
  return conversation.name
}

function preferDefaultConversationId(channels: readonly ChannelConversation[]) {
  return (
    channels.find((item) => !item.isDm)?.id ?? channels[0]?.id ?? ''
  )
}

function startOfLocalDay(date: Date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime()
}

function formatMessageDayLabel(iso: string) {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ''
  const today = startOfLocalDay(new Date())
  const day = startOfLocalDay(date)
  const dayMs = 24 * 60 * 60 * 1000
  if (day === today) return i18n.t('common.today')
  if (day === today - dayMs) return i18n.t('common.yesterday')
  const locale = parseAppLocale(i18n.resolvedLanguage ?? i18n.language)
  return date.toLocaleDateString(locale, {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: date.getFullYear() === new Date().getFullYear() ? undefined : 'numeric',
  })
}

function formatMessageTime(iso: string) {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ''
  const locale = parseAppLocale(i18n.resolvedLanguage ?? i18n.language)
  return date.toLocaleTimeString(locale, {
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatMessageTimestampTitle(iso: string) {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ''
  const locale = parseAppLocale(i18n.resolvedLanguage ?? i18n.language)
  return date.toLocaleString(locale, {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

function nextLaterToday(): Date {
  const d = new Date()
  d.setHours(15, 0, 0, 0)
  if (d.getTime() <= Date.now() + 60_000) {
    d.setDate(d.getDate() + 1)
  }
  return d
}

function nextTomorrowMorning(): Date {
  const d = new Date()
  d.setDate(d.getDate() + 1)
  d.setHours(9, 0, 0, 0)
  return d
}

function nextMondayMorning(): Date {
  const d = new Date()
  const day = d.getDay()
  const daysUntilMonday = day === 0 ? 1 : 8 - day
  d.setDate(d.getDate() + daysUntilMonday)
  d.setHours(9, 0, 0, 0)
  return d
}

function toDatetimeLocalValue(date: Date) {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function lastWorkspaceStorageKey(userId: string, serverUrl: string) {
  return `dagr.lastWorkspace.${userId}.${serverUrl}`
}

const EMPTY_LOCATION: ShellLocation = {
  sessionId: '',
  workspaceId: '',
  conversationId: '',
  view: 'chat',
}

const MESSAGE_SKELETON_ROWS = [
  { name: 'w-24', lines: ['w-3/4', 'w-1/2'] },
  { name: 'w-32', lines: ['w-5/6'] },
  { name: 'w-20', lines: ['w-2/3', 'w-4/5', 'w-1/3'] },
  { name: 'w-28', lines: ['w-1/2', 'w-3/5'] },
  { name: 'w-36', lines: ['w-4/5'] },
  { name: 'w-24', lines: ['w-2/3', 'w-1/2'] },
] as const

function ChannelMessagesSkeleton() {
  const { t } = useLocale()
  return (
    <div
      className="flex flex-col gap-4"
      aria-busy="true"
      aria-label={t('chat.loadingMessages')}
    >
      {MESSAGE_SKELETON_ROWS.map((row, index) => (
        <div key={index} className="flex items-start gap-3 px-1 py-1.5">
          <Skeleton className="size-9 shrink-0 rounded-md" />
          <div className="flex min-w-0 flex-1 flex-col gap-2 pt-0.5">
            <div className="flex items-center gap-2">
              <Skeleton className={cn('h-3.5', row.name)} />
              <Skeleton className="h-3 w-10" />
            </div>
            {row.lines.map((line, lineIndex) => (
              <Skeleton key={lineIndex} className={cn('h-3.5', line)} />
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

function UserMenu({
  serverUrl,
  token,
  presence,
  onProfileSaved,
  onSwitchAccount,
  editProfileOpen,
  onEditProfileOpenChange,
}: {
  serverUrl?: string
  token?: string
  presence?: PresenceState
  onProfileSaved?: () => void
  onSwitchAccount?: (sessionId: string) => void
  editProfileOpen: boolean
  onEditProfileOpenChange: (open: boolean) => void
}) {
  const { t } = useLocale()
  const { isMobile } = useSidebar()
  const {
    session,
    sessions,
    signOut,
    removeSession,
    setAddingAccount,
  } = useAuth()
  const [setStatusOpen, setSetStatusOpen] = useState(false)
  const [resendingVerification, setResendingVerification] = useState(false)
  const displayName = session?.displayName ?? CURRENT_USER.name
  const email = session?.email ?? CURRENT_USER.email
  const emailVerified = session?.emailVerified !== false
  const customStatus = formatCustomStatus(
    session?.statusEmoji,
    session?.statusText,
    session?.statusExpiresAt,
  )

  const handleResendVerification = () => {
    if (!serverUrl || !token || resendingVerification) return
    setResendingVerification(true)
    void (async () => {
      try {
        await resendVerificationEmail(serverUrl, token)
        toast.success(t('profile.verificationSent'))
      } catch (err) {
        const message =
          formatUserError(err, t('profile.verificationError'))
        toast.error(message)
      } finally {
        setResendingVerification(false)
      }
    })()
  }

  return (
    <>
      <SidebarMenu className="my-auto items-center overflow-visible">
        <SidebarMenuItem className="overflow-visible">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <SidebarMenuButton
                size="lg"
                className="size-8! overflow-visible! p-0! data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
                tooltip={displayName}
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
                  className="size-8 rounded-md after:rounded-md"
                  imageClassName="rounded-md"
                  fallbackClassName="rounded-md bg-primary text-xs font-semibold text-primary-foreground"
                />
                <span className="sr-only">{displayName}</span>
              </SidebarMenuButton>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              className="min-w-64 rounded-lg"
              side={isMobile ? 'bottom' : 'right'}
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
              <DropdownMenuItem onSelect={() => onEditProfileOpenChange(true)}>
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
        </SidebarMenuItem>
      </SidebarMenu>
      <EditProfileDialog
        open={editProfileOpen}
        onOpenChange={onEditProfileOpenChange}
        serverUrl={serverUrl}
        token={token}
        onSaved={onProfileSaved}
      />
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

function EditProfileDialog({
  open,
  onOpenChange,
  serverUrl,
  token,
  onSaved,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  serverUrl?: string
  token?: string
  onSaved?: () => void
}) {
  const { t } = useLocale()
  const { session, signIn } = useAuth()
  const [displayName, setDisplayName] = useState('')
  const [notificationLevel, setNotificationLevel] =
    useState<NotificationLevel>('mentions')
  const [submitting, setSubmitting] = useState(false)
  const [avatarBusy, setAvatarBusy] = useState(false)
  const [resendingVerification, setResendingVerification] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (open && session) {
      setDisplayName(session.displayName)
      setNotificationLevel(session.notificationLevel)
      setSubmitting(false)
    }
  }, [open, session])

  if (!session) return null

  const emailVerified = session.emailVerified !== false

  const applyUser = (user: ApiUser) => {
    signIn({
      ...session,
      displayName: user.displayName,
      notificationLevel: user.notificationLevel,
      locale: parseAppLocale(user.locale),
      emailVerified: Boolean(user.emailVerified),
      statusEmoji: user.statusEmoji ?? '',
      statusText: user.statusText ?? '',
      statusExpiresAt: user.statusExpiresAt ?? null,
      hasAvatar: Boolean(user.hasAvatar),
      avatarUpdatedAt: user.avatarUpdatedAt ?? null,
    })
  }

  const handleResendVerification = () => {
    if (!serverUrl || !token || resendingVerification) return
    setResendingVerification(true)
    void (async () => {
      try {
        await resendVerificationEmail(serverUrl, token)
        toast.success(t('profile.verificationSent'))
      } catch (err) {
        const message =
          formatUserError(err, t('profile.verificationError'))
        toast.error(message)
      } finally {
        setResendingVerification(false)
      }
    })()
  }

  const handleAvatarFile = async (file: File) => {
    if (!serverUrl || !token) return
    setAvatarBusy(true)
    try {
      const result = await uploadAvatar(serverUrl, token, file)
      applyUser(result.user)
      onSaved?.()
      toast.success(t('profile.avatarUpdated'))
    } catch (err) {
      const message =
        formatUserError(err, t('profile.avatarUploadError'))
      toast.error(message)
    } finally {
      setAvatarBusy(false)
    }
  }

  const handleRemoveAvatar = async () => {
    if (!serverUrl || !token) return
    setAvatarBusy(true)
    try {
      const result = await removeAvatar(serverUrl, token)
      applyUser(result.user)
      onSaved?.()
      toast.success(t('profile.avatarRemoved'))
    } catch (err) {
      const message =
        formatUserError(err, t('profile.avatarRemoveError'))
      toast.error(message)
    } finally {
      setAvatarBusy(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('profile.edit')}</DialogTitle>
          <DialogDescription>
            {t('profile.editDescription')}
          </DialogDescription>
        </DialogHeader>
        <form
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault()
            if (!displayName.trim() || !serverUrl || !token || submitting) {
              return
            }
            setSubmitting(true)
            void (async () => {
              try {
                const result = await updateProfile(serverUrl, token, {
                  displayName: displayName.trim(),
                  notificationLevel,
                })
                applyUser(result.user)
                onSaved?.()
                onOpenChange(false)
                toast.success(t('profile.updated'))
              } catch (err) {
                const message =
                  formatUserError(err, t('profile.updateError'))
                toast.error(message)
              } finally {
                setSubmitting(false)
              }
            })()
          }}
        >
          <div className="flex flex-col gap-1.5">
            <Label>{t('common.email')}</Label>
            <p className="text-sm text-muted-foreground">{session.email}</p>
            {!emailVerified ? (
              <div className="flex flex-col gap-2 rounded-lg border border-amber-500/30 bg-amber-500/10 p-3">
                <div className="flex items-start gap-2 text-sm">
                  <WarningCircleIcon
                    strokeWidth={2}
                    className="mt-0.5 size-4 shrink-0 text-amber-700 dark:text-amber-400"
                    aria-hidden
                  />
                  <p className="text-pretty">
                    {t('profile.emailUnverifiedHint')}
                  </p>
                </div>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  className="self-start"
                  disabled={resendingVerification}
                  onClick={handleResendVerification}
                >
                  {resendingVerification
                    ? t('profile.sending')
                    : t('profile.resendVerification')}
                </Button>
              </div>
            ) : null}
          </div>
          <div className="flex flex-col gap-3">
            <Label>{t('profile.picture')}</Label>
            <div className="flex items-center gap-4">
              <UserAvatarMark
                userId={session.userId}
                name={displayName.trim() || session.displayName}
                hasAvatar={session.hasAvatar}
                avatarUpdatedAt={session.avatarUpdatedAt}
                serverUrl={serverUrl}
                token={token}
                className="size-14 rounded-md after:rounded-md"
                imageClassName="rounded-md"
                fallbackClassName="rounded-md bg-primary text-lg font-semibold text-primary-foreground"
              />
              <div className="flex flex-col gap-1">
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/*"
                  className="hidden"
                  onChange={(event) => {
                    const file = event.target.files?.[0]
                    event.target.value = ''
                    if (file) void handleAvatarFile(file)
                  }}
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={avatarBusy}
                  onClick={() => fileInputRef.current?.click()}
                >
                  {t('profile.uploadPhoto')}
                </Button>
                {session.hasAvatar ? (
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    disabled={avatarBusy}
                    onClick={() => void handleRemoveAvatar()}
                  >
                    {t('profile.removePhoto')}
                  </Button>
                ) : null}
              </div>
            </div>
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="profile-display-name">{t('profile.displayName')}</Label>
            <Input
              id="profile-display-name"
              value={displayName}
              onChange={(event) => setDisplayName(event.target.value)}
              maxLength={80}
              disabled={submitting}
            />
          </div>
          <div className="flex flex-col gap-1">
            <p className="px-1 text-xs font-medium tracking-wide text-muted-foreground uppercase">
              {t('profile.defaultNotifications')}
            </p>
            <SettingRow
              id="profile-notify-all"
              label={t('profile.notifyAll')}
              description={t('profile.notifyAllHint')}
              checked={notificationLevel === 'all'}
              onCheckedChange={(checked) => {
                if (checked) setNotificationLevel('all')
              }}
            />
            <SettingRow
              id="profile-notify-mentions"
              label={t('profile.notifyMentions')}
              description={t('profile.notifyMentionsHint')}
              checked={notificationLevel === 'mentions'}
              onCheckedChange={(checked) => {
                if (checked) setNotificationLevel('mentions')
              }}
            />
            <SettingRow
              id="profile-notify-nothing"
              label={t('profile.notifyNothing')}
              description={t('profile.notifyNothingHint')}
              checked={notificationLevel === 'nothing'}
              onCheckedChange={(checked) => {
                if (checked) setNotificationLevel('nothing')
              }}
            />
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={submitting}
            >
              {t('common.cancel')}
            </Button>
            <Button type="submit" disabled={!displayName.trim() || submitting}>
              {submitting ? t('common.saving') : t('common.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function WorkspaceRail({
  workspaces,
  activeRailKey,
  onSelectWorkspace,
  onAddWorkspace,
  activeView,
  onOpenNotifications,
  notificationUnreadCount,
  onOpenAppSettings,
  onProfileSaved,
  onSwitchAccount,
  presence,
  serverUrl,
  token,
  editProfileOpen,
  onEditProfileOpenChange,
}: {
  workspaces: readonly RailWorkspace[]
  activeRailKey: string
  onSelectWorkspace: (railKey: string) => void
  onAddWorkspace: () => void
  activeView: ShellLocation['view']
  onOpenNotifications: () => void
  notificationUnreadCount: number
  onOpenAppSettings: () => void
  onProfileSaved?: () => void
  onSwitchAccount?: (sessionId: string) => void
  presence?: PresenceState
  serverUrl?: string
  token?: string
  editProfileOpen: boolean
  onEditProfileOpenChange: (open: boolean) => void
}) {
  const { t } = useLocale()
  const { isMobile, setOpen } = useSidebar()

  return (
    <Sidebar
      collapsible="none"
      className="w-14! min-w-14! max-w-14! shrink-0 overflow-visible border-r"
    >
      <SidebarHeader className="z-10 flex h-14 items-center justify-center overflow-visible border-b p-0">
        <UserMenu
          serverUrl={serverUrl}
          token={token}
          presence={presence}
          onProfileSaved={onProfileSaved}
          onSwitchAccount={onSwitchAccount}
          editProfileOpen={editProfileOpen}
          onEditProfileOpenChange={onEditProfileOpenChange}
        />
      </SidebarHeader>

      <SidebarContent className="overflow-x-hidden py-2">
        <SidebarGroup className="px-0 py-0">
          <SidebarGroupContent className="px-0">
            <SidebarMenu className="items-center gap-2.5">
              {workspaces.map((workspace) => (
                <SidebarMenuItem key={workspace.railKey} className="flex justify-center">
                  <SidebarMenuButton
                    tooltip={{
                      children:
                        workspaces.length > 1 &&
                        workspaces.some(
                          (item) => item.serverLabel !== workspace.serverLabel,
                        )
                          ? `${workspace.name} · ${workspace.serverLabel}`
                          : workspace.name,
                      hidden: isMobile,
                    }}
                    onClick={() => {
                      onSelectWorkspace(workspace.railKey)
                      setOpen(true)
                    }}
                    isActive={activeRailKey === workspace.railKey}
                    aria-label={workspace.name}
                    className="size-8! p-0! hover:bg-transparent data-active:bg-transparent border-muted-foreground/10 border"
                  >
                    <WorkspaceIconMark
                      workspaceId={workspace.id}
                      name={workspace.name}
                      hasIcon={workspace.hasIcon}
                      iconUpdatedAt={workspace.iconUpdatedAt}
                      serverUrl={workspace.serverUrl}
                      token={workspace.token}
                      className={cn(
                        'flex size-8 items-center justify-center overflow-hidden rounded-md text-xs font-semibold transition-colors',
                        activeRailKey === workspace.railKey &&
                          'ring-2 ring-primary',
                        workspace.hasIcon
                          ? ''
                          : activeRailKey === workspace.railKey
                            ? 'bg-primary text-primary-foreground'
                            : 'bg-muted text-muted-foreground hover:bg-accent hover:text-accent-foreground',
                      )}
                      initialsClassName="flex size-8 items-center justify-center"
                    />
                    <span className="sr-only">{workspace.name}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
              <SidebarMenuItem className="mt-1 flex justify-center">
                <SidebarMenuButton
                  tooltip={{ children: t('chat.addWorkspace'), hidden: isMobile }}
                  aria-label={t('chat.addWorkspace')}
                  className="size-8! p-0! hover:bg-transparent"
                  onClick={onAddWorkspace}
                >
                  <span className="flex size-8 items-center justify-center rounded-md border border-dashed border-muted-foreground/50 text-muted-foreground transition-colors hover:border-foreground/40 hover:bg-accent hover:text-foreground">
                    <PlusIcon className="size-4" />
                  </span>
                  <span className="sr-only">{t('chat.addWorkspace')}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter className="overflow-visible p-0 pb-2">
        <SidebarGroup className="overflow-visible px-0 py-0">
          <SidebarGroupContent className="overflow-visible px-0">
            <SidebarMenu className="items-center gap-2.5 overflow-visible">
              {(
                [
                  {
                    key: 'notifications',
                    label: t('notifications.title'),
                    icon: BellIcon,
                    onClick: onOpenNotifications,
                    isActive: activeView === 'notifications',
                  },
                  {
                    key: 'settings',
                    label: t('common.settings'),
                    icon: GearIcon,
                    onClick: onOpenAppSettings,
                    isActive: activeView === 'app-settings',
                  },
                ] as const
              ).map((item) => {
                const Icon = item.icon
                const isActive = item.isActive
                return (
                  <SidebarMenuItem
                    key={item.key}
                    className="flex justify-center overflow-visible"
                  >
                    <SidebarMenuButton
                      tooltip={{ children: item.label, hidden: isMobile }}
                      aria-label={
                        item.key === 'notifications' && notificationUnreadCount > 0
                          ? t('notifications.sidebarLabel', {
                              label: item.label,
                              count: notificationUnreadCount,
                            })
                          : item.label
                      }
                      aria-current={isActive ? 'page' : undefined}
                      className="size-8! overflow-visible! p-0! hover:bg-transparent"
                      onClick={'onClick' in item ? item.onClick : undefined}
                    >
                      <span
                        className={cn(
                          'relative flex size-8 items-center justify-center overflow-visible rounded-md transition-colors',
                          isActive
                            ? 'bg-accent text-accent-foreground'
                            : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
                        )}
                      >
                        <Icon />
                        {item.key === 'notifications' &&
                        notificationUnreadCount > 0 ? (
                          <span
                            aria-hidden
                            className="absolute -top-1 -right-1 z-10 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold leading-none text-primary-foreground"
                          >
                            {notificationUnreadCount > 99
                              ? t('common.unreadOverflow')
                              : notificationUnreadCount}
                          </span>
                        ) : null}
                      </span>
                      <span className="sr-only">{item.label}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                )
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarFooter>
    </Sidebar>
  )
}

function ConversationPanel({
  workspace,
  channels,
  channelsLoading,
  activeConversationId,
  onSelectConversation,
  onOpenWorkspaceSettings,
  onCreateChannel,
  onInviteToWorkspace,
  dmMembers,
  onSelectDmMember,
  serverUrl,
  token,
}: {
  workspace: RailWorkspace | null
  channels: readonly ChannelConversation[]
  channelsLoading: boolean
  activeConversationId: string
  onSelectConversation: (id: string) => void
  onOpenWorkspaceSettings: () => void
  onCreateChannel: () => void
  onInviteToWorkspace: () => void
  dmMembers: readonly ChatUserRef[]
  onSelectDmMember: (user: ChatUserRef) => void
  serverUrl?: string
  token?: string
}) {
  const { t } = useLocale()
  const channelItems = channels.filter((item) => !item.isDm)
  const dmByPeerId = new Map(
    channels
      .filter((item) => item.isDm && item.peerUserId)
      .map((item) => [item.peerUserId!, item]),
  )
  const dmRows = dmMembers.map((member) => {
    const existing = dmByPeerId.get(member.userId)
    return {
      member,
      conversationId: existing?.id,
      unreadCount: existing?.unreadCount ?? 0,
    }
  })

  const renderChannelMenu = (items: readonly ChannelConversation[]) => (
    <SidebarMenu>
      {items.map((channel) => (
        <SidebarMenuItem key={channel.id}>
          <SidebarMenuButton
            isActive={activeConversationId === channel.id}
            onClick={() => onSelectConversation(channel.id)}
            className="px-2"
          >
            {channel.isPrivate ? <LockKeyIcon /> : <HashIcon />}
            <span
              className={cn(
                'truncate',
                channel.unreadCount > 0 && 'font-semibold text-foreground',
              )}
            >
              {channel.name}
            </span>
          </SidebarMenuButton>
          {channel.unreadCount > 0 ? (
            <SidebarMenuBadge className="bg-primary text-primary-foreground">
              {channel.unreadCount}
            </SidebarMenuBadge>
          ) : null}
        </SidebarMenuItem>
      ))}
    </SidebarMenu>
  )

  if (!workspace) {
    return (
      <Sidebar collapsible="none" className="hidden flex-1 md:flex">
        <SidebarHeader className="flex h-14 flex-row items-center border-b px-4">
          <span className="text-sm font-semibold">{t('chat.noWorkspace')}</span>
        </SidebarHeader>
        <SidebarContent className="px-4 py-6">
          <p className="text-sm text-muted-foreground">
            {t('chat.noWorkspaceHint')}
          </p>
        </SidebarContent>
      </Sidebar>
    )
  }

  return (
    <Sidebar collapsible="none" className="hidden flex-1 md:flex">
      <SidebarHeader className="flex h-14 flex-row items-center justify-between border-b px-2">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              type="button"
              variant="ghost"
              className="h-auto min-w-0 flex-1 justify-start gap-1 px-2 py-1.5"
              aria-label={t('chat.workspaceMenu', { name: workspace.name })}
            >
              <div className="flex min-w-0 flex-1 flex-col items-start gap-0.5">
                <span className="flex min-w-0 items-center gap-1">
                  <span className="truncate text-sm font-semibold">{workspace.name}</span>
                  <CaretDownIcon className="size-3.5 shrink-0 text-muted-foreground" />
                </span>
              </div>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="min-w-52">
            <DropdownMenuLabel>{workspace.name}</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={onInviteToWorkspace}>
              <UserPlusIcon />
              {t('chat.invitePeople')}
            </DropdownMenuItem>
            <DropdownMenuItem onSelect={onOpenWorkspaceSettings}>
              <GearIcon />
              {t('workspace.settings')}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <Button
          variant="ghost"
          size="icon-sm"
          aria-label={t('chat.newChannel')}
          onClick={onCreateChannel}
        >
          <PlusIcon data-icon />
        </Button>
      </SidebarHeader>

      <SidebarContent className="overflow-hidden">
        <ScrollArea className="min-h-0 flex-1">
          <div className="flex flex-col gap-4 px-2 py-3">
            <SidebarGroup className="p-0">
              <SidebarGroupLabel className="px-2 text-xs font-medium tracking-wide text-muted-foreground">
                {t('chat.channels')}
              </SidebarGroupLabel>
              <SidebarGroupContent>
                {channelsLoading ? (
                  <p className="px-2 py-1 text-xs text-muted-foreground">{t('chat.loadingChannels')}</p>
                ) : channelItems.length === 0 ? (
                  <p className="px-2 py-1 text-xs text-muted-foreground">{t('chat.noChannels')}</p>
                ) : (
                  renderChannelMenu(channelItems)
                )}
              </SidebarGroupContent>
            </SidebarGroup>

            <SidebarGroup className="p-0">
              <SidebarGroupLabel className="px-2 text-xs font-medium tracking-wide text-muted-foreground">
                {t('chat.directMessages')}
              </SidebarGroupLabel>
              <SidebarGroupContent>
                {dmRows.length === 0 ? (
                  <p className="px-2 py-1 text-xs text-muted-foreground">
                    {t('chat.inviteTeammates')}
                  </p>
                ) : (
                  <SidebarMenu>
                    {dmRows.map(({ member, conversationId, unreadCount }) => {
                      const label =
                        member.displayName.trim() ||
                        (member.handle ? `@${member.handle}` : t('common.member'))
                      const externalLabel = member.isExternal
                        ? member.homeWorkspaceName?.trim() || t('common.external')
                        : ''
                      return (
                        <SidebarMenuItem key={member.userId}>
                          <SidebarMenuButton
                            isActive={
                              Boolean(conversationId) &&
                              activeConversationId === conversationId
                            }
                            onClick={() => onSelectDmMember(member)}
                            className="px-2"
                            tooltip={
                              externalLabel
                                ? `${label} · ${externalLabel}`
                                : label
                            }
                          >
                            {serverUrl && token ? (
                              <UserAvatarMark
                                userId={member.userId}
                                name={label}
                                hasAvatar={member.hasAvatar}
                                avatarUpdatedAt={member.avatarUpdatedAt}
                                presence={member.presence}
                                showPresence
                                serverUrl={serverUrl}
                                token={token}
                                size="sm"
                                className="size-5 shrink-0"
                                imageClassName="rounded-md"
                                fallbackClassName="rounded-md text-[9px]"
                              />
                            ) : (
                              <UserCircleIcon />
                            )}
                            <span className="flex min-w-0 items-baseline gap-1.5">
                              <span
                                className={cn(
                                  'truncate',
                                  unreadCount > 0 &&
                                    'font-semibold text-foreground',
                                )}
                              >
                                {label}
                              </span>
                              {externalLabel ? (
                                <span className="truncate text-xs font-normal text-muted-foreground">
                                  {externalLabel}
                                </span>
                              ) : null}
                            </span>
                          </SidebarMenuButton>
                          {unreadCount > 0 ? (
                            <SidebarMenuBadge className="bg-primary text-primary-foreground">
                              {unreadCount}
                            </SidebarMenuBadge>
                          ) : null}
                        </SidebarMenuItem>
                      )
                    })}
                  </SidebarMenu>
                )}
              </SidebarGroupContent>
            </SidebarGroup>
          </div>
        </ScrollArea>
      </SidebarContent>
    </Sidebar>
  )
}

function CreateChannelDialog({
  open,
  onOpenChange,
  onCreated,
  serverUrl,
  token,
  workspaceId,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreated: (channel: ApiChannel) => void
  serverUrl: string
  token: string
  workspaceId: string
}) {
  const { t } = useLocale()
  const [name, setName] = useState('')
  const [topic, setTopic] = useState('')
  const [isPrivate, setIsPrivate] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (!open) {
      setName('')
      setTopic('')
      setIsPrivate(false)
      setSubmitting(false)
    }
  }, [open])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('chat.createChannel')}</DialogTitle>
          <DialogDescription>
            {t('chat.createChannelHint')}
          </DialogDescription>
        </DialogHeader>
        <form
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault()
            const trimmed = name.trim()
            if (!trimmed || submitting) return
            setSubmitting(true)
            void (async () => {
              try {
                const result = await createChannel(serverUrl, token, workspaceId, {
                  name: trimmed,
                  topic: topic.trim() || undefined,
                  isPrivate,
                })
                onCreated(result.channel)
                onOpenChange(false)
                toast.success(t('chat.createdChannel', { name: result.channel.name }))
              } catch (err) {
                const message =
                  formatUserError(err, t('chat.createChannelError'))
                toast.error(message)
                setSubmitting(false)
              }
            })()
          }}
        >
          <div className="flex flex-col gap-2">
            <Label htmlFor="channel-name">{t('common.name')}</Label>
            <Input
              id="channel-name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder={t('chat.channelNamePlaceholder')}
              autoFocus
              maxLength={80}
              disabled={submitting}
            />
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="channel-topic">{t('chat.topicOptional')}</Label>
            <Input
              id="channel-topic"
              value={topic}
              onChange={(event) => setTopic(event.target.value)}
              placeholder={t('chat.topicPlaceholder')}
              maxLength={250}
              disabled={submitting}
            />
          </div>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={isPrivate}
              onChange={(event) => setIsPrivate(event.target.checked)}
              disabled={submitting}
            />
            {t('chat.makePrivate')}
          </label>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={submitting}
            >
              {t('common.cancel')}
            </Button>
            <Button type="submit" disabled={!name.trim() || submitting}>
              {submitting ? t('common.creating') : t('common.create')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function EditChannelDialog({
  open,
  onOpenChange,
  channel,
  serverUrl,
  token,
  onUpdated,
  onDeleted,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  channel: ChannelConversation | null
  serverUrl: string
  token: string
  onUpdated: (channel: ApiChannel) => void
  onDeleted: (channelId: string) => void
}) {
  const { t } = useLocale()
  const [name, setName] = useState('')
  const [topic, setTopic] = useState('')
  const [isPrivate, setIsPrivate] = useState(false)
  const [memberEmail, setMemberEmail] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (open && channel) {
      setName(channel.name)
      setTopic(channel.topic ?? '')
      setIsPrivate(channel.isPrivate)
      setMemberEmail('')
      setSubmitting(false)
    }
  }, [open, channel])

  if (!channel) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('chat.channelSettings')}</DialogTitle>
          <DialogDescription>
            {isPrivate ? t('chat.editChannelHintMembers') : t('chat.editChannelHint')}
          </DialogDescription>
        </DialogHeader>
        <form
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault()
            if (!name.trim() || submitting) return
            setSubmitting(true)
            void (async () => {
              try {
                const result = await updateChannel(serverUrl, token, channel.id, {
                  name: name.trim(),
                  topic: topic.trim(),
                  isPrivate,
                })
                onUpdated(result.channel)
                onOpenChange(false)
                toast.success(t('chat.channelUpdated'))
              } catch (err) {
                const message =
                  formatUserError(err, t('chat.updateChannelError'))
                toast.error(message)
                setSubmitting(false)
              }
            })()
          }}
        >
          <div className="flex flex-col gap-2">
            <Label htmlFor="edit-channel-name">{t('common.name')}</Label>
            <Input
              id="edit-channel-name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              maxLength={80}
              disabled={submitting}
            />
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="edit-channel-topic">{t('chat.topic')}</Label>
            <Input
              id="edit-channel-topic"
              value={topic}
              onChange={(event) => setTopic(event.target.value)}
              maxLength={250}
              disabled={submitting}
            />
          </div>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={isPrivate}
              onChange={(event) => setIsPrivate(event.target.checked)}
              disabled={submitting}
            />
            {t('chat.makePrivate')}
          </label>
          {isPrivate && (
            <div className="flex flex-col gap-2">
              <Label htmlFor="channel-member-email">{t('chat.addMemberByEmail')}</Label>
              <div className="flex gap-2">
                <Input
                  id="channel-member-email"
                  type="email"
                  value={memberEmail}
                  onChange={(event) => setMemberEmail(event.target.value)}
                  placeholder={t('chat.memberEmailPlaceholder')}
                  disabled={submitting}
                />
                <Button
                  type="button"
                  variant="outline"
                  disabled={!memberEmail.trim() || submitting}
                  onClick={() => {
                    void (async () => {
                      try {
                        await addChannelMember(
                          serverUrl,
                          token,
                          channel.id,
                          memberEmail.trim(),
                        )
                        toast.success(t('chat.memberAdded'))
                        setMemberEmail('')
                      } catch (err) {
                        const message =
                          formatUserError(err, t('chat.addMemberError'))
                        toast.error(message)
                      }
                    })()
                  }}
                >
                  {t('common.add')}
                </Button>
              </div>
            </div>
          )}
          <DialogFooter className="sm:justify-between">
            <Button
              type="button"
              variant="destructive"
              disabled={submitting}
              onClick={() => {
                void (async () => {
                  setSubmitting(true)
                  try {
                    await deleteChannel(serverUrl, token, channel.id)
                    onDeleted(channel.id)
                    onOpenChange(false)
                    toast.success(t('chat.channelDeleted'))
                  } catch (err) {
                    const message =
                      formatUserError(err, t('chat.deleteChannelError'))
                    toast.error(message)
                    setSubmitting(false)
                  }
                })()
              }}
            >
              <TrashIcon data-icon="inline-start" />
              {t('common.delete')}
            </Button>
            <div className="flex gap-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
                disabled={submitting}
              >
                {t('common.cancel')}
              </Button>
              <Button type="submit" disabled={!name.trim() || submitting}>
                {submitting ? t('common.saving') : t('common.save')}
              </Button>
            </div>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function InviteWorkspaceDialog({
  open,
  onOpenChange,
  serverUrl,
  token,
  workspaceId,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  serverUrl: string
  token: string
  workspaceId: string
}) {
  const { t, locale } = useLocale()
  const [email, setEmail] = useState('')
  const [inviteLink, setInviteLink] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [billing, setBilling] = useState<ApiWorkspaceBilling | null>(null)

  useEffect(() => {
    if (!open) {
      setEmail('')
      setInviteLink('')
      setSubmitting(false)
      setBilling(null)
      return
    }
    const controller = new AbortController()
    void getWorkspaceBilling(serverUrl, token, workspaceId, controller.signal)
      .then((result) => {
        if (!controller.signal.aborted) setBilling(result.billing)
      })
      .catch(() => {
        if (!controller.signal.aborted) setBilling(null)
      })
    return () => controller.abort()
  }, [open, serverUrl, token, workspaceId])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('chat.inviteToWorkspace')}</DialogTitle>
          <DialogDescription>
            {t('chat.inviteHint')}
            {billing?.enabled && billing.entitlements.plan === 'pro' ? (
              <span className="mt-2 block">
                {t('chat.inviteBillingHint', {
                  amount: formatPlanAmount(
                    Math.round(
                      ((billing.nextAmountCents ||
                        billing.monthlyAmountCents ||
                        700) /
                        Math.max(billing.billableSeats || 1, 1)) *
                        ((billing.billableSeats || 1) + 1),
                    ),
                    billing.currency || 'EUR',
                    locale,
                  ),
                })}
              </span>
            ) : null}
          </DialogDescription>
        </DialogHeader>
        <form
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault()
            if (!email.trim() || submitting) return
            setSubmitting(true)
            void (async () => {
              try {
                const result = await inviteToWorkspace(
                  serverUrl,
                  token,
                  workspaceId,
                  { email: email.trim() },
                )
                if (result.status === 'added') {
                  toast.success(t('chat.memberAddedWorkspace'))
                  onOpenChange(false)
                } else if (result.invite) {
                  const link = `${window.location.origin}${result.invite.acceptPath}`
                  setInviteLink(link)
                  toast.success(t('chat.inviteCreated'))
                }
              } catch (err) {
                const message =
                  formatUserError(err, t('chat.sendInviteError'))
                toast.error(message)
              } finally {
                setSubmitting(false)
              }
            })()
          }}
        >
          <div className="flex flex-col gap-2">
            <Label htmlFor="invite-email">{t('common.email')}</Label>
            <Input
              id="invite-email"
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder={t('chat.inviteEmailPlaceholder')}
              autoFocus
              disabled={submitting}
            />
          </div>
          {inviteLink && (
            <div className="flex flex-col gap-2">
              <Label htmlFor="invite-link">{t('chat.inviteLink')}</Label>
              <div className="flex gap-2">
                <Input id="invite-link" value={inviteLink} readOnly />
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    void navigator.clipboard.writeText(inviteLink)
                    toast.success(t('chat.inviteLinkCopied'))
                  }}
                >
                  {t('common.copy')}
                </Button>
              </div>
            </div>
          )}
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={submitting}
            >
              {t('common.close')}
            </Button>
            <Button type="submit" disabled={!email.trim() || submitting}>
              {submitting ? t('chat.inviting') : t('common.invite')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function AcceptInviteDialog({
  open,
  onOpenChange,
  serverUrl,
  token,
  initialToken,
  onAccepted,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  serverUrl: string
  token: string
  initialToken: string
  onAccepted: (workspace: ApiWorkspace) => void
}) {
  const { t } = useLocale()
  const [inviteToken, setInviteToken] = useState(initialToken)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (open) {
      setInviteToken(initialToken)
      setSubmitting(false)
    }
  }, [open, initialToken])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('workspace.invite.acceptTitle')}</DialogTitle>
          <DialogDescription>
            {t('workspace.invite.acceptHint')}
          </DialogDescription>
        </DialogHeader>
        <form
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault()
            if (!inviteToken.trim() || submitting) return
            setSubmitting(true)
            void (async () => {
              try {
                const result = await acceptInvite(
                  serverUrl,
                  token,
                  inviteToken.trim(),
                )
                onAccepted(result.workspace)
                onOpenChange(false)
                toast.success(t('workspace.invite.joined', { name: result.workspace.name }))
              } catch (err) {
                const message =
                  formatUserError(err, t('workspace.invite.acceptError'))
                toast.error(message)
                setSubmitting(false)
              }
            })()
          }}
        >
          <div className="flex flex-col gap-2">
            <Label htmlFor="accept-token">{t('workspace.invite.token')}</Label>
            <Input
              id="accept-token"
              value={inviteToken}
              onChange={(event) => setInviteToken(event.target.value)}
              placeholder={t('workspace.invite.tokenPlaceholder')}
              autoFocus
              disabled={submitting}
            />
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={submitting}
            >
              {t('common.cancel')}
            </Button>
            <Button type="submit" disabled={!inviteToken.trim() || submitting}>
              {submitting ? t('workspace.invite.joining') : t('workspace.invite.accept')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function CustomScheduleDialog({
  open,
  onOpenChange,
  onConfirm,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (sendAt: Date) => void
}) {
  const { t } = useLocale()
  const [value, setValue] = useState(() =>
    toDatetimeLocalValue(new Date(Date.now() + 60 * 60 * 1000)),
  )

  useEffect(() => {
    if (open) {
      setValue(toDatetimeLocalValue(new Date(Date.now() + 60 * 60 * 1000)))
    }
  }, [open])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('chat.customSchedule')}</DialogTitle>
          <DialogDescription>
            {t('chat.customScheduleHint')}
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-2">
          <Label htmlFor="custom-send-at">{t('chat.sendAt')}</Label>
          <Input
            id="custom-send-at"
            type="datetime-local"
            value={value}
            onChange={(event) => setValue(event.target.value)}
          />
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.cancel')}
          </Button>
          <Button
            type="button"
            onClick={() => {
              const sendAt = new Date(value)
              if (Number.isNaN(sendAt.getTime()) || sendAt.getTime() <= Date.now() + 30_000) {
                toast.error(t('chat.scheduleFuture'))
                return
              }
              onConfirm(sendAt)
              onOpenChange(false)
            }}
          >
            {t('chat.schedule')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function CreateWorkspaceDialog({
  open,
  onOpenChange,
  onCreated,
  serverUrl,
  token,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreated: (workspace: ApiWorkspace, channels: ApiChannel[]) => void
  serverUrl: string
  token: string
}) {
  const { t } = useLocale()
  const [name, setName] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (!open) {
      setName('')
      setSubmitting(false)
    }
  }, [open])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('chat.createWorkspace')}</DialogTitle>
          <DialogDescription>
            {t('chat.createWorkspaceHint')}
          </DialogDescription>
        </DialogHeader>
        <form
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault()
            const trimmed = name.trim()
            if (!trimmed || submitting) return
            const capitalised =
              trimmed.charAt(0).toLocaleUpperCase() + trimmed.slice(1)
            setSubmitting(true)
            void (async () => {
              try {
                const result = await createWorkspace(serverUrl, token, {
                  name: capitalised,
                })
                onCreated(result.workspace, result.channels)
                onOpenChange(false)
                toast.success(t('chat.createdWorkspace', { name: result.workspace.name }))
              } catch (err) {
                const message =
                  formatUserError(err, t('chat.createWorkspaceError'))
                toast.error(message)
                setSubmitting(false)
              }
            })()
          }}
        >
          <div className="flex flex-col gap-2">
            <Label htmlFor="workspace-name">{t('common.name')}</Label>
            <Input
              id="workspace-name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder={t('chat.workspaceNamePlaceholder')}
              autoFocus
              maxLength={80}
              disabled={submitting}
            />
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={submitting}
            >
              {t('common.cancel')}
            </Button>
            <Button type="submit" disabled={!name.trim() || submitting}>
              {submitting ? t('common.creating') : t('common.create')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function SettingRow({
  id,
  label,
  description,
  checked,
  onCheckedChange,
}: {
  id: string
  label: string
  description: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}) {
  return (
    <div className="flex items-start justify-between gap-3 rounded-md px-1 py-2">
      <div className="flex min-w-0 flex-col gap-1">
        <Label htmlFor={id}>{label}</Label>
        <p className="text-xs text-muted-foreground">{description}</p>
      </div>
      <Switch id={id} checked={checked} onCheckedChange={onCheckedChange} />
    </div>
  )
}

function ChatShellLayout() {
  const { t, locale, formatDateTime } = useLocale()
  const { session, sessions, switchSession, signIn } = useAuth()
  const { offline, noteSuccess, noteFailure } = useServerConnection()
  const [workspaces, setWorkspaces] = useState<RailWorkspace[]>([])
  const [channelsByWorkspace, setChannelsByWorkspace] = useState<
    Record<string, ChannelConversation[]>
  >({})
  const [workspacesLoading, setWorkspacesLoading] = useState(true)
  const [channelsLoading, setChannelsLoading] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [createChannelOpen, setCreateChannelOpen] = useState(false)
  const [editChannelOpen, setEditChannelOpen] = useState(false)
  const [inviteOpen, setInviteOpen] = useState(false)
  const [acceptInviteOpen, setAcceptInviteOpen] = useState(false)
  const [acceptInviteToken, setAcceptInviteToken] = useState('')
  const [customScheduleOpen, setCustomScheduleOpen] = useState(false)
  const [editProfileOpen, setEditProfileOpen] = useState(false)
  const [resendingVerification, setResendingVerification] = useState(false)
  const [billingRefreshToken, setBillingRefreshToken] = useState(0)
  const verifyEmailHandledRef = useRef<string | null>(null)
  const [messages, setMessages] = useState<ApiMessage[]>([])
  const [historyLimited, setHistoryLimited] = useState(false)
  const [historyRetentionDays, setHistoryRetentionDays] = useState<number | null>(
    null,
  )
  const [messagesLoading, setMessagesLoading] = useState(false)
  const [sending, setSending] = useState(false)
  const [editingMessageId, setEditingMessageId] = useState<string | null>(null)
  const [editDraft, setEditDraft] = useState('')
  const [messageBusyId, setMessageBusyId] = useState<string | null>(null)
  const [deleteMessageId, setDeleteMessageId] = useState<string | null>(null)
  const [selectedMessageId, setSelectedMessageId] = useState<string | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const messagesPaneRef = useRef<HTMLDivElement>(null)
  const [focusMessageId, setFocusMessageId] = useState<string | null>(null)
  const [highlightedMessageId, setHighlightedMessageId] = useState<string | null>(
    null,
  )
  // Keeps notification deep-links from being pulled to the bottom by auto-scroll.
  const pinScrollToMessageRef = useRef(false)
  // After mark-as-unread, skip auto mark-read until the user scrolls.
  const skipAutoReadRef = useRef(false)
  const latestMessageId = messages[messages.length - 1]?.id
  const [notificationUnreadCount, setNotificationUnreadCount] = useState(0)
  const seenNotificationIdsRef = useRef<Set<string>>(new Set())
  const desktopNotifiedIdsRef = useRef<Set<string>>(new Set())
  const notificationsBootstrappedRef = useRef(false)
  const [navigation, setNavigation] = useState({
    entries: [EMPTY_LOCATION],
    index: 0,
  })

  const location = navigation.entries[navigation.index] ?? EMPTY_LOCATION
  const {
    sessionId: activeSessionId,
    workspaceId: activeWorkspaceId,
    conversationId: activeConversationId,
    view: activeView,
  } = location
  const sessionsKey = sessions.map((item) => item.id).join('|')

  const [draft, setDraft] = useState('')
  const composerInputRef = useRef<HTMLTextAreaElement>(null)
  const [detailsOpen, setDetailsOpen] = useState(false)
  const canShowDetails = useShowDetailsPanel()
  const detailsVisible = activeView === 'chat' && detailsOpen && canShowDetails

  const workspace = useMemo(
    () =>
      workspaces.find(
        (item) =>
          item.id === activeWorkspaceId &&
          (!activeSessionId || item.sessionId === activeSessionId),
      ) ?? null,
    [workspaces, activeWorkspaceId, activeSessionId],
  )
  const activeRailKey = workspace?.railKey ?? ''
  const channels = channelsByWorkspace[activeWorkspaceId] ?? []
  const [membersByHandle, setMembersByHandle] = useState(
    () => new Map<string, ChatUserRef>(),
  )
  const [workspaceMembers, setWorkspaceMembers] = useState<ChatUserRef[]>([])
  const [membersVersion, setMembersVersion] = useState(0)
  const dmMembers = useMemo(() => {
    return workspaceMembers
      .filter((member) => !session?.userId || member.userId !== session.userId)
      .sort((a, b) =>
        a.displayName.localeCompare(b.displayName, locale, { sensitivity: 'base' }),
      )
  }, [workspaceMembers, session?.userId, locale])

  const desktopUnreadCount = useMemo(() => {
    if (!session) return 0
    let total = 0
    for (const list of Object.values(channelsByWorkspace)) {
      for (const channel of list) {
        total += channel.unreadCount
      }
    }
    return total
  }, [session, channelsByWorkspace])

  useEffect(() => {
    void setDesktopBadgeCount(desktopUnreadCount)
  }, [desktopUnreadCount])

  useEffect(() => {
    return () => {
      void setDesktopBadgeCount(0)
    }
  }, [])

  const ownPresence = useOwnPresence(session?.serverUrl, session?.token)

  useEffect(() => {
    if (!session || !activeWorkspaceId) {
      setMembersByHandle(new Map())
      setWorkspaceMembers([])
      return
    }
    const controller = new AbortController()
    void listWorkspaceMembers(
      session.serverUrl,
      session.token,
      activeWorkspaceId,
      controller.signal,
    )
      .then((result) => {
        if (controller.signal.aborted) return
        const next = new Map<string, ChatUserRef>()
        const members: ChatUserRef[] = []
        for (const member of result.members) {
          const ref: ChatUserRef = {
            userId: member.userId,
            displayName: member.displayName,
            handle: member.handle ?? '',
            statusEmoji: member.statusEmoji ?? '',
            statusText: member.statusText ?? '',
            statusExpiresAt: member.statusExpiresAt ?? null,
            presence: member.presence ?? 'offline',
            hasAvatar: Boolean(member.hasAvatar),
            avatarUpdatedAt: member.avatarUpdatedAt ?? null,
            isExternal: Boolean(member.isExternal),
            homeWorkspaceName: member.homeWorkspaceName,
            homeWorkspaceIconUrl: member.homeWorkspaceIconUrl,
            homeServerUrl: member.homeServerUrl,
          }
          members.push(ref)
          if (!member.handle) continue
          next.set(member.handle.toLowerCase(), ref)
          for (const former of member.formerHandles ?? []) {
            const key = former.trim().toLowerCase()
            if (!key || next.has(key)) continue
            next.set(key, ref)
          }
        }
        setMembersByHandle(next)
        setWorkspaceMembers(members)
      })
      .catch(() => {
        if (!controller.signal.aborted) {
          setMembersByHandle(new Map())
          setWorkspaceMembers([])
        }
      })
    return () => controller.abort()
  }, [session, activeWorkspaceId, membersVersion])

  useEffect(() => {
    if (!session || !activeWorkspaceId) return
    const id = window.setInterval(() => {
      setMembersVersion((value) => value + 1)
    }, 60_000)
    return () => window.clearInterval(id)
  }, [session, activeWorkspaceId])

  useEffect(() => {
    if (!session) return
    const params = new URLSearchParams(window.location.search)
    const tokenFromQuery = params.get('token')
    if (window.location.pathname.includes('/invites/accept') && tokenFromQuery) {
      setAcceptInviteToken(tokenFromQuery)
      setAcceptInviteOpen(true)
    }
  }, [session])

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const tokenFromQuery = params.get('token')
    if (
      !window.location.pathname.includes('/verify-email') ||
      !tokenFromQuery
    ) {
      return
    }
    if (verifyEmailHandledRef.current === tokenFromQuery) return
    verifyEmailHandledRef.current = tokenFromQuery

    const serverUrl =
      session?.serverUrl || resolveServerUrl(readStoredServerHost())
    const clearVerifyQuery = () => {
      const url = new URL(window.location.href)
      url.pathname = url.pathname.replace(/\/verify-email\/?$/, '/') || '/'
      url.searchParams.delete('token')
      window.history.replaceState({}, '', url.toString())
    }

    void (async () => {
      try {
        const result = await verifyEmail(serverUrl, tokenFromQuery)
        if (session && result.user.id === session.userId) {
          signIn({
            ...session,
            email: result.user.email,
            displayName: result.user.displayName,
            notificationLevel: result.user.notificationLevel,
            locale: parseAppLocale(result.user.locale),
            emailVerified: Boolean(result.user.emailVerified),
            statusEmoji: result.user.statusEmoji ?? '',
            statusText: result.user.statusText ?? '',
            statusExpiresAt: result.user.statusExpiresAt ?? null,
            hasAvatar: Boolean(result.user.hasAvatar),
            avatarUpdatedAt: result.user.avatarUpdatedAt ?? null,
          })
        } else if (session) {
          try {
            const meResult = await me(session.serverUrl, session.token)
            signIn({
              ...session,
              email: meResult.user.email,
              displayName: meResult.user.displayName,
              notificationLevel: meResult.user.notificationLevel,
              locale: parseAppLocale(meResult.user.locale),
              emailVerified: Boolean(meResult.user.emailVerified),
              statusEmoji: meResult.user.statusEmoji ?? '',
              statusText: meResult.user.statusText ?? '',
              statusExpiresAt: meResult.user.statusExpiresAt ?? null,
              hasAvatar: Boolean(meResult.user.hasAvatar),
              avatarUpdatedAt: meResult.user.avatarUpdatedAt ?? null,
            })
          } catch {
            // Verification succeeded even if session refresh failed.
          }
        }
        toast.success(t('profile.verified'))
      } catch (err) {
        const message =
          formatUserError(err, t('profile.verifyError'))
        toast.error(message)
      } finally {
        clearVerifyQuery()
      }
    })()
  }, [session, signIn])

  useEffect(() => {
    if (sessions.length === 0) {
      setNotificationUnreadCount(0)
      seenNotificationIdsRef.current = new Set()
      desktopNotifiedIdsRef.current = new Set()
      notificationsBootstrappedRef.current = false
      return
    }
    let cancelled = false

    const notifyDesktop = async (
      items: ApiNotification[],
      level: string | undefined,
    ) => {
      if (level === 'nothing') return
      for (const item of items) {
        const key = `${item.id}`
        if (desktopNotifiedIdsRef.current.has(key)) continue
        const result = await showDesktopNotification({
          id: item.id,
          title: item.actorName || 'Dagr',
          body: item.body,
        })
        if (result.shown || result.reason !== 'focused') {
          desktopNotifiedIdsRef.current.add(key)
        }
      }
    }

    const poll = async () => {
      try {
        const results = await Promise.all(
          sessions.map(async (item) => {
            try {
              const result = await listNotifications(item.serverUrl, item.token, {
                filter: 'unread',
                limit: 20,
              })
              return { session: item, result, error: null as unknown }
            } catch (error) {
              return { session: item, result: null, error }
            }
          }),
        )
        if (cancelled) return
        let totalUnread = 0
        const unseen: { item: ApiNotification; level?: string }[] = []
        let anySuccess = false
        for (const entry of results) {
          if (!entry.result) {
            if (
              entry.session.id === session?.id &&
              entry.error &&
              isServerUnavailable(entry.error)
            ) {
              noteFailure(entry.error)
            }
            continue
          }
          anySuccess = true
          totalUnread += entry.result.unreadCount
          for (const item of entry.result.notifications) {
            const seenKey = `${entry.session.id}:${item.id}`
            if (!seenNotificationIdsRef.current.has(seenKey)) {
              unseen.push({
                item,
                level: entry.session.notificationLevel,
              })
            }
            seenNotificationIdsRef.current.add(seenKey)
          }
        }
        if (anySuccess) noteSuccess()
        setNotificationUnreadCount(totalUnread)
        if (!notificationsBootstrappedRef.current) {
          for (const entry of results) {
            for (const item of entry.result?.notifications ?? []) {
              desktopNotifiedIdsRef.current.add(item.id)
            }
          }
          notificationsBootstrappedRef.current = true
          return
        }
        if (unseen.length > 0) {
          await notifyDesktop(
            unseen.slice(0, 3).map((entry) => entry.item),
            unseen[0]?.level,
          )
        }
      } catch (error) {
        if (!cancelled) noteFailure(error)
      }
    }
    void poll()
    const timer = window.setInterval(() => {
      void poll()
    }, 5000)
    return () => {
      cancelled = true
      window.clearInterval(timer)
    }
  }, [sessions, sessionsKey, session?.id, noteFailure, noteSuccess])

  const navigateTo = (next: ShellLocation) => {
    const withSession: ShellLocation = {
      ...next,
      sessionId: next.sessionId || session?.id || '',
    }
    if (
      withSession.sessionId &&
      session?.id &&
      withSession.sessionId !== session.id
    ) {
      switchSession(withSession.sessionId)
    }
    setNavigation(({ entries, index }) => {
      const present = entries[index] ?? EMPTY_LOCATION
      if (sameLocation(withSession, present)) return { entries, index }
      return {
        entries: [...entries.slice(0, index + 1), withSession],
        index: index + 1,
      }
    })
  }

  const goBack = () => {
    setNavigation(({ entries, index }) => {
      const nextIndex = Math.max(0, index - 1)
      const next = entries[nextIndex]
      if (next?.sessionId && next.sessionId !== session?.id) {
        switchSession(next.sessionId)
      }
      return { entries, index: nextIndex }
    })
  }

  const goForward = () => {
    setNavigation(({ entries, index }) => {
      const nextIndex = Math.min(entries.length - 1, index + 1)
      const next = entries[nextIndex]
      if (next?.sessionId && next.sessionId !== session?.id) {
        switchSession(next.sessionId)
      }
      return { entries, index: nextIndex }
    })
  }

  useEffect(() => {
    if (!activeSessionId || !session || activeSessionId === session.id) return
    switchSession(activeSessionId)
  }, [activeSessionId, session, switchSession])

  const selectWorkspace = (
    nextWorkspace: RailWorkspace,
    nextChannels: ChannelConversation[],
  ) => {
    const conversationId = preferDefaultConversationId(nextChannels)
    const next: ShellLocation = {
      sessionId: nextWorkspace.sessionId,
      workspaceId: nextWorkspace.id,
      conversationId,
      view: 'chat',
    }
    sessionStorage.setItem(
      lastWorkspaceStorageKey(
        sessions.find((item) => item.id === nextWorkspace.sessionId)?.userId ??
          session?.userId ??
          '',
        nextWorkspace.serverUrl,
      ),
      nextWorkspace.id,
    )
    navigateTo(next)
  }

  useEffect(() => {
    return subscribeDeepLink((raw) => {
      const link = parseDagrDeepLink(raw)
      if (!link) {
        consumeDeepLink()
        return
      }
      if (!session) return
      if (link.kind === 'verified') {
        consumeDeepLink()
        void me(session.serverUrl, session.token)
          .then((result) => {
            signIn({
              ...session,
              email: result.user.email,
              displayName: result.user.displayName,
              notificationLevel: result.user.notificationLevel,
              locale: parseAppLocale(result.user.locale),
              emailVerified: Boolean(result.user.emailVerified),
              statusEmoji: result.user.statusEmoji ?? '',
              statusText: result.user.statusText ?? '',
              statusExpiresAt: result.user.statusExpiresAt ?? null,
              hasAvatar: Boolean(result.user.hasAvatar),
              avatarUpdatedAt: result.user.avatarUpdatedAt ?? null,
            })
            toast.success(t('profile.verified'))
          })
          .catch(() => {
            toast.success(t('profile.verified'))
          })
        return
      }
      if (workspacesLoading) return
      consumeDeepLink()
      const target = workspaces.find((item) => item.id === link.workspaceId)
      navigateTo({
        sessionId: target?.sessionId || session.id,
        workspaceId: link.workspaceId || activeWorkspaceId,
        conversationId: target
          ? preferDefaultConversationId(channelsByWorkspace[target.id] ?? [])
          : activeConversationId,
        view: 'workspace-settings',
      })
      setBillingRefreshToken((value) => value + 1)
      toast.success(t('chat.checkoutReturned'))
    })
  }, [
    session,
    signIn,
    workspaces,
    workspacesLoading,
    channelsByWorkspace,
    activeWorkspaceId,
    activeConversationId,
  ])

  useEffect(() => {
    if (sessions.length === 0) {
      setWorkspaces([])
      setChannelsByWorkspace({})
      setWorkspacesLoading(false)
      return
    }

    const controller = new AbortController()
    setWorkspacesLoading(true)

    void (async () => {
      try {
        const results = await Promise.all(
          sessions.map(async (item) => {
            try {
              const result = await listWorkspaces(
                item.serverUrl,
                item.token,
                controller.signal,
              )
              return {
                session: item,
                workspaces: result.workspaces.map((ws) =>
                  toRailWorkspace(ws, item),
                ),
                error: null as unknown,
              }
            } catch (error) {
              return { session: item, workspaces: [] as RailWorkspace[], error }
            }
          }),
        )
        if (controller.signal.aborted) return
        const nextWorkspaces = results.flatMap((entry) => entry.workspaces)
        const failedActive = results.find(
          (entry) =>
            entry.session.id === session?.id &&
            entry.error &&
            entry.workspaces.length === 0,
        )
        if (failedActive?.error) {
          noteFailure(failedActive.error)
          const message =
            formatUserError(failedActive.error, t('chat.loadWorkspacesError'))
          toast.error(message)
        } else {
          noteSuccess()
        }
        setWorkspaces(nextWorkspaces)

        if (nextWorkspaces.length === 0) {
          setNavigation((nav) => {
            const present = nav.entries[nav.index] ?? EMPTY_LOCATION
            if (!present.workspaceId) return nav
            return { entries: [EMPTY_LOCATION], index: 0 }
          })
          return
        }

        setNavigation((nav) => {
          const present = nav.entries[nav.index] ?? EMPTY_LOCATION
          const stillValid =
            present.workspaceId &&
            nextWorkspaces.some(
              (item) =>
                item.id === present.workspaceId &&
                (!present.sessionId || item.sessionId === present.sessionId),
            )
          if (stillValid) return nav

          const activeSessionWorkspaces = nextWorkspaces.filter(
            (item) => item.sessionId === (session?.id || present.sessionId),
          )
          const pool =
            activeSessionWorkspaces.length > 0
              ? activeSessionWorkspaces
              : nextWorkspaces
          const preferredSession = sessions.find(
            (item) => item.id === pool[0]?.sessionId,
          )
          const storedId = preferredSession
            ? sessionStorage.getItem(
                lastWorkspaceStorageKey(
                  preferredSession.userId,
                  preferredSession.serverUrl,
                ),
              )
            : null
          const preferred =
            pool.find((item) => item.id === storedId) ?? pool[0]
          return {
            entries: [
              {
                sessionId: preferred.sessionId,
                workspaceId: preferred.id,
                conversationId: '',
                view: 'chat',
              },
            ],
            index: 0,
          }
        })
      } finally {
        if (!controller.signal.aborted) {
          setWorkspacesLoading(false)
        }
      }
    })()

    return () => controller.abort()
  }, [sessionsKey, noteFailure, noteSuccess, session?.id, sessions])

  const channelsByWorkspaceRef = useRef(channelsByWorkspace)
  channelsByWorkspaceRef.current = channelsByWorkspace

  useEffect(() => {
    if (!session || !activeWorkspaceId) {
      setChannelsLoading(false)
      return
    }

    if (Object.hasOwn(channelsByWorkspaceRef.current, activeWorkspaceId)) {
      const existing = channelsByWorkspaceRef.current[activeWorkspaceId] ?? []
      const defaultId = preferDefaultConversationId(existing)
      if (!activeConversationId && defaultId) {
        setNavigation((nav) => {
          const present = nav.entries[nav.index] ?? EMPTY_LOCATION
          if (present.workspaceId !== activeWorkspaceId || present.conversationId) {
            return nav
          }
          const next = { ...present, conversationId: defaultId }
          const entries = [...nav.entries]
          entries[nav.index] = next
          return { ...nav, entries }
        })
      }
      setChannelsLoading(false)
      return
    }

    const controller = new AbortController()
    setChannelsLoading(true)

    void (async () => {
      try {
        const result = await listChannels(
          session.serverUrl,
          session.token,
          activeWorkspaceId,
          controller.signal,
        )
        if (controller.signal.aborted) return
        noteSuccess()
        const nextChannels = result.channels.map(toChannelConversation)
        setChannelsByWorkspace((prev) => ({
          ...prev,
          [activeWorkspaceId]: nextChannels,
        }))
        const defaultId = preferDefaultConversationId(nextChannels)
        if (!activeConversationId && defaultId) {
          setNavigation((nav) => {
            const present = nav.entries[nav.index] ?? EMPTY_LOCATION
            if (present.workspaceId !== activeWorkspaceId || present.conversationId) {
              return nav
            }
            const next = { ...present, conversationId: defaultId }
            const entries = [...nav.entries]
            entries[nav.index] = next
            return { ...nav, entries }
          })
        }
      } catch (err) {
        if (controller.signal.aborted) return
        noteFailure(err)
        const message =
          formatUserError(err, t('chat.loadChannelsError'))
        toast.error(message)
        // Do not cache an empty list on failure; that blocks later retries.
      } finally {
        if (!controller.signal.aborted) {
          setChannelsLoading(false)
        }
      }
    })()

    return () => controller.abort()
  }, [
    session,
    activeWorkspaceId,
    activeConversationId,
    noteFailure,
    noteSuccess,
  ])

  const conversation = useMemo(() => {
    return (
      channels.find((item) => item.id === activeConversationId) ??
      channels.find((item) => !item.isDm) ??
      channels[0] ??
      null
    )
  }, [channels, activeConversationId])

  const applyChannelUnread = (
    channelId: string,
    unread: ChannelUnreadResponse,
  ) => {
    if (!activeWorkspaceId) return
    setChannelsByWorkspace((prev) => ({
      ...prev,
      [activeWorkspaceId]: (prev[activeWorkspaceId] ?? []).map((item) =>
        item.id === channelId
          ? {
              ...item,
              unreadCount: unread.unreadCount,
              firstUnreadMessageId: unread.firstUnreadMessageId,
            }
          : item,
      ),
    }))
  }

  const refreshWorkspaceChannels = async (opts?: { silent?: boolean }) => {
    if (!session || !activeWorkspaceId) return
    try {
      const result = await listChannels(
        session.serverUrl,
        session.token,
        activeWorkspaceId,
      )
      noteSuccess()
      setChannelsByWorkspace((prev) => ({
        ...prev,
        [activeWorkspaceId]: result.channels.map(toChannelConversation),
      }))
    } catch (err) {
      noteFailure(err)
      if (!opts?.silent) {
        const message =
          formatUserError(err, t('chat.loadChannelsError'))
        toast.error(message)
      }
    }
  }

  const retryServerConnection = async () => {
    if (!session) {
      throw new ApiError(0, 'network_error', t('chat.serverUnreachable'))
    }
    await me(session.serverUrl, session.token)
    if (activeWorkspaceId) {
      const result = await listChannels(
        session.serverUrl,
        session.token,
        activeWorkspaceId,
      )
      setChannelsByWorkspace((prev) => ({
        ...prev,
        [activeWorkspaceId]: result.channels.map(toChannelConversation),
      }))
    }
    if (conversation && activeView === 'chat') {
      const result = await listMessages(
        session.serverUrl,
        session.token,
        conversation.id,
        { limit: 100 },
      )
      setMessages(result.messages)
      setHistoryLimited(Boolean(result.historyLimited))
      setHistoryRetentionDays(result.historyRetentionDays ?? null)
    }
  }

  const markConversationRead = async (messageId?: string) => {
    if (!session || !conversation) return
    try {
      const unread = await markChannelRead(
        session.serverUrl,
        session.token,
        conversation.id,
        messageId,
      )
      applyChannelUnread(conversation.id, unread)
    } catch {
      // Best-effort; unread badges refresh on the next channel poll.
    }
  }

  const markMessageUnread = async (messageId: string) => {
    if (!session || !conversation) return
    if (unreadMessageIds.has(messageId)) return
    skipAutoReadRef.current = true
    try {
      const unread = await markChannelUnread(
        session.serverUrl,
        session.token,
        conversation.id,
        messageId,
      )
      applyChannelUnread(conversation.id, unread)
      toast.success(t('chat.markedUnread'))
    } catch (err) {
      const message =
        formatUserError(err, t('chat.markUnreadError'))
      toast.error(message)
    }
  }

  const toggleReaction = async (messageId: string, emoji: string) => {
    if (!session || messageBusyId) return
    setMessageBusyId(messageId)
    try {
      const result = await toggleMessageReaction(
        session.serverUrl,
        session.token,
        messageId,
        emoji,
      )
      setMessages((prev) =>
        prev.map((item) => (item.id === messageId ? result.message : item)),
      )
    } catch (err) {
      const message =
        formatUserError(err, t('chat.reactionError'))
      toast.error(message)
    } finally {
      setMessageBusyId(null)
    }
  }

  useEffect(() => {
    if (!session || !activeWorkspaceId) return
    const poll = () => {
      void refreshWorkspaceChannels({ silent: true })
    }
    poll()
    const timer = window.setInterval(poll, 5000)
    const onVisible = () => {
      if (document.visibilityState === 'visible') poll()
    }
    window.addEventListener('focus', poll)
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      window.clearInterval(timer)
      window.removeEventListener('focus', poll)
      document.removeEventListener('visibilitychange', onVisible)
    }
    // refreshWorkspaceChannels closes over the latest session/workspace each render.
    // eslint-disable-next-line react-hooks/exhaustive-deps -- poll while workspace is open
  }, [session, activeWorkspaceId])

  useEffect(() => {
    if (!session || !conversation || activeView !== 'chat') {
      setMessages([])
      setHistoryLimited(false)
      setHistoryRetentionDays(null)
      setMessagesLoading(false)
      return
    }
    let cancelled = false
    setMessages([])
    setMessagesLoading(true)
    const load = async (showLoading: boolean) => {
      if (showLoading) setMessagesLoading(true)
      try {
        const result = await listMessages(
          session.serverUrl,
          session.token,
          conversation.id,
          { limit: 100 },
        )
        if (cancelled) return
        noteSuccess()
        setMessages(result.messages)
        setHistoryLimited(Boolean(result.historyLimited))
        setHistoryRetentionDays(result.historyRetentionDays ?? null)
      } catch (err) {
        if (cancelled) return
        noteFailure(err)
        if (showLoading) {
          const message =
            formatUserError(err, t('chat.loadMessagesError'))
          toast.error(message)
          if (!isServerUnavailable(err)) {
            setMessages([])
          }
        }
      } finally {
        if (!cancelled && showLoading) setMessagesLoading(false)
      }
    }
    void load(true)
    const timer = window.setInterval(() => {
      void load(false)
    }, 4000)
    return () => {
      cancelled = true
      window.clearInterval(timer)
    }
  }, [session, conversation?.id, activeView, noteFailure, noteSuccess])

  useEffect(() => {
    // Channel switches reset bottom-scroll pinning, unless a notification
    // deep-link is already targeting a message in the new channel.
    if (focusMessageId) {
      pinScrollToMessageRef.current = true
      return
    }
    pinScrollToMessageRef.current = false
    skipAutoReadRef.current = false
    setHighlightedMessageId(null)
    setEditingMessageId(null)
    setEditDraft('')
    setDeleteMessageId(null)
    setSelectedMessageId(null)
    // Only intentionally keyed on conversation id; focusMessageId is read from
    // the same commit as the channel switch (batched with navigateTo).
    // eslint-disable-next-line react-hooks/exhaustive-deps -- channel switch only
  }, [conversation?.id])

  useEffect(() => {
    if (
      activeView !== 'chat' ||
      !conversation ||
      focusMessageId ||
      pinScrollToMessageRef.current ||
      messagesLoading
    ) {
      return
    }
    const firstUnreadId = conversation.firstUnreadMessageId
    const hasUnread = conversation.unreadCount > 0 && Boolean(firstUnreadId)
    const frame = window.requestAnimationFrame(() => {
      if (hasUnread && firstUnreadId) {
        const target = messagesPaneRef.current?.querySelector(
          `[data-message-id="${firstUnreadId}"]`,
        )
        if (target instanceof HTMLElement) {
          target.scrollIntoView({ behavior: 'auto', block: 'center' })
          return
        }
      }
      messagesEndRef.current?.scrollIntoView({
        behavior: 'auto',
        block: 'end',
      })
    })
    return () => window.cancelAnimationFrame(frame)
    // Position once when the channel's messages finish loading.
    // eslint-disable-next-line react-hooks/exhaustive-deps -- channel open only
  }, [conversation?.id, messagesLoading, activeView, focusMessageId])

  useEffect(() => {
    if (
      activeView !== 'chat' ||
      !conversation ||
      focusMessageId ||
      pinScrollToMessageRef.current ||
      conversation.unreadCount > 0
    ) {
      return
    }
    const frame = window.requestAnimationFrame(() => {
      messagesEndRef.current?.scrollIntoView({
        behavior: messagesLoading ? 'auto' : 'smooth',
        block: 'end',
      })
    })
    return () => window.cancelAnimationFrame(frame)
  }, [
    latestMessageId,
    conversation?.id,
    conversation?.unreadCount,
    activeView,
    messagesLoading,
    focusMessageId,
  ])

  useEffect(() => {
    if (activeView !== 'chat' || !conversation || !latestMessageId || !session) {
      return
    }
    const end = messagesEndRef.current
    if (!end) return

    let timer: number | undefined
    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries.some((entry) => entry.isIntersecting)
        if (!visible) return
        if (skipAutoReadRef.current) return
        window.clearTimeout(timer)
        timer = window.setTimeout(() => {
          if (skipAutoReadRef.current) return
          void markConversationRead(latestMessageId)
        }, 700)
      },
      { threshold: 0.1 },
    )
    observer.observe(end)

    const onScroll = () => {
      skipAutoReadRef.current = false
    }
    const viewport =
      messagesPaneRef.current
        ?.closest('[data-slot="scroll-area"]')
        ?.querySelector('[data-slot="scroll-area-viewport"]') ?? null
    viewport?.addEventListener('scroll', onScroll, { passive: true })

    return () => {
      observer.disconnect()
      window.clearTimeout(timer)
      viewport?.removeEventListener('scroll', onScroll)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- catch-up while viewing latest
  }, [
    activeView,
    conversation?.id,
    latestMessageId,
    session,
    messagesLoading,
  ])

  useEffect(() => {
    if (
      !focusMessageId ||
      activeView !== 'chat' ||
      messagesLoading ||
      !conversation
    ) {
      return
    }

    const targetId = focusMessageId
    pinScrollToMessageRef.current = true
    const frame = window.requestAnimationFrame(() => {
      const el = document.querySelector<HTMLElement>(
        `[data-message-id="${CSS.escape(targetId)}"]`,
      )
      if (!el) {
        toast.message(t('chat.notInHistory'))
        pinScrollToMessageRef.current = false
        setFocusMessageId(null)
        return
      }
      el.scrollIntoView({ behavior: 'smooth', block: 'center' })
      setHighlightedMessageId(targetId)
      setFocusMessageId(null)
    })

    return () => window.cancelAnimationFrame(frame)
  }, [
    focusMessageId,
    messages,
    messagesLoading,
    activeView,
    conversation?.id,
  ])

  useEffect(() => {
    if (!highlightedMessageId) return
    const timer = window.setTimeout(() => {
      setHighlightedMessageId(null)
    }, 2400)
    return () => window.clearTimeout(timer)
  }, [highlightedMessageId])

  const refreshChannels = async () => {
    if (!session || !activeWorkspaceId) return
    const result = await listChannels(
      session.serverUrl,
      session.token,
      activeWorkspaceId,
    )
    setChannelsByWorkspace((prev) => ({
      ...prev,
      [activeWorkspaceId]: result.channels.map(toChannelConversation),
    }))
  }

  const focusComposer = () => {
    requestAnimationFrame(() => {
      composerInputRef.current?.focus()
    })
  }

  const navigableMessages = useMemo(
    () =>
      messages.filter(
        (message) => message.contentType !== 'application/x-dagr-system',
      ),
    [messages],
  )

  const unreadMessageIds = useMemo(() => {
    const ids = new Set<string>()
    if (!conversation || conversation.unreadCount <= 0) return ids
    const firstUnreadId = conversation.firstUnreadMessageId
    if (!firstUnreadId) return ids
    const start = messages.findIndex((message) => message.id === firstUnreadId)
    if (start < 0) return ids
    for (let i = start; i < messages.length; i += 1) {
      const message = messages[i]
      if (message.contentType === 'application/x-dagr-system') continue
      ids.add(message.id)
    }
    return ids
  }, [
    conversation?.unreadCount,
    conversation?.firstUnreadMessageId,
    messages,
  ])

  const scrollMessageIntoView = (messageId: string) => {
    requestAnimationFrame(() => {
      const el = document.querySelector<HTMLElement>(
        `[data-message-id="${CSS.escape(messageId)}"]`,
      )
      el?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
    })
  }

  const selectMessage = (messageId: string | null) => {
    setSelectedMessageId(messageId)
    if (!messageId) return
    pinScrollToMessageRef.current = true
    scrollMessageIntoView(messageId)
    requestAnimationFrame(() => {
      messagesPaneRef.current?.focus()
    })
  }

  const moveMessageSelection = (direction: -1 | 1) => {
    if (navigableMessages.length === 0) return
    const currentIndex = selectedMessageId
      ? navigableMessages.findIndex((message) => message.id === selectedMessageId)
      : -1

    if (direction === -1) {
      const nextIndex =
        currentIndex === -1
          ? navigableMessages.length - 1
          : Math.max(0, currentIndex - 1)
      selectMessage(navigableMessages[nextIndex]?.id ?? null)
      return
    }

    if (currentIndex === -1) return
    if (currentIndex >= navigableMessages.length - 1) {
      setSelectedMessageId(null)
      pinScrollToMessageRef.current = false
      focusComposer()
      return
    }
    selectMessage(navigableMessages[currentIndex + 1]?.id ?? null)
  }

  const selectedMessage = useMemo(
    () => messages.find((message) => message.id === selectedMessageId) ?? null,
    [messages, selectedMessageId],
  )
  const selectedIsOwn =
    Boolean(session?.userId) &&
    Boolean(selectedMessage) &&
    selectedMessage?.authorId === session?.userId

  const beginEditMessage = (message: ApiMessage) => {
    setSelectedMessageId(message.id)
    setEditingMessageId(message.id)
    setEditDraft(message.body)
  }

  const cancelEditMessage = () => {
    setEditingMessageId(null)
    setEditDraft('')
    setSelectedMessageId(null)
    pinScrollToMessageRef.current = false
    focusComposer()
  }

  useEffect(() => {
    if (
      selectedMessageId &&
      !messages.some((message) => message.id === selectedMessageId)
    ) {
      setSelectedMessageId(null)
    }
  }, [messages, selectedMessageId])

  const clearMessageSelection = () => {
    setSelectedMessageId(null)
    pinScrollToMessageRef.current = false
  }

  const shouldKeepMessageSelection = (target: EventTarget | null) => {
    const element =
      target instanceof Element
        ? target
        : target instanceof Node
          ? target.parentElement
          : null
    if (!element) return false
    if (element.closest('[data-slot="context-menu-content"]')) return true
    if (element.closest('[data-slot="context-menu-sub-content"]')) return true
    if (element.closest('[data-slot="dropdown-menu-content"]')) return true
    if (element.closest('[data-slot="popover-content"]')) return true
    if (element.closest('[data-slot="dropdown-menu-trigger"]')) return true
    if (element.closest('[role="menu"]')) return true
    return false
  }

  useEffect(() => {
    if (
      !selectedMessageId ||
      editingMessageId ||
      deleteMessageId ||
      activeView !== 'chat'
    ) {
      return
    }

    const onPointerDown = (event: PointerEvent) => {
      if (shouldKeepMessageSelection(event.target)) return
      clearMessageSelection()
    }

    window.addEventListener('pointerdown', onPointerDown, true)
    return () => window.removeEventListener('pointerdown', onPointerDown, true)
  }, [selectedMessageId, editingMessageId, deleteMessageId, activeView])

  useEffect(() => {
    if (
      !selectedMessageId ||
      editingMessageId ||
      deleteMessageId ||
      activeView !== 'chat'
    ) {
      return
    }

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.isComposing) return
      const target = event.target as HTMLElement | null
      if (
        target &&
        (target.tagName === 'INPUT' ||
          target.tagName === 'TEXTAREA' ||
          target.isContentEditable)
      ) {
        return
      }

      if (event.key === 'ArrowUp') {
        event.preventDefault()
        moveMessageSelection(-1)
        return
      }
      if (event.key === 'ArrowDown') {
        event.preventDefault()
        moveMessageSelection(1)
        return
      }
      if (event.key === 'Escape') {
        event.preventDefault()
        clearMessageSelection()
        focusComposer()
        return
      }
      if (!selectedMessage) return
      if (
        (event.key === 'u' || event.key === 'U') &&
        !selectedIsOwn &&
        !unreadMessageIds.has(selectedMessage.id)
      ) {
        event.preventDefault()
        void markMessageUnread(selectedMessage.id)
        return
      }
      if (!selectedIsOwn) return
      if (event.key === 'e' || event.key === 'E') {
        event.preventDefault()
        beginEditMessage(selectedMessage)
        return
      }
      if (event.key === 'Delete' || event.key === 'Backspace') {
        event.preventDefault()
        setDeleteMessageId(selectedMessage.id)
      }
    }

    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
    // Navigation helpers close over the latest selection state each render.
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentional selection keyboard mode
  }, [
    selectedMessageId,
    selectedMessage,
    selectedIsOwn,
    editingMessageId,
    deleteMessageId,
    activeView,
    navigableMessages,
    unreadMessageIds,
  ])

  const saveEditMessage = async (messageId: string) => {
    if (!session || messageBusyId) return
    const body = editDraft.trim()
    if (!body) return
    setMessageBusyId(messageId)
    try {
      const result = await updateMessage(
        session.serverUrl,
        session.token,
        messageId,
        body,
      )
      setMessages((prev) =>
        prev.map((item) => (item.id === messageId ? result.message : item)),
      )
      cancelEditMessage()
      toast.success(t('chat.messageUpdated'))
    } catch (err) {
      const message =
        formatUserError(err, t('chat.updateMessageError'))
      toast.error(message)
    } finally {
      setMessageBusyId(null)
    }
  }

  const confirmDeleteMessage = async () => {
    if (!session || !deleteMessageId || messageBusyId) return
    const messageId = deleteMessageId
    setMessageBusyId(messageId)
    try {
      await deleteMessage(session.serverUrl, session.token, messageId)
      setMessages((prev) => prev.filter((item) => item.id !== messageId))
      if (editingMessageId === messageId) cancelEditMessage()
      if (selectedMessageId === messageId) setSelectedMessageId(null)
      setDeleteMessageId(null)
      toast.success(t('chat.messageDeleted'))
      focusComposer()
    } catch (err) {
      const message =
        formatUserError(err, t('chat.deleteMessageError'))
      toast.error(message)
    } finally {
      setMessageBusyId(null)
    }
  }

  const sendDraft = async () => {
    if (!session || !conversation || !draft.trim() || sending || offline) return
    setSending(true)
    pinScrollToMessageRef.current = false
    try {
      const result = await postMessage(
        session.serverUrl,
        session.token,
        conversation.id,
        draft.trim(),
      )
      noteSuccess()
      setMessages((prev) =>
        prev.some((item) => item.id === result.message.id)
          ? prev
          : [...prev, result.message],
      )
      setDraft('')
      skipAutoReadRef.current = false
      void markConversationRead(result.message.id)
    } catch (err) {
      noteFailure(err)
      const message = formatUserError(err, t('chat.sendError'))
      toast.error(message)
    } finally {
      setSending(false)
      focusComposer()
    }
  }

  const scheduleDraft = async (sendAt: Date) => {
    if (!session || !conversation || !draft.trim() || sending) return
    setSending(true)
    try {
      await scheduleMessage(session.serverUrl, session.token, conversation.id, {
        body: draft.trim(),
        sendAt: sendAt.toISOString(),
      })
      toast.success(t('chat.scheduledFor', { datetime: formatDateTime(sendAt) }))
      setDraft('')
    } catch (err) {
      const message =
        formatUserError(err, t('chat.scheduleError'))
      toast.error(message)
    } finally {
      setSending(false)
      focusComposer()
    }
  }

  const title = conversation
    ? conversationDisplayName(conversation)
    : 'conversation'
  const composerPlaceholder = conversation?.isDm
    ? t('chat.messageUser', { name: title })
    : t('chat.messageChannel', { name: title })
  const hasWorkspaces = workspaces.length > 0
  const titleBarSubtitle = (() => {
    if (activeView === 'notifications') return t('notifications.title')
    if (activeView === 'app-settings') return t('common.settings')
    if (activeView === 'workspace-settings') return t('workspace.settings')
    if (!conversation) return undefined
    if (conversation.isDm) return title
    return conversation.isPrivate ? conversation.name : `#${conversation.name}`
  })()

  const upsertConversation = (channel: ApiChannel) => {
    if (!activeWorkspaceId) return
    const next = toChannelConversation(channel)
    setChannelsByWorkspace((prev) => {
      const current = prev[activeWorkspaceId] ?? []
      const index = current.findIndex((item) => item.id === next.id)
      if (index === -1) {
        return { ...prev, [activeWorkspaceId]: [...current, next] }
      }
      const copy = [...current]
      copy[index] = { ...copy[index], ...next }
      return { ...prev, [activeWorkspaceId]: copy }
    })
  }

  const startDirectMessageWithUser = async (user: ChatUserRef) => {
    if (!session || !activeWorkspaceId) return
    if (user.userId === session.userId) return
    try {
      const result = await openDirectMessage(
        session.serverUrl,
        session.token,
        activeWorkspaceId,
        user.userId,
      )
      upsertConversation(result.channel)
      setDetailsOpen(false)
      navigateTo({
        workspaceId: activeWorkspaceId,
        conversationId: result.channel.id,
        view: 'chat',
      })
    } catch (err) {
      noteFailure(err)
      const message =
        formatUserError(err, t('chat.openDmError'))
      toast.error(message)
    }
  }

  const resizeComposer = () => {
    const el = composerInputRef.current
    if (!el) return
    el.style.height = '0px'
    el.style.height = `${Math.min(el.scrollHeight, 160)}px`
  }

  const applyComposerEdit = (edit: ComposerSelectionEdit) => {
    setDraft(edit.next)
    requestAnimationFrame(() => {
      const el = composerInputRef.current
      if (!el) return
      el.focus()
      el.setSelectionRange(edit.selectionStart, edit.selectionEnd)
      resizeComposer()
    })
  }

  const insertComposerText = (text: string) => {
    const input = composerInputRef.current
    const start = input?.selectionStart ?? draft.length
    const end = input?.selectionEnd ?? draft.length
    applyComposerEdit({
      next: `${draft.slice(0, start)}${text}${draft.slice(end)}`,
      selectionStart: start + text.length,
      selectionEnd: start + text.length,
    })
  }

  useEffect(() => {
    resizeComposer()
  }, [draft])

  return (
    <TrustedDomainsProvider
      serverUrl={session?.serverUrl}
      token={session?.token}
      workspaces={workspaces}
    >
    <DirectMessageProvider
      value={{
        currentUserId: session?.userId,
        onMessageUser: (user) => {
          void startDirectMessageWithUser(user)
        },
      }}
    >
    <div
      className="flex h-full min-h-0 w-full flex-col overflow-hidden"
      onPointerDownCapture={(event) => {
        if (
          !selectedMessageId ||
          editingMessageId ||
          deleteMessageId ||
          activeView !== 'chat'
        ) {
          return
        }
        if (shouldKeepMessageSelection(event.target)) return
        clearMessageSelection()
      }}
    >
      <TitleBar
        title={workspace?.name ?? 'Dagr'}
        subtitle={titleBarSubtitle}
        initials={workspace?.initials ?? 'D'}
        brandMark={
          workspace && session ? (
            <WorkspaceIconMark
              workspaceId={workspace.id}
              name={workspace.name}
              hasIcon={workspace.hasIcon}
              iconUpdatedAt={workspace.iconUpdatedAt}
              serverUrl={session.serverUrl}
              token={session.token}
              className="size-4 rounded-[4px] object-cover"
              initialsClassName="flex size-4 items-center justify-center rounded-[4px] bg-primary text-[9px] font-semibold text-primary-foreground"
            />
          ) : undefined
        }
        canGoBack={navigation.index > 0}
        canGoForward={navigation.index < navigation.entries.length - 1}
        onBack={goBack}
        onForward={goForward}
      />

      <ServerConnectionBanner onRetry={retryServerConnection} />
      <EmailVerificationBanner
        visible={Boolean(session && session.emailVerified === false)}
        resending={resendingVerification}
        onOpenProfile={() => setEditProfileOpen(true)}
        onResend={() => {
          if (!session || resendingVerification) return
          setResendingVerification(true)
          void (async () => {
            try {
              await resendVerificationEmail(session.serverUrl, session.token)
              toast.success(t('profile.verificationSent'))
            } catch (err) {
              const message =
                formatUserError(err, t('profile.verificationError'))
              toast.error(message)
            } finally {
              setResendingVerification(false)
            }
          })()
        }}
      />

      {session && (
        <>
          <CreateWorkspaceDialog
            open={createOpen}
            onOpenChange={setCreateOpen}
            serverUrl={session.serverUrl}
            token={session.token}
            onCreated={(created, createdChannels) => {
              const rail = toRailWorkspace(created, session)
              const nextChannels = createdChannels.map(toChannelConversation)
              setWorkspaces((prev) => {
                if (prev.some((item) => item.railKey === rail.railKey)) return prev
                return [...prev, rail]
              })
              setChannelsByWorkspace((prev) => ({
                ...prev,
                [rail.id]: nextChannels,
              }))
              selectWorkspace(rail, nextChannels)
            }}
          />
          <Dialog
            open={Boolean(deleteMessageId)}
            onOpenChange={(open) => {
              if (!open && !messageBusyId) setDeleteMessageId(null)
            }}
          >
            <DialogContent>
              <DialogHeader>
                <DialogTitle>{t('chat.deleteMessage')}</DialogTitle>
                <DialogDescription>
                  {t('chat.deleteMessageHint')}
                </DialogDescription>
              </DialogHeader>
              <DialogFooter>
                <Button
                  type="button"
                  variant="outline"
                  disabled={Boolean(messageBusyId)}
                  onClick={() => setDeleteMessageId(null)}
                >
                  {t('common.cancel')}
                </Button>
                <Button
                  type="button"
                  variant="destructive"
                  disabled={Boolean(messageBusyId)}
                  onClick={() => void confirmDeleteMessage()}
                >
                  {messageBusyId && messageBusyId === deleteMessageId
                    ? t('common.deleting')
                    : t('common.delete')}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
          {activeWorkspaceId && (
            <>
              <CreateChannelDialog
                open={createChannelOpen}
                onOpenChange={setCreateChannelOpen}
                serverUrl={session.serverUrl}
                token={session.token}
                workspaceId={activeWorkspaceId}
                onCreated={(channel) => {
                  const next = toChannelConversation(channel)
                  setChannelsByWorkspace((prev) => ({
                    ...prev,
                    [activeWorkspaceId]: [...(prev[activeWorkspaceId] ?? []), next],
                  }))
                  navigateTo({
                    workspaceId: activeWorkspaceId,
                    conversationId: channel.id,
                    view: 'chat',
                  })
                }}
              />
              <InviteWorkspaceDialog
                open={inviteOpen}
                onOpenChange={setInviteOpen}
                serverUrl={session.serverUrl}
                token={session.token}
                workspaceId={activeWorkspaceId}
              />
            </>
          )}
          <EditChannelDialog
            open={editChannelOpen}
            onOpenChange={setEditChannelOpen}
            channel={conversation}
            serverUrl={session.serverUrl}
            token={session.token}
            onUpdated={(channel) => {
              const next = toChannelConversation(channel)
              setChannelsByWorkspace((prev) => ({
                ...prev,
                [activeWorkspaceId]: (prev[activeWorkspaceId] ?? []).map((item) =>
                  item.id === next.id ? next : item,
                ),
              }))
            }}
            onDeleted={(channelId) => {
              setChannelsByWorkspace((prev) => {
                const list = (prev[activeWorkspaceId] ?? []).filter(
                  (item) => item.id !== channelId,
                )
                return { ...prev, [activeWorkspaceId]: list }
              })
              navigateTo({
                workspaceId: activeWorkspaceId,
                conversationId: '',
                view: 'chat',
              })
              void refreshChannels()
            }}
          />
          <AcceptInviteDialog
            open={acceptInviteOpen}
            onOpenChange={setAcceptInviteOpen}
            serverUrl={session.serverUrl}
            token={session.token}
            initialToken={acceptInviteToken}
            onAccepted={(created) => {
              const rail = toRailWorkspace(created, session)
              setWorkspaces((prev) => {
                if (prev.some((item) => item.railKey === rail.railKey)) return prev
                return [...prev, rail]
              })
              setChannelsByWorkspace((prev) => {
                const next = { ...prev }
                delete next[rail.id]
                return next
              })
              navigateTo({
                sessionId: rail.sessionId,
                workspaceId: rail.id,
                conversationId: '',
                view: 'chat',
              })
            }}
          />
          <CustomScheduleDialog
            open={customScheduleOpen}
            onOpenChange={setCustomScheduleOpen}
            onConfirm={(sendAt) => {
              void scheduleDraft(sendAt)
            }}
          />
        </>
      )}

      <div className="flex min-h-0 flex-1 overflow-hidden">
        <div className="flex h-full min-h-0 w-(--sidebar-width) shrink-0 flex-row overflow-visible border-r bg-sidebar text-sidebar-foreground">
          <WorkspaceRail
            workspaces={workspaces}
            activeRailKey={activeRailKey}
            onSelectWorkspace={(railKey) => {
              const nextWorkspace = workspaces.find(
                (item) => item.railKey === railKey,
              )
              if (!nextWorkspace) return
              const nextChannels = channelsByWorkspace[nextWorkspace.id] ?? []
              selectWorkspace(nextWorkspace, nextChannels)
            }}
            onAddWorkspace={() => setCreateOpen(true)}
            activeView={activeView}
            notificationUnreadCount={notificationUnreadCount}
            onOpenNotifications={() =>
              navigateTo({
                sessionId: workspace?.sessionId || session?.id || '',
                workspaceId: activeWorkspaceId,
                conversationId: conversation?.id ?? activeConversationId,
                view: 'notifications',
              })
            }
            onOpenAppSettings={() =>
              navigateTo({
                sessionId: workspace?.sessionId || session?.id || '',
                workspaceId: activeWorkspaceId,
                conversationId: conversation?.id ?? activeConversationId,
                view: 'app-settings',
              })
            }
            onProfileSaved={() => setMembersVersion((value) => value + 1)}
            onSwitchAccount={(sessionId) => {
              switchSession(sessionId)
              const preferred =
                workspaces.find((item) => item.sessionId === sessionId) ?? null
              if (!preferred) return
              const nextChannels = channelsByWorkspace[preferred.id] ?? []
              selectWorkspace(preferred, nextChannels)
            }}
            presence={ownPresence}
            serverUrl={session?.serverUrl}
            token={session?.token}
            editProfileOpen={editProfileOpen}
            onEditProfileOpenChange={setEditProfileOpen}
          />
          <ConversationPanel
            workspace={workspace}
            channels={channels}
            channelsLoading={channelsLoading}
            activeConversationId={
              activeView === 'chat' ? (conversation?.id ?? '') : ''
            }
            onSelectConversation={(id) => {
              const next = channels.find((item) => item.id === id)
              if (next?.isDm) setDetailsOpen(false)
              navigateTo({
                workspaceId: activeWorkspaceId,
                conversationId: id,
                view: 'chat',
              })
            }}
            onOpenWorkspaceSettings={() => {
              if (!workspace) return
              navigateTo({
                workspaceId: workspace.id,
                conversationId: conversation?.id ?? activeConversationId,
                view: 'workspace-settings',
              })
            }}
            onCreateChannel={() => setCreateChannelOpen(true)}
            onInviteToWorkspace={() => setInviteOpen(true)}
            dmMembers={dmMembers}
            onSelectDmMember={(user) => {
              const existing = channels.find(
                (item) => item.isDm && item.peerUserId === user.userId,
              )
              if (existing) {
                setDetailsOpen(false)
                navigateTo({
                  workspaceId: activeWorkspaceId,
                  conversationId: existing.id,
                  view: 'chat',
                })
                return
              }
              void startDirectMessageWithUser(user)
            }}
            serverUrl={session?.serverUrl}
            token={session?.token}
          />
        </div>

        {activeView === 'notifications' ? (
          <NotificationsPage
            onUnreadCountChange={setNotificationUnreadCount}
            onOpenNotification={(item) => {
              if (item.sessionId && item.sessionId !== session?.id) {
                switchSession(item.sessionId)
              }
              if (item.workspaceId && item.channelId) {
                if (item.messageId) {
                  pinScrollToMessageRef.current = true
                  setFocusMessageId(item.messageId)
                } else {
                  pinScrollToMessageRef.current = false
                  setFocusMessageId(null)
                  setHighlightedMessageId(null)
                }
                navigateTo({
                  sessionId: item.sessionId || session?.id || '',
                  workspaceId: item.workspaceId,
                  conversationId: item.channelId,
                  view: 'chat',
                })
                return
              }
              setFocusMessageId(null)
              setHighlightedMessageId(null)
              if (item.workspaceId) {
                navigateTo({
                  sessionId: item.sessionId || session?.id || '',
                  workspaceId: item.workspaceId,
                  conversationId: '',
                  view: 'chat',
                })
              }
            }}
          />
        ) : activeView === 'app-settings' ? (
          <AppSettingsPage
            onBack={() =>
              navigateTo({
                sessionId: workspace?.sessionId || session?.id || '',
                workspaceId: activeWorkspaceId,
                conversationId: conversation?.id ?? activeConversationId,
                view: 'chat',
              })
            }
          />
        ) : activeView === 'workspace-settings' && workspace && session ? (
          <WorkspaceSettingsPage
            workspace={{
              id: workspace.id,
              name: workspace.name,
              slug: workspace.slug,
              role: workspace.role,
              hasIcon: workspace.hasIcon,
              iconUpdatedAt: workspace.iconUpdatedAt,
            }}
            serverUrl={workspace.serverUrl}
            token={workspace.token}
            canManage={workspace.role === 'owner' || workspace.role === 'admin'}
            billingRefreshToken={billingRefreshToken}
            currentUserId={session.userId}
            onBack={() =>
              navigateTo({
                sessionId: workspace.sessionId,
                workspaceId: workspace.id,
                conversationId: conversation?.id ?? activeConversationId,
                view: 'chat',
              })
            }
            onLeftWorkspace={() => {
              const remaining = workspaces.filter(
                (item) => item.railKey !== workspace.railKey,
              )
              setWorkspaces(remaining)
              setChannelsByWorkspace((prev) => {
                const next = { ...prev }
                delete next[workspace.id]
                return next
              })
              if (remaining[0]) {
                const nextChannels = channelsByWorkspace[remaining[0].id] ?? []
                selectWorkspace(remaining[0], nextChannels)
              } else {
                setNavigation({ entries: [EMPTY_LOCATION], index: 0 })
              }
            }}
            onRenamed={(updated) => {
              const rail = toRailWorkspace(updated, {
                id: workspace.sessionId,
                serverUrl: workspace.serverUrl,
                serverLabel: workspace.serverLabel,
                token: workspace.token,
              })
              setWorkspaces((prev) =>
                prev.map((item) =>
                  item.railKey === workspace.railKey ? rail : item,
                ),
              )
            }}
            onIconChanged={(updated) => {
              const rail = toRailWorkspace(updated, {
                id: workspace.sessionId,
                serverUrl: workspace.serverUrl,
                serverLabel: workspace.serverLabel,
                token: workspace.token,
              })
              setWorkspaces((prev) =>
                prev.map((item) =>
                  item.railKey === workspace.railKey ? rail : item,
                ),
              )
            }}
            onDeleted={(workspaceId) => {
              const remaining = workspaces.filter(
                (item) =>
                  !(
                    item.id === workspaceId &&
                    item.sessionId === workspace.sessionId
                  ),
              )
              setWorkspaces(remaining)
              setChannelsByWorkspace((prev) => {
                const next = { ...prev }
                delete next[workspaceId]
                return next
              })
              sessionStorage.removeItem(
                lastWorkspaceStorageKey(
                  session.userId,
                  workspace.serverUrl,
                ),
              )
              if (remaining[0]) {
                const nextChannels = channelsByWorkspace[remaining[0].id] ?? []
                selectWorkspace(remaining[0], nextChannels)
              } else {
                setNavigation({ entries: [EMPTY_LOCATION], index: 0 })
              }
            }}
          />
        ) : workspacesLoading ? (
          <AppLoadingScreen
            label={t('app.loadingWorkspaces')}
            className="flex-1"
          />
        ) : !hasWorkspaces ? (
          <div className="flex flex-1 flex-col items-center justify-center gap-4 px-6 text-center">
            <div className="flex max-w-sm flex-col gap-2">
              <h1 className="text-lg font-semibold">{t('chat.firstWorkspaceTitle')}</h1>
              <p className="text-sm text-muted-foreground">
                {t('chat.firstWorkspaceHint')}
              </p>
            </div>
            <Button onClick={() => setCreateOpen(true)}>
              <PlusIcon strokeWidth={2} data-icon="inline-start" />
              {t('chat.createWorkspace')}
            </Button>
          </div>
        ) : (
          <ResizablePanelGroup
            orientation="horizontal"
            className="min-h-0 min-w-0 flex-1"
          >
            <ResizablePanel
              id="chat"
              defaultSize={detailsVisible ? '70%' : '100%'}
              minSize="40%"
            >
              <SidebarInset className="flex h-full min-h-0 min-w-0 flex-col">
                <header className="flex h-14 shrink-0 items-center gap-2 border-b bg-background px-3">
                  {conversation?.isDm && conversation.peerUserId && session ? (
                    <UserAvatarMark
                      userId={conversation.peerUserId}
                      name={title}
                      hasAvatar={conversation.peerHasAvatar}
                      avatarUpdatedAt={conversation.peerAvatarUpdatedAt}
                      serverUrl={session.serverUrl}
                      token={session.token}
                      className="size-7 after:hidden"
                      imageClassName="rounded-md"
                      fallbackClassName="rounded-md text-[10px]"
                    />
                  ) : conversation?.isPrivate ? (
                    <LockKeyIcon strokeWidth={2} className="text-muted-foreground" />
                  ) : (
                    <HashIcon strokeWidth={2} className="text-muted-foreground" />
                  )}
                  <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                    <div className="flex min-w-0 items-center gap-2">
                      <h1 className="truncate text-sm font-semibold">{title}</h1>
                    </div>
                    <p className="truncate text-xs text-muted-foreground">
                      {conversation?.isDm
                        ? conversation.peerHandle
                          ? `@${conversation.peerHandle}`
                          : t('chat.directMessage')
                        : conversation?.topic?.trim()
                          ? conversation.topic
                          : `${workspace?.name ?? t('chat.workspace')} · ${conversation?.isPrivate ? t('chat.privateChannel') : t('chat.publicChannel')}`}
                    </p>
                  </div>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        aria-label={
                          conversation?.isDm ? t('chat.conversationMenu') : t('chat.channelMenu')
                        }
                      >
                        <DotsThreeOutlineIcon strokeWidth={2} data-icon />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" className="min-w-48">
                      {!conversation?.isDm ? (
                        <DropdownMenuItem onSelect={() => setEditChannelOpen(true)}>
                          <GearIcon strokeWidth={2} />
                          {t('chat.channelSettings')}
                        </DropdownMenuItem>
                      ) : null}
                      <DropdownMenuItem onSelect={() => setInviteOpen(true)}>
                        <UserPlusIcon strokeWidth={2} />
                        {t('chat.inviteToWorkspace')}
                      </DropdownMenuItem>
                      <DropdownMenuItem onSelect={() => setAcceptInviteOpen(true)}>
                        <CheckCircleIcon strokeWidth={2} />
                        {t('chat.acceptInviteEllipsis')}
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                  {canShowDetails && !conversation?.isDm ? (
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label={detailsOpen ? t('chat.hideDetails') : t('chat.showDetails')}
                      aria-pressed={detailsOpen}
                      onClick={() => setDetailsOpen((open) => !open)}
                    >
                      <SidebarSimpleIcon strokeWidth={2} data-icon />
                    </Button>
                  ) : null}
                </header>

                <ScrollArea className="min-h-0 flex-1">
                  <div
                    ref={messagesPaneRef}
                    tabIndex={-1}
                    className="flex flex-col gap-4 p-4 outline-none"
                    aria-label={t('chat.messages')}
                  >
                    {messagesLoading ? (
                      <ChannelMessagesSkeleton />
                    ) : messages.length === 0 ? (
                      <p className="text-sm text-muted-foreground">
                        {t('chat.noMessages', { name: title })}
                      </p>
                    ) : (
                      <>
                      {historyLimited ? (
                        <div className="rounded-lg border bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
                          {t('chat.historyLimit', { count: historyRetentionDays ?? 90 })}
                        </div>
                      ) : null}
                      {messages.map((message, index) => {
                        let member: ChatUserRef | undefined
                        for (const item of membersByHandle.values()) {
                          if (item.userId === message.authorId) {
                            member = item
                            break
                          }
                        }
                        const authorUser: ChatUserRef = {
                          userId: message.authorId,
                          displayName:
                            message.authorName ||
                            member?.displayName ||
                            t('common.member'),
                          handle:
                            message.authorHandle || member?.handle || '',
                          statusEmoji: member?.statusEmoji ?? '',
                          statusText: member?.statusText ?? '',
                          statusExpiresAt: member?.statusExpiresAt ?? null,
                          presence: member?.presence ?? 'offline',
                          hasAvatar:
                            Boolean(message.authorHasAvatar) ||
                            Boolean(member?.hasAvatar),
                          avatarUpdatedAt:
                            message.authorAvatarUpdatedAt ??
                            member?.avatarUpdatedAt ??
                            null,
                        }
                        const authorCustomStatus = formatCustomStatus(
                          authorUser.statusEmoji,
                          authorUser.statusText,
                          authorUser.statusExpiresAt,
                        )
                        const previous = messages[index - 1]
                        const showDayDivider =
                          !previous ||
                          startOfLocalDay(new Date(previous.createdAt)) !==
                            startOfLocalDay(new Date(message.createdAt))
                        const dayLabel = formatMessageDayLabel(message.createdAt)
                        const timeLabel = formatMessageTime(message.createdAt)
                        const isSystem =
                          message.contentType === 'application/x-dagr-system'
                        const isHighlighted = highlightedMessageId === message.id
                        const isSelected = selectedMessageId === message.id
                        const isOwn =
                          Boolean(session?.userId) &&
                          message.authorId === session?.userId
                        const isEditing = editingMessageId === message.id
                        const isBusy = messageBusyId === message.id
                        const showUnreadDivider =
                          !isSystem &&
                          Boolean(conversation?.firstUnreadMessageId) &&
                          conversation?.firstUnreadMessageId === message.id &&
                          (conversation?.unreadCount ?? 0) > 0
                        const displayBody = messageBodyWithoutGifLinks(
                          message.body,
                          message.linkPreviews,
                        )
                        const isUnread = unreadMessageIds.has(message.id)
                        const messageRow = (
                          <div
                            data-message-id={message.id}
                            className={cn(
                              'group/message relative flex items-start gap-3 rounded-lg px-1 py-1.5 transition-colors hover:bg-muted/50',
                              isHighlighted && 'message-highlight',
                              isSelected && 'message-selected',
                            )}
                          >
                            {!isEditing ? (
                              <MessageActionBar
                                isOwn={isOwn}
                                isUnread={isUnread}
                                disabled={isBusy}
                                forceVisible={isSelected}
                                messageBody={message.body}
                                onReact={(emoji) => {
                                  void toggleReaction(message.id, emoji)
                                }}
                                onMarkUnread={() => {
                                  void markMessageUnread(message.id)
                                }}
                                onEdit={() => beginEditMessage(message)}
                                onDelete={() => {
                                  setSelectedMessageId(message.id)
                                  setDeleteMessageId(message.id)
                                }}
                              />
                            ) : null}
                            <UserHandle
                              user={authorUser}
                              serverUrl={session?.serverUrl}
                              token={session?.token}
                              className="shrink-0 self-start rounded-md font-normal hover:no-underline"
                            >
                              <UserAvatarMark
                                userId={authorUser.userId}
                                name={authorUser.displayName}
                                hasAvatar={authorUser.hasAvatar}
                                avatarUpdatedAt={authorUser.avatarUpdatedAt}
                                presence={authorUser.presence}
                                showPresence
                                serverUrl={session?.serverUrl}
                                token={session?.token}
                                className="size-9 rounded-md after:rounded-md"
                                imageClassName="rounded-md"
                                fallbackClassName="rounded-md text-xs"
                              />
                            </UserHandle>
                            <div className="flex min-w-0 flex-1 flex-col gap-1">
                              <div className="flex min-w-0 flex-wrap items-baseline gap-x-2 gap-y-0.5">
                                <UserHandle
                                  user={authorUser}
                                  serverUrl={session?.serverUrl}
                                  token={session?.token}
                                  className="text-sm"
                                >
                                  {authorUser.displayName}
                                </UserHandle>
                                {hasCustomStatus(
                                  authorUser.statusEmoji,
                                  authorUser.statusText,
                                  authorUser.statusExpiresAt,
                                ) ? (
                                  <span
                                    className="truncate text-xs text-muted-foreground"
                                    title={authorCustomStatus}
                                  >
                                    {authorCustomStatus}
                                  </span>
                                ) : null}
                                {timeLabel ? (
                                  <time
                                    dateTime={message.createdAt}
                                    title={formatMessageTimestampTitle(
                                      message.createdAt,
                                    )}
                                    className="text-xs text-muted-foreground"
                                  >
                                    {timeLabel}
                                  </time>
                                ) : null}
                                {message.editedAt ? (
                                  <span className="text-xs text-muted-foreground">
                                    {t('chat.edited')}
                                  </span>
                                ) : null}
                              </div>
                              {isEditing ? (
                                <form
                                  className="flex flex-col gap-2"
                                  onSubmit={(event) => {
                                    event.preventDefault()
                                    void saveEditMessage(message.id)
                                  }}
                                >
                                  <Textarea
                                    value={editDraft}
                                    onChange={(event) =>
                                      setEditDraft(event.target.value)
                                    }
                                    onKeyDown={(event) => {
                                      if (
                                        event.key === 'Escape' &&
                                        !event.nativeEvent.isComposing
                                      ) {
                                        event.preventDefault()
                                        cancelEditMessage()
                                        return
                                      }
                                      if (
                                        event.key === 'Enter' &&
                                        (event.metaKey || event.ctrlKey) &&
                                        !event.nativeEvent.isComposing
                                      ) {
                                        event.preventDefault()
                                        void saveEditMessage(message.id)
                                      }
                                    }}
                                    rows={3}
                                    autoFocus
                                    disabled={isBusy}
                                    aria-label={t('chat.editMessage')}
                                    className="min-h-16 resize-y"
                                  />
                                  <div className="flex flex-wrap items-center gap-2">
                                    <Button
                                      type="submit"
                                      size="sm"
                                      disabled={!editDraft.trim() || isBusy}
                                    >
                                      {isBusy ? t('common.saving') : t('common.save')}
                                    </Button>
                                    <Button
                                      type="button"
                                      size="sm"
                                      variant="ghost"
                                      disabled={isBusy}
                                      onClick={cancelEditMessage}
                                    >
                                      {t('common.cancel')}
                                    </Button>
                                    <span className="text-xs text-muted-foreground">
                                      {t('chat.editHint')}
                                    </span>
                                  </div>
                                </form>
                              ) : (
                                <>
                                  {displayBody ? (
                                    <MessageMarkdown
                                      body={displayBody}
                                      usersByHandle={membersByHandle}
                                      serverUrl={session?.serverUrl}
                                      token={session?.token}
                                    />
                                  ) : null}
                                  <MessageLinkPreviews
                                    body={message.body}
                                    previews={message.linkPreviews}
                                  />
                                  <MessageReactions
                                    reactions={message.reactions}
                                    disabled={isBusy}
                                    onToggle={(emoji) => {
                                      void toggleReaction(message.id, emoji)
                                    }}
                                  />
                                </>
                              )}
                            </div>
                          </div>
                        )
                        return (
                          <div key={message.id} className="flex flex-col gap-4">
                            {showDayDivider && dayLabel ? (
                              <div
                                className="flex items-center gap-3 py-1"
                                role="separator"
                                aria-label={dayLabel}
                              >
                                <div className="h-px flex-1 bg-border" />
                                <span className="shrink-0 text-xs font-medium text-muted-foreground">
                                  {dayLabel}
                                </span>
                                <div className="h-px flex-1 bg-border" />
                              </div>
                            ) : null}
                            {showUnreadDivider ? (
                              <div
                                className="flex items-center gap-3 py-1"
                                role="separator"
                                aria-label={t('chat.newMessages')}
                              >
                                <div className="h-px flex-1 bg-primary/50" />
                                <span className="shrink-0 text-xs font-semibold text-primary">
                                  {t('chat.new')}
                                </span>
                                <div className="h-px flex-1 bg-primary/50" />
                              </div>
                            ) : null}
                            {isSystem ? (
                              <div
                                data-message-id={message.id}
                                className={cn(
                                  'flex items-center justify-center gap-2 px-4 py-1 text-center',
                                  isHighlighted && 'message-highlight -mx-1 px-5',
                                  isSelected && 'message-selected -mx-1 px-5',
                                )}
                              >
                                <p className="text-xs text-muted-foreground">
                                  {message.body}
                                </p>
                                {timeLabel ? (
                                  <time
                                    dateTime={message.createdAt}
                                    title={formatMessageTimestampTitle(
                                      message.createdAt,
                                    )}
                                    className="shrink-0 text-xs text-muted-foreground/80"
                                  >
                                    {timeLabel}
                                  </time>
                                ) : null}
                              </div>
                            ) : (
                              <ContextMenu>
                                <ContextMenuTrigger asChild>
                                  {messageRow}
                                </ContextMenuTrigger>
                                <ContextMenuContent className="min-w-52">
                                  {!isOwn && !isUnread ? (
                                    <>
                                      <ContextMenuItem
                                        disabled={isBusy}
                                        onSelect={() => {
                                          void markMessageUnread(message.id)
                                        }}
                                      >
                                        <EnvelopeSimpleIcon strokeWidth={2} />
                                        {t('chat.markUnread')}
                                        <ContextMenuShortcut>U</ContextMenuShortcut>
                                      </ContextMenuItem>
                                      <ContextMenuSeparator />
                                    </>
                                  ) : null}
                                  <ContextMenuItem
                                    disabled={isBusy}
                                    onSelect={() => {
                                      void navigator.clipboard
                                        .writeText(message.body)
                                        .then(() => toast.success(t('chat.messageCopied')))
                                        .catch(() =>
                                          toast.error(t('chat.copyMessageError')),
                                        )
                                    }}
                                  >
                                    <CopyIcon strokeWidth={2} />
                                    {t('chat.copyMessage')}
                                    <ContextMenuShortcut>⌘C</ContextMenuShortcut>
                                  </ContextMenuItem>
                                  {isOwn ? (
                                    <>
                                      <ContextMenuSeparator />
                                      <ContextMenuItem
                                        disabled={isBusy}
                                        onSelect={() => beginEditMessage(message)}
                                      >
                                        <PencilSimpleIcon strokeWidth={2} />
                                        {t('chat.editMessage')}
                                        <ContextMenuShortcut>E</ContextMenuShortcut>
                                      </ContextMenuItem>
                                      <ContextMenuItem
                                        variant="destructive"
                                        disabled={isBusy}
                                        onSelect={() => {
                                          setSelectedMessageId(message.id)
                                          setDeleteMessageId(message.id)
                                        }}
                                      >
                                        <TrashIcon strokeWidth={2} />
                                        {t('chat.deleteMessageEllipsis')}
                                        <ContextMenuShortcut>⌫</ContextMenuShortcut>
                                      </ContextMenuItem>
                                    </>
                                  ) : null}
                                </ContextMenuContent>
                              </ContextMenu>
                            )}
                          </div>
                        )
                      })}
                      </>
                    )}
                    <div ref={messagesEndRef} aria-hidden className="h-px w-full shrink-0" />
                  </div>
                </ScrollArea>

                <div className="border-t border-border p-3">
                  <div className="flex flex-col gap-2">
                    {(() => {
                      if (!conversation) return null
                      const externals = workspaceMembers.filter((member) => {
                        if (!member.isExternal) return false
                        if (conversation.isDm) {
                          return member.userId === conversation.peerUserId
                        }
                        // Same-server guests: show when any external can see this channel.
                        return !conversation.isPrivate
                      })
                      if (externals.length === 0) return null
                      const origins = [
                        ...new Set(
                          externals
                            .map((item) => item.homeWorkspaceName?.trim())
                            .filter(Boolean) as string[],
                        ),
                      ]
                      const originLabel =
                        origins.length === 1
                          ? origins[0]
                          : origins.length > 1
                            ? t('chat.otherWorkspaces')
                            : t('chat.anotherWorkspace')
                      const subject =
                        conversation.isDm || externals.length === 1
                          ? externals[0].displayName
                          : t('chat.externalPeople', { count: externals.length })
                      const verb =
                        conversation.isDm || externals.length === 1
                          ? t('chat.isFrom')
                          : t('chat.areFrom')
                      return (
                        <div className="flex items-center gap-2 rounded-md bg-amber-950/80 px-3 py-2 text-sm text-amber-50">
                          <span className="flex size-6 shrink-0 items-center justify-center rounded-md bg-amber-100/15 text-[10px] font-semibold tracking-wide uppercase">
                            {(originLabel === t('chat.otherWorkspaces') ||
                            originLabel === t('chat.anotherWorkspace')
                              ? 'EX'
                              : originLabel
                            )
                              .slice(0, 2)
                              .toUpperCase()}
                          </span>
                          <p className="min-w-0 truncate">
                            <span className="font-medium text-sky-300">{subject}</span>
                            {verb}
                            <span className="font-semibold">{originLabel}</span>
                          </p>
                        </div>
                      )
                    })()}
                    <ComposerMarkdownToolbar
                      disabled={sending || !conversation || offline}
                      value={draft}
                      getSelection={() => {
                        const input = composerInputRef.current
                        return {
                          start: input?.selectionStart ?? draft.length,
                          end: input?.selectionEnd ?? draft.length,
                        }
                      }}
                      onApply={applyComposerEdit}
                    />
                    <form
                      className="flex items-end gap-2"
                      onSubmit={(event) => {
                        event.preventDefault()
                        void sendDraft()
                      }}
                    >
                    <InputGroup className="h-auto min-h-8 flex-1 items-end">
                      <InputGroupTextarea
                        ref={composerInputRef}
                        value={draft}
                        onChange={(event) => setDraft(event.target.value)}
                        onFocus={() => {
                          if (selectedMessageId && !editingMessageId) {
                            clearMessageSelection()
                          }
                        }}
                        onKeyDown={(event) => {
                          if (event.nativeEvent.isComposing) return
                          if (event.key === 'ArrowUp') {
                            const caret = event.currentTarget.selectionStart ?? 0
                            const end = event.currentTarget.selectionEnd ?? 0
                            if (caret === 0 && end === 0) {
                              event.preventDefault()
                              moveMessageSelection(-1)
                            }
                            return
                          }
                          if (event.key !== 'Enter') return
                          if (event.shiftKey) return
                          event.preventDefault()
                          if (
                            !draft.trim() ||
                            sending ||
                            !conversation ||
                            offline
                          ) {
                            return
                          }
                          void sendDraft()
                        }}
                        placeholder={
                          offline
                            ? t('chat.waitingForServer')
                            : composerPlaceholder
                        }
                        aria-label={t('chat.composer')}
                        rows={1}
                        disabled={sending || !conversation || offline}
                        className="max-h-40 min-h-8 overflow-y-auto py-1.5"
                      />
                      <InputGroupAddon align="inline-end" className="self-end pb-1">
                        <EmojiPickerButton
                          disabled={sending || !conversation || offline}
                          onSelect={(shortcode) => {
                            const input = composerInputRef.current
                            const start = input?.selectionStart ?? draft.length
                            const needsLeadingSpace =
                              start > 0 && !/\s/.test(draft[start - 1] ?? '')
                            const needsTrailingSpace =
                              start >= draft.length ||
                              !/\s/.test(draft[start] ?? '')
                            insertComposerText(
                              `${needsLeadingSpace ? ' ' : ''}${shortcode}${needsTrailingSpace ? ' ' : ''}`,
                            )
                          }}
                        />
                      </InputGroupAddon>
                    </InputGroup>
                    <ButtonGroup>
                      <Button
                        type="submit"
                        disabled={
                          !draft.trim() || sending || !conversation || offline
                        }
                        aria-label={t('chat.sendMessage')}
                      >
                        <PaperPlaneRightIcon strokeWidth={2} data-icon="inline-start" />
                        {t('chat.send')}
                      </Button>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            type="button"
                            disabled={
                              !draft.trim() || sending || !conversation || offline
                            }
                            className="pl-2!"
                            aria-label={t('chat.scheduleOptions')}
                          >
                            <CaretDownIcon strokeWidth={2} />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end" className="min-w-56">
                          <DropdownMenuLabel>{t('chat.scheduleSend')}</DropdownMenuLabel>
                          <DropdownMenuSeparator />
                          <DropdownMenuGroup>
                            <DropdownMenuItem
                              onSelect={() => {
                                void scheduleDraft(nextLaterToday())
                              }}
                            >
                              <ClockIcon strokeWidth={2} />
                              {t('chat.laterToday')}
                              <DropdownMenuShortcut>{t('chat.threePm')}</DropdownMenuShortcut>
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onSelect={() => {
                                void scheduleDraft(nextTomorrowMorning())
                              }}
                            >
                              <CalendarBlankIcon strokeWidth={2} />
                              {t('chat.tomorrowMorning')}
                              <DropdownMenuShortcut>{t('chat.nineAm')}</DropdownMenuShortcut>
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onSelect={() => {
                                void scheduleDraft(nextMondayMorning())
                              }}
                            >
                              <CalendarBlankIcon strokeWidth={2} />
                              {t('chat.mondayMorning')}
                              <DropdownMenuShortcut>{t('chat.nineAm')}</DropdownMenuShortcut>
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onSelect={() => setCustomScheduleOpen(true)}
                            >
                              <ClockCountdownIcon strokeWidth={2} />
                              {t('chat.customTime')}
                            </DropdownMenuItem>
                          </DropdownMenuGroup>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </ButtonGroup>
                    </form>
                  </div>
                </div>
              </SidebarInset>
            </ResizablePanel>

            {detailsVisible && !conversation?.isDm ? (
              <>
                <ResizableHandle
                  withHandle
                  aria-label={t('chat.resizeDetails')}
                  className="w-1.5 after:w-3"
                />
                <ResizablePanel
                  id="details"
                  defaultSize={320}
                  minSize={280}
                  maxSize={560}
                >
                  <ChannelDetailsSidebar
                    title={title}
                    topic={conversation?.topic}
                    isPrivate={conversation?.isPrivate ?? false}
                    createdBy={conversation?.createdBy}
                    channelId={conversation?.id}
                    currentUserId={session?.userId}
                    serverUrl={session?.serverUrl}
                    token={session?.token}
                    onInvite={() => {
                      if (conversation?.isPrivate) {
                        setEditChannelOpen(true)
                      } else {
                        setInviteOpen(true)
                      }
                    }}
                    onEditChannel={() => setEditChannelOpen(true)}
                  />
                </ResizablePanel>
              </>
            ) : null}
          </ResizablePanelGroup>
        )}
      </div>
    </div>
    </DirectMessageProvider>
    </TrustedDomainsProvider>
  )
}

export function ChatShell() {
  return (
    <SidebarProvider
      style={
        {
          '--sidebar-width': '20rem',
        } as CSSProperties
      }
      className="h-full min-h-0 flex-col overflow-hidden"
    >
      <ChatShellLayout />
    </SidebarProvider>
  )
}
