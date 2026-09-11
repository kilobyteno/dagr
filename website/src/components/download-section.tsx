import { DownloadButtons } from '@/components/download-buttons'
import { RELEASES_PAGE_URL, type LatestRelease } from '@/lib/github'
import { DOCS_URL } from '@/lib/site'

export function DownloadSection({
  release,
  fetchFailed,
}: {
  release: LatestRelease | null
  fetchFailed: boolean
}) {
  return (
    <section id="download" className="scroll-mt-20 border-t border-line bg-card/40">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-10 px-6 py-20">
        <div className="flex max-w-2xl flex-col gap-3">
          <h2 className="m-0 text-3xl font-semibold tracking-tight">
            Download Dagr
          </h2>
          <p className="m-0 text-base leading-relaxed text-muted">
            After install, sign in to Dagr Cloud or enter the address of a
            server you host.
          </p>
        </div>
        {fetchFailed ? (
          <div className="flex flex-col gap-3">
            <p className="m-0 text-sm text-muted">
              Could not load the latest release. Try again in a moment, or open
              GitHub.
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
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="rounded-xl border border-line bg-card p-5">
            <h3 className="m-0 text-base font-semibold">Install on macOS</h3>
            <ol className="mb-0 mt-3 list-decimal space-y-2 pl-5 text-sm leading-relaxed text-muted">
              <li>Open the disk image and drag Dagr into Applications.</li>
              <li>
                If macOS blocks the first launch, open System Settings, then
                Privacy and Security, and choose Open Anyway.
              </li>
            </ol>
          </div>
          <div className="rounded-xl border border-line bg-card p-5">
            <h3 className="m-0 text-base font-semibold">Install on Windows</h3>
            <ol className="mb-0 mt-3 list-decimal space-y-2 pl-5 text-sm leading-relaxed text-muted">
              <li>Run the installer and follow the steps.</li>
              <li>
                If SmartScreen appears, choose More info, then Run anyway.
              </li>
            </ol>
          </div>
        </div>
        <p className="m-0 max-w-xl text-sm leading-relaxed text-muted">
          See the{' '}
          <a href={DOCS_URL} className="text-ink underline underline-offset-2">
            documentation
          </a>{' '}
          for self hosting.
        </p>
      </div>
    </section>
  )
}
