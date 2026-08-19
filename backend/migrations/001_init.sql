-- 001_init.sql — initial schema for the SS multi-state tax compliance engine.

CREATE EXTENSION IF NOT EXISTS "pgcrypto"; -- for gen_random_uuid()

-- accountants (the paying users)
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    firm_name     TEXT DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- the accountant's clients
CREATE TABLE IF NOT EXISTS clients (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    filing_status       TEXT NOT NULL,
    state               TEXT NOT NULL DEFAULT '',
    age                 INT  NOT NULL DEFAULT 0,
    at_fra              BOOLEAN NOT NULL DEFAULT false,
    ss_benefits         NUMERIC NOT NULL DEFAULT 0,
    other_income        NUMERIC NOT NULL DEFAULT 0,
    tax_exempt_interest NUMERIC NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_clients_user ON clients(user_id);

-- saved calculation scenarios per client (what-if history)
CREATE TABLE IF NOT EXISTS scenarios (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id     UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    tax_year      INT NOT NULL,
    label         TEXT DEFAULT '',
    inputs_json   JSONB NOT NULL,
    results_json  JSONB NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_scenarios_client ON scenarios(client_id);

-- versioned tax rules (the business moat)
CREATE TABLE IF NOT EXISTS tax_rules (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tax_year       INT NOT NULL,
    jurisdiction   TEXT NOT NULL,
    rules_json     JSONB NOT NULL,
    effective_from DATE,
    UNIQUE (tax_year, jurisdiction)
);
