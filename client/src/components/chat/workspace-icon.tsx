import { useEffect, useState } from 'react'

import { GifAwareImage } from '@/components/chat/gif-aware-image'
import {
  fetchWorkspaceIconObjectUrl,
  workspaceInitials,
} from '@/lib/api/workspaces'
import { cn } from '@/lib/utils'

export function WorkspaceIconMark({
  workspaceId,
  name,
  hasIcon,
  iconUpdatedAt,
  serverUrl,
  token,
  className,
  initialsClassName,
}: {
  workspaceId: string
  name: string
  hasIcon?: boolean
  iconUpdatedAt?: string | null
  serverUrl?: string
  token?: string
  className?: string
  initialsClassName?: string
}) {
  const [icon, setIcon] = useState<{ url: string; isGif: boolean } | null>(null)
  const initials = workspaceInitials(name)

  useEffect(() => {
    if (!hasIcon || !serverUrl || !token || !workspaceId) {
      setIcon(null)
      return
    }
    const controller = new AbortController()
    let objectUrl: string | null = null
    void fetchWorkspaceIconObjectUrl(
      serverUrl,
      token,
      workspaceId,
      controller.signal,
    ).then((result) => {
      if (controller.signal.aborted) {
        if (result?.url) URL.revokeObjectURL(result.url)
        return
      }
      if (!result) {
        setIcon(null)
        return
      }
      objectUrl = result.url
      setIcon({
        url: result.url,
        isGif: result.contentType.toLowerCase().includes('gif'),
      })
    })
    return () => {
      controller.abort()
      if (objectUrl) URL.revokeObjectURL(objectUrl)
    }
  }, [hasIcon, iconUpdatedAt, serverUrl, token, workspaceId])

  if (icon) {
    return (
      <GifAwareImage
        src={icon.url}
        alt=""
        isGif={icon.isGif}
        showBadge={false}
        className={cn('object-cover', className)}
      />
    )
  }

  return (
    <span className={cn(initialsClassName, className)} aria-hidden>
      {initials}
    </span>
  )
}
