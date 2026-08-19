import { useEffect } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { useAuth } from '../App.jsx'
import LoginForm from '../components/LoginForm.jsx'

const STATES = ['Colorado', 'Connecticut', 'Minnesota', 'Montana', 'New Mexico', 'Rhode Island', 'Utah', 'Vermont']

const STEPS = [
  {
    n: '1',
    title: 'Enter the client’s numbers',
    body: 'Filing status, age, Social Security benefits, other income, and tax-exempt interest. The same figures already on the return.',
  },
  {
    n: '2',
    title: 'We run the worksheet',
    body: 'The full IRS Publication 915 worksheet and the client’s state rule run on our server in seconds.',
  },
  {
    n: '3',
    title: 'Show the client why',
    body: 'Every number comes with the rule that produced it. You get a plain-English explanation and an audit trail, not just a total.',
  },
]

const FEATURES = [
  {
    title: 'Federal, done right',
    body: 'Not the rough “50% or 85%” shortcut. We run the actual Publication 915 Worksheet 1, so the taxable amount matches the filed return.',
  },
  {
    title: 'All eight taxing states',
    body: 'Colorado, Connecticut, Minnesota, Montana, New Mexico, Rhode Island, Utah, and Vermont. Each one has its own thresholds, phase-outs, and age rules. We handle them.',
  },
  {
    title: 'A reason for every number',
    body: 'Each result tells you the rule and threshold behind it in plain English. Now you can explain the figure to the client and defend it later.',
  },
  {
    title: 'Rules kept current',
    body: 'Thresholds and rates are stored by tax year. When a state changes its numbers, we update them. You don’t have to track it.',
  },
  {
    title: 'Save client scenarios',
    body: 'Model a Roth conversion or a bigger withdrawal, save it, and come back to it. See the tax impact before it happens.',
  },
  {
    title: 'Made for your practice',
    body: 'Many clients, fast data entry, numbers you can trust. Built for professionals who serve retirees, not for consumers.',
  },
]

const PLANS = [
  {
    name: 'Free',
    price: '$0',
    cadence: 'forever',
    blurb: 'For trying it out or handling a few clients.',
    cta: { label: 'Get started', to: '/register', primary: false },
    features: [
      'Up to 3 saved clients',
      '1 saved scenario per client',
      'Current tax year only',
      'Full federal + 8-state engine',
      'Rule-by-rule audit trail',
      'Community support',
    ],
    highlight: false,
  },
  {
    name: 'Pro',
    price: '$39',
    cadence: 'per month',
    blurb: 'For practices running these numbers all season.',
    cta: { label: 'Upgrade to Pro', to: '/register', primary: true },
    features: [
      'Unlimited clients',
      'Unlimited scenarios per client',
      'Multi-year what-if planning',
      'All available tax years',
      'Everything in Free',
      'Priority support',
    ],
    highlight: true,
  },
]

