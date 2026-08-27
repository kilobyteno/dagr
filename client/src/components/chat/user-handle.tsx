import { ChatCircleIcon, HashIcon, LockKeyIcon } from '@phosphor-icons/react'
import {
  createContext,
  useContext,
  type ReactNode,
} from 'react'

import { MessageEmoji } from '@/components/chat/message-emoji'
import { UserAvatarMark } from '@/components/chat/user-avatar'
import { Button } from '@/components/ui/button'
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from '@/components/ui/hover-card'
import {
  isSkinToneShortcode,
  matchEmoticonAt,
  resolveEmoji,
  type CustomEmoji,
} from '@/lib/emoji'
import { formatCustomStatus, hasCustomStatus } from '@/lib/presence'
import { cn } from '@/lib/utils'

export type ChatUserRef = {
  userId: string
  displayName: string
  handle: string
  formerHandles?: string[]
  statusEmoji?: string
  statusText?: string
  statusExpiresAt?: string | null
  presence?: 'online' | 'away' | 'offline'
  hasAvatar?: boolean
  avatarUpdatedAt?: string | null
  isExternal?: boolean
  homeWorkspaceName?: string
  homeWorkspaceIconUrl?: string
  homeServerUrl?: string
}

export type ChatChannelRef = {
  id: string
  name: string
  isPrivate: boolean
  topic?: string
}

type DirectMessageActions = {
  currentUserId?: string
  onMessageUser: (user: ChatUserRef) => void
}

const DirectMessageContext = createContext<DirectMessageActions | null>(null)

export function DirectMessageProvider({
  value,
  children,
}: {
  value: DirectMessageActions
  children: ReactNode
}) {
  return (
    <DirectMessageContext.Provider value={value}>
      {children}
    </DirectMessageContext.Provider>
  )
}

function useDirectMessageActions() {
  return useContext(DirectMessageContext)
}

type ChannelLinkActions = {
  channelsByName: Map<string, ChatChannelRef>
  onOpenChannel: (channelId: string) => void
}

const ChannelLinkContext = createContext<ChannelLinkActions | null>(null)

export function ChannelLinkProvider({
  value,
  children,
}: {
  value: ChannelLinkActions
  children: ReactNode
}) {
  return (
    <ChannelLinkContext.Provider value={value}>
      {children}
    </ChannelLinkContext.Provider>
  )
}

