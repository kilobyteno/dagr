/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_DAGR_CLOUD_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

interface Window {
  dagr?: {
    platform: NodeJS.Platform
    invoke: (channel: string, ...args: unknown[]) => Promise<unknown>
    onDeepLink?: (callback: (url: string) => void) => () => void
  }
}
