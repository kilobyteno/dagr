import { DOCS_COMPARE_URL } from '@/lib/site'

const rows = [
  {
    label: 'Who holds the messages',
    dagr: 'You, on your server or Dagr Cloud',
    slack: 'Slack',
  },
  {
    label: 'Self host',
    dagr: 'Yes, one Compose file',
    slack: 'No',
  },
  {
    label: 'History',
    dagr: 'Unlimited on self host and Cloud Pro',
    slack: '90 days on Free, then pay',
  },
  {
    label: 'Price',
    dagr: 'Your infra, or Cloud Pro at €7 per seat',
    slack: 'Paid seats to keep history and the full product',
  },
  {
    label: 'Licence',
    dagr: 'Apache 2.0',
    slack: 'Proprietary',
  },
]

export function CompareTeaser() {
  return (
    <section id="compare" className="scroll-mt-20 border-y border-line bg-card/40">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-10 px-6 py-20">
        <div className="flex max-w-2xl flex-col gap-3">
          <h2 className="m-0 text-3xl font-semibold tracking-tight">
            Slack you run yourself
          </h2>
          <p className="m-0 text-base leading-relaxed text-muted">
            Keep the model your team already knows. Drop the vendor. Mattermost,
            Rocket.Chat, Zulip, and Element are compared in the docs.
          </p>
        </div>
        <div className="overflow-x-auto rounded-xl border border-line bg-card">
          <table className="w-full min-w-[36rem] border-collapse text-left text-sm">
            <thead>
              <tr className="border-b border-line text-muted">
                <th className="px-5 py-3 font-medium" />
                <th className="px-5 py-3 font-semibold text-ink">Dagr</th>
                <th className="px-5 py-3 font-medium">Slack</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.label} className="border-b border-line last:border-b-0">
                  <th className="px-5 py-3 font-medium text-muted">{row.label}</th>
                  <td className="px-5 py-3 text-ink">{row.dagr}</td>
                  <td className="px-5 py-3 text-muted">{row.slack}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <a
          href={DOCS_COMPARE_URL}
          className="w-fit text-sm font-medium text-ink underline underline-offset-2"
        >
          Why teams leave Slack and other team chat
        </a>
      </div>
    </section>
  )
}
