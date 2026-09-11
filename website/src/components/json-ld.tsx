import {
  SITE_DESCRIPTION,
  SITE_NAME,
  SITE_TAGLINE,
  SITE_URL,
} from '@/lib/site'

export function JsonLd() {
  const data = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: SITE_NAME,
    applicationCategory: 'CommunicationApplication',
    operatingSystem: 'macOS, Windows',
    description: SITE_DESCRIPTION,
    slogan: SITE_TAGLINE,
    url: SITE_URL,
    downloadUrl: `${SITE_URL}/#download`,
    license: 'https://www.apache.org/licenses/LICENSE-2.0',
    offers: [
      {
        '@type': 'Offer',
        name: 'Self host',
        price: '0',
        priceCurrency: 'EUR',
      },
      {
        '@type': 'Offer',
        name: 'Cloud Free',
        price: '0',
        priceCurrency: 'EUR',
      },
      {
        '@type': 'Offer',
        name: 'Cloud Pro',
        price: '7',
        priceCurrency: 'EUR',
        unitText: 'seat per month',
      },
    ],
  }

  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{ __html: JSON.stringify(data) }}
    />
  )
}
