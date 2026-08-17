import { GifAwareImage } from '@/components/chat/gif-aware-image'
import { SafeExternalLink } from '@/components/chat/trusted-domains'
import type { ApiLinkPreview } from '@/lib/api/messages'
import { isLikelyGifUrl } from '@/lib/app-preferences'
import { cn } from '@/lib/utils'

const MESSAGE_URL_PATTERN = /https?:\/\/[^\s<>"'`]+/gi

function hostFromURL(raw: string) {
  try {
    return new URL(raw).hostname.replace(/^www\./, '')
  } catch {
    return raw
  }
}

function trimTrailingURLPunctuation(raw: string) {
  return raw.replace(/[.,;:!?)]}>"']+$/g, '')
}

function normalizePreviewURL(raw: string) {
  try {
    const url = new URL(raw.trim())
    url.hash = ''
    url.hostname = url.hostname.toLowerCase()
    if (
      (url.protocol === 'http:' && url.port === '80') ||
      (url.protocol === 'https:' && url.port === '443')
    ) {
      url.port = ''
    }
    if (!url.pathname) url.pathname = '/'
    return url.toString()
  } catch {
    return raw.trim()
  }
}

function extractMessageURLs(body: string): string[] {
  const matches = body.match(MESSAGE_URL_PATTERN) ?? []
  const seen = new Set<string>()
  const out: string[] = []
  for (const match of matches) {
    const cleaned = trimTrailingURLPunctuation(match)
    const key = normalizePreviewURL(cleaned)
    if (!cleaned || seen.has(key)) continue
    seen.add(key)
    out.push(cleaned)
  }
  return out
}

export function isGifLinkPreview(preview: ApiLinkPreview): boolean {
  if (preview.status !== 'ready' && preview.status !== 'pending') {
    return false
  }
  const mediaSrc = preview.imageUrl || preview.url
  return (
    Boolean(mediaSrc) &&
    (isLikelyGifUrl(preview.url) || isLikelyGifUrl(mediaSrc))
  )
}

function shouldHideURLInBody(raw: string, previews: ApiLinkPreview[]): boolean {
  if (isLikelyGifUrl(raw)) return true
  return previews.some(
    (preview) =>
      isGifLinkPreview(preview) &&
      normalizePreviewURL(preview.url) === normalizePreviewURL(raw),
  )
}

/** Drop GIF URLs from message text; they render as embeds instead. */
export function messageBodyWithoutGifLinks(
  body: string,
  previews?: ApiLinkPreview[],
): string {
  const list = previews ?? []
  const next = body.replace(MESSAGE_URL_PATTERN, (match) => {
    const cleaned = trimTrailingURLPunctuation(match)
    const trailing = match.slice(cleaned.length)
    if (!shouldHideURLInBody(cleaned, list)) return match
    return trailing
  })

  return next
    .replace(/[^\S\n]+/g, ' ')
    .replace(/ ?\n ?/g, '\n')
    .replace(/\n{3,}/g, '\n\n')
    .trim()
}

function GifEmbed({
  href,
  src,
  label,
  className,
}: {
  href: string
  src: string
  label: string
  className?: string
}) {
  return (
    <SafeExternalLink
      href={href}
      className={cn(
        'block max-w-sm overflow-hidden rounded-lg bg-muted/40 text-left ring-1 ring-border transition-opacity hover:opacity-95',
        className,
      )}
      aria-label={label}
    >
      <GifAwareImage
        src={src}
        alt={label}
        isGif
        className="max-h-80 w-full object-contain"
        loading="lazy"
        referrerPolicy="no-referrer"
        onError={(event) => {
          event.currentTarget.hidden = true
        }}
      />
    </SafeExternalLink>
  )
}

