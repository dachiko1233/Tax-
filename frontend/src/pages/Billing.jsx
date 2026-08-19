import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { api } from '../api/client'
import { useAuth } from '../App.jsx'
import { Card, Button, StatRow } from '../components/ui.jsx'

const PRO_FEATURES = [
  'Unlimited clients',
  'Unlimited saved scenarios per client',
  'Multi-year what-if planning',
  'All available tax years',
  'Priority support',
]

export default function Billing() {
  const { setPlan } = useAuth()
  const [params] = useSearchParams()
  const [status, setStatus] = useState(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState('')
  const [notice, setNotice] = useState('')

  // Safe default so the page always renders the Free plan + Upgrade button,
  // even if the live status call fails (e.g. transient network / stale backend).
  const FALLBACK_STATUS = { plan: 'free', subscription: null, billing_configured: false }

  const returnedFromCheckout = params.get('status') === 'success'

  async function load() {
    try {
      const s = await api.billingStatus()
      setStatus(s)
      setPlan(s.plan)
      setLoadError('')
    } catch (e) {
      // Degrade gracefully: a free user with no subscription (or an
      // unreachable billing service) should still see their plan and an
      // Upgrade button, never a bare error page.
      setStatus((prev) => prev || FALLBACK_STATUS)
      setLoadError(
        e.status === 404
          ? 'Live billing status is unavailable on this server build. Showing the Free plan.'
          : 'Could not load live billing status — showing your last-known plan.'
      )
    } finally {
      setLoading(false)
    }
  }
  useEffect(() => { load() }, [])

  async function upgrade() {
    setError(''); setBusy('checkout')
    try {
      const { checkout_url } = await api.checkout()
      window.location.href = checkout_url
    } catch (e) {
      setError(e.message)
      setBusy('')
    }
  }

  async function reconcile() {
    setError(''); setNotice(''); setBusy('verify')
    try {
      const res = await api.billingVerify()
      setPlan(res.plan)
      setNotice(res.message || `Plan is now ${res.plan}.`)
      await load()
    } catch (e) {
      setError(e.message)
    } finally {
      setBusy('')
    }
  }

  if (loading && !status) return <Shell><p className="text-stone-500">Loading…</p></Shell>

  const view = status || FALLBACK_STATUS
  const isPro = view.plan === 'pro'

  return (
    <Shell>
      {loadError && (
        <div className="mb-6 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800">
          {loadError}{' '}
          <button onClick={load} className="font-semibold underline">Retry</button>
        </div>
      )}

      {returnedFromCheckout && !isPro && (
        <div className="mb-6 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800">
          Thanks! If your plan still shows Free, your subscription webhook may not
          have arrived yet (common on localhost). Click{' '}
          <button onClick={reconcile} className="font-semibold underline">reconcile now</button>.
        </div>
      )}

      <div className="grid gap-6 md:grid-cols-2">
        <Card title="Current plan" accent={isPro}>
          <StatRow label="Plan" value={isPro ? 'Pro' : 'Free'} strong />
          {view.subscription && (
            <>
              <StatRow label="Subscription status" value={view.subscription.status} />
              {view.subscription.current_period_end && (
                <StatRow
                  label="Renews / ends"
                  value={new Date(view.subscription.current_period_end).toLocaleDateString()}
                />
              )}
            </>
          )}
          {!view.billing_configured && (
            <p className="mt-3 rounded bg-stone-100 p-2 text-xs text-stone-500">
              Billing provider is not configured on this server. Use “reconcile”
              below to grant Pro locally for development.
            </p>
          )}
        </Card>

        <Card title="Pro — $39/month">
          <ul className="space-y-2">
            {PRO_FEATURES.map((f) => (
              <li key={f} className="flex gap-2 text-sm text-stone-700">
                <span className="text-teal-600">✓</span>
                <span>{f}</span>
              </li>
            ))}
          </ul>
          <div className="mt-5 flex flex-wrap gap-2">
            {isPro ? (
              <span className="rounded-lg bg-teal-50 px-3 py-2 text-sm font-medium text-teal-700">
                You're on Pro — thank you!
              </span>
            ) : (
              <Button onClick={upgrade} disabled={busy === 'checkout'}>
                {busy === 'checkout' ? 'Redirecting…' : 'Upgrade to Pro'}
              </Button>
            )}
            <Button variant="ghost" onClick={reconcile} disabled={busy === 'verify'}>
              {busy === 'verify' ? 'Checking…' : 'Reconcile subscription'}
            </Button>
          </div>
          {notice && <p className="mt-3 text-sm text-teal-700">{notice}</p>}
          {error && <p className="mt-3 text-sm text-red-600">{error}</p>}
        </Card>
      </div>

      <p className="mt-6 text-xs text-stone-500">
        Payments are processed by Dodo Payments, a Merchant of Record that handles
        applicable sales tax and VAT. Final pricing is set in the Dodo dashboard.
      </p>
    </Shell>
  )
}

function Shell({ children }) {
  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-stone-900">Billing &amp; plan</h1>
        <p className="text-sm text-stone-600">Manage your subscription and entitlements.</p>
      </div>
      {children}
    </div>
  )
}
