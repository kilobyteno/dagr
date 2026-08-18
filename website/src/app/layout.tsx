import type { Metadata } from 'next'

import './globals.css'

export const metadata: Metadata = {
  title: 'Dagr',
  description:
    'Privacy-centric, self-hostable team chat. A Slack alternative you run yourself.',
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="en-GB">
      <body className="min-h-screen font-sans antialiased">{children}</body>
    </html>
  )
}
