import { useEffect, useRef, useState, type ImgHTMLAttributes } from 'react'

import { isLikelyGifUrl, useAppPreferences } from '@/lib/app-preferences'
import { cn } from '@/lib/utils'

type GifAwareImageProps = Omit<
  ImgHTMLAttributes<HTMLImageElement>,
  'src' | 'onMouseEnter' | 'onMouseLeave' | 'onFocus' | 'onBlur'
> & {
  src: string
  /** Show a small GIF badge while frozen. Default true. */
  showBadge?: boolean
  /**
   * Force GIF handling when the URL alone is inconclusive
   * (for example blob: object URLs).
   */
  isGif?: boolean
}

/**
 * Renders an image, optionally freezing GIFs until hover/focus when the
 * "GIFs on hover only" preference is enabled.
 */
export function GifAwareImage({
  src,
  className,
  alt = '',
  showBadge = true,
  isGif,
  ...props
}: GifAwareImageProps) {
  const { preferences } = useAppPreferences()
  const freezeGifs =
    preferences.gifsOnHoverOnly && (isGif ?? isLikelyGifUrl(src))
  const [active, setActive] = useState(false)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [frameReady, setFrameReady] = useState(false)

  useEffect(() => {
    if (!freezeGifs || active) {
      setFrameReady(false)
      return
    }

    let cancelled = false
    const image = new Image()
    image.decoding = 'async'
    image.onload = () => {
      if (cancelled) return
      const canvas = canvasRef.current
      if (!canvas) return
      const width = image.naturalWidth || image.width
      const height = image.naturalHeight || image.height
      if (!width || !height) return
      canvas.width = width
      canvas.height = height
      const context = canvas.getContext('2d')
      if (!context) return
      context.clearRect(0, 0, width, height)
      context.drawImage(image, 0, 0, width, height)
      setFrameReady(true)
    }
    image.onerror = () => {
      if (!cancelled) setFrameReady(false)
    }
    image.src = src

    return () => {
      cancelled = true
    }
  }, [src, freezeGifs, active])

  if (!freezeGifs) {
    return <img src={src} alt={alt} className={className} {...props} />
  }

  return (
    <span
      className={cn('relative inline-block overflow-hidden bg-muted', className)}
      onMouseEnter={() => setActive(true)}
      onMouseLeave={() => setActive(false)}
      onFocus={() => setActive(true)}
      onBlur={() => setActive(false)}
      tabIndex={-1}
    >
      {active ? (
        <img src={src} alt={alt} className="size-full object-cover" {...props} />
      ) : (
        <>
          <canvas
            ref={canvasRef}
            role={frameReady ? 'img' : undefined}
            aria-label={frameReady ? alt || 'GIF' : undefined}
            className={cn(
              'size-full max-h-full max-w-full object-cover',
              !frameReady && 'opacity-0',
            )}
          />
          {showBadge ? (
            <span className="pointer-events-none absolute right-1 bottom-1 rounded bg-background/85 px-1 py-0.5 text-[10px] font-semibold tracking-wide text-foreground uppercase shadow-sm ring-1 ring-border">
              GIF
            </span>
          ) : null}
        </>
      )}
    </span>
  )
}
