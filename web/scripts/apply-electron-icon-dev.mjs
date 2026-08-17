/**
 * Prepare a Dagr-branded macOS app bundle for development:
 * - Build an opaque, padded .icns from build/icon.png
 * - Mirror Electron.app → Dagr.app (so Dock/About say "Dagr", not "Electron")
 * - Install icons + Info.plist branding
 * - Point electron/path.txt at Dagr.app
 */
import { execFileSync } from 'node:child_process'
import { createRequire } from 'node:module'
import {
  copyFileSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from 'node:fs'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

if (process.platform !== 'darwin') {
  process.exit(0)
}

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const webRoot = path.join(__dirname, '..')
const pngPath = path.join(webRoot, 'build', 'icon.png')
const icnsPath = path.join(webRoot, 'build', 'icon.icns')
const dockPngPath = path.join(webRoot, 'build', 'icon-dock.png')
const padScript = path.join(__dirname, 'pad-app-icon.swift')

if (!existsSync(pngPath)) {
  console.warn(`[apply-electron-icon-dev] Missing ${pngPath}; skipping.`)
  process.exit(0)
}

function buildIcns() {
  const work = mkdtempSync(path.join(tmpdir(), 'dagr-icon-'))
  const setDir = path.join(work, 'icon.iconset')
  const padded = path.join(work, 'padded.png')
  mkdirSync(setDir)

  try {
    execFileSync('swift', [padScript, pngPath, padded], { stdio: 'pipe' })
    copyFileSync(padded, dockPngPath)

    const entries = [
      [16, 'icon_16x16.png'],
      [32, 'icon_16x16@2x.png'],
      [32, 'icon_32x32.png'],
      [64, 'icon_32x32@2x.png'],
      [128, 'icon_128x128.png'],
      [256, 'icon_128x128@2x.png'],
      [256, 'icon_256x256.png'],
      [512, 'icon_256x256@2x.png'],
      [512, 'icon_512x512.png'],
      [1024, 'icon_512x512@2x.png'],
    ]

    for (const [px, name] of entries) {
      execFileSync(
        'sips',
        ['-z', String(px), String(px), padded, '--out', path.join(setDir, name)],
        { stdio: 'pipe' },
      )
    }
    execFileSync('iconutil', ['-c', 'icns', setDir, '-o', icnsPath], {
      stdio: 'pipe',
    })
    console.log(`[apply-electron-icon-dev] Wrote ${icnsPath}`)
  } finally {
    rmSync(work, { recursive: true, force: true })
  }
}

function patchPlist(infoPlist) {
  if (!existsSync(infoPlist)) return
  let next = readFileSync(infoPlist, 'utf8')
  const replacements = [
    [/(<key>CFBundleIconFile<\/key>\s*<string>)[^<]+(<\/string>)/, '$1dagr.icns$2'],
    [/(<key>CFBundleDisplayName<\/key>\s*<string>)[^<]+(<\/string>)/, '$1Dagr$2'],
    [/(<key>CFBundleName<\/key>\s*<string>)[^<]+(<\/string>)/, '$1Dagr$2'],
    [
      /(<key>CFBundleIdentifier<\/key>\s*<string>)[^<]+(<\/string>)/,
      '$1no.kilobyte.dagr.dev$2',
    ],
  ]
  for (const [pattern, replacement] of replacements) {
    next = next.replace(pattern, replacement)
  }
  writeFileSync(infoPlist, next)
}

function brandAppBundle(appPath, { touch = true } = {}) {
  const resources = path.join(appPath, 'Contents', 'Resources')
  const infoPlist = path.join(appPath, 'Contents', 'Info.plist')
  copyFileSync(icnsPath, path.join(resources, 'electron.icns'))
  copyFileSync(icnsPath, path.join(resources, 'dagr.icns'))
  patchPlist(infoPlist)
  if (touch) {
    execFileSync('touch', [appPath], { stdio: 'pipe' })
  }
}

try {
  buildIcns()
} catch (err) {
  console.warn(
    '[apply-electron-icon-dev] Could not build icon.icns:',
    err instanceof Error ? err.message : err,
  )
  process.exit(0)
}

const require = createRequire(import.meta.url)
const electronRoot = path.dirname(require.resolve('electron/package.json'))
const electronPkg = JSON.parse(
  readFileSync(path.join(electronRoot, 'package.json'), 'utf8'),
)
const distDir = path.join(electronRoot, 'dist')
const electronApp = path.join(distDir, 'Electron.app')
const dagrApp = path.join(distDir, 'Dagr.app')
const stampPath = path.join(distDir, '.dagr-app-stamp')
const pathTxt = path.join(electronRoot, 'path.txt')

if (!existsSync(electronApp)) {
  console.warn('[apply-electron-icon-dev] Electron.app missing; skipping.')
  process.exit(0)
}

const stamp = String(electronPkg.version)

try {
  const previous = existsSync(stampPath) ? readFileSync(stampPath, 'utf8').trim() : ''
  if (previous !== stamp || !existsSync(dagrApp)) {
    rmSync(dagrApp, { recursive: true, force: true })
    execFileSync('cp', ['-R', electronApp, dagrApp], { stdio: 'pipe' })
    writeFileSync(stampPath, stamp)
    console.log('[apply-electron-icon-dev] Created Dagr.app from Electron.app')
  }

  brandAppBundle(dagrApp)
  // Keep Electron.app branded too, in case something still launches it.
  brandAppBundle(electronApp, { touch: false })

  // electron/index.js does not trim path.txt; a trailing newline breaks spawn.
  writeFileSync(pathTxt, 'Dagr.app/Contents/MacOS/Electron')
  console.log('[apply-electron-icon-dev] Electron launcher → Dagr.app')
} catch (err) {
  console.warn(
    '[apply-electron-icon-dev] Could not prepare Dagr.app:',
    err instanceof Error ? err.message : err,
  )
}
