import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import { useAuth } from '../App.jsx'
import {
  Card, Field, TextInput, Select, Button, money, PlanLimitNotice,
  FILING_STATUSES, US_STATES, FREE_MAX_CLIENTS,
} from '../components/ui.jsx'

const emptyClient = {
  name: '', filing_status: 'single', state: '', age: 0, at_fra: false,
  ss_benefits: 0, other_income: 0, tax_exempt_interest: 0,
}

export default function Dashboard() {
  const { auth } = useAuth()
  const [clients, setClients] = useState([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [error, setError] = useState('')

  const isFree = auth?.plan !== 'pro'
  const atClientLimit = isFree && clients.length >= FREE_MAX_CLIENTS

  async function load() {
    setLoading(true)
    try {
      setClients(await api.listClients())
    } catch (e) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }
  useEffect(() => { load() }, [])

  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-stone-900">Clients</h1>
          <p className="text-sm text-stone-600">Your retiree clients and their saved details.</p>
        </div>
        <Button
          onClick={() => setShowForm((v) => !v)}
          disabled={atClientLimit && !showForm}
          title={atClientLimit ? `Free plan is limited to ${FREE_MAX_CLIENTS} clients` : undefined}
        >
          {showForm ? 'Close' : '+ Add client'}
        </Button>
      </div>

      {error && <p className="mb-4 text-sm text-red-600">{error}</p>}

      {atClientLimit && !showForm && (
        <PlanLimitNotice
          className="mb-6"
          error={{
            upgrade: true,
            message: `You've reached the Free plan limit of ${FREE_MAX_CLIENTS} clients. Upgrade to Pro for unlimited clients.`,
          }}
        />
      )}

      {showForm && (
        <div className="mb-6">
          <ClientForm
            onSaved={() => { setShowForm(false); load() }}
          />
        </div>
      )}

      {loading ? (
        <p className="text-stone-500">Loading…</p>
      ) : clients.length === 0 ? (
        <Card>
          <div className="flex flex-col items-start gap-3">
            <p className="text-stone-600">No clients yet. Add your first client to get started.</p>
            <Button onClick={() => setShowForm((v) => !v)}>
              {showForm ? 'Close' : 'Create one'}
            </Button>
          </div>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {clients.map((c) => (
            <Link key={c.id} to={`/app/clients/${c.id}`}>
              <Card>
                <div className="flex items-start justify-between">
                  <h3 className="text-lg font-semibold text-stone-900">{c.name}</h3>
                  <span className="rounded bg-teal-50 px-2 py-0.5 text-xs font-medium text-teal-700">
                    {c.state || '—'}
                  </span>
                </div>
                <p className="mt-1 text-sm text-stone-500">
                  {FILING_STATUSES.find((f) => f.value === c.filing_status)?.label}
                </p>
                <div className="mt-3 text-sm text-stone-600">
                  SS benefits: <span className="font-medium">{money(c.ss_benefits)}</span>
                </div>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}

export function ClientForm({ initial, onSaved, submitLabel = 'Save client' }) {
  const [form, setForm] = useState(initial || emptyClient)
  const [error, setError] = useState(null)
  const [busy, setBusy] = useState(false)

  const set = (k, numeric = false) => (e) => {
    const v = e.target.type === 'checkbox' ? e.target.checked
      : numeric ? Number(e.target.value) : e.target.value
    setForm({ ...form, [k]: v })
  }

  async function submit(e) {
    e.preventDefault()
    setError(null); setBusy(true)
    try {
      if (initial?.id) await api.updateClient(initial.id, form)
      else await api.createClient(form)
      onSaved?.()
    } catch (err) {
      setError(err)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Card title={initial?.id ? 'Edit client' : 'New client'}>
      <form onSubmit={submit} className="grid gap-4 sm:grid-cols-2">
        <Field label="Name"><TextInput value={form.name} onChange={set('name')} required /></Field>
        <Field label="Filing status">
          <Select value={form.filing_status} onChange={set('filing_status')}>
            {FILING_STATUSES.map((f) => <option key={f.value} value={f.value}>{f.label}</option>)}
          </Select>
        </Field>
        <Field label="State of residence">
          <Select value={form.state} onChange={set('state')}>
            <option value="">— none —</option>
            {US_STATES.map((s) => <option key={s} value={s}>{s}</option>)}
          </Select>
        </Field>
        <Field label="Age">
          <TextInput type="number" min="0" value={form.age} onChange={set('age', true)} />
        </Field>
        <Field label="SS benefits (annual)">
          <TextInput type="number" min="0" step="100" value={form.ss_benefits} onChange={set('ss_benefits', true)} />
        </Field>
        <Field label="Other income (AGI excl. SS)">
          <TextInput type="number" min="0" step="100" value={form.other_income} onChange={set('other_income', true)} />
        </Field>
        <Field label="Tax-exempt interest">
          <TextInput type="number" min="0" step="100" value={form.tax_exempt_interest} onChange={set('tax_exempt_interest', true)} />
        </Field>
        <label className="flex items-center gap-2 self-end pb-2 text-sm text-stone-700">
          <input type="checkbox" checked={form.at_fra} onChange={set('at_fra')} className="h-4 w-4 rounded border-stone-300 text-teal-600" />
          At full retirement age
        </label>

        {error?.upgrade ? (
          <div className="sm:col-span-2"><PlanLimitNotice error={error} /></div>
        ) : error ? (
          <p className="text-sm text-red-600 sm:col-span-2">{error.message}</p>
        ) : null}
        <div className="sm:col-span-2">
          <Button type="submit" disabled={busy}>{busy ? 'Saving…' : submitLabel}</Button>
        </div>
      </form>
    </Card>
  )
}
