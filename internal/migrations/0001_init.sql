-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE coach (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT UNIQUE NOT NULL,
  name TEXT,
  tz TEXT NOT NULL DEFAULT 'Europe/Berlin',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE session_store (
  token TEXT PRIMARY KEY,
  data BYTEA NOT NULL,
  expiry TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS session_store;
DROP TABLE IF EXISTS coach;
