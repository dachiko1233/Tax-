# SS Tax Engine — Social Security Multi-State Tax Compliance (SaaS)

A subscription web app for accountants that computes how much of a retiree
client's **Social Security benefit is taxable**, federally (IRS Publication 915)
and in the **8 states** that tax it (CO, CT, MN, MT, NM, RI, UT, VT), and
**explains the rule behind every number** as an audit trail.

- **Frontend:** React (Vite) + TailwindCSS — public marketing site + authenticated app
- **Backend / API:** Go + Gin
- **Database:** PostgreSQL (numbered SQL migrations)
- **Auth:** JWT (email/password, bcrypt) with **email verification**
- **Email:** [Resend](https://resend.com) for transactional email
- **Billing:** [Dodo Payments](https://dodopayments.com) — hosted checkout + signature-verified webhooks (a Merchant of Record, so it handles sales tax / VAT)
- **Ops:** Docker Compose + Makefile

## Quick start

```bash
cp .env.example .env          # then edit JWT_SECRET (email/billing keys optional)
make up                       # builds + starts db, backend, frontend
# backend auto-runs: migrate -> seed -> serve
```

- Frontend: http://localhost:5174
- API: http://localhost:8080  (health: `GET /healthz`)

> Host ports are mapped to 5174 (frontend) and 5433 (Postgres) in
> `docker-compose.yml` to avoid clashing with a local Vite/Postgres already on
> 5173/5432. Change them there if you prefer the defaults.

**First run, no external keys needed.** With `RESEND_API_KEY` and the Dodo vars
empty, the app runs in a dev-friendly mode:

- **Email verification** links are logged to the backend console (`make logs`)
  *and* returned by the register call as `dev_verify_link` (shown on the
  "check your email" screen), so you can click through without an email provider.
- **Billing** checkout is disabled; the Billing page's **Reconcile subscription**
  button grants Pro locally so you can exercise the Pro experience without paying.

Typical local flow: **Get started → register → click the dev verify link →
sign in → add a client → open it → run a calculation → Save scenario**. Visit
**Billing → Reconcile** to unlock Pro limits.

### Makefile targets

| Target | Action |
|---|---|
| `make up` | Build + start the whole stack (one command) |
| `make down` | Stop everything |
| `make build` | Build backend + frontend images |
| `make migrate` | Run DB migrations |
| `make seed` | Load the `tax_rules` table for the current tax year |
| `make test` | Run the Go unit tests (engine + webhook verification) |
| `make logs` | Tail all container logs |
| `make fmt` | Format Go + frontend code |

## The tax engine

All tax logic lives in `backend/internal/engine/` as **pure, unit-tested Go**
with no HTTP or DB dependencies (inputs in, results out).

- `federal.go` — the full IRS Pub. 915 Worksheet 1 (not the simplified 50/85%
  summary), so numbers match a real return.
- `state.go` — the 8 taxing states, each as a tagged rule *kind* whose
  thresholds/rates are pure data.
- `rules.go` — **versioned rulesets** keyed by tax year. This is the business
  moat: the yearly update is a data edit, seeded into the `tax_rules` table.

```bash
cd backend && go test ./...
```

> ⚠️ **Verify before shipping.** The federal thresholds are statutory (fixed
> since 1994). The **state** thresholds and the representative per-state rates in
> `rules.go` are a **2026 starting point** and must be confirmed against each
> state revenue department for the target tax year — they change yearly. They are
> data-driven precisely so that confirmation is a config change, not a rewrite.

## Accounts, plans & entitlement

Two plans, **enforced server-side** (never by hiding UI alone):

| | Free ($0) | Pro (~$39/mo) |
|---|---|---|
| Saved clients | 3 | Unlimited |
| Scenarios per client | 1 | Unlimited |
| Tax years | Current year only | All available years |
| Multi-year what-if | — | ✓ |
| Support | Community | Priority |

The user's `plan` is kept current by Dodo webhooks (the source of truth). The
handlers read it and reject over-limit actions with `402 Payment Required` and
`{"upgrade": true}`, which the UI turns into an upgrade prompt.

## Email verification (Resend)

- Register creates the user with `email_verified = false`, bcrypts the password,
  generates a random token, stores **only its SHA-256 hash** with a 24h expiry,
  and emails a link to `${APP_BASE_URL}/verify-email?token=…` via Resend.
- `GET /api/auth/verify?token=…` atomically validates + consumes the token and
  sets `email_verified = true`.
- **Login is rejected (403) until the email is verified.**
- `POST /api/auth/resend` re-issues the email, rate-limited to 3/hour/user.

Client + templates live in `backend/internal/email/`.

**Setup for real delivery:** create an API key at Resend, set `RESEND_API_KEY`,
and set `EMAIL_FROM`. For production the sender domain must be **verified in
Resend (SPF/DKIM)**; `onboarding@resend.dev` works for local testing.

## Billing (Dodo Payments)

Code in `backend/internal/billing/` (client + Standard-Webhooks HMAC verifier).

| Endpoint | Purpose |
|---|---|
| `POST /api/billing/checkout` | Create a Dodo hosted checkout for the Pro product (user id/email attached as metadata); returns `checkout_url` to redirect to. `success_url` → `/billing?status=success`. |
| `POST /api/webhooks/dodo` | Receive webhooks. **Signature is verified before anything is trusted.** Handles `subscription.active/renewed/on_hold/failed/updated`. Webhooks are the source of truth for entitlement, not the browser redirect. |
| `GET /api/billing/status` | Return the user's plan + subscription state. |
| `POST /api/billing/verify` | Manual reconcile for local dev (localhost webhooks are often blocked). |

**Env:** `DODO_PAYMENTS_API_KEY`, `DODO_PAYMENTS_WEBHOOK_SECRET`,
`DODO_PRO_PRODUCT_ID`, `DODO_ENV` (`test_mode` | `live_mode`).

**Setup for test mode:**
1. In the Dodo dashboard (test mode), create a **Pro product** (recurring,
   ~$39/mo). Copy its id into `DODO_PRO_PRODUCT_ID`.
2. Create an **API key** → `DODO_PAYMENTS_API_KEY`.
3. Create a **webhook endpoint** pointing at `${public}/api/webhooks/dodo` and
   copy its signing secret into `DODO_PAYMENTS_WEBHOOK_SECRET`. For local testing,
   expose port 8080 with a tunnel (e.g. ngrok/cloudflared), or rely on
   `POST /api/billing/verify` to reconcile.

## API

All endpoints except `/api/auth/*` and `/api/webhooks/*` require
`Authorization: Bearer <jwt>`.

```
POST   /api/auth/register             # creates unverified user, sends verify email
POST   /api/auth/login                # rejected until email verified
GET    /api/auth/verify?token=...     # consume token, mark verified
POST   /api/auth/resend               # rate-limited resend of verification email

GET    /api/clients
POST   /api/clients                   # Free: capped at 3
GET    /api/clients/:id
PUT    /api/clients/:id
DELETE /api/clients/:id

POST   /api/calculate                 # run engine, no save (Free: current year only)
POST   /api/clients/:id/scenarios     # save a scenario (Free: 1/client, current year)
GET    /api/clients/:id/scenarios

POST   /api/billing/checkout          # -> { checkout_url }
GET    /api/billing/status            # -> { plan, subscription, billing_configured }
POST   /api/billing/verify            # manual reconcile (dev)
POST   /api/webhooks/dodo             # Dodo -> signature verified, source of truth
```

The engine always runs on the backend — client-side numbers are never trusted.

## Security notes

- Passwords are bcrypt-hashed; JWTs are HS256-signed.
- Email-verification tokens are random and stored **only as SHA-256 hashes**,
  single-use, with expiry.
- Dodo webhooks are authenticated with an HMAC-SHA256 signature (Standard
  Webhooks scheme) and a timestamp-tolerance check before any state is trusted;
  unsigned/unknown-secret requests are refused.
- Login and resend-verification are rate-limited.

## Disclaimer

This app produces **estimates to assist a professional**, not filing advice.
Verify every threshold and rate against official IRS and state sources.

## Out of scope for v1

PDF export, multi-user firms, e-file integration.
