import { DownloadButtons } from '@/components/download-buttons'
import {
  DEFAULT_GITHUB_REPO,
  RELEASES_PAGE_URL,
  fetchLatestRelease,
} from '@/lib/github'

const DOCS_URL = 'https://docs.page/kilobyteno/dagr'
const GITHUB_URL = `https://github.com/${DEFAULT_GITHUB_REPO}`

export default async function HomePage() {
  let release = null
  let fetchFailed = false
  try {
    release = await fetchLatestRelease()
  } catch {
    fetchFailed = true
  }

  return (
    <div className="mx-auto flex min-h-screen w-full max-w-3xl flex-col px-6 py-8">
      <header className="flex items-center justify-between gap-4">
        <a href="/" className="flex items-center gap-3 no-underline text-ink">
          <img
            src="/app-icon.png"
            alt=""
            width={40}
            height={40}
            className="size-10 rounded-md"
          />
          <span className="text-lg font-semibold tracking-tight">Dagr</span>
        </a>
        <nav className="flex items-center gap-4 text-sm">
          <a href={DOCS_URL} className="text-muted no-underline hover:text-ink">
            Docs
          </a>
          <a href={GITHUB_URL} className="text-muted no-underline hover:text-ink">
            GitHub
          </a>
        </nav>
      </header>

      <main className="flex flex-1 flex-col justify-center gap-10 py-16">
        <div className="flex flex-col gap-4">
          <h1 className="m-0 text-4xl font-semibold tracking-tight text-balance sm:text-5xl">
            Team chat you host yourself.
          </h1>
          <p className="m-0 max-w-xl text-base leading-relaxed text-muted text-pretty">
            Privacy-centric, self-hostable team chat. A Slack alternative you
            run yourself.
          </p>
        </div>

        <section className="flex flex-col gap-4">
          {fetchFailed ? (
            <div className="flex flex-col gap-3">
              <p className="m-0 text-sm text-muted">
                Could not load the latest release. Try again in a moment, or
                open GitHub.
              </p>
              <a
                href={RELEASES_PAGE_URL}
                className="inline-flex w-fit items-center justify-center rounded-lg bg-brand px-5 py-3 text-sm font-semibold text-white no-underline hover:bg-[#d95a1c]"
              >
                View releases on GitHub
              </a>
            </div>
          ) : (
            <DownloadButtons
              release={release}
              releasesPageUrl={RELEASES_PAGE_URL}
            />
          )}
        </section>

        <section className="grid gap-4 sm:grid-cols-2">
          <div className="rounded-xl border border-line bg-card p-5">
            <h2 className="m-0 text-base font-semibold">Install on macOS</h2>
            <ol className="mb-0 mt-3 list-decimal space-y-2 pl-5 text-sm leading-relaxed text-muted">
              <li>Open the disk image and drag Dagr into Applications.</li>
              <li>
                If macOS blocks the first launch, open System Settings, then
                Privacy and Security, and choose Open Anyway.
              </li>
            </ol>
          </div>
          <div className="rounded-xl border border-line bg-card p-5">
            <h2 className="m-0 text-base font-semibold">Install on Windows</h2>
            <ol className="mb-0 mt-3 list-decimal space-y-2 pl-5 text-sm leading-relaxed text-muted">
              <li>Run the installer and follow the steps.</li>
              <li>
                If SmartScreen appears, choose More info, then Run anyway.
              </li>
            </ol>
          </div>
        </section>

        <p className="m-0 max-w-xl text-sm leading-relaxed text-muted">
          After install, sign in to Dagr Cloud or enter the address of a server
          you host. See the{' '}
          <a href={DOCS_URL} className="text-ink underline underline-offset-2">
            documentation
          </a>{' '}
          for self hosting.
        </p>
      </main>

      <footer className="flex flex-col gap-2 border-t border-line pt-6 text-xs text-muted sm:flex-row sm:items-center sm:justify-between">
        <p className="m-0">Kilobyte AS · Apache 2.0</p>
        <p className="m-0">
          <a href={DOCS_URL} className="text-muted no-underline hover:text-ink">
            Docs
          </a>
          {' · '}
          <a href={GITHUB_URL} className="text-muted no-underline hover:text-ink">
            GitHub
          </a>
        </p>
      </footer>
    </div>
  )
}
