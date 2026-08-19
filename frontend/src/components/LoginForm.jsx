import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import { useAuth } from '../App.jsx'
import { Card, Field, TextInput, Button } from './ui.jsx'

// The sign-in form, extracted so it can live in the landing-page hero. Same
// API call, JWT handling, redirect, and error/unverified display the standalone
// login page used.
export default function LoginForm() {
  const { login } = useAuth()
  const nav = useNavigate()
  const [form, setForm] = useState({ email: '', password: '' })
  const [error, setError] = useState('')
  const [unverified, setUnverified] = useState(false)
  const [resent, setResent] = useState('')
  const [busy, setBusy] = useState(false)

  const set = (k) => (e) => setForm({ ...form, [k]: e.target.value })

  async function submit(e) {
    e.preventDefault()
    setError(''); setUnverified(false); setResent(''); setBusy(true)
    try {
      const res = await api.login(form)
      login({ token: res.token, email: res.email, plan: res.plan })
      nav('/app')
    } catch (err) {
      if (err.status === 403) {
        setUnverified(true)
        setError('Your email address is not verified yet.')
      } else {
        setError(err.message)
      }
    } finally {
      setBusy(false)
    }
  }

  async function resend() {
    setResent('')
    try {
      await api.resendVerification(form.email)
      setResent('If that account exists and is unverified, a new link has been sent.')
    } catch (err) {
      setResent(err.message)
    }
  }

  return (
    <Card>
      <form onSubmit={submit} className="space-y-4">
        <div>
          <h2 className="font-serif text-xl font-semibold text-teal-900">Sign in</h2>
          <p className="text-sm text-stone-600">Access your dashboard and clients.</p>
        </div>

        <Field label="Email">
          <TextInput type="email" value={form.email} onChange={set('email')} required />
        </Field>
        <Field label="Password">
          <TextInput type="password" value={form.password} onChange={set('password')} required />
        </Field>

        {error && <p className="text-sm text-red-600">{error}</p>}
        {unverified && (
          <button type="button" onClick={resend} className="text-sm text-teal-700 underline">
            Resend verification email
          </button>
        )}
        {resent && <p className="text-sm text-teal-700">{resent}</p>}

        <Button type="submit" disabled={busy} className="w-full">
          {busy ? 'Signing in…' : 'Sign in'}
        </Button>

        <p className="text-center text-sm text-stone-600">
          Don't have an account?{' '}
          <Link to="/register" className="font-medium text-teal-700 hover:underline">
            Create one
          </Link>
        </p>
      </form>
    </Card>
  )
}
