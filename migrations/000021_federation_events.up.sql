CREATE TABLE IF NOT EXISTS federated_event_refs (
	origin_server_id TEXT NOT NULL,
	origin_event_id TEXT NOT NULL,
	kind TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (origin_server_id, origin_event_id)
);
