import {
  DOCS_COMPARE_URL,
  DOCS_PLANS_URL,
  DOCS_URL,
  GITHUB_URL,
} from '@/lib/site'

export function SiteFooter() {
  return (
    <footer className="border-t border-line">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-4 px-6 py-8 text-xs text-muted sm:flex-row sm:items-center sm:justify-between">
        <p className="m-0">Kilobyte AS · Apache 2.0</p>
        <p className="m-0 flex flex-wrap items-center gap-x-4 gap-y-2">
          <a href={DOCS_URL} className="text-muted no-underline hover:text-ink">
            Docs
          </a>
          <a
            href={GITHUB_URL}
            className="text-muted no-underline hover:text-ink"
          >
            GitHub
          </a>
          <a
            href={DOCS_COMPARE_URL}
            className="text-muted no-underline hover:text-ink"
          >
            Compare
          </a>
          <a
            href={DOCS_PLANS_URL}
            className="text-muted no-underline hover:text-ink"
          >
            Plans
          </a>
        </p>
      </div>
    </footer>
  )
}
