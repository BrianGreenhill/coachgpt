-- internal/migrations/0002_athlete.sql
-- +goose Up
CREATE TABLE IF NOT EXISTS athlete (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  coach_id             UUID NOT NULL REFERENCES coach(id) ON DELETE CASCADE,
  name                 TEXT NOT NULL,
  email                TEXT,                                  -- optional
  tz                   TEXT NOT NULL DEFAULT 'Europe/Berlin',
  strava_athlete_id    BIGINT,
  strava_access_token  TEXT,
  strava_refresh_token TEXT,
  strava_token_expiry  TIMESTAMPTZ,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Prevent duplicates, but allow NULLs
CREATE UNIQUE INDEX IF NOT EXISTS uniq_athlete_email
  ON athlete (email) WHERE email IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_athlete_strava_id
  ON athlete (strava_athlete_id) WHERE strava_athlete_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS uniq_athlete_strava_id;
DROP INDEX IF EXISTS uniq_athlete_email;
DROP TABLE IF EXISTS athlete;
