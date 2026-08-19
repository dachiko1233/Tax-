import { useEffect, useRef, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { api } from '../api/client'
import { Card } from '../components/ui.jsx'

export default function VerifyEmail() {
  const [params] = useSearchParams()
  const token = params.get('token')
  const [state, setState] = useState('verifying') // verifying | ok | error
  const [message, setMessage] = useState('')
  const ran = useRef(false)

  useEffect(() => {
    if (ran.current) return // guard against React 18 StrictMode double-invoke
    ran.current = true
    if (!token) {
      setState('error')
      setMessage('This link is missing its verification token.')
      return
    }
    api
      .verifyEmail(token)
      .then((res) => { setState('ok'); setMessage(res.message || 'Email verified.') })
      .catch((err) => { setState('error'); setMessage(err.message || 'Verification failed.') })
  }, [token])

  const ui = {
    verifying: { icon: '⏳', title: 'Verifying…', tone: 'text-stone-600' },
    ok: { icon: '✅', title: 'Email verified', tone: 'text-teal-700' },
    error: { icon: '⚠️', title: 'Verification problem', tone: 'text-red-600' },
  }[state]

  return (
    <div className="mx-auto flex min-h-[70vh] max-w-md items-center px-4">
      <div className="w-full">
        <Card>
          <div className="text-center">
            <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-stone-100 text-2xl">
              {ui.icon}
            </div>
            <h1 className="font-serif text-2xl font-semibold text-teal-900">{ui.title}</h1>
            <p className={`mt-2 text-sm ${ui.tone}`}>{message}</p>
            {state !== 'verifying' && (
              <Link
                to="/"
                className="mt-5 inline-block rounded-lg bg-teal-700 px-4 py-2 text-sm font-medium text-white hover:bg-teal-600"
              >
                Continue to sign in
              </Link>
            )}
          </div>
        </Card>
      </div>
    </div>
  )
}
