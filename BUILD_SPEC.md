# Build Spec — Social Security Multi-State Tax Compliance Engine (SaaS)

> Hand this file to Claude (or any developer) as the build brief. It describes
> **what to build, with which stack, and how to run it.** Follow it section by
> section. Ask before deviating from the stack.

---

## 1. What we are building (plain summary)

A subscription web app (**SaaS**) for **accountants and tax professionals** in the
United States. Their retiree clients receive **Social Security** benefits, and a
portion of those benefits can be taxed — at the **federal** level (IRS provisional
income rules) and, in **8 states**, at the **state** level, each with its own
thresholds that change yearly.

The app takes a client's income details and instantly computes **how much of the
Social Security benefit is taxable, federally and by state**, and **explains the
rule behind every number** (an audit trail). The value is: saved time, fewer
errors, and something the accountant can show or send to their client.

**Business model:** monthly subscription per accountant (e.g. $29–79/month). The
moat is a **versioned tax-rules table** updated every tax year — customers pay to
never have to track rule changes themselves.

There is already a working **React prototype** of the calculation UI
(`SSTaxEngine.jsx`). This spec describes turning that prototype into a **full
product** with a real backend, database, authentication, and saved client records.

---

## 2. Tech stack (required — do not substitute without asking)

| Layer | Technology |
|---|---|
| Frontend | **React** (Vite) + **TailwindCSS** |
| Backend / API | **Go** with the **Gin** web framework |
| Database | **PostgreSQL** |
| Build automation | **Makefile** |
| Containerization | **Docker** + **docker-compose** |
| Auth | JWT-based email/password login |

---

## 3. Repository structure

```
ss-tax-engine/
├── Makefile                  # one-command build / run / test / migrate
├── docker-compose.yml        # spins up postgres + backend + frontend
├── .env.example              # documented environment variables
├── README.md                 # setup + run instructions
│
├── backend/                  # Go + Gin API
│   ├── Dockerfile
│   ├── go.mod
│   ├── main.go               # entrypoint, router setup
│   ├── internal/
│   │   ├── api/              # HTTP handlers (Gin)
│   │   ├── auth/             # JWT, password hashing (bcrypt)
│   │   ├── engine/           # THE TAX RULES ENGINE (federal + per-state)
│   │   ├── models/           # DB structs
│   │   ├── db/               # connection + queries
│   │   └── config/           # env loading
│   └── migrations/           # SQL schema migrations (numbered)
│
└── frontend/                 # React + Tailwind (Vite)
    ├── Dockerfile
    ├── package.json
    ├── tailwind.config.js
    ├── index.html
    └── src/
        ├── main.jsx
        ├── App.jsx
        ├── api/              # fetch wrappers to the backend
        ├── pages/            # Login, Dashboard, ClientDetail, Calculator
        └── components/       # reusable UI (adapt SSTaxEngine.jsx here)
```

---

## 4. The tax engine (the core — build this first and carefully)

Put all tax logic in `backend/internal/engine/`. It must be **pure, testable Go**
with **no HTTP or DB dependencies** — just inputs in, results out. This is the
part that must be correct and unit-tested.

### 4.1 Federal calculation (IRS Publication 915)

Input: filing status, annual SS benefits, other income (AGI excluding SS),
tax-exempt (municipal) interest.

Logic:
- `provisional_income = other_income + tax_exempt_interest + 0.5 * ss_benefits`
- Thresholds: Single/HoH `$25,000` base / `$34,000` upper. MFJ `$32,000` /
  `$44,000`. MFS living together: 85% taxable at any income.
- Below base → **0%** taxable.
- Between base and upper → up to **50%** of benefits taxable.
- Above upper → up to **85%** of benefits taxable.
- Return: provisional income, taxable amount, which tier applied, and a
  human-readable explanation string.

> Note: these federal thresholds are **not inflation-adjusted** (fixed since 1994),
> so hardcoding them for the current year is acceptable — but still store them in
> the versioned rules table (section 4.3) so future law changes are easy.

### 4.2 State calculation (8 states, each its own rule)

Model each taxing state as its own ruleset. Non-taxing states (42 + DC) return 0.
For the **2026 tax year**, implement these (verify current figures against official
state sources at build time — they change yearly):

- **Colorado** — age 65+ get a full subtraction of federally-taxed benefits; under
  65 taxed at flat 4.4%.
- **Connecticut** — fully exempt if AGI under ~$75k (single) / ~$100k (MFJ);
  partial phase-in above.
- **Minnesota** — full subtraction if AGI at/below ~$86,410 (single) / ~$110,780
  (MFJ); phases out above.
- **Montana** — follows federal inclusion; age 65+ get a ~$5,500 subtraction.
- **New Mexico** — exempt if AGI at/below ~$100k (single) / ~$150k (MFJ).
- **Rhode Island** — exempt if at full retirement age AND AGI at/below ~$104,200
  (single) / ~$130,250 (MFJ).
- **Utah** — flat 4.45%; a Social Security credit offsets the tax below AGI caps
  (~$54k single / ~$90k MFJ), phasing out above.
- **Vermont** — full exemption if AGI at/below ~$65k (single) / ~$80k (MFJ);
  partial band above; no exemption past the upper cap.

Each state rule returns: state-taxable amount, applicable rate, estimated state
tax, and an explanation string.

> **Do not treat these numbers as final.** Confirm each state's current-year
> thresholds and rates from the official state revenue department before shipping.

### 4.3 Versioned rules table (the business moat)