export default function Landing() {
  const loc = useLocation()
  const { auth } = useAuth()
  const proTo = auth ? '/app/billing' : '/register'

  // Smooth-scroll to the pricing section when arriving with #pricing
  // (e.g. clicking "Pricing" in the header from another page).
  useEffect(() => {
    if (loc.hash === '#pricing') {
      requestAnimationFrame(() =>
        document.getElementById('pricing')?.scrollIntoView({ behavior: 'smooth' })
      )
    }
  }, [loc.hash])

  return (
    <div>
      {/* Hero */}
      <section className="border-b border-stone-200 bg-white">
        <div className="mx-auto grid max-w-6xl items-center gap-10 px-4 py-16 md:grid-cols-2 md:py-20">
          {/* Left: product pitch */}
          <div className="text-center md:text-left">
            <span className="inline-block rounded-full bg-teal-50 px-3 py-1 text-xs font-medium text-teal-700">
              For accountants &amp; tax professionals
            </span>
            <h1 className="mt-5 text-4xl font-semibold leading-tight text-stone-900 sm:text-5xl">
              How much of your client’s Social Security is taxable?
              <span className="text-teal-700"> Know in seconds.</span>
            </h1>
            <p className="mt-5 text-lg text-stone-600">
              SS Tax Engine runs the full IRS worksheet and every taxing state’s
              rule for you, then shows the reason behind each dollar. Fewer
              mistakes, less time, an answer you can hand to the client.
            </p>
            <div className="mt-8 flex items-center justify-center gap-3 md:justify-start">
              <Link
                to="/register"
                className="rounded-lg bg-teal-700 px-6 py-3 text-base font-medium text-white hover:bg-teal-600"
              >
                Get started free
              </Link>
            </div>
            <p className="mt-4 text-sm text-stone-500">Free plan for up to 3 clients. No card required.</p>
          </div>

          {/* Right: sign-in form */}
          <div className="mx-auto w-full max-w-md md:mx-0 md:ml-auto">
            <LoginForm />
          </div>
        </div>
      </section>

      {/* Problem */}
      <section className="mx-auto max-w-6xl px-4 py-16">
        <div className="grid items-center gap-10 md:grid-cols-2">
          <div>
            <h2 className="text-3xl font-semibold text-stone-900">
              This number is easy to get wrong
            </h2>
            <p className="mt-4 text-stone-600">
              The federal share hinges on “provisional income” — where half the
              benefit and all tax-exempt interest can quietly push a retiree over a
              threshold. Then eight states add rules of their own: some exempt
              benefits below an income cap, some phase the break out, some turn on
              age or full retirement age.
            </p>
            <p className="mt-4 text-stone-600">
              Miss one threshold and the whole figure is off. And the thresholds
              change every year. A calculator that just says “85%” isn’t something
              you can put in front of a client.
            </p>
          </div>
          <div className="rounded-2xl border border-stone-200 bg-white p-6 shadow-sm">
            <p className="text-sm font-medium text-stone-500">States that tax Social Security benefits</p>
            <div className="mt-3 flex flex-wrap gap-2">
              {STATES.map((s) => (
                <span key={s} className="rounded-lg bg-teal-50 px-3 py-1.5 text-sm font-medium text-teal-800">
                  {s}
                </span>
              ))}
            </div>
            <p className="mt-4 text-sm text-stone-500">
              The other 42 states and DC don’t tax it. For those, the engine
              returns $0 and tells you why.
            </p>
          </div>
        </div>
      </section>

      {/* How it works */}
      <section className="border-y border-stone-200 bg-white">
        <div className="mx-auto max-w-6xl px-4 py-16">
          <h2 className="text-center text-3xl font-semibold text-stone-900">How it works</h2>
          <div className="mt-10 grid gap-8 md:grid-cols-3">
            {STEPS.map((s) => (
              <div key={s.n}>
                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-teal-700 font-serif text-lg font-semibold text-white">
                  {s.n}
                </div>
                <h3 className="mt-4 text-xl font-semibold text-stone-900">{s.title}</h3>
                <p className="mt-2 text-stone-600">{s.body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="mx-auto max-w-6xl px-4 py-16">
        <h2 className="text-center text-3xl font-semibold text-stone-900">
          Everything you need to answer with confidence
        </h2>
        <div className="mt-10 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {FEATURES.map((f) => (
            <div key={f.title} className="rounded-xl border border-stone-200 bg-white p-6 shadow-sm">
              <h3 className="text-lg font-semibold text-stone-900">{f.title}</h3>
              <p className="mt-2 text-sm text-stone-600">{f.body}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Pricing */}
      <section id="pricing" className="scroll-mt-20 border-y border-stone-200 bg-white">
        <div className="mx-auto max-w-5xl px-4 py-16">
          <div className="text-center">
            <h2 className="text-3xl font-semibold text-stone-900">Simple pricing</h2>
            <p className="mx-auto mt-3 max-w-xl text-stone-600">
              Start free and upgrade when your client list grows. Final Pro pricing
              is set at checkout; taxes are handled by our Merchant of Record.
            </p>
          </div>

          <div className="mt-12 grid gap-6 md:grid-cols-2">
            {PLANS.map((p) => {
              const to = p.name === 'Pro' ? proTo : p.cta.to
              return (
                <div
                  key={p.name}
                  className={`rounded-2xl border bg-white p-8 shadow-sm ${
                    p.highlight ? 'border-2 border-teal-600' : 'border-stone-200'
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <h3 className="text-2xl font-semibold text-stone-900">{p.name}</h3>
                    {p.highlight && (
                      <span className="rounded-full bg-teal-50 px-3 py-1 text-xs font-medium text-teal-700">
                        Most popular
                      </span>
                    )}
                  </div>
                  <p className="mt-4">
                    <span className="text-4xl font-semibold text-stone-900">{p.price}</span>{' '}
                    <span className="text-stone-500">{p.cadence}</span>
                  </p>
                  <p className="mt-2 text-sm text-stone-600">{p.blurb}</p>

                  <Link
                    to={to}
                    className={`mt-6 block rounded-lg px-4 py-2.5 text-center text-sm font-medium ${
                      p.cta.primary
                        ? 'bg-teal-700 text-white hover:bg-teal-600'
                        : 'border border-stone-300 text-stone-700 hover:bg-stone-100'
                    }`}
                  >
                    {p.cta.label}
                  </Link>

                  <ul className="mt-6 space-y-2.5">
                    {p.features.map((f) => (
                      <li key={f} className="flex gap-2 text-sm text-stone-700">
                        <span className="text-teal-600">✓</span>
                        <span>{f}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              )
            })}
          </div>

          <p className="mx-auto mt-10 max-w-2xl text-center text-xs text-stone-500">
            SS Tax Engine produces estimates to assist a tax professional — not
            filing advice. Every threshold and rate must be verified against
            official IRS and state sources for the tax year in question.
          </p>
        </div>
      </section>

      {/* Final CTA */}
      <section className="mx-auto max-w-4xl px-4 py-20 text-center">
        <h2 className="text-3xl font-semibold text-stone-900">Give your client a number you can stand behind</h2>
        <p className="mx-auto mt-4 max-w-xl text-stone-600">
          Create a free account and run your first calculation in under a minute.
        </p>
        <Link
          to="/register"
          className="mt-8 inline-block rounded-lg bg-teal-700 px-6 py-3 text-base font-medium text-white hover:bg-teal-600"
        >
          Get started free
        </Link>
        <p className="mt-6 text-xs text-stone-500">
          SS Tax Engine produces estimates to assist a professional — not filing
          advice. Verify every threshold against official IRS and state sources.
        </p>
      </section>
    </div>
  )
}
