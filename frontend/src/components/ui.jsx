// Small shared UI primitives + domain constants, styled in the teal/stone palette.

import { Link } from 'react-router-dom'

// Free-plan limits — mirror the backend (internal/api/server.go). The backend is
// the real gate; these just let the UI annotate limits before the user hits them.
export const CURRENT_TAX_YEAR = 2026
export const FREE_MAX_CLIENTS = 3
export const FREE_MAX_SCENARIOS_PER_CLIENT = 1

export const FILING_STATUSES = [
  { value: 'single', label: 'Single' },
  { value: 'hoh', label: 'Head of Household' },
  { value: 'mfj', label: 'Married Filing Jointly' },
  { value: 'mfs_apart', label: 'Married Filing Separately (lived apart)' },
  { value: 'mfs_together', label: 'Married Filing Separately (lived together)' },
]

// The 8 states that tax Social Security + a "none" sentinel. Everything else
// returns $0 from the backend.
export const TAXING_STATES = ['CO', 'CT', 'MN', 'MT', 'NM', 'RI', 'UT', 'VT']
export const US_STATES = [
  'AL','AK','AZ','AR','CA','CO','CT','DE','FL','GA','HI','ID','IL','IN','IA','KS',
  'KY','LA','ME','MD','MA','MI','MN','MS','MO','MT','NE','NV','NH','NJ','NM','NY',
  'NC','ND','OH','OK','OR','PA','RI','SC','SD','TN','TX','UT','VT','VA','WA','WV',
  'WI','WY','DC',
]

export const money = (n) =>
  (n ?? 0).toLocaleString('en-US', { style: 'currency', currency: 'USD' })

export function Card({ title, children, accent }) {
  return (
    <div className={`rounded-xl border bg-white shadow-sm ${accent ? 'border-teal-300' : 'border-stone-200'}`}>
      {title && (
        <div className="border-b border-stone-100 px-5 py-3">
          <h3 className="text-base font-semibold text-stone-800">{title}</h3>
        </div>
      )}
      <div className="p-5">{children}</div>
    </div>
  )
}

// PlanLimitNotice renders the friendly upgrade prompt shown when the backend
// rejects a write with a plan_limit error. Pass the caught error (which carries
// `.upgrade` and `.message` from the api client) — it only renders for those.
export function PlanLimitNotice({ error, className = '' }) {
  if (!error?.upgrade) return null
  return (
    <div className={`rounded-lg border border-amber-200 bg-amber-50 p-4 ${className}`}>
      <p className="text-sm text-amber-900">{error.message}</p>
      <Link
        to="/app/billing"
        className="mt-3 inline-block rounded-lg bg-teal-700 px-4 py-2 text-sm font-medium text-white transition hover:bg-teal-600"
      >
        Upgrade to Pro
      </Link>
    </div>
  )
}

// ProBadge marks a feature as Pro-only for Free users.
export function ProBadge({ className = '' }) {
  return (
    <span className={`rounded-full bg-amber-100 px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide text-amber-800 ${className}`}>
      Pro
    </span>
  )
}

export function Field({ label, children, hint }) {
  return (
    <label className="block">
      <span className="mb-1 block text-sm font-medium text-stone-700">{label}</span>
      {children}
      {hint && <span className="mt-1 block text-xs text-stone-500">{hint}</span>}
    </label>
  )
}

const inputCls =
  'w-full rounded-lg border border-stone-300 px-3 py-2 text-stone-800 ' +
  'focus:border-teal-500 focus:ring-1 focus:ring-teal-500'

export function TextInput(props) {
  return <input className={inputCls} {...props} />
}
export function Select({ children, ...props }) {
  return <select className={inputCls} {...props}>{children}</select>
}

export function Button({ variant = 'primary', className = '', ...props }) {
  const styles = {
    primary: 'bg-teal-700 text-white hover:bg-teal-600 disabled:opacity-50',
    ghost: 'border border-stone-300 text-stone-700 hover:bg-stone-100',
    danger: 'border border-red-300 text-red-700 hover:bg-red-50',
  }[variant]
  return (
    <button
      className={`rounded-lg px-4 py-2 text-sm font-medium transition ${styles} ${className}`}
      {...props}
    />
  )
}

export function StatRow({ label, value, strong }) {
  return (
    <div className="flex items-baseline justify-between py-1">
      <span className="text-sm text-stone-600">{label}</span>
      <span className={strong ? 'text-lg font-semibold text-stone-900' : 'text-stone-800'}>
        {value}
      </span>
    </div>
  )
}

// Renders the federal + state result cards and the "Rule applied" panels.
export function ResultView({ result }) {
  if (!result) return null
  const { federal, state } = result
  return (
    <div className="space-y-4">
      <div className="grid gap-4 md:grid-cols-2">
        <Card title="Federal (IRS Pub. 915)" accent>
          <StatRow label="Provisional income" value={money(federal.provisional_income)} />
          <StatRow label="Taxable SS benefit" value={money(federal.taxable_amount)} strong />
          <StatRow label="Taxable percent" value={`${federal.taxable_percent ?? 0}%`} />
          <StatRow label="Tier" value={tierLabel(federal.tier)} />
        </Card>

        <Card title={`State${state.state ? ` — ${state.state}` : ''}`}>
          {state.taxable ? (
            <>
              <StatRow label="State-taxable amount" value={money(state.taxable_amount)} />
              <StatRow label="Applicable rate" value={`${(state.rate * 100).toFixed(2)}%`} />
              <StatRow label="Estimated state tax" value={money(state.estimated_tax)} strong />
            </>
          ) : (
            <p className="text-sm text-stone-600">
              {state.explanation || 'This state does not tax Social Security benefits.'}
            </p>
          )}
        </Card>
      </div>

      <Card title="Rule applied — audit trail">
        <ul className="space-y-2">
          {result.explanations.map((e, i) => (
            <li key={i} className="flex gap-2 text-sm text-stone-700">
              <span className="mt-0.5 text-teal-600">▸</span>
              <span>{e}</span>
            </li>
          ))}
        </ul>
      </Card>
    </div>
  )
}

function tierLabel(t) {
  return { none: 'None taxable', up_to_50: 'Up to 50%', up_to_85: 'Up to 85%' }[t] || t
}
