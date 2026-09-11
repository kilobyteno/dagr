import { CompareTeaser } from '@/components/compare-teaser'
import { DownloadSection } from '@/components/download-section'
import { Features } from '@/components/features'
import { Hero } from '@/components/hero'
import { JsonLd } from '@/components/json-ld'
import { Pricing } from '@/components/pricing'
import { SiteFooter } from '@/components/site-footer'
import { SiteHeader } from '@/components/site-header'
import { TrustStrip } from '@/components/trust-strip'
import { fetchLatestRelease } from '@/lib/github'

export default async function HomePage() {
  let release = null
  let fetchFailed = false
  try {
    release = await fetchLatestRelease()
  } catch {
    fetchFailed = true
  }

  return (
    <div className="flex min-h-screen flex-col">
      <JsonLd />
      <a href="#features" className="skip-link">
        Skip to content
      </a>
      <SiteHeader />
      <main className="flex-1">
        <Hero release={release} fetchFailed={fetchFailed} />
        <TrustStrip />
        <Features />
        <CompareTeaser />
        <Pricing />
        <DownloadSection release={release} fetchFailed={fetchFailed} />
      </main>
      <SiteFooter />
    </div>
  )
}
