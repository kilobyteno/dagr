import { DOCS_PLANS_URL, DOCS_SELF_HOST_URL } from '@/lib/site'

const plans = [
  {
    name: 'Self host',
    price: 'Your infra',
    detail: 'No plan. No seat tax.',
    href: DOCS_SELF_HOST_URL,
    cta: 'Read the hosting guide',
    highlight: false,
    points: [
      'Unlimited history',
      'One Compose file',
      'Desktop and web against your API',
    ],
  },
  {
    name: 'Cloud Free',
    price: '€0',
    detail: 'Hosted by Kilobyte.',
    href: DOCS_PLANS_URL,
    cta: 'See Cloud plans',
    highlight: false,
    points: ['90 days of history', '10 apps and integrations', 'Home members only'],
  },
  {
    name: 'Cloud Pro',
    price: '€7',
    detail: 'Per seat per month.',
    href: DOCS_PLANS_URL,
    cta: 'See Cloud plans',
    highlight: true,
    points: [
      'Unlimited history',
      'Unlimited apps',
      'Yearly billing 10% off',
    ],
  },
]

export function Pricing() {
  return (
    <section id="pricing" className="scroll-mt-20">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-10 px-6 py-20">
        <div className="flex max-w-2xl flex-col gap-3">
          <h2 className="m-0 text-3xl font-semibold tracking-tight">
            Host it, or let us
          </h2>
          <p className="m-0 text-base leading-relaxed text-muted">
            Self hosted workspaces are not on a plan. Cloud is the same product
            when you would rather not run a server.
          </p>
        </div>
        <div className="grid gap-4 lg:grid-cols-3">
          {plans.map((plan) => (
            <article
              key={plan.name}
              className={
                plan.highlight
                  ? 'flex flex-col gap-5 rounded-xl border border-brand bg-card p-6'
                  : 'flex flex-col gap-5 rounded-xl border border-line bg-card p-6'
              }
            >
              <div className="flex flex-col gap-1">
                <h3 className="m-0 text-base font-semibold">{plan.name}</h3>
                <p className="m-0 text-3xl font-semibold tracking-tight">
                  {plan.price}
                </p>
                <p className="m-0 text-sm text-muted">{plan.detail}</p>
              </div>
              <ul className="m-0 flex flex-1 list-none flex-col gap-2 p-0 text-sm leading-relaxed text-muted">
                {plan.points.map((point) => (
                  <li key={point}>{point}</li>
                ))}
              </ul>
              <a href={plan.href} className="btn-secondary w-fit">
                {plan.cta}
              </a>
            </article>
          ))}
        </div>
        <p className="m-0 max-w-2xl text-sm leading-relaxed text-muted">
          During early access, the first 3 months of Cloud Pro are 50% off.
          External guests do not add to the bill.{' '}
          <a
            href={DOCS_PLANS_URL}
            className="text-ink underline underline-offset-2"
          >
            Seat rules and yearly pricing
          </a>
        </p>
      </div>
    </section>
  )
}
