import {
  AtIcon,
  BellIcon,
  HashIcon,
  SmileyIcon,
  UserPlusIcon,
  UsersThreeIcon,
} from '@phosphor-icons/react'
import { useEffect, useState } from 'react'

import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { formatUserError, isServerUnavailable } from '@/lib/api/client'
import {
  listNotifications,
  markAllNotificationsRead,
  markNotificationRead,
  type ApiNotification,
  type ApiNotificationKind,
} from '@/lib/api/notifications'
import { useAuth, type AuthSession } from '@/lib/auth'
import { useServerConnection } from '@/lib/server-connection'
import { cn } from '@/lib/utils'
import { toast } from 'sonner'

const KIND_ICON: Record<
  ApiNotificationKind,
  typeof AtIcon
> = {
  mention: AtIcon,
  message: BellIcon,
  reaction: SmileyIcon,
  channel_invite: HashIcon,
  workspace_invite: UserPlusIcon,
  workspace_join: UsersThreeIcon,
}

type Filter = 'all' | 'unread' | 'mentions'

export type MergedNotification = ApiNotification & {
  sessionId: string
  serverLabel: string
  serverUrl: string
  token: string
}

function initialsFromName(name: string) {
  return (
    name
      .split(/\s+/)
      .map((part) => part[0])
      .join('')
      .slice(0, 2)
      .toUpperCase() || '?'
  )
}

function formatRelativeTime(iso: string) {
  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) return ''
  const seconds = Math.round((Date.now() - then) / 1000)
  if (seconds < 60) return 'Just now'
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes} minute${minutes === 1 ? '' : 's'} ago`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return `${hours} hour${hours === 1 ? '' : 's'} ago`
  const days = Math.round(hours / 24)
  if (days < 7) return `${days} day${days === 1 ? '' : 's'} ago`
  return new Date(iso).toLocaleDateString()
}

function contextLabel(item: ApiNotification) {
  if (item.isDm) {
    return item.channelName || 'Direct message'
  }
  if (item.channelName?.startsWith('dm_')) {
    return 'Direct message'
  }
  if (item.channelName) return `#${item.channelName}`
  if (item.kind === 'workspace_invite' || item.kind === 'workspace_join') {
    return 'Workspace'
  }
  return ''
}

function mergeNotifications(
  sessions: AuthSession[],
  results: { session: AuthSession; notifications: ApiNotification[] }[],
): MergedNotification[] {
  const merged: MergedNotification[] = []
  for (const entry of results) {
    const label = entry.session.serverLabel || entry.session.serverUrl
    for (const item of entry.notifications) {
      merged.push({
        ...item,
        sessionId: entry.session.id,
        serverLabel: label,
        serverUrl: entry.session.serverUrl,
        token: entry.session.token,
      })
    }
  }
  merged.sort(
    (a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt),
  )
  return merged
}

