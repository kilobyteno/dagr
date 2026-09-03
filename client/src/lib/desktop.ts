/** True when running inside the Electron shell on macOS. */
export function isElectronMac() {
  return window.dagr?.platform === 'darwin'
}

/** True when running inside the Electron shell on Windows. */
export function isElectronWindows() {
  return window.dagr?.platform === 'win32'
}

export function isElectron() {
  return typeof window !== 'undefined' && Boolean(window.dagr?.platform)
}

export function onDesktopDeepLink(callback: (url: string) => void) {
  if (!isElectron() || !window.dagr?.onDeepLink) {
    return () => {}
  }
  return window.dagr.onDeepLink(callback)
}

export type DesktopNotificationResult = {
  shown: boolean
  reason?: string
  id?: string
}

export async function showDesktopNotification(input: {
  title: string
  body: string
  id?: string
}): Promise<DesktopNotificationResult> {
  if (!isElectron() || !window.dagr?.invoke) {
    return { shown: false, reason: 'not_electron' }
  }
  try {
    const result = (await window.dagr.invoke(
      'notifications:show',
      input,
    )) as DesktopNotificationResult | undefined
    return result ?? { shown: false, reason: 'empty_result' }
  } catch (error) {
    const reason = error instanceof Error ? error.message : 'invoke_failed'
    console.warn('[dagr] OS notification invoke failed:', reason)
    return { shown: false, reason }
  }
}

/** Keep Electron / macOS traffic lights in sync with the app colour scheme. */
export async function setDesktopTheme(theme: 'light' | 'dark') {
  if (!isElectron() || !window.dagr?.invoke) return
  try {
    await window.dagr.invoke('theme:set', theme)
  } catch (error) {
    const reason = error instanceof Error ? error.message : 'invoke_failed'
    console.warn('[dagr] theme:set invoke failed:', reason)
  }
}

/** Sync unread count onto the OS dock / launcher badge (macOS and Linux). */
export async function setDesktopBadgeCount(count: number) {
  if (!isElectron() || !window.dagr?.invoke) return
  const next = Number.isFinite(count) ? Math.max(0, Math.floor(count)) : 0
  try {
    await window.dagr.invoke('badge:set', next)
  } catch (error) {
    const reason = error instanceof Error ? error.message : 'invoke_failed'
    console.warn('[dagr] badge:set invoke failed:', reason)
  }
}
