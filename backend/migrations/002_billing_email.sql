-- 002_billing_email.sql — email verification + subscription billing.
-- Adds account state (verified email, plan, Dodo customer id), a table of
-- single-use email-verification tokens (hashed at rest), and a subscriptions
-- table that Dodo webhooks keep current as the source of truth for entitlement.

-- --- account state on the existing users table ---
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS email_verified    BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS plan              TEXT    NOT NULL DEFAULT 'free',
    ADD COLUMN IF NOT EXISTS dodo_customer_id  TEXT    NOT NULL DEFAULT '';

-- --- email verification tokens (token is stored only as a SHA-256 hash) ---
CREATE TABLE IF NOT EXISTS email_verification_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_evt_user ON email_verification_tokens(user_id);

-- --- subscriptions (Dodo webhooks are the source of truth) ---
CREATE TABLE IF NOT EXISTS subscriptions (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    dodo_subscription_id TEXT NOT NULL UNIQUE,
    status               TEXT NOT NULL,               -- active | on_hold | failed | cancelled | ...
    current_period_end   TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_subs_user ON subscriptions(user_id);