function ChannelHandle({
  channel,
  children,
}: {
  channel: ChatChannelRef
  children: ReactNode
}) {
  const links = useContext(ChannelLinkContext)
  return (
    <button
      type="button"
      className="inline-flex items-baseline gap-0.5 rounded-sm font-medium text-primary underline-offset-2 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      title={channel.isPrivate ? `#${channel.name}` : undefined}
      onClick={(event) => {
        event.stopPropagation()
        links?.onOpenChannel(channel.id)
      }}
    >
      {channel.isPrivate ? (
        <LockKeyIcon
          strokeWidth={2}
          className="inline size-[0.95em] translate-y-px"
          aria-hidden
        />
      ) : (
        <HashIcon
          strokeWidth={2}
          className="inline size-[0.95em] translate-y-px"
          aria-hidden
        />
      )}
      <span>{typeof children === 'string' ? children.replace(/^#/, '') : children}</span>
    </button>
  )
}

export function UserHandle({
  user,
  className,
  children,
  serverUrl,
  token,
}: {
  user: ChatUserRef
  className?: string
  children?: ReactNode
  serverUrl?: string
  token?: string
}) {
  const dm = useDirectMessageActions()
  const label =
    children ??
    (user.handle ? `@${user.handle}` : user.displayName || 'Member')
  const title = user.displayName || user.handle || 'Member'
  const isSelf = Boolean(dm?.currentUserId && user.userId === dm.currentUserId)

  const messageUser = () => {
    if (isSelf || !dm) return
    dm.onMessageUser(user)
  }

  return (
    <HoverCard openDelay={200} closeDelay={100}>
      <HoverCardTrigger asChild>
        <button
          type="button"
          className={cn(
            'inline rounded-sm text-left font-medium text-foreground underline-offset-2 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
            className,
          )}
          onClick={messageUser}
        >
          {label}
        </button>
      </HoverCardTrigger>
      <HoverCardContent align="start" className="w-72 p-3">
        <div className="flex flex-col gap-3">
          <div className="flex items-start gap-3">
            <UserAvatarMark
              userId={user.userId}
              name={title}
              hasAvatar={user.hasAvatar}
              avatarUpdatedAt={user.avatarUpdatedAt}
              presence={user.presence}
              showPresence
              serverUrl={serverUrl}
              token={token}
              size="sm"
            />
            <div className="grid min-w-0 flex-1 gap-0.5">
              <p className="truncate text-sm font-semibold">{title}</p>
              {user.handle ? (
                <p className="truncate text-xs text-muted-foreground">
                  @{user.handle}
                </p>
              ) : null}
              {hasCustomStatus(
                user.statusEmoji,
                user.statusText,
                user.statusExpiresAt,
              ) ? (
                <p className="truncate text-xs text-muted-foreground">
                  {formatCustomStatus(
                    user.statusEmoji,
                    user.statusText,
                    user.statusExpiresAt,
                  )}
                </p>
              ) : null}
            </div>
          </div>
          {!isSelf ? (
            <Button
              type="button"
              size="sm"
              variant="outline"
              className="w-full"
              onClick={messageUser}
            >
              <ChatCircleIcon strokeWidth={2} data-icon="inline-start" />
              Message
            </Button>
          ) : null}
        </div>
      </HoverCardContent>
    </HoverCard>
  )
}

const INLINE_TOKEN =
  /(#[a-z0-9](?:[a-z0-9_-]{0,78}[a-z0-9])?)\b|(@[a-z0-9][a-z0-9_]{1,31})\b|:([a-zA-Z0-9_+-]+):/g

function pushTextWithEmoticons(
  parts: ReactNode[],
  text: string,
  keyPrefix: string,
) {
  let index = 0
  let buffer = ''
  const flush = () => {
    if (!buffer) return
    parts.push(buffer)
    buffer = ''
  }
  while (index < text.length) {
    const hit = matchEmoticonAt(text, index)
    if (hit) {
      flush()
      parts.push(
        <MessageEmoji
          key={`${keyPrefix}-emoticon-${index}`}
          name={hit.id}
        />,
      )
      index += hit.token.length
      continue
    }
    buffer += text[index]
    index += 1
  }
  flush()
}

/**
 * Renders message text with Slack-style :emoji: shortcodes, classic
 * emoticons such as :) and :D, and @handles.
 * Custom workspace emoji can be supplied later via customByName.
 */
export function MessageBodyWithHandles({
  body,
  usersByHandle,
  customByName,
  serverUrl,
  token,
}: {
  body: string
  usersByHandle: Map<string, ChatUserRef>
  customByName?: Map<string, CustomEmoji>
  serverUrl?: string
  token?: string
}) {
  const channelLinks = useContext(ChannelLinkContext)
  const parts: ReactNode[] = []
  let lastIndex = 0
  const re = new RegExp(INLINE_TOKEN.source, 'g')
  let match: RegExpExecArray | null

  while ((match = re.exec(body)) !== null) {
    if (match.index > lastIndex) {
      pushTextWithEmoticons(
        parts,
        body.slice(lastIndex, match.index),
        `${lastIndex}`,
      )
    }

    const channelToken = match[1]
    const mention = match[2]
    const emojiName = match[3]

    if (channelToken) {
      const atBoundary =
        match.index === 0 || /\s/.test(body[match.index - 1] ?? '')
      const name = channelToken.slice(1).toLowerCase()
      const channel = atBoundary
        ? channelLinks?.channelsByName.get(name)
        : undefined
      if (channel) {
        parts.push(
          <ChannelHandle
            key={`${match.index}-${channel.id}-${name}`}
            channel={channel}
          >
            {channelToken}
          </ChannelHandle>,
        )
      } else {
        parts.push(channelToken)
      }
      lastIndex = match.index + channelToken.length
      continue
    }

    if (mention) {
      const handle = mention.slice(1).toLowerCase()
      const user = usersByHandle.get(handle)
      if (user) {
        parts.push(
          <UserHandle
            key={`${match.index}-${user.userId}-${handle}`}
            user={user}
            serverUrl={serverUrl}
            token={token}
            className="font-medium text-primary"
          >
            {mention}
          </UserHandle>,
        )
      } else {
        parts.push(mention)
      }
      lastIndex = match.index + mention.length
      continue
    }

    if (emojiName) {
      const token = match[0]
      const skinAlone = isSkinToneShortcode(emojiName)
      if (skinAlone !== null) {
        // Orphan skin tone with no preceding emoji shortcode.
        parts.push(token)
        lastIndex = match.index + token.length
        continue
      }

      let skinTone = 1
      let consumed = token.length
      const afterEmoji = match.index + token.length
      const next = new RegExp(INLINE_TOKEN.source, 'g')
      next.lastIndex = afterEmoji
      const peek = next.exec(body)
      if (
        peek &&
        peek.index === afterEmoji &&
        peek[2] &&
        isSkinToneShortcode(peek[2]) !== null
      ) {
        skinTone = isSkinToneShortcode(peek[2]) ?? 1
        consumed += peek[0].length
        re.lastIndex = afterEmoji + peek[0].length
      }

      const resolved = resolveEmoji(emojiName, { skinTone, customByName })
      if (resolved) {
        parts.push(
          <MessageEmoji
            key={`${match.index}-${emojiName}-${skinTone}`}
            name={emojiName}
            skinTone={skinTone}
            customByName={customByName}
          />,
        )
      } else {
        parts.push(body.slice(match.index, match.index + consumed))
      }
      lastIndex = match.index + consumed
    }
  }

  if (lastIndex < body.length) {
    pushTextWithEmoticons(parts, body.slice(lastIndex), `${lastIndex}`)
  }
  return <>{parts}</>
}
