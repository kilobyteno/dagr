import { useEffect, useState } from 'react'

import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { fetchUserAvatarObjectUrl, userInitials } from '@/lib/api/auth'
import { presenceDotClass } from '@/lib/presence'
import { cn } from '@/lib/utils'

export function UserAvatarMark({
  userId,
  name,
  hasAvatar,
  avatarUpdatedAt,
  iconUrl,
  presence,
  showPresence = false,
  serverUrl,
  token,
  className,
  fallbackClassName,
  imageClassName,
  size = 'default',
}: {
  userId: string
  name: string
  hasAvatar?: boolean
  avatarUpdatedAt?: string | null
  iconUrl?: string | null
  presence?: 'online' | 'away' | 'offline' | string | null
  showPresence?: boolean
  serverUrl?: string
  token?: string
  className?: string
  fallbackClassName?: string
  imageClassName?: string
  size?: 'default' | 'sm' | 'lg'
}) {
  const [src, setSrc] = useState<string | null>(null)
  const initials = userInitials(name)

  useEffect(() => {
    if (iconUrl) {
      setSrc(iconUrl)
      return
    }
    if (!hasAvatar || !serverUrl || !token || !userId) {
      setSrc(null)
      return
    }
    const controller = new AbortController()
    let objectUrl: string | null = null
    void fetchUserAvatarObjectUrl(
      serverUrl,
      token,
      userId,
      controller.signal,
    ).then((url) => {
      if (controller.signal.aborted) {
        if (url) URL.revokeObjectURL(url)
        return
      }
      objectUrl = url
      setSrc(url)
    })
    return () => {
      controller.abort()
      if (objectUrl) URL.revokeObjectURL(objectUrl)
    }
  }, [hasAvatar, avatarUpdatedAt, iconUrl, serverUrl, token, userId])

  return (
    <span className="relative inline-flex shrink-0 overflow-visible">
      <Avatar size={size} className={className}>
        {src ? (
          <AvatarImage src={src} alt="" className={imageClassName} />
        ) : null}
        <AvatarFallback className={cn(fallbackClassName)}>
          {initials}
        </AvatarFallback>
      </Avatar>
      {showPresence ? (
        <span
          aria-hidden
          className={cn(
            'pointer-events-none absolute -right-px -bottom-px z-10 size-2 rounded-full ring-1 ring-background',
            presenceDotClass(presence),
          )}
          title={
            presence === 'online'
              ? 'Online'
              : presence === 'away'
                ? 'Away'
                : 'Offline'
          }
        />
      ) : null}
    </span>
  )
}