export function NotificationsPage({
  onOpenNotification,
  onUnreadCountChange,
}: {
  onOpenNotification?: (item: MergedNotification) => void
  onUnreadCountChange?: (count: number) => void
}) {
  const { sessions } = useAuth()
  const { noteSuccess, noteFailure } = useServerConnection()
  const [filter, setFilter] = useState<Filter>('all')
  const [items, setItems] = useState<MergedNotification[]>([])
  const [unreadCount, setUnreadCount] = useState(0)
  const [loading, setLoading] = useState(true)
  const showServerLabels = sessions.length > 1
  const sessionsKey = sessions.map((item) => item.id).join('|')

  useEffect(() => {
    if (sessions.length === 0) {
      setItems([])
      setUnreadCount(0)
      setLoading(false)
      return
    }
    let cancelled = false
    const load = async (showLoading: boolean) => {
      if (showLoading) setLoading(true)
      try {
        const results = await Promise.all(
          sessions.map(async (session) => {
            try {
              const result = await listNotifications(
                session.serverUrl,
                session.token,
                { filter, limit: 50 },
              )
              return {
                session,
                notifications: result.notifications,
                unreadCount: result.unreadCount,
                error: null as unknown,
              }
            } catch (error) {
              return {
                session,
                notifications: [] as ApiNotification[],
                unreadCount: 0,
                error,
              }
            }
          }),
        )
        if (cancelled) return
        const success = results.filter((entry) => !entry.error)
        if (success.length > 0) noteSuccess()
        const firstError = results.find((entry) => entry.error)?.error
        if (firstError && success.length === 0) {
          noteFailure(firstError)
          if (showLoading) {
            const message =
              formatUserError(firstError, 'Could not load notifications')
            toast.error(message)
            if (!isServerUnavailable(firstError)) {
              setItems([])
            }
          }
          return
        }
        const nextItems = mergeNotifications(
          sessions,
          success.map((entry) => ({
            session: entry.session,
            notifications: entry.notifications,
          })),
        )
        const nextUnread = success.reduce(
          (sum, entry) => sum + entry.unreadCount,
          0,
        )
        setItems(nextItems)
        setUnreadCount(nextUnread)
        onUnreadCountChange?.(nextUnread)
      } finally {
        if (!cancelled && showLoading) setLoading(false)
      }
    }
    void load(true)
    const timer = window.setInterval(() => {
      void load(false)
    }, 5000)
    return () => {
      cancelled = true
      window.clearInterval(timer)
    }
  }, [sessions, sessionsKey, filter, onUnreadCountChange, noteFailure, noteSuccess])

  if (sessions.length === 0) {
    return null
  }

  return (
    <div className="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-background">
      <header className="flex h-14 shrink-0 items-center gap-3 border-b px-4">
        <BellIcon strokeWidth={2} className="size-4 text-muted-foreground" />
        <div className="flex min-w-0 flex-1 flex-col gap-0.5">
          <div className="flex items-center gap-2">
            <h1 className="truncate text-sm font-semibold">Notifications</h1>
            {unreadCount > 0 && (
              <Badge variant="secondary" className="h-5 px-1.5 text-[10px]">
                {unreadCount} unread
              </Badge>
            )}
          </div>
          <p className="truncate text-xs text-muted-foreground">
            {showServerLabels
              ? 'Mentions, invites, and activity across your accounts'
              : 'Mentions, invites, and workspace activity'}
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          disabled={unreadCount === 0}
          onClick={() => {
            void (async () => {
              try {
                await Promise.all(
                  sessions.map((session) =>
                    markAllNotificationsRead(session.serverUrl, session.token),
                  ),
                )
                setItems((current) =>
                  current.map((item) => ({ ...item, unread: false })),
                )
                setUnreadCount(0)
                onUnreadCountChange?.(0)
              } catch (err) {
                const message =
                  formatUserError(err, 'Could not mark notifications as read')
                toast.error(message)
              }
            })()
          }}
        >
          Mark all as read
        </Button>
      </header>

      <div className="flex shrink-0 items-center border-b px-4 py-2">
        <Tabs
          value={filter}
          onValueChange={(value) => setFilter(value as Filter)}
        >
          <TabsList variant="line">
            <TabsTrigger value="all">All</TabsTrigger>
            <TabsTrigger value="unread">Unread</TabsTrigger>
            <TabsTrigger value="mentions">Mentions</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      <ScrollArea className="min-h-0 flex-1">
        {loading && items.length === 0 ? (
          <p className="px-4 py-8 text-sm text-muted-foreground">
            Loading notifications…
          </p>
        ) : items.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-2 px-6 py-20 text-center">
            <BellIcon
              strokeWidth={2}
              className="size-8 text-muted-foreground/50"
            />
            <p className="text-sm font-medium">You are all caught up</p>
            <p className="max-w-xs text-sm text-muted-foreground">
              Mentions, channel invites, and workspace joins will show up here.
            </p>
          </div>
        ) : (
          <ul className="flex flex-col">
            {items.map((item) => {
              const Icon = KIND_ICON[item.kind] ?? BellIcon
              const actor = item.actorName || 'Someone'
              return (
                <li key={`${item.sessionId}:${item.id}`}>
                  <button
                    type="button"
                    className={cn(
                      'flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-muted/50',
                      item.unread && 'bg-muted/30',
                    )}
                    onClick={() => {
                      void (async () => {
                        if (item.unread) {
                          try {
                            await markNotificationRead(
                              item.serverUrl,
                              item.token,
                              item.id,
                            )
                            setItems((current) =>
                              current.map((entry) =>
                                entry.sessionId === item.sessionId &&
                                entry.id === item.id
                                  ? { ...entry, unread: false }
                                  : entry,
                              ),
                            )
                            setUnreadCount((count) => Math.max(0, count - 1))
                            onUnreadCountChange?.(Math.max(0, unreadCount - 1))
                          } catch {
                            // still allow navigation
                          }
                        }
                        onOpenNotification?.(item)
                      })()
                    }}
                  >
                    <span
                      className={cn(
                        'mt-2 size-2 shrink-0 rounded-full',
                        item.unread ? 'bg-primary' : 'bg-transparent',
                      )}
                      aria-hidden
                    />
                    <Avatar size="sm">
                      <AvatarFallback>{initialsFromName(actor)}</AvatarFallback>
                    </Avatar>
                    <div className="flex min-w-0 flex-1 flex-col gap-1">
                      <div className="flex items-center gap-2">
                        <Icon className="size-3.5 shrink-0 text-muted-foreground" />
                        <span className="truncate text-sm font-medium">
                          {actor}
                        </span>
                        <span className="truncate text-xs text-muted-foreground">
                          {contextLabel(item)}
                        </span>
                      </div>
                      <p className="text-sm text-muted-foreground">{item.body}</p>
                      <p className="text-xs text-muted-foreground">
                        {formatRelativeTime(item.createdAt)}
                        {showServerLabels ? ` · ${item.serverLabel}` : ''}
                      </p>
                    </div>
                  </button>
                </li>
              )
            })}
          </ul>
        )}
      </ScrollArea>
    </div>
  )
}
