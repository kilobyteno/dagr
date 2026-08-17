import {
  FileTextIcon,
  GearIcon,
  HashIcon,
  LinkSimpleIcon,
  LockKeyIcon,
  PencilSimpleIcon,
  UserPlusIcon,
  UsersIcon,
} from '@phosphor-icons/react'
import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type ComponentType,
  type SVGProps,
} from 'react'
import { toast } from 'sonner'

import { UserAvatarMark } from '@/components/chat/user-avatar'
import {
  UserHandle,
  type ChatUserRef,
} from '@/components/chat/user-handle'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  getChannelNotificationSettings,
  listChannelMembers,
  updateChannelNotificationSettings,
  type ApiWorkspaceMember,
  type ChannelNotificationLevel,
} from '@/lib/api/channels'
import { formatUserError } from '@/lib/api/client'
import { cn } from '@/lib/utils'

const DETAILS_TAB_LABEL_MIN_WIDTH = 360

const DETAILS_TABS: {
  value: string
  label: string
  icon: ComponentType<SVGProps<SVGSVGElement>>
}[] = [
  { value: 'members', label: 'Members', icon: UsersIcon },
  { value: 'files', label: 'Files', icon: FileTextIcon },
  { value: 'links', label: 'Links', icon: LinkSimpleIcon },
  { value: 'settings', label: 'Settings', icon: GearIcon },
]

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
      <div className="min-w-0 flex-1">
        <label htmlFor={id} className="text-sm font-medium text-foreground">
          {label}
        </label>
        <p className="text-xs text-muted-foreground">{description}</p>
      </div>
      <Switch id={id} checked={checked} onCheckedChange={onCheckedChange} />
    </div>
  )
}

function ConversationSettings({
  title,
  channelId,
  serverUrl,
  token,
  onEditChannel,
}: {
  title: string
  channelId?: string
  serverUrl?: string
  token?: string
  onEditChannel?: () => void
}) {
  const [notifyLevel, setNotifyLevel] =
    useState<ChannelNotificationLevel>('mentions')
  const [notifyLoading, setNotifyLoading] = useState(Boolean(channelId))
  const [notifySaving, setNotifySaving] = useState(false)

  useEffect(() => {
    if (!channelId || !serverUrl || !token) {
      setNotifyLoading(false)
      return
    }
    const controller = new AbortController()
    setNotifyLoading(true)
    void getChannelNotificationSettings(serverUrl, token, channelId)
      .then((result) => {
        if (!controller.signal.aborted) setNotifyLevel(result.level)
      })
      .catch((err: unknown) => {
        if (controller.signal.aborted) return
        const message =
          formatUserError(err, 'Could not load notification settings')
        toast.error(message)
      })
      .finally(() => {
        if (!controller.signal.aborted) setNotifyLoading(false)
      })
    return () => controller.abort()
  }, [channelId, serverUrl, token])

  const saveNotifyLevel = async (level: ChannelNotificationLevel) => {
    if (!channelId || !serverUrl || !token || level === notifyLevel) return
    const previous = notifyLevel
    setNotifyLevel(level)
    setNotifySaving(true)
    try {
      const result = await updateChannelNotificationSettings(
        serverUrl,
        token,
        channelId,
        level,
      )
      setNotifyLevel(result.level)
    } catch (err) {
      setNotifyLevel(previous)
      const message =
        formatUserError(err, 'Could not update notification settings')
      toast.error(message)
    } finally {
      setNotifySaving(false)
    }
  }

  return (
    <div className="flex flex-col gap-5 p-3">
      <section className="flex flex-col gap-1">
        <p className="px-1 text-xs font-medium tracking-wide text-muted-foreground uppercase">
          Notifications
        </p>
        {notifyLoading ? (
          <p className="px-1 py-2 text-sm text-muted-foreground">
            Loading notification settings…
          </p>
        ) : (
          <>
            <p className="px-1 pb-1 text-sm text-muted-foreground">
              Choose what notifies you in #{title}. Limited by your account
              preference in Edit profile. Changes save immediately.
            </p>
            <SettingRow
              id="notify-all"
              label="All messages"
              description="Notify me about every new message."
              checked={notifyLevel === 'all'}
              onCheckedChange={(checked) => {
                if (checked) void saveNotifyLevel('all')
              }}
            />
            <SettingRow
              id="notify-mentions"
              label="Mentions only"
              description="Only notify me when I am mentioned."
              checked={notifyLevel === 'mentions'}
              onCheckedChange={(checked) => {
                if (checked) void saveNotifyLevel('mentions')
              }}
            />
            <SettingRow
              id="mute-conversation"
              label="Nothing"
              description="Silence message notifications from this channel."
              checked={notifyLevel === 'nothing'}
              onCheckedChange={(checked) => {
                if (checked) void saveNotifyLevel('nothing')
              }}
            />
            {notifySaving ? (
              <p className="px-1 pt-1 text-xs text-muted-foreground">Saving…</p>
            ) : null}
          </>
        )}
      </section>

      {onEditChannel ? (
        <>
          <Separator />
          <section className="flex flex-col gap-2">
            <Button
              type="button"
              variant="outline"
              className="w-full"
              onClick={onEditChannel}
            >
              <PencilSimpleIcon strokeWidth={2} data-icon="inline-start" />
              Channel settings
            </Button>
          </section>
        </>
      ) : null}
    </div>
  )
}

