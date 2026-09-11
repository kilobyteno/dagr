import { DOCS_URL, GITHUB_URL } from '@/lib/site'

const nav = [
  { href: '#features', label: 'Features' },
  { href: '#compare', label: 'Compare' },
  { href: '#pricing', label: 'Pricing' },
  { href: '#download', label: 'Download' },
]

export function SiteHeader() {
  return (
    <header className="sticky top-0 z-20 border-b border-line/80 bg-page/80 backdrop-blur-md">
      <div className="mx-auto flex h-16 w-full max-w-6xl items-center justify-between gap-4 px-6">
        <a href="/" className="flex items-center gap-3 text-ink no-underline">
          <img
            src="/app-icon.png"
            alt=""
            width={36}
            height={36}
            className="size-9 rounded-md"
          />
          <span className="text-lg font-semibold tracking-tight">Dagr</span>
        </a>
        <nav className="flex items-center gap-4 text-sm sm:gap-5">
          {nav.map((item) => (
            <a
              key={item.href}
              href={item.href}
              className="hidden text-muted no-underline hover:text-ink md:inline"
            >
              {item.label}
            </a>
          ))}
          <a
            href={DOCS_URL}
            className="text-muted no-underline hover:text-ink"
          >
            Docs
          </a>
          <a
            href={GITHUB_URL}
            className="text-muted no-underline hover:text-ink"
          >
            GitHub
          </a>
        </nav>
      </div>
    </header>
  )
}