export function LinkPreviewCard({
  preview,
  className,
}: {
  preview: ApiLinkPreview
  className?: string
}) {
  if (preview.status === 'pending') {
    const pendingGif = isGifLinkPreview(preview)
    return (
      <div
        className={cn(
          pendingGif
            ? 'max-w-sm overflow-hidden rounded-lg bg-muted/40 ring-1 ring-border'
            : 'max-w-md overflow-hidden rounded-lg border border-border bg-muted/30',
          className,
        )}
        aria-busy="true"
      >
        {pendingGif ? (
          <div className="h-40 w-full animate-pulse bg-muted" />
        ) : (
          <div className="flex gap-3 p-3">
            <div className="size-16 shrink-0 animate-pulse rounded-md bg-muted" />
            <div className="flex min-w-0 flex-1 flex-col justify-center gap-2">
              <div className="h-3 w-2/3 animate-pulse rounded bg-muted" />
              <div className="h-3 w-full animate-pulse rounded bg-muted" />
              <div className="h-3 w-1/3 animate-pulse rounded bg-muted" />
            </div>
          </div>
        )}
      </div>
    )
  }

  if (preview.status !== 'ready') {
    return null
  }

  const title = preview.title || preview.siteName || hostFromURL(preview.url)
  const site = preview.siteName || hostFromURL(preview.url)
  const mediaSrc = preview.imageUrl || preview.url
  const isGifEmbed = isGifLinkPreview(preview)

  if (isGifEmbed) {
    return (
      <GifEmbed
        href={preview.url}
        src={mediaSrc}
        label={title ? `GIF from ${site}: ${title}` : `GIF from ${site}`}
        className={className}
      />
    )
  }

  return (
    <SafeExternalLink
      href={preview.url}
      className={cn(
        'flex max-w-md overflow-hidden rounded-lg border border-border bg-card text-left transition-colors hover:bg-accent/40',
        className,
      )}
    >
      {preview.imageUrl ? (
        <GifAwareImage
          src={preview.imageUrl}
          alt=""
          className="size-24 shrink-0 object-cover sm:size-28"
          loading="lazy"
          referrerPolicy="no-referrer"
          onError={(event) => {
            event.currentTarget.hidden = true
          }}
        />
      ) : (
        <div className="flex size-24 shrink-0 items-center justify-center bg-muted text-xs font-semibold uppercase tracking-wide text-muted-foreground sm:size-28">
          {site.slice(0, 2)}
        </div>
      )}
      <div className="flex min-w-0 flex-1 flex-col justify-center gap-1 p-3">
        <p className="truncate text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {site}
        </p>
        <p className="line-clamp-2 text-sm font-medium text-foreground">{title}</p>
        {preview.description ? (
          <p className="line-clamp-2 text-xs text-muted-foreground">
            {preview.description}
          </p>
        ) : (
          <p className="truncate text-xs text-muted-foreground">{preview.url}</p>
        )}
      </div>
    </SafeExternalLink>
  )
}

export function MessageLinkPreviews({
  body,
  previews,
}: {
  body?: string
  previews?: ApiLinkPreview[]
}) {
  const visible = (previews ?? []).filter(
    (preview) => preview.status === 'ready' || preview.status === 'pending',
  )
  const covered = new Set(
    visible.map((preview) => normalizePreviewURL(preview.url)),
  )
  // Once the server has a GIF preview, prefer it (final URL may differ after redirects).
  const hasServerGif = visible.some(isGifLinkPreview)
  const orphanGifURLs = hasServerGif
    ? []
    : extractMessageURLs(body ?? '').filter((url) => {
        if (!isLikelyGifUrl(url)) return false
        return !covered.has(normalizePreviewURL(url))
      })

  if (!visible.length && !orphanGifURLs.length) return null

  return (
    <div className="flex flex-col gap-2">
      {visible.map((preview) => (
        <LinkPreviewCard key={preview.id} preview={preview} />
      ))}
      {orphanGifURLs.map((url) => (
        <GifEmbed
          key={url}
          href={url}
          src={url}
          label={`GIF from ${hostFromURL(url)}`}
        />
      ))}
    </div>
  )
}
