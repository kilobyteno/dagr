const items = [
  {
    title: 'Apache 2.0',
    body: 'One product. No enterprise upsell to stay complete.',
  },
  {
    title: 'One Compose file',
    body: 'API, Postgres, Redis, and MinIO on infrastructure you control.',
  },
  {
    title: 'No required SaaS',
    body: 'Email and object storage are optional to configure.',
  },
  {
    title: 'Desktop and web',
    body: 'Sign in from the app, a browser, or several servers at once.',
  },
]

export function TrustStrip() {
  return (
    <section className="border-y border-line bg-card/40">
      <div className="mx-auto grid w-full max-w-6xl gap-6 px-6 py-10 sm:grid-cols-2 lg:grid-cols-4">
        {items.map((item) => (
          <div key={item.title} className="flex flex-col gap-2">
            <p className="m-0 text-sm font-semibold tracking-tight">{item.title}</p>
            <p className="m-0 text-sm leading-relaxed text-muted">{item.body}</p>
          </div>
        ))}
      </div>
    </section>
  )
}
