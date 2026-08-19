import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import { useAuth } from '../App.jsx'
import { Card, Button, StatRow, money, FREE_MAX_SCENARIOS_PER_CLIENT } from '../components/ui.jsx'
import { ClientForm } from './Dashboard.jsx'
import CalcPanel from '../components/CalcPanel.jsx'

export default function ClientDetail() {
  const { id } = useParams()
  const nav = useNavigate()
  const { auth } = useAuth()
  const [client, setClient] = useState(null)
  const [scenarios, setScenarios] = useState([])
  const [editing, setEditing] = useState(false)
  const [error, setError] = useState('')

  const isFree = auth?.plan !== 'pro'
  const atScenarioLimit = isFree && scenarios.length >= FREE_MAX_SCENARIOS_PER_CLIENT

  async function load() {
    try {
      const [c, s] = await Promise.all([api.getClient(id), api.listScenarios(id)])
      setClient(c)
      setScenarios(s)
    } catch (e) {
      setError(e.message)
    }
  }
  useEffect(() => { load() }, [id])

  async function remove() {
    if (!confirm('Delete this client and all saved scenarios?')) return
    try {
      await api.deleteClient(id)
      nav('/app')
    } catch (e) {
      setError(e.message)
    }
  }

  if (error) return <p className="mx-auto max-w-6xl px-4 py-8 text-red-600">{error}</p>
  if (!client) return <p className="mx-auto max-w-6xl px-4 py-8 text-stone-500">Loading…</p>

  const calcInitial = {
    filing_status: client.filing_status, state: client.state, age: client.age,
    at_fra: client.at_fra, ss_benefits: client.ss_benefits,
    other_income: client.other_income, tax_exempt_interest: client.tax_exempt_interest,
    tax_year: 2026,
  }

  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <button onClick={() => nav('/app')} className="mb-4 text-sm text-teal-700 hover:underline">
        ← Back to clients
      </button>

      <div className="mb-6 flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-stone-900">{client.name}</h1>
          <p className="text-sm text-stone-600">
            {client.state || 'No state'} · age {client.age}
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="ghost" onClick={() => setEditing((v) => !v)}>
            {editing ? 'Close' : 'Edit'}
          </Button>
          <Button variant="danger" onClick={remove}>Delete</Button>
        </div>
      </div>

      {editing ? (
        <div className="mb-8">
          <ClientForm initial={client} submitLabel="Update client"
            onSaved={() => { setEditing(false); load() }} />
        </div>
      ) : (
        <Card title="Saved details">
          <div className="grid gap-x-8 sm:grid-cols-2">
            <StatRow label="SS benefits" value={money(client.ss_benefits)} />
            <StatRow label="Other income" value={money(client.other_income)} />
            <StatRow label="Tax-exempt interest" value={money(client.tax_exempt_interest)} />
            <StatRow label="At full retirement age" value={client.at_fra ? 'Yes' : 'No'} />
          </div>
        </Card>
      )}

      <h2 className="mb-3 mt-8 text-xl font-semibold text-stone-900">Run a calculation</h2>
      <CalcPanel initial={calcInitial} clientId={id} onScenarioSaved={load} atScenarioLimit={atScenarioLimit} />

      <h2 className="mb-3 mt-10 text-xl font-semibold text-stone-900">Scenario history</h2>
      {scenarios.length === 0 ? (
        <Card><p className="text-stone-500">No saved scenarios yet.</p></Card>
      ) : (
        <div className="space-y-3">
          {scenarios.map((s) => (
            <Card key={s.id}>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <p className="font-medium text-stone-900">{s.label || 'Scenario'}</p>
                  <p className="text-xs text-stone-500">
                    Tax year {s.tax_year} · {new Date(s.created_at).toLocaleString()}
                  </p>
                </div>
                <div className="text-right text-sm">
                  <p className="text-stone-600">
                    Federal taxable: <span className="font-medium">{money(s.results?.federal?.taxable_amount)}</span>
                  </p>
                  <p className="text-stone-600">
                    State tax: <span className="font-medium">{money(s.results?.state?.estimated_tax)}</span>
                  </p>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
