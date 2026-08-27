#!/usr/bin/env node
import { spawn } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const viteUrl = process.env.VITE_DEV_SERVER_URL || 'http://localhost:5173'

async function waitForVite(url, timeoutMs = 60_000) {
  const started = Date.now()
  while (Date.now() - started < timeoutMs) {
    try {
      const res = await fetch(url, { signal: AbortSignal.timeout(1500) })
      if (res.ok || res.status === 404) return
    } catch {
      // retry until timeout
    }
    await new Promise((resolve) => setTimeout(resolve, 400))
  }
  throw new Error(
    `Vite is not reachable at ${url}. Start instance 1 with make client-dev first.`,
  )
}

await waitForVite(viteUrl)

const electronBin = path.join(root, 'node_modules', '.bin', 'electron')
const mainEntry = path.join(root, 'dist-electron', 'main', 'index.js')

const childEnv = {
  ...process.env,
  DAGR_INSTANCE: '2',
  VITE_DEV_SERVER_URL: viteUrl,
}
// Cursor/IDE shells often set this; it makes Electron run as plain Node and break imports.
delete childEnv.ELECTRON_RUN_AS_NODE

const child = spawn(electronBin, [mainEntry], {
  cwd: root,
  stdio: 'inherit',
  env: childEnv,
})

child.on('exit', (code, signal) => {
  if (signal) process.kill(process.pid, signal)
  process.exit(code ?? 1)
})
