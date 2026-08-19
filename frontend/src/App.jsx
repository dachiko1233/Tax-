import { createContext, useContext, useState } from 'react'
import { Routes, Route, Navigate, Link, NavLink, useNavigate, useLocation } from 'react-router-dom'
import { setToken, getToken } from './api/client'
import Landing from './pages/Landing.jsx'
import Register from './pages/Register.jsx'
import VerifyEmail from './pages/VerifyEmail.jsx'
import Dashboard from './pages/Dashboard.jsx'
import ClientDetail from './pages/ClientDetail.jsx'
import Calculator from './pages/Calculator.jsx'
import Billing from './pages/Billing.jsx'

const AuthCtx = createContext(null)
export const useAuth = () => useContext(AuthCtx)

export default function App() {
  const [auth, setAuth] = useState(() => {
    const t = getToken()
    return t
      ? {
          token: t,
          email: localStorage.getItem('email') || '',
          plan: localStorage.getItem('plan') || 'free',
        }
      : null
  })

  function login({ token, email, plan }) {
    setToken(token)
    localStorage.setItem('email', email || '')
    localStorage.setItem('plan', plan || 'free')
    setAuth({ token, email, plan: plan || 'free' })
  }
  function logout() {
    setToken(null)
    localStorage.removeItem('email')
    localStorage.removeItem('plan')
    setAuth(null)
  }
  function setPlan(plan) {
    localStorage.setItem('plan', plan)
    setAuth((a) => (a ? { ...a, plan } : a))
  }

  return (
    <AuthCtx.Provider value={{ auth, login, logout, setPlan }}>
      <Routes>
        <Route path="/" element={<PublicLayout><Landing /></PublicLayout>} />
        <Route
          path="/register"
          element={auth ? <Navigate to="/app" /> : <PublicLayout><Register /></PublicLayout>}
        />
        <Route path="/verify-email" element={<PublicLayout><VerifyEmail /></PublicLayout>} />

        <Route path="/app" element={<Protected><AppLayout><Dashboard /></AppLayout></Protected>} />
        <Route path="/app/calculator" element={<Protected><AppLayout><Calculator /></AppLayout></Protected>} />
        <Route path="/app/clients/:id" element={<Protected><AppLayout><ClientDetail /></AppLayout></Protected>} />
        <Route path="/app/billing" element={<Protected><AppLayout><Billing /></AppLayout></Protected>} />

        <Route path="*" element={<Navigate to="/" />} />
      </Routes>
    </AuthCtx.Provider>
  )
}

function Protected({ children }) {
  const { auth } = useAuth()
  return auth ? children : <Navigate to="/" replace />
}

function PublicLayout({ children }) {
  return (
    <div className="flex min-h-screen flex-col bg-stone-50">
      <PublicNav />
      <main className="flex-1">{children}</main>
      <Footer />
    </div>
  )
}

function AppLayout({ children }) {
  return (
    <div className="flex min-h-screen flex-col bg-stone-50">
      <AppNav />
      <main className="flex-1">{children}</main>
      <Footer />
    </div>
  )
}

function PublicNav() {
  const loc = useLocation()
  const nav = useNavigate()
  const { auth } = useAuth()

  // "Pricing" smooth-scrolls to the pricing section on the landing page. From
  // any other page, navigate home with #pricing so the landing page scrolls
  // once it mounts.
  function onPricing(e) {
    e.preventDefault()
    if (loc.pathname === '/') {
      document.getElementById('pricing')?.scrollIntoView({ behavior: 'smooth' })
      window.history.replaceState(null, '', '/#pricing')
    } else {
      nav('/#pricing')
    }
  }

  return (
    <header className="border-b border-stone-200 bg-white">
      <div className="mx-auto flex max-w-6xl items-center justify-between gap-3 px-4 py-3">
        <Link to="/" className="shrink-0 truncate font-serif text-base font-semibold tracking-tight text-teal-900 sm:text-lg">
          SS Tax Engine
        </Link>
        <nav className="flex shrink-0 items-center gap-3 text-sm sm:gap-5">
          <a href="/#pricing" onClick={onPricing} className="text-stone-600 hover:text-stone-900">
            Pricing
          </a>
          {auth ? (
            <Link
              to="/app"
              className="rounded-lg bg-teal-700 px-3 py-1.5 font-medium text-white hover:bg-teal-600 sm:px-3.5"
            >
              Go to app
            </Link>
          ) : (
            <Link
              to="/register"
              className="rounded-lg bg-teal-700 px-3 py-1.5 font-medium text-white hover:bg-teal-600 sm:px-3.5"
            >
              Get started
            </Link>
          )}
        </nav>
      </div>
    </header>
  )
}

function AppNav() {
  const { auth, logout } = useAuth()
  const nav = useNavigate()
  const isPro = auth.plan === 'pro'
  const linkCls = ({ isActive }) =>
    `rounded-lg px-3 py-1.5 text-sm font-medium transition ${
      isActive
        ? 'bg-white/15 text-white shadow-inner'
        : 'text-teal-100/80 hover:bg-white/10 hover:text-white'
    }`
  return (
    <header className="sticky top-0 z-20 border-b border-teal-800/60 bg-gradient-to-r from-teal-950 to-teal-900 text-teal-50 shadow-sm backdrop-blur">
      <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-2.5">
        <Link to="/app" className="flex items-center gap-2.5">
          <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-white/15 font-serif text-sm font-bold tracking-tight text-white ring-1 ring-white/20">
            SS
          </span>
          <span className="font-serif text-lg font-semibold tracking-tight">Tax Engine</span>
        </Link>

        <nav className="flex flex-1 items-center gap-1">
          <NavLink end to="/app" className={linkCls}>Clients</NavLink>
          <NavLink to="/app/calculator" className={linkCls}>Calculator</NavLink>
        </nav>

        <div className="flex items-center gap-2">
          <Link
            to="/app/billing"
            title="Billing & plan"
            className="flex items-center gap-2 rounded-full border border-white/15 bg-white/5 py-1 pl-1.5 pr-1.5 text-sm transition hover:bg-white/10 sm:pl-3"
          >
            <span className="hidden max-w-[14ch] truncate text-teal-100/90 sm:inline">{auth.email}</span>
            <span
              className={`rounded-full px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide ${
                isPro
                  ? 'bg-amber-400 text-amber-950'
                  : 'bg-white/15 text-teal-50'
              }`}
            >
              {isPro ? 'Pro' : 'Free'}
            </span>
          </Link>
          <button
            onClick={() => { logout(); nav('/') }}
            className="rounded-lg border border-white/15 px-3 py-1.5 text-sm font-medium text-teal-50 transition hover:bg-white/10"
          >
            Sign out
          </button>
        </div>
      </div>
    </header>
  )
}

function Footer() {
  return (
    <footer className="border-t border-stone-200 bg-white">
      <div className="mx-auto max-w-6xl px-4 py-4 text-xs text-stone-500">
        Estimates to assist a tax professional — not filing advice. Verify all
        figures against official IRS and state sources for the tax year.
      </div>
    </footer>
  )
}
