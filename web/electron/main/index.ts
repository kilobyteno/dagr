import { Notification, app, BrowserWindow, ipcMain, nativeTheme, shell } from 'electron'
import { checkForUpdates, openUpdateUrl } from './updates'
import { existsSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import os from 'node:os'

const WINDOW_BG_LIGHT = '#f5f5f5'
const WINDOW_BG_DARK = '#0a0a0a'

function windowBackgroundForDark(dark: boolean) {
  return dark ? WINDOW_BG_DARK : WINDOW_BG_LIGHT
}

function applyNativeTheme(theme: 'light' | 'dark') {
  nativeTheme.themeSource = theme
  win?.setBackgroundColor(windowBackgroundForDark(theme === 'dark'))
}

const __dirname = path.dirname(fileURLToPath(import.meta.url))

process.env.APP_ROOT = path.join(__dirname, '../..')

export const MAIN_DIST = path.join(process.env.APP_ROOT, 'dist-electron')
export const RENDERER_DIST = path.join(process.env.APP_ROOT, 'dist')
export const VITE_DEV_SERVER_URL = process.env.VITE_DEV_SERVER_URL

process.env.VITE_PUBLIC = VITE_DEV_SERVER_URL
  ? path.join(process.env.APP_ROOT, 'public')
  : RENDERER_DIST

const instanceRaw = (process.env.DAGR_INSTANCE || '1').trim()
const instance = /^\d+$/.test(instanceRaw) ? instanceRaw : '1'

const appName = instance === '1' ? 'Dagr' : `Dagr (instance ${instance})`
app.setName(appName)

function readAppVersion(): string {
  try {
    const pkgPath = path.join(process.env.APP_ROOT!, 'package.json')
    const pkg = JSON.parse(readFileSync(pkgPath, 'utf8')) as { version?: string }
    if (typeof pkg.version === 'string' && pkg.version.trim()) {
      return pkg.version.trim()
    }
  } catch {
    // Fall back to Electron's reported version.
  }
  return app.getVersion()
}

function resolveAppIcon(): string | undefined {
  const root = process.env.APP_ROOT!
  const candidates = [
    path.join(root, 'build', process.platform === 'darwin' ? 'icon.icns' : ''),
    path.join(root, 'build', process.platform === 'win32' ? 'icon.ico' : 'icon.png'),
    path.join(root, 'build', 'icon.png'),
    path.join(root, 'build', 'icon.icns'),
    path.join(process.env.VITE_PUBLIC!, 'app-icon.png'),
    path.join(process.env.VITE_PUBLIC!, 'favicon.ico'),
  ]
  return candidates.find((candidate) => candidate && existsSync(candidate))
}

/** Keep macOS Dagr → About Dagr in sync with package.json and the app icon. */
function configureAboutPanel() {
  const version = readAppVersion()
  const iconPath = resolveAppIcon()
  app.setAboutPanelOptions({
    applicationName: 'Dagr',
    applicationVersion: version,
    version,
    copyright: 'Copyright © Kilobyte AS',
    // Linux / Windows About dialogs; macOS uses the app / Dock icon instead.
    ...(iconPath ? { iconPath } : {}),
  })
}

configureAboutPanel()

// Separate Chromium profile (and sessionStorage) per instance for local multi-user testing.
app.setPath(
  'userData',
  path.join(app.getPath('userData'), `dagr-instance-${instance}`),
)

if (process.platform === 'win32' && os.release().startsWith('6.1')) {
  app.disableHardwareAcceleration()
}

if (process.platform === 'win32') {
  app.setAppUserModelId(`no.kilobyte.dagr.${instance}`)
}

const useSingleInstanceLock = instance === '1'
if (useSingleInstanceLock) {
  if (!app.requestSingleInstanceLock()) {
    app.quit()
    process.exit(0)
  }
}

const PROTOCOL = 'dagr'

if (process.defaultApp) {
  if (process.argv.length >= 2) {
    app.setAsDefaultProtocolClient(PROTOCOL, process.execPath, [
      path.resolve(process.argv[1]),
    ])
  }
} else {
  app.setAsDefaultProtocolClient(PROTOCOL)
}

function isDagrDeepLink(value: string) {
  return value.startsWith(`${PROTOCOL}://`)
}

function extractDeepLink(argv: string[]) {
  for (const raw of argv) {
    const value = raw.replace(/^"+|"+$/g, '')
    if (isDagrDeepLink(value)) return value
  }
  return null
}

let pendingDeepLink = extractDeepLink(process.argv)

function sendDeepLink(url: string) {
  if (!isDagrDeepLink(url)) return
  if (win && !win.isDestroyed()) {
    win.webContents.send('deep-link', url)
    focusMainWindow()
    return
  }
  pendingDeepLink = url
}

let win: BrowserWindow | null = null
const preload = path.join(__dirname, '../preload/index.mjs')
const indexHtml = path.join(RENDERER_DIST, 'index.html')
const windowTitle = instance === '1' ? 'Dagr' : `Dagr (instance ${instance})`

async function createWindow() {
  const icon = resolveAppIcon()
  win = new BrowserWindow({
    title: windowTitle,
    width: 1280,
    height: 800,
    minWidth: 900,
    minHeight: 600,
    backgroundColor: windowBackgroundForDark(nativeTheme.shouldUseDarkColors),
    ...(icon ? { icon } : {}),
    ...(process.platform === 'darwin'
      ? {
          titleBarStyle: 'hidden' as const,
          trafficLightPosition: { x: 16, y: 10 },
        }
      : {}),
    webPreferences: {
      preload,
      contextIsolation: true,
      nodeIntegration: false,
    },
  })

  win.setTitle(windowTitle)

  if (VITE_DEV_SERVER_URL) {
    await win.loadURL(VITE_DEV_SERVER_URL)
    win.webContents.openDevTools({ mode: 'detach' })
  } else {
    await win.loadFile(indexHtml)
  }

  win.webContents.setWindowOpenHandler(({ url }) => {
    if (url.startsWith('https:') || url.startsWith('http:')) {
      shell.openExternal(url)
    }
    return { action: 'deny' }
  })
}

function focusMainWindow() {
  if (!win) return
  if (win.isMinimized()) win.restore()
  win.show()
  win.focus()
}

app.whenReady().then(() => {
  configureAboutPanel()

  ipcMain.handle('theme:set', async (_event, theme: unknown) => {
    if (theme !== 'light' && theme !== 'dark') return { ok: false }
    applyNativeTheme(theme)
    return { ok: true }
  })

  ipcMain.handle('badge:set', async (_event, count: unknown) => {
    const next =
      typeof count === 'number' && Number.isFinite(count)
        ? Math.max(0, Math.floor(count))
        : 0
    if (process.platform === 'darwin' && app.dock) {
      // String badge supports a compact 99+ form on the macOS Dock.
      app.dock.setBadge(next > 0 ? String(next > 99 ? '99+' : next) : '')
    } else {
      app.setBadgeCount(next)
    }
    return { ok: true, count: next }
  })

  ipcMain.handle('updates:check', async (_event, payload: unknown) => {
    const body =
      typeof payload === 'object' && payload !== null
        ? (payload as { force?: unknown; channel?: unknown })
        : {}
    return checkForUpdates(readAppVersion(), {
      force: Boolean(body.force),
      channel: body.channel,
    })
  })

  ipcMain.handle('updates:open', async (_event, target: unknown) => {
    const url = typeof target === 'string' ? target : undefined
    return openUpdateUrl(url)
  })

  ipcMain.handle(
    'notifications:show',
    async (_event, payload: { title?: string; body?: string; id?: string }) => {
      if (!Notification.isSupported()) {
        return { shown: false, reason: 'unsupported' }
      }
      const title = payload?.title?.trim() || 'Dagr'
      const body = payload?.body?.trim() || ''
      const notification = new Notification({
        title,
        body,
        silent: false,
      })

      const outcome = await new Promise<{
        shown: boolean
        reason?: string
        id?: string
      }>((resolve) => {
        let settled = false
        const finish = (result: { shown: boolean; reason?: string; id?: string }) => {
          if (settled) return
          settled = true
          resolve(result)
        }

        notification.on('show', () => {
          finish({ shown: true, id: payload?.id })
        })
        notification.on('failed', (_event, error) => {
          const message = String(error)
          console.warn('[dagr] OS notification failed:', message)
          finish({ shown: false, reason: message, id: payload?.id })
        })
        notification.on('click', () => {
          focusMainWindow()
        })

        // If neither show nor failed arrives quickly, do not hang the IPC call.
        setTimeout(() => {
          finish({ shown: true, id: payload?.id, reason: 'timeout' })
        }, 1500)

        notification.show()
      })

      return outcome
    },
  )
  void createWindow().then(() => {
    if (!pendingDeepLink) return
    const url = pendingDeepLink
    pendingDeepLink = null
    sendDeepLink(url)
  })
})

app.on('open-url', (event, url) => {
  event.preventDefault()
  sendDeepLink(url)
})

function clearAppBadge() {
  if (process.platform === 'darwin' && app.dock) {
    app.dock.setBadge('')
  } else {
    app.setBadgeCount(0)
  }
}

app.on('window-all-closed', () => {
  win = null
  clearAppBadge()
  if (process.platform !== 'darwin') app.quit()
})

if (useSingleInstanceLock) {
  app.on('second-instance', (_event, commandLine) => {
    const url = extractDeepLink(commandLine)
    if (url) {
      sendDeepLink(url)
      return
    }
    focusMainWindow()
  })
}

app.on('activate', () => {
  const allWindows = BrowserWindow.getAllWindows()
  if (allWindows.length) {
    allWindows[0].focus()
  } else {
    createWindow()
  }
})