function memberToUserRef(member: ApiWorkspaceMember): ChatUserRef {
  return {
    userId: member.userId,
    displayName: member.displayName,
    handle: member.handle,
    statusEmoji: member.statusEmoji ?? '',
    statusText: member.statusText ?? '',
    statusExpiresAt: member.statusExpiresAt ?? null,
    presence: member.presence ?? 'offline',
    hasAvatar: Boolean(member.hasAvatar),
    avatarUpdatedAt: member.avatarUpdatedAt ?? null,
  }
}

export function ChannelDetailsSidebar({
  title,
  topic,
  isPrivate,
  createdBy,
  channelId,
  currentUserId,
  serverUrl,
  token,
  onInvite,
  onEditChannel,
}: {
  title: string
  topic?: string
  isPrivate: boolean
  createdBy?: string
  channelId?: string
  currentUserId?: string
  serverUrl?: string
  token?: string
  onInvite?: () => void
  onEditChannel?: () => void
}) {
  const asideRef = useRef<HTMLElement>(null)
  const [compactTabs, setCompactTabs] = useState(true)
  const [members, setMembers] = useState<ApiWorkspaceMember[]>([])
  const [membersLoading, setMembersLoading] = useState(Boolean(channelId))

  useEffect(() => {
    const node = asideRef.current
    if (!node) return

    const update = (width: number) => {
      setCompactTabs(width < DETAILS_TAB_LABEL_MIN_WIDTH)
    }

    update(node.getBoundingClientRect().width)

    const observer = new ResizeObserver((entries) => {
      const width = entries[0]?.contentRect.width
      if (typeof width === 'number') update(width)
    })

    observer.observe(node)
    return () => observer.disconnect()
  }, [])

  useEffect(() => {
    if (!channelId || !serverUrl || !token) {
      setMembers([])
      setMembersLoading(false)
      return
    }
    const controller = new AbortController()
    setMembersLoading(true)
    void listChannelMembers(serverUrl, token, channelId, controller.signal)
      .then((result) => {
        if (controller.signal.aborted) return
        setMembers(result.members)
      })
      .catch((err: unknown) => {
        if (controller.signal.aborted) return
        setMembers([])
        const message =
          formatUserError(err, 'Could not load channel members')
        toast.error(message)
      })
      .finally(() => {
        if (!controller.signal.aborted) setMembersLoading(false)
      })
    return () => controller.abort()
  }, [channelId, serverUrl, token])

  const creator = useMemo(() => {
    if (!createdBy) return null
    return members.find((member) => member.userId === createdBy) ?? null
  }, [createdBy, members])

  const sortedMembers = useMemo(() => {
    return [...members].sort((a, b) => {
      if (a.userId === currentUserId) return -1
      if (b.userId === currentUserId) return 1
      return a.displayName.localeCompare(b.displayName, 'en-GB')
    })
  }, [members, currentUserId])

  return (
    <aside
      ref={asideRef}
      className="flex h-full min-h-0 w-full flex-col bg-sidebar text-sidebar-foreground"
    >
      <div className="flex h-14 shrink-0 items-center gap-2 border-b border-border px-3">
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="truncate text-sm font-semibold">Details</span>
          <span className="truncate text-xs text-muted-foreground">
            #{title}
          </span>
        </div>
      </div>

      <div className="border-b border-border px-3 py-3">
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-2">
            <span className="flex size-8 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
              {isPrivate ? (
                <LockKeyIcon strokeWidth={2} />
              ) : (
                <HashIcon strokeWidth={2} />
              )}
            </span>
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-semibold">#{title}</p>
              <p className="truncate text-xs text-muted-foreground">
                {isPrivate ? 'Private channel' : 'Public channel'}
              </p>
            </div>
          </div>
          {topic ? (
            <p className="text-sm text-muted-foreground">{topic}</p>
          ) : (
            <p className="text-sm text-muted-foreground italic">No topic set</p>
          )}
          {creator ? (
            <p className="text-xs text-muted-foreground">
              Created by{' '}
              <span className="font-medium text-foreground">
                {creator.displayName}
              </span>
            </p>
          ) : null}
        </div>
      </div>

      <Tabs defaultValue="members" className="flex min-h-0 flex-1 flex-col gap-0">
        <div className="border-b border-border px-2 py-2">
          <TabsList variant="line" className="w-full">
            {DETAILS_TABS.map((tab) => {
              const Icon = tab.icon
              return (
                <TabsTrigger
                  key={tab.value}
                  value={tab.value}
                  className="flex-1 px-2"
                  title={tab.label}
                  aria-label={tab.label}
                >
                  <Icon />
                  <span className={cn(compactTabs && 'sr-only')}>{tab.label}</span>
                </TabsTrigger>
              )
            })}
          </TabsList>
        </div>

        <ScrollArea className="min-h-0 flex-1">
          <TabsContent value="members" className="mt-0 p-3">
            <div className="mb-3 flex items-center justify-between gap-2">
              <p className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                {membersLoading
                  ? 'Members'
                  : `${sortedMembers.length} ${
                      sortedMembers.length === 1 ? 'member' : 'members'
                    }`}
              </p>
              {onInvite ? (
                <Button variant="outline" size="sm" onClick={onInvite}>
                  <UserPlusIcon strokeWidth={2} data-icon="inline-start" />
                  Invite
                </Button>
              ) : null}
            </div>
            {membersLoading ? (
              <p className="px-1 py-2 text-sm text-muted-foreground">
                Loading members…
              </p>
            ) : sortedMembers.length === 0 ? (
              <p className="px-1 py-2 text-sm text-muted-foreground">
                No members to show yet.
              </p>
            ) : (
              <div className="flex flex-col gap-1">
                {sortedMembers.map((member) => {
                  const user = memberToUserRef(member)
                  const isYou = member.userId === currentUserId
                  return (
                    <div
                      key={member.userId}
                      className="flex w-full items-center gap-3 rounded-md px-2 py-1.5"
                    >
                      <UserHandle
                        user={user}
                        serverUrl={serverUrl}
                        token={token}
                        className="flex min-w-0 flex-1 items-center gap-3 font-normal hover:no-underline"
                      >
                        <UserAvatarMark
                          userId={user.userId}
                          name={user.displayName}
                          hasAvatar={user.hasAvatar}
                          avatarUpdatedAt={user.avatarUpdatedAt}
                          presence={user.presence}
                          showPresence
                          serverUrl={serverUrl}
                          token={token}
                          className="size-8 rounded-md after:rounded-md"
                          imageClassName="rounded-md"
                          fallbackClassName="rounded-md text-xs"
                        />
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2">
                            <span className="truncate text-sm font-medium">
                              {member.displayName}
                            </span>
                            {isYou ? (
                              <Badge
                                variant="secondary"
                                className="h-5 px-1.5 text-[10px]"
                              >
                                You
                              </Badge>
                            ) : null}
                          </div>
                          <p className="truncate text-xs text-muted-foreground">
                            @{member.handle}
                            {member.role === 'owner' || member.role === 'admin'
                              ? ` · ${member.role}`
                              : ''}
                          </p>
                        </div>
                      </UserHandle>
                    </div>
                  )
                })}
              </div>
            )}
          </TabsContent>

          <TabsContent value="files" className="mt-0 p-3">
            <div className="flex flex-col items-start gap-2 px-1 py-6">
              <span className="flex size-10 items-center justify-center rounded-md bg-muted text-muted-foreground">
                <FileTextIcon strokeWidth={2} />
              </span>
              <div className="flex flex-col gap-1">
                <p className="text-sm font-medium">Files coming soon</p>
                <p className="text-sm text-muted-foreground">
                  Shared files in this channel will appear here.
                </p>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="links" className="mt-0 p-3">
            <div className="flex flex-col items-start gap-2 px-1 py-6">
              <span className="flex size-10 items-center justify-center rounded-md bg-muted text-muted-foreground">
                <LinkSimpleIcon strokeWidth={2} />
              </span>
              <div className="flex flex-col gap-1">
                <p className="text-sm font-medium">Links coming soon</p>
                <p className="text-sm text-muted-foreground">
                  Links shared in this channel will appear here.
                </p>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="settings" className="mt-0">
            <ConversationSettings
              title={title}
              channelId={channelId}
              serverUrl={serverUrl}
              token={token}
              onEditChannel={onEditChannel}
            />
          </TabsContent>
        </ScrollArea>
      </Tabs>
    </aside>
  )
}
