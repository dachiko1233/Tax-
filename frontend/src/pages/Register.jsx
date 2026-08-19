import { useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import { Card, Field, TextInput, Button } from '../components/ui.jsx'

export default function Register() {
  const [form, setForm] = useState({ email: '', password: '', firm_name: '' })
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [done, setDone] = useState(null) // { email, dev_verify_link? }

  const set = (k) => (e) => setForm({ ...form, [k]: e.target.value })

  async function submit(e) {
    e.preventDefault()
    setError(''); setBusy(true)
    try {
      const res = await api.register(form)
      setDone({ email: res.email, devLink: res.dev_verify_link })
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  if (done) {
    return (
      <div className="mx-auto flex min-h-[70vh] max-w-md items-center px-4">
        <Card>
          <div className="text-center">
            <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-teal-50 text-2xl">
              ✉️
            </div>
            <h1 className="font-serif text-2xl font-semibold text-teal-900">Check your email</h1>
            <p className="mt-2 text-sm text-stone-600">
              We sent a verification link to <span className="font-medium">{done.email}</span>.
              Click it to activate your account, then sign in.
            </p>
            {done.devLink && (
              <div className="mt-4 rounded-lg border border-amber-200 bg-amber-50 p-3 text-left text-xs text-amber-800">
                <p className="mb-1 font-semibold">Developer mode (no email provider configured):</p>
                <a href={done.devLink} className="break-all font-mono text-teal-700 underline">
                  {done.devLink}
                </a>
              </div>
            )}
            <Link
              to="/"
              className="mt-5 inline-block rounded-lg bg-teal-700 px-4 py-2 text-sm font-medium text-white hover:bg-teal-600"
            >
              Go to sign in
            </Link>
          </div>
        </Card>
      </div>
    )
  }

  return (
    <div className="mx-auto flex min-h-[70vh] max-w-md items-center px-4">
      <div className="w-full">
        <div className="mb-6 text-center">
          <h1 className="font-serif text-3xl font-semibold text-teal-900">Create your account</h1>
          <p className="mt-1 text-sm text-stone-600">
            Start computing Social Security taxability in minutes. No card required.
          </p>
        </div>
        <Card>
          <form onSubmit={submit} className="space-y-4">
            <Field label="Work email">
              <TextInput type="email" value={form.email} onChange={set('email')} required />
            </Field>
            <Field label="Password" hint="Minimum 8 characters.">
              <TextInput type="password" value={form.password} onChange={set('password')} required />
            </Field>
            <Field label="Firm name (optional)">
              <TextInput value={form.firm_name} onChange={set('firm_name')} />
            </Field>

            {error && <p className="text-sm text-red-600">{error}</p>}

            <Button type="submit" disabled={busy} className="w-full">
              {busy ? 'Creating account…' : 'Create account'}
            </Button>
          </form>
        </Card>
        <p className="mt-4 text-center text-sm text-stone-600">
          Already have an account?{' '}
          <Link to="/" className="font-medium text-teal-700 hover:underline">
            Sign in
          </Link>
        </p>
      </div>
    </div>
  )
}
