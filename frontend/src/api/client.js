// Thin fetch wrapper around the Go backend. The JWT is held in memory and
// mirrored to localStorage so a page refresh stays logged in.

const BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080'

let token = localStorage.getItem('token') || null

export function setToken(t) {
  token = t
  if (t) localStorage.setItem('token', t)
  else localStorage.removeItem('token')
}

export function getToken() {
  return token
}

async function request(method, path, body) {
  const headers = { 'Content-Type': 'application/json' }
  if (token) headers.Authorization = `Bearer ${token}`

  const res = await fetch(`${BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  })

  if (res.status === 204) return null
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    // Plan-limit rejections come back as {error:"plan_limit", message, limit,
    // upgrade}. Surface the human message and carry the structured fields so
    // the UI can show the right upgrade prompt instead of a raw error.
    const isPlanLimit = data.error === 'plan_limit'
    const err = new Error(data.message || data.error || `Request failed (${res.status})`)
    err.status = res.status
    err.code = data.error
    err.limit = data.limit
    err.upgrade = data.upgrade === true || isPlanLimit
    throw err
  }
  return data
}

export const api = {
  register: (body) => request('POST', '/api/auth/register', body),
  login: (body) => request('POST', '/api/auth/login', body),
  verifyEmail: (token) =>
    request('GET', `/api/auth/verify?token=${encodeURIComponent(token)}`),
  resendVerification: (email) => request('POST', '/api/auth/resend', { email }),

  listClients: () => request('GET', '/api/clients'),
  getClient: (id) => request('GET', `/api/clients/${id}`),
  createClient: (body) => request('POST', '/api/clients', body),
  updateClient: (id, body) => request('PUT', `/api/clients/${id}`, body),
  deleteClient: (id) => request('DELETE', `/api/clients/${id}`),

  calculate: (body) => request('POST', '/api/calculate', body),

  createScenario: (clientId, body) =>
    request('POST', `/api/clients/${clientId}/scenarios`, body),
  listScenarios: (clientId) =>
    request('GET', `/api/clients/${clientId}/scenarios`),

  billingStatus: () => request('GET', '/api/billing/status'),
  checkout: () => request('POST', '/api/billing/checkout'),
  billingVerify: () => request('POST', '/api/billing/verify'),
}
