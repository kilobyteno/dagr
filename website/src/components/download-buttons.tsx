'use client'

import { useEffect, useMemo, useState } from 'react'

import type { LatestRelease } from '@/lib/github'

type Os = 'mac' | 'windows' | 'other'

function detectOs(): Os {
  const ua = navigator.userAgent
  if (/Mac|iPhone|iPad/.test(ua)) return 'mac'
  if (/Win/.test(ua)) return 'windows'
  return 'other'
}

function DownloadLink({
  href,
  label,
  primary,
}: {
  href: string
  label: string
  primary?: boolean
}) {
  return (
    <a
      href={href}
      className={primary ? 'btn-primary' : 'btn-secondary'}
    >
      {label}
    </a>
  )
}

export function DownloadButtons({
  release,
  releasesPageUrl,
}: {
  release: LatestRelease | null
  releasesPageUrl: string
}) {
  const [os, setOs] = useState<Os>('other')

  useEffect(() => {
    setOs(detectOs())
  }, [])

  const buttons = useMemo(() => {
    const mac = release?.macDmgUrl
      ? { href: release.macDmgUrl, label: `Download for macOS${release.version ? ` (${release.version})` : ''}` }
      : null
    const windows = release?.windowsExeUrl
      ? { href: release.windowsExeUrl, label: `Download for Windows${release.version ? ` (${release.version})` : ''}` }
      : null

    if (!mac && !windows) return []

    if (os === 'windows') {
      return [
        windows ? { ...windows, primary: true } : null,
        mac ? { ...mac, primary: false } : null,
      ].filter(Boolean) as { href: string; label: string; primary: boolean }[]
    }

    return [
      mac ? { ...mac, primary: os === 'mac' } : null,
      windows ? { ...windows, primary: os !== 'mac' } : null,
    ].filter(Boolean) as { href: string; label: string; primary: boolean }[]
  }, [os, release])

  if (!release || buttons.length === 0) {
    return (
      <div className="flex flex-col gap-3">
        <p className="m-0 text-sm text-muted">
          Releases are not published yet. Installers will appear here once a
          GitHub release is published.
        </p>
        <DownloadLink href={releasesPageUrl} label="View releases on GitHub" />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:flex-wrap">
      {buttons.map((button) => (
        <DownloadLink
          key={button.href}
          href={button.href}
          label={button.label}
          primary={button.primary}
        />
      ))}
    </div>
  )
}
