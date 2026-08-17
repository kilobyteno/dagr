CREATE TABLE workspace_billing (
	workspace_id UUID PRIMARY KEY REFERENCES workspaces (id) ON DELETE CASCADE,
	plan TEXT NOT NULL DEFAULT 'free' CHECK (plan IN ('free', 'pro')),
	status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'past_due', 'cancelled')),
	billing_interval TEXT CHECK (billing_interval IN ('monthly', 'yearly')),
	billable_seats INTEGER NOT NULL DEFAULT 1 CHECK (billable_seats >= 1),
	current_period_end TIMESTAMPTZ,
	cancel_at_period_end BOOLEAN NOT NULL DEFAULT false,
	early_access_ends_at TIMESTAMPTZ,
	mollie_customer_id TEXT,
	mollie_mandate_id TEXT,
	mollie_subscription_id TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX workspace_billing_plan_status_idx
	ON workspace_billing (plan, status);

CREATE TABLE billing_events (
	id UUID PRIMARY KEY DEFAULT uuidv7(),
	workspace_id UUID REFERENCES workspaces (id) ON DELETE SET NULL,
	mollie_payment_id TEXT,
	event_type TEXT NOT NULL,
	payload JSONB,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX billing_events_mollie_payment_uidx
	ON billing_events (mollie_payment_id)
	WHERE mollie_payment_id IS NOT NULL;

CREATE INDEX billing_events_workspace_id_idx ON billing_events (workspace_id);

INSERT INTO workspace_billing (workspace_id, plan, status, billable_seats)
SELECT w.id, 'free', 'active', GREATEST(1, (
	SELECT COUNT(*)::int
	FROM workspace_members m
	WHERE m.workspace_id = w.id AND m.kind = 'member'
))
FROM workspaces w
ON CONFLICT (workspace_id) DO NOTHING;
