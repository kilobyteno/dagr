-- UUIDv7 helper for Postgres 16 (native uuidv7() arrives in Postgres 18).
-- Replaces the first 48 bits of a UUIDv4 with Unix time in milliseconds and sets version 7.
CREATE OR REPLACE FUNCTION uuidv7() RETURNS uuid
LANGUAGE sql
VOLATILE
PARALLEL SAFE
AS $$
  SELECT encode(
    set_bit(
      set_bit(
        overlay(
          uuid_send(gen_random_uuid())
          placing substring(int8send((extract(epoch from clock_timestamp()) * 1000)::bigint) from 3)
          from 1 for 6
        ),
        52,
        1
      ),
      53,
      1
    ),
    'hex'
  )::uuid;
$$;

ALTER TABLE users ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE sessions ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE workspaces ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE channels ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE workspace_domains ALTER COLUMN id SET DEFAULT uuidv7();
