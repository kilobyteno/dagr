import { MessageMarkdown } from '@/components/chat/message-markdown'
import { SafeExternalLink } from '@/components/chat/trusted-domains'
import type { ChatUserRef } from '@/components/chat/user-handle'
import { useLocale } from '@/lib/i18n'
import { cn } from '@/lib/utils'

export type RichField = {
  name: string
  value: string
  inline?: boolean
}

export type RichEmbedActor = {
  name?: string
  url?: string
  iconUrl?: string
}

export type RichEmbedFooter = {
  text?: string
  iconUrl?: string
}

export type RichEmbed = {
  title?: string
  url?: string
  description?: string
  color?: string
  author?: RichEmbedActor
  fields?: RichField[]
  thumbnailUrl?: string
  imageUrl?: string
  footer?: RichEmbedFooter
  timestamp?: string
}

export type RichPayload = {
  text?: string
  username?: string
  iconUrl?: string
  embeds?: RichEmbed[]
}

function embedColor(color?: string) {
  if (!color) return 'var(--border)'
  return color
}

export function RichMessage({
  payload,
  usersByHandle,
  serverUrl,
  token,
}: {
  payload?: RichPayload | null
  usersByHandle?: Map<string, ChatUserRef>
  serverUrl?: string
  token?: string
}) {
  if (!payload) return null
  const embeds = payload.embeds ?? []
  const handles = usersByHandle ?? new Map<string, ChatUserRef>()
  return (
    <div className="flex min-w-0 flex-col gap-2">
      {payload.text ? (
        <MessageMarkdown
          body={payload.text}
          usersByHandle={handles}
          serverUrl={serverUrl}
          token={token}
        />
      ) : null}
      {embeds.map((embed, index) => (
        <RichEmbedCard
          key={`${embed.title ?? 'embed'}-${index}`}
          embed={embed}
          usersByHandle={handles}
          serverUrl={serverUrl}
          token={token}
        />
      ))}
    </div>
  )
}

function RichEmbedCard({
  embed,
  usersByHandle,
  serverUrl,
  token,
}: {
  embed: RichEmbed
  usersByHandle: Map<string, ChatUserRef>
  serverUrl?: string
  token?: string
}) {
  const { t } = useLocale()
  const inlineFields = (embed.fields ?? []).filter((field) => field.inline)
  const stackedFields = (embed.fields ?? []).filter((field) => !field.inline)
  const stamp = embed.timestamp ? new Date(embed.timestamp) : null

  return (
    <article
      className="flex max-w-lg overflow-hidden rounded-lg border border-border bg-card"
    >
      <div
        aria-hidden
        className="w-1 shrink-0"
        style={{ backgroundColor: embedColor(embed.color) }}
      />
      <div className="flex min-w-0 flex-1 gap-3 p-3">
        <div className="flex min-w-0 flex-1 flex-col gap-2">
          {embed.author?.name ? (
            <div className="flex items-center gap-2">
              {embed.author.iconUrl ? (
                <img
                  src={embed.author.iconUrl}
                  alt=""
                  className="size-4 rounded-sm object-cover"
                  referrerPolicy="no-referrer"
                />
              ) : null}
              {embed.author.url ? (
                <SafeExternalLink
                  href={embed.author.url}
                  className="truncate text-xs font-medium text-foreground hover:underline"
                >
                  {embed.author.name}
                </SafeExternalLink>
              ) : (
                <span className="truncate text-xs font-medium">{embed.author.name}</span>
              )}
            </div>
          ) : null}
          {embed.title ? (
            embed.url ? (
              <SafeExternalLink
                href={embed.url}
                className="text-sm font-semibold text-foreground hover:underline"
              >
                {embed.title}
              </SafeExternalLink>
            ) : (
              <h3 className="text-sm font-semibold">{embed.title}</h3>
            )
          ) : null}
          {embed.description ? (
            <MessageMarkdown
              body={embed.description}
              usersByHandle={usersByHandle}
              serverUrl={serverUrl}
              token={token}
            />
          ) : null}
          {inlineFields.length > 0 ? (
            <dl className="grid grid-cols-2 gap-2">
              {inlineFields.map((field) => (
                <div key={field.name} className="min-w-0">
                  <dt className="text-xs font-medium text-muted-foreground">{field.name}</dt>
                  <dd className="text-sm">
                    <MessageMarkdown
                      body={field.value}
                      usersByHandle={usersByHandle}
                      serverUrl={serverUrl}
                      token={token}
                    />
                  </dd>
                </div>
              ))}
            </dl>
          ) : null}
          {stackedFields.map((field) => (
            <div key={field.name} className="min-w-0">
              <p className="text-xs font-medium text-muted-foreground">{field.name}</p>
              <MessageMarkdown
                body={field.value}
                usersByHandle={usersByHandle}
                serverUrl={serverUrl}
                token={token}
              />
            </div>
          ))}
          {embed.imageUrl ? (
            <img
              src={embed.imageUrl}
              alt=""
              className="max-h-64 w-full rounded-md object-cover"
              referrerPolicy="no-referrer"
            />
          ) : null}
          {embed.footer?.text || stamp ? (
            <footer className="flex items-center gap-2 text-xs text-muted-foreground">
              {embed.footer?.iconUrl ? (
                <img
                  src={embed.footer.iconUrl}
                  alt=""
                  className="size-3.5 rounded-sm object-cover"
                  referrerPolicy="no-referrer"
                />
              ) : null}
              {embed.footer?.text ? <span>{embed.footer.text}</span> : null}
              {stamp && !Number.isNaN(stamp.getTime()) ? (
                <time dateTime={stamp.toISOString()}>
                  {stamp.toLocaleString()}
                </time>
              ) : null}
            </footer>
          ) : null}
        </div>
        {embed.thumbnailUrl ? (
          <img
            src={embed.thumbnailUrl}
            alt={t('apps.thumbnail')}
            className="size-16 shrink-0 rounded-md object-cover"
            referrerPolicy="no-referrer"
          />
        ) : null}
      </div>
    </article>
  )
}