Do **not** hardcode thresholds inline in logic. Store them in a
`tax_rules` table (or a versioned config file loaded at startup), keyed by
`tax_year` (e.g. `2026`) and `jurisdiction` (`FEDERAL`, `CO`, `CT`, ...). The
engine looks up the ruleset for the requested year. This makes the **yearly update**
— the thing customers pay for — a data change, not a code rewrite.

### 4.4 Unit tests (required)

Write table-driven Go tests in `engine/` covering: below/at/above each federal
threshold; each state's exempt / partial / taxable bands; the municipal-interest
effect on provisional income. Aim for known-correct example cases.

---

## 5. Database schema (PostgreSQL)

Write numbered SQL migrations in `backend/migrations/`.

```sql
-- accountants (the paying users)
users (
  id UUID PK, email TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL,
  firm_name TEXT, created_at TIMESTAMPTZ DEFAULT now()
)

-- the accountant's clients
clients (
  id UUID PK, user_id UUID FK -> users(id),
  name TEXT NOT NULL, filing_status TEXT NOT NULL, state TEXT NOT NULL,
  age INT, at_fra BOOLEAN,
  ss_benefits NUMERIC, other_income NUMERIC, tax_exempt_interest NUMERIC,
  created_at TIMESTAMPTZ DEFAULT now(), updated_at TIMESTAMPTZ DEFAULT now()
)

-- saved calculation scenarios per client (for what-if history)
scenarios (
  id UUID PK, client_id UUID FK -> clients(id),
  tax_year INT NOT NULL, label TEXT,
  inputs_json JSONB NOT NULL, results_json JSONB NOT NULL,
  created_at TIMESTAMPTZ DEFAULT now()
)

-- versioned tax rules (the moat)
tax_rules (
  id UUID PK, tax_year INT NOT NULL, jurisdiction TEXT NOT NULL,
  rules_json JSONB NOT NULL, effective_from DATE,
  UNIQUE(tax_year, jurisdiction)
)
```

---

## 6. API endpoints (Gin)

All endpoints except auth require a valid JWT (Authorization: Bearer).

```
POST   /api/auth/register     -> create accountant account
POST   /api/auth/login        -> return JWT

GET    /api/clients           -> list this user's clients
POST   /api/clients           -> create a client
GET    /api/clients/:id       -> get one client
PUT    /api/clients/:id       -> update a client
DELETE /api/clients/:id       -> delete a client

POST   /api/calculate         -> run the engine on given inputs (no save)
                                 body: { filing_status, state, age, at_fra,
                                         ss_benefits, other_income,
                                         tax_exempt_interest, tax_year }
                                 returns: { federal:{...}, state:{...},
                                            combined, explanations:[...] }

POST   /api/clients/:id/scenarios  -> save a scenario for a client
GET    /api/clients/:id/scenarios  -> list saved scenarios
```

Validate all inputs. Never trust client-side numbers for the final calculation —
the engine runs on the backend and is the source of truth.

---

## 7. Frontend (React + Tailwind)

- **Login / Register** page (stores JWT, e.g. in memory + refresh flow).
- **Dashboard**: list of the accountant's clients, "add client" button.
- **Client detail**: the client's saved info + their scenario history.
- **Calculator**: adapt the existing `SSTaxEngine.jsx` prototype — it already has
  the input form, the federal + state result cards, and the "Rule applied"
  explanation panels. Wire its inputs to `POST /api/calculate` instead of doing
  math in the browser, and add a "Save scenario" action.
- Keep the prototype's visual style (clean, professional, serif headings +
  Inter body, teal/stone palette). Tailwind utility classes only.
- Must be responsive down to mobile, with visible keyboard focus states.

---

## 8. Makefile targets (required)

```make
make up            # docker-compose up (postgres + backend + frontend)
make down          # stop everything
make build         # build backend + frontend images
make migrate       # run DB migrations
make seed          # load the tax_rules table for the current tax year
make test          # run Go engine unit tests
make logs          # tail all container logs
make fmt           # format Go + frontend code
```

---

## 9. Docker

- `docker-compose.yml` defines three services: `db` (postgres:16), `backend`
  (Go/Gin), `frontend` (React served via Vite preview or nginx).
- Backend waits for postgres to be healthy before starting.
- Use environment variables from `.env` (documented in `.env.example`):
  `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `DATABASE_URL`,
  `JWT_SECRET`, `PORT`.
- `make up` should bring the whole stack online with one command.

---

## 10. Build order (do it in this sequence)

1. **Engine + tests first.** Get `backend/internal/engine/` correct and fully
   unit-tested before anything else. This is the product's core.
2. **Database + migrations.** Schema from section 5.
3. **Backend API.** Auth, then clients CRUD, then `/api/calculate`, then scenarios.
4. **Docker + Makefile.** Get `make up` working end-to-end locally.
5. **Frontend.** Adapt the prototype, wire it to the API, build the dashboard.
6. **Seed the rules table** for the current tax year and verify against the
   prototype's numbers.

---

## 11. Important correctness & scope notes

- This app produces **estimates to assist a professional**, not filing advice.
  Include a clear disclaimer in the UI.
- **Verify every tax threshold and rate** against official IRS and state revenue
  sources for the target tax year before shipping. The numbers in section 4 are a
  2026 starting point and must be confirmed — they change yearly.
- Keep the engine pure and the rules data-driven, so next year's update is a
  `tax_rules` change plus new tests, not a rewrite.
- Out of scope for v1 (note for later): billing/Stripe, PDF report export,
  multi-user firms, e-file integration. Build these after the core is validated.
```
