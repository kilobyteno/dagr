const features = [
  {
    title: 'Channels and DMs',
    body: 'Public and private channels, mentions, reactions, and scheduled send. People open Dagr and already know where to type.',
    icon: (
      <path
        d="M5 8.5A3.5 3.5 0 0 1 8.5 5h7A3.5 3.5 0 0 1 19 8.5v4A3.5 3.5 0 0 1 15.5 16H12l-4 3v-3H8.5A3.5 3.5 0 0 1 5 12.5z"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinejoin="round"
      />
    ),
  },
  {
    title: 'Workspace docs',
    body: 'A markdown wiki beside chat. Mention pages with [[slug]] so the conversation and the write-up stay together.',
    icon: (
      <path
        d="M8 4.5h5.2L18 9.3V19a1.5 1.5 0 0 1-1.5 1.5h-8.5A1.5 1.5 0 0 1 6.5 19V6A1.5 1.5 0 0 1 8 4.5zM13 4.5V9h4.5"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinejoin="round"
      />
    ),
  },
  {
    title: 'Notifications',
    body: 'In-app inbox and OS alerts. Silence a channel or keep mentions loud, per account.',
    icon: (
      <path
        d="M12 4.5a5 5 0 0 1 5 5v3.2l1.3 2.3H5.7L7 12.7V9.5a5 5 0 0 1 5-5zm-2.2 13.2a2.2 2.2 0 0 0 4.4 0"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinejoin="round"
      />
    ),
  },
  {
    title: 'Incoming webhooks',
    body: 'Post into a channel from the tools you already run. Native JSON or Slack-compatible payloads.',
    icon: (
      <path
        d="M8 8.5h3.2a3 3 0 0 1 0 6H8m8-6h-3.2a3 3 0 0 0 0 6H16M6.5 11.5h11"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
      />
    ),
  },
  {
    title: 'Cloud or self host',
    body: 'Run the stack yourself with no seat tax, or use Dagr Cloud when you would rather not operate a server.',
    icon: (
      <path
        d="M6 16.5h12.5A2.5 2.5 0 0 0 21 14c0-1.8-1.4-3.2-3.1-3.4A4.5 4.5 0 0 0 9.2 8.4 3.5 3.5 0 0 0 4 11.7c0 .2 0 .5.1.7H6"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinejoin="round"
      />
    ),
  },
  {
    title: 'Multi-account desktop',
    body: 'macOS and Windows app, plus the same UI in a browser. Stay signed in to Cloud and a server you host.',
    icon: (
      <path
        d="M5 7.5A1.5 1.5 0 0 1 6.5 6h11A1.5 1.5 0 0 1 19 7.5v8A1.5 1.5 0 0 1 17.5 17h-11A1.5 1.5 0 0 1 5 15.5zM8 20h8"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    ),
  },
]

export function Features() {
  return (
    <section id="features" className="scroll-mt-20">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-10 px-6 py-20">
        <div className="flex max-w-2xl flex-col gap-3">
          <h2 className="m-0 text-3xl font-semibold tracking-tight">
            Slack-like, on your server
          </h2>
          <p className="m-0 text-base leading-relaxed text-muted">
            Channels, DMs, docs, and a desktop app your team already
            understands. Postgres, Redis, and files stay on infrastructure you
            control.
          </p>
        </div>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {features.map((feature) => (
            <article
              key={feature.title}
              className="flex flex-col gap-3 rounded-xl border border-line bg-card p-5"
            >
              <span className="flex size-10 items-center justify-center rounded-lg bg-brand/10 text-brand">
                <svg viewBox="0 0 24 24" className="size-5" aria-hidden>
                  {feature.icon}
                </svg>
              </span>
              <h3 className="m-0 text-base font-semibold">{feature.title}</h3>
              <p className="m-0 text-sm leading-relaxed text-muted">
                {feature.body}
              </p>
            </article>
          ))}
        </div>
      </div>
    </section>
  )
}
