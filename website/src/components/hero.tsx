import { DownloadButtons } from '@/components/download-buttons'
import { RELEASES_PAGE_URL, type LatestRelease } from '@/lib/github'
import { DOCS_SELF_HOST_URL } from '@/lib/site'

export function Hero({
  release,
  fetchFailed,
}: {
  release: LatestRelease | null
  fetchFailed: boolean
}) {
  return (
    <section className="relative overflow-hidden">
      <div className="hero-glow pointer-events-none absolute inset-0" aria-hidden />
      <div className="hero-grid pointer-events-none absolute inset-0" aria-hidden />
      <div className="relative mx-auto flex w-full max-w-6xl flex-col gap-12 px-6 py-16 lg:py-24">
        <div className="flex max-w-2xl flex-col gap-6">
          <h1 className="m-0 text-4xl font-semibold tracking-tight text-balance sm:text-5xl lg:text-6xl">
            Team chat you host yourself.
          </h1>
          <p className="m-0 max-w-xl text-lg leading-relaxed text-muted text-pretty">
            Privacy-centric, self-hostable team chat. A Slack alternative you
            run yourself.
          </p>
          <div className="flex flex-col gap-4">
            {fetchFailed ? (
              <div className="flex flex-col gap-3">
                <p className="m-0 text-sm text-muted">
                  Could not load the latest release. Try again in a moment, or
                  open GitHub.
                </p>
                <a href={RELEASES_PAGE_URL} className="btn-primary w-fit">
                  View releases on GitHub
                </a>
              </div>
            ) : (
              <DownloadButtons
                release={release}
                releasesPageUrl={RELEASES_PAGE_URL}
              />
            )}
            <a
              href={DOCS_SELF_HOST_URL}
              className="w-fit text-sm text-ink underline underline-offset-2"
            >
              Self host with Docker Compose
            </a>
          </div>
          <p className="m-0 max-w-xl text-sm leading-relaxed text-muted">
            UI is changing during early development, and might not be acurate.
          </p>
        </div>
        <div className="overflow-hidden rounded-xl border border-line bg-card shadow-[0_24px_80px_rgba(0,0,0,0.45)]">
          <div className="flex items-center gap-2 border-b border-line px-4 py-2.5">
            <span className="size-2.5 rounded-full bg-[#ff5f57]" />
            <span className="size-2.5 rounded-full bg-[#febc2e]" />
            <span className="size-2.5 rounded-full bg-[#28c840]" />
            <span className="ml-3 text-xs text-muted">Kilobyte · #general</span>
          </div>
          <img
            src="/screenshots/chat.svg"
            alt="Dagr workspace with channels, a docs mention, and team messages"
            width={1440}
            height={900}
            className="block h-auto w-full"
          />
        </div>
      </div>
    </section>
  )
}
