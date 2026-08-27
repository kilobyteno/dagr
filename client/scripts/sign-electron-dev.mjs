/**
 * Electron 42+ on macOS uses UNNotification, which rejects the stock
 * linker-signed Electron.app. Re-sign for local OS notifications.
 *
 * Prefer a stable identity named "Electron Dev" (self-signed code signing
 * cert in the login keychain). Falls back to ad-hoc signing otherwise.
 *
 * Also signs Dagr.app when present (dev launcher points at Dagr.app).
 */
import { execFileSync } from 'node:child_process'
import { createRequire } from 'node:module'
import path from 'node:path'
import { existsSync } from 'node:fs'

if (process.platform !== 'darwin') {
  process.exit(0)
}

const require = createRequire(import.meta.url)
const electronRoot = path.dirname(require.resolve('electron/package.json'))
const distDir = path.join(electronRoot, 'dist')
const appPaths = [
  path.join(distDir, 'Dagr.app'),
  path.join(distDir, 'Electron.app'),
].filter((appPath) => existsSync(appPath))

if (appPaths.length === 0) {
  console.warn(`[sign-electron-dev] No Electron/Dagr app found under ${distDir}`)
  process.exit(0)
}

function codesignIdentities() {
  try {
    const out = execFileSync('security', ['find-identity', '-v', '-p', 'codesigning'], {
      encoding: 'utf8',
    })
    return out
  } catch {
    return ''
  }
}

const identities = codesignIdentities()
const preferred = process.env.ELECTRON_DEV_IDENTITY?.trim() || 'Electron Dev'
const hasPreferred = identities.includes(`"${preferred}"`)
const identity = hasPreferred ? preferred : '-'

for (const appPath of appPaths) {
  try {
    execFileSync(
      'codesign',
      ['--force', '--deep', '--sign', identity, appPath],
      { stdio: 'inherit' },
    )
  } catch (err) {
    console.warn(
      `[sign-electron-dev] codesign failed for ${path.basename(appPath)}:`,
      err instanceof Error ? err.message : err,
    )
  }
}

if (hasPreferred) {
  console.log(`[sign-electron-dev] Signed app bundle(s) with "${preferred}"`)
} else {
  console.log(
    '[sign-electron-dev] Signed app bundle(s) ad-hoc. For a stable identity (fewer Keychain prompts), create a self-signed "Electron Dev" code signing certificate in Keychain Access, then re-run pnpm sign:dev.',
  )
}
