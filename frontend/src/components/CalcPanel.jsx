import { useState } from 'react'
import { api } from '../api/client'
import { useAuth } from '../App.jsx'
import {
  Card, Field, TextInput, Select, Button, ResultView, PlanLimitNotice, ProBadge,
  FILING_STATUSES, US_STATES, CURRENT_TAX_YEAR,
} from './ui.jsx'

const defaults = {
  filing_status: 'single', state: 'CO', age: 67, at_fra: true,
  ss_benefits: 24000, other_income: 40000, tax_exempt_interest: 0,
  tax_year: 2026,
}

// The core calculator. Wires inputs to POST /api/calculate — all math happens
// on the backend, which is the source of truth. When clientId is provided it
// also offers "Save scenario".
export default function CalcPanel({ initial, clientId, onScenarioSaved, atScenarioLimit = false }) {
  const { auth } = useAuth()
  const [form, setForm] = useState({ ...defaults, ...(initial || {}) })
  const [result, setResult] = useState(null)
  const [error, setError] = useState(null)
  const [busy, setBusy] = useState(false)
  const [label, setLabel] = useState('')
  const [savedMsg, setSavedMsg] = useState('')

  const isFree = auth?.plan !== 'pro'
  // Multi-year what-if is Pro-only; Free is limited to the current tax year.
  const multiYearBlocked = isFree && Number(form.tax_year) !== CURRENT_TAX_YEAR

  const set = (k, numeric = false) => (e) => {
    const v = e.target.type === 'checkbox' ? e.target.checked
      : numeric ? Number(e.target.value) : e.target.value
    setForm({ ...form, [k]: v })
  }

  async function calculate(e) {
    e.preventDefault()
    setError(null); setSavedMsg(''); setBusy(true)
    try {
      setResult(await api.calculate(form))
    } catch (err) {
      setError(err)
    } finally {
      setBusy(false)
    }
  }

  async function saveScenario() {
    setError(null); setSavedMsg('')
    try {
      await api.createScenario(clientId, { label: label || 'Scenario', inputs: form })
      setLabel(''); setSavedMsg('Scenario saved.')
      onScenarioSaved?.()
    } catch (err) {
      setError(err)
    }
  }

  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <Card title="Inputs">
        <form onSubmit={calculate} className="grid gap-4 sm:grid-cols-2">
          <Field label="Filing status">
            <Select value={form.filing_status} onChange={set('filing_status')}>
              {FILING_STATUSES.map((f) => <option key={f.value} value={f.value}>{f.label}</option>)}
            </Select>
          </Field>
          <Field label="State">
            <Select value={form.state} onChange={set('state')}>
              <option value="">— none —</option>
              {US_STATES.map((s) => <option key={s} value={s}>{s}</option>)}
            </Select>
          </Field>
          <Field label="Age">
            <TextInput type="number" min="0" value={form.age} onChange={set('age', true)} />
          </Field>
          <Field
            label={<span className="inline-flex items-center gap-1.5">Tax year {isFree && <ProBadge />}</span>}
            hint={isFree ? `Free plan: ${CURRENT_TAX_YEAR} only. Upgrade to Pro for multi-year planning.` : undefined}
          >
            <TextInput type="number" value={form.tax_year} onChange={set('tax_year', true)} />
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
            <Button type="submit" disabled={busy}>{busy ? 'Calculating…' : 'Calculate'}</Button>
            {multiYearBlocked && (
              <span className="ml-3 text-xs text-amber-700">
                Tax year {form.tax_year} needs Pro.
              </span>
            )}
          </div>
        </form>

        {clientId && result && (
          <div className="mt-4 border-t border-stone-100 pt-4">
            {isFree && atScenarioLimit ? (
              <PlanLimitNotice
                error={{
                  upgrade: true,
                  message: 'The Free plan allows one saved scenario per client. Upgrade to Pro for unlimited scenarios.',
                }}
              />
            ) : (
              <div className="flex items-end gap-2">
                <Field label="Save this scenario">
                  <TextInput value={label} onChange={(e) => setLabel(e.target.value)} placeholder="e.g. Baseline 2026" />
                </Field>
                <Button variant="ghost" type="button" onClick={saveScenario}>Save</Button>
              </div>
            )}
            {savedMsg && <p className="mt-2 text-sm text-teal-700">{savedMsg}</p>}
          </div>
        )}
      </Card>

      <div>
        {result ? (
          <ResultView result={result} />
        ) : (
          <Card>
            <p className="text-stone-500">Enter the client's income details and press Calculate.</p>
          </Card>
        )}
      </div>
    </div>
  )
}
